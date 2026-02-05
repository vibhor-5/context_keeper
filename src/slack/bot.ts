/**
 * Slack Bot HTTP Server
 * Handles Slack slash commands and integrates with MCP server
 */

import express, { Express, Request, Response } from 'express';
import { createHmac, timingSafeEqual } from 'crypto';
import { Server } from 'http';
import { SlackBotConfig } from '../types/config';
import { SlackBot, SlackCommand, SlackResponse } from '../types/slack';
import { SlackMCPClient } from './mcpClient';
import { logger } from '../utils/logger';
import { 
  executeWithRetry, 
  executeWithTimeout, 
  CircuitBreaker,
  createSlackErrorMessage,
  classifyError,
  DEFAULT_RETRY_CONFIG,
  DEFAULT_TIMEOUT_CONFIG
} from '../utils/errorHandling';

export class ContextKeeperSlackBot implements SlackBot {
  private config: SlackBotConfig;
  private app: Express;
  private server: Server | null = null;
  private isRunning: boolean = false;
  private mcpClient: SlackMCPClient;
  private mcpCircuitBreaker: CircuitBreaker;

  constructor(config: SlackBotConfig) {
    this.config = config;
    this.app = express();
    
    // Initialize MCP client
    this.mcpClient = new SlackMCPClient({
      mcpServerUrl: config.mcpServerUrl,
      timeout: DEFAULT_TIMEOUT_CONFIG.operation,
      retryAttempts: config.retryAttempts,
      retryDelay: config.retryDelay
    });
    
    // Initialize circuit breaker for MCP client
    this.mcpCircuitBreaker = new CircuitBreaker({
      failureThreshold: 5,
      resetTimeout: 60000,
      monitoringPeriod: 300000
    }, 'MCP Server');
    
    this.setupMiddleware();
    this.setupRoutes();
  }

  private setupMiddleware(): void {
    // Raw body parser for Slack signature verification
    this.app.use('/slack', express.raw({ type: 'application/x-www-form-urlencoded' }));
    
    // JSON parsing for other endpoints
    this.app.use(express.json({ limit: '10mb' }));
    
    // URL encoded parsing for form data
    this.app.use(express.urlencoded({ extended: true }));

    // Request logging
    this.app.use((req, res, next) => {
      logger.debug('Slack Bot HTTP request', {
        method: req.method,
        path: req.path,
        userAgent: req.get('User-Agent')
      });
      next();
    });
  }

  private setupRoutes(): void {
    // Health check endpoint
    this.app.get('/health', (req: Request, res: Response) => {
      res.json({
        status: 'healthy',
        timestamp: new Date().toISOString(),
        version: '1.0.0',
        service: 'slack-bot'
      });
    });

    // Slack slash commands endpoint
    this.app.post('/slack/commands', async (req: Request, res: Response) => {
      try {
        // Verify Slack signature
        if (!this.verifySlackSignature(req)) {
          logger.warn('Invalid Slack signature', {
            headers: req.headers,
            timestamp: req.headers['x-slack-request-timestamp']
          });
          return res.status(401).json({ error: 'Unauthorized' });
        }

        // Parse the form data
        const body = new URLSearchParams(req.body.toString());
        const slackCommand: SlackCommand = {
          command: body.get('command') || '',
          text: body.get('text') || '',
          userId: body.get('user_id') || '',
          channelId: body.get('channel_id') || '',
          teamId: body.get('team_id') || '',
          userName: body.get('user_name') || '',
          responseUrl: body.get('response_url') || ''
        };

        logger.info('Processing Slack command', {
          command: slackCommand.command,
          userId: slackCommand.userId,
          channelId: slackCommand.channelId,
          textLength: slackCommand.text.length
        });

        // Handle the command with timeout and retry logic
        const response = await executeWithTimeout(
          () => executeWithRetry(
            () => this.handleCommand(slackCommand),
            {
              maxAttempts: 2, // Limited retries for user-facing commands
              baseDelay: 1000,
              maxDelay: 5000
            },
            `Slack command ${slackCommand.command}`
          ),
          DEFAULT_TIMEOUT_CONFIG.operation,
          `Slack command ${slackCommand.command}`
        );
        
        // Send response
        res.json(response);

      } catch (error) {
        logger.error('Error processing Slack command', {}, error as Error);
        
        const errorMessage = createSlackErrorMessage(error, 'command processing');
        const errorResponse: SlackResponse = {
          text: `${errorMessage}\n\nPlease try again later or contact your team administrator if the problem persists.`,
          responseType: 'ephemeral'
        };
        
        res.status(500).json(errorResponse);
      }
    });

    // Slack interactive components endpoint (for buttons, menus, etc.)
    this.app.post('/slack/interactive', async (req: Request, res: Response) => {
      try {
        // Verify Slack signature
        if (!this.verifySlackSignature(req)) {
          logger.warn('Invalid Slack signature for interactive component');
          return res.status(401).json({ error: 'Unauthorized' });
        }

        // Parse the payload
        const body = new URLSearchParams(req.body.toString());
        const payload = JSON.parse(body.get('payload') || '{}');

        logger.info('Processing Slack interactive component', {
          type: payload.type,
          userId: payload.user?.id,
          actionId: payload.actions?.[0]?.action_id
        });

        // For now, just acknowledge the interaction
        res.json({ text: 'Interaction received' });

      } catch (error) {
        logger.error('Error processing Slack interactive component', {}, error as Error);
        res.status(500).json({ error: 'Internal server error' });
      }
    });

    // Handle unsupported endpoints
    this.app.all('*', (req: Request, res: Response) => {
      res.status(404).json({
        error: 'Not found',
        message: `Endpoint ${req.path} not found`
      });
    });
  }

  async start(): Promise<void> {
    if (this.isRunning) {
      logger.warn('Slack Bot is already running');
      return;
    }

    try {
      logger.info('Starting Slack Bot...', {
        port: this.config.port,
        mcpServerUrl: this.config.mcpServerUrl
      });

      await new Promise<void>((resolve, reject) => {
        this.server = this.app.listen(this.config.port, (error?: Error) => {
          if (error) {
            reject(error);
          } else {
            resolve();
          }
        });

        this.server.on('error', (error: Error) => {
          logger.error('Slack Bot server error', {}, error);
          reject(error);
        });
      });
      
      this.isRunning = true;
      logger.info('Slack Bot started successfully', { port: this.config.port });
      
    } catch (error) {
      logger.error('Failed to start Slack Bot', {}, error as Error);
      throw error;
    }
  }

  async stop(): Promise<void> {
    if (!this.isRunning) {
      logger.warn('Slack Bot is not running');
      return;
    }

    try {
      logger.info('Stopping Slack Bot...');
      
      if (this.server) {
        // Set a timeout for graceful shutdown
        const shutdownTimeout = 15000; // 15 seconds to allow Slack commands to complete
        
        await new Promise<void>((resolve, reject) => {
          const timeout = setTimeout(() => {
            logger.warn('Slack Bot shutdown timeout, forcing close');
            this.server?.close();
            resolve();
          }, shutdownTimeout);
          
          this.server!.close((error?: Error) => {
            clearTimeout(timeout);
            if (error) {
              reject(error);
            } else {
              resolve();
            }
          });
        });
        
        this.server = null;
      }
      
      this.isRunning = false;
      logger.info('Slack Bot stopped successfully');
      
    } catch (error) {
      logger.error('Failed to stop Slack Bot', {}, error as Error);
      throw error;
    }
  }

  async handleCommand(command: SlackCommand): Promise<SlackResponse> {
    logger.debug('Handling Slack command', { 
      command: command.command,
      text: command.text.substring(0, 100)
    });

    try {
      // Route commands to appropriate handlers
      switch (command.command) {
        case '/context':
          return await this.handleContextCommand(command);
        
        case '/onboard':
          return await this.handleOnboardCommand(command);
        
        case '/recent':
          return await this.handleRecentCommand(command);
        
        case '/status':
          return await this.handleStatusCommand(command);
        
        default:
          return this.createHelpResponse(command.command);
      }

    } catch (error) {
      logger.error('Error handling command', { command: command.command }, error as Error);
      
      return {
        text: `⚠️ Error processing ${command.command}: ${(error as Error).message}`,
        responseType: 'ephemeral'
      };
    }
  }

  private async handleContextCommand(command: SlackCommand): Promise<SlackResponse> {
    try {
      logger.info('Processing /context command', {
        userId: command.userId,
        channelId: command.channelId,
        textLength: command.text.length
      });

      // Parse command parameters
      const text = command.text.trim();
      if (!text) {
        return {
          text: '❓ Please provide a query for context search.\n\n**Usage:** `/context <query> [repo:<repository>]`\n\n**Examples:**\n• `/context What are the main components?`\n• `/context How does authentication work? repo:my-app`',
          responseType: 'ephemeral'
        };
      }

      // Parse repository parameter if provided
      let query = text;
      let repositoryId: string | undefined;
      
      const repoMatch = text.match(/\brepo:(\S+)/);
      if (repoMatch) {
        repositoryId = repoMatch[1];
        query = text.replace(/\brepo:\S+/, '').trim();
      }

      // Use default repository if configured and none specified
      if (!repositoryId && this.config.defaultRepository) {
        repositoryId = this.config.defaultRepository;
      }

      if (!query) {
        return {
          text: '❓ Please provide a query after specifying the repository.\n\n**Usage:** `/context <query> [repo:<repository>]`',
          responseType: 'ephemeral'
        };
      }

      logger.debug('Parsed context command', {
        query: query.substring(0, 100),
        repositoryId,
        queryLength: query.length
      });

      // Query context via MCP client with circuit breaker
      const contextResult = await this.mcpCircuitBreaker.execute(async () => {
        return await executeWithTimeout(
          () => this.mcpClient.queryContext(query, repositoryId),
          DEFAULT_TIMEOUT_CONFIG.operation,
          'context query'
        );
      });

      // Format response for Slack
      let responseText = `🔍 **Context Search Results**\n\n${contextResult}`;
      
      // Add metadata
      const metadata = [];
      if (repositoryId) {
        metadata.push(`Repository: ${repositoryId}`);
      }
      metadata.push(`Query: "${query.substring(0, 50)}${query.length > 50 ? '...' : ''}"`);
      
      if (metadata.length > 0) {
        responseText += `\n\n_${metadata.join(' • ')}_`;
      }

      // Truncate if too long
      if (responseText.length > this.config.maxResponseLength) {
        const truncateLength = this.config.maxResponseLength - 100;
        responseText = responseText.substring(0, truncateLength) + '\n\n_[Response truncated due to length]_';
      }

      logger.info('Context command completed successfully', {
        userId: command.userId,
        repositoryId,
        responseLength: responseText.length
      });

      return {
        text: responseText,
        responseType: 'ephemeral'
      };

    } catch (error) {
      logger.error('Context command failed', {
        userId: command.userId,
        text: command.text
      }, error as Error);

      const errorMessage = createSlackErrorMessage(error, 'context search');
      
      return {
        text: `${errorMessage}\n\n**Usage:** \`/context <query> [repo:<repository>]\`\n\nNeed help? Contact your team administrator.`,
        responseType: 'ephemeral'
      };
    }
  }

  private async handleOnboardCommand(command: SlackCommand): Promise<SlackResponse> {
    try {
      logger.info('Processing /onboard command', {
        userId: command.userId,
        channelId: command.channelId,
        textLength: command.text.length
      });

      // Parse command parameters
      const text = command.text.trim();
      let repositoryId = text;

      // Use default repository if none specified
      if (!repositoryId && this.config.defaultRepository) {
        repositoryId = this.config.defaultRepository;
      }

      if (!repositoryId) {
        return {
          text: '❓ Please specify a repository for onboarding.\n\n**Usage:** `/onboard <repository>`\n\n**Examples:**\n• `/onboard my-app`\n• `/onboard frontend-service`',
          responseType: 'ephemeral'
        };
      }

      logger.debug('Parsed onboard command', {
        repositoryId,
        userId: command.userId
      });

      // Get onboarding summary via MCP client with circuit breaker
      const onboardingSummary = await this.mcpCircuitBreaker.execute(async () => {
        return await executeWithTimeout(
          () => this.mcpClient.getOnboardingSummary(repositoryId),
          DEFAULT_TIMEOUT_CONFIG.operation,
          'onboarding summary'
        );
      });

      // Format response for Slack
      let responseText = `👋 **Onboarding Guide for ${repositoryId}**\n\n${onboardingSummary}`;
      
      // Add metadata
      const metadata = [`Repository: ${repositoryId}`, `Generated for: <@${command.userId}>`];
      responseText += `\n\n_${metadata.join(' • ')}_`;

      // Truncate if too long
      if (responseText.length > this.config.maxResponseLength) {
        const truncateLength = this.config.maxResponseLength - 100;
        responseText = responseText.substring(0, truncateLength) + '\n\n_[Guide truncated due to length - contact your team for the full onboarding guide]_';
      }

      logger.info('Onboard command completed successfully', {
        userId: command.userId,
        repositoryId,
        responseLength: responseText.length
      });

      return {
        text: responseText,
        responseType: 'ephemeral'
      };

    } catch (error) {
      logger.error('Onboard command failed', {
        userId: command.userId,
        text: command.text
      }, error as Error);

      const errorMessage = createSlackErrorMessage(error, 'onboarding guide generation');

      return {
        text: `${errorMessage}\n\n**Usage:** \`/onboard <repository>\`\n\nNeed help? Contact your team administrator.`,
        responseType: 'ephemeral'
      };
    }
  }

  private async handleRecentCommand(command: SlackCommand): Promise<SlackResponse> {
    try {
      logger.info('Processing /recent command', {
        userId: command.userId,
        channelId: command.channelId,
        textLength: command.text.length
      });

      // Parse command parameters
      const text = command.text.trim();
      let repositoryId: string | undefined;
      let days = 7; // Default to 7 days

      if (text) {
        // Parse parameters: can be "repo-name" or "repo-name days:5" or "days:5"
        const parts = text.split(/\s+/);
        
        for (const part of parts) {
          if (part.startsWith('days:')) {
            const daysValue = parseInt(part.split(':')[1]);
            if (!isNaN(daysValue) && daysValue > 0 && daysValue <= 30) {
              days = daysValue;
            }
          } else if (!repositoryId && !part.includes(':')) {
            // First non-parameter part is repository name
            repositoryId = part;
          }
        }
      }

      // Use default repository if none specified
      if (!repositoryId && this.config.defaultRepository) {
        repositoryId = this.config.defaultRepository;
      }

      if (!repositoryId) {
        return {
          text: '❓ Please specify a repository to view recent activity.\n\n**Usage:** `/recent <repository> [days:<number>]`\n\n**Examples:**\n• `/recent my-app`\n• `/recent frontend-service days:14`\n• `/recent my-app days:3`',
          responseType: 'ephemeral'
        };
      }

      logger.debug('Parsed recent command', {
        repositoryId,
        days,
        userId: command.userId
      });

      // Get recent activity via MCP client with circuit breaker
      const recentActivity = await this.mcpCircuitBreaker.execute(async () => {
        return await executeWithTimeout(
          () => this.mcpClient.getRecentActivity(repositoryId!, days),
          DEFAULT_TIMEOUT_CONFIG.operation,
          'recent activity query'
        );
      });

      // Format response for Slack
      let responseText = `📅 **Recent Activity for ${repositoryId}** (last ${days} days)\n\n${recentActivity}`;
      
      // Add metadata
      const metadata = [`Repository: ${repositoryId}`, `Period: ${days} days`];
      responseText += `\n\n_${metadata.join(' • ')}_`;

      // Truncate if too long
      if (responseText.length > this.config.maxResponseLength) {
        const truncateLength = this.config.maxResponseLength - 100;
        responseText = responseText.substring(0, truncateLength) + '\n\n_[Activity list truncated due to length - use a shorter time period for more details]_';
      }

      logger.info('Recent command completed successfully', {
        userId: command.userId,
        repositoryId,
        days,
        responseLength: responseText.length
      });

      return {
        text: responseText,
        responseType: 'ephemeral'
      };

    } catch (error) {
      logger.error('Recent command failed', {
        userId: command.userId,
        text: command.text
      }, error as Error);

      const errorMessage = createSlackErrorMessage(error, 'recent activity retrieval');

      return {
        text: `${errorMessage}\n\n**Usage:** \`/recent <repository> [days:<number>]\`\n\nNeed help? Contact your team administrator.`,
        responseType: 'ephemeral'
      };
    }
  }

  private async handleStatusCommand(command: SlackCommand): Promise<SlackResponse> {
    try {
      logger.info('Processing /status command', {
        userId: command.userId,
        channelId: command.channelId
      });

      // Get system status via MCP client with circuit breaker
      const systemStatus = await this.mcpCircuitBreaker.execute(async () => {
        return await executeWithTimeout(
          () => this.mcpClient.getSystemStatus(),
          DEFAULT_TIMEOUT_CONFIG.operation,
          'system status query'
        );
      });

      // Format response for Slack
      let responseText = '📊 **System Status**\n\n';

      // MCP Server Status
      const mcpStatusEmoji = systemStatus.mcpServer.status === 'healthy' ? '✅' : 
                            systemStatus.mcpServer.status === 'degraded' ? '⚠️' : '❌';
      responseText += `${mcpStatusEmoji} **MCP Server**: ${systemStatus.mcpServer.status}\n`;
      responseText += `   • Version: ${systemStatus.mcpServer.version}\n`;
      if (systemStatus.mcpServer.uptime > 0) {
        responseText += `   • Uptime: ${Math.floor(systemStatus.mcpServer.uptime / 3600)}h ${Math.floor((systemStatus.mcpServer.uptime % 3600) / 60)}m\n`;
      }

      // Go Backend Status
      const backendStatusEmoji = systemStatus.goBackend.status === 'healthy' ? '✅' : 
                                systemStatus.goBackend.status === 'degraded' ? '⚠️' : '❌';
      responseText += `\n${backendStatusEmoji} **Context Backend**: ${systemStatus.goBackend.status}\n`;
      if (systemStatus.goBackend.responseTime > 0) {
        responseText += `   • Response Time: ${systemStatus.goBackend.responseTime}ms\n`;
      }
      responseText += `   • Last Check: ${new Date(systemStatus.goBackend.lastCheck).toLocaleTimeString()}\n`;

      // Repository Status
      responseText += `\n📚 **Repositories**\n`;
      responseText += `   • Total: ${systemStatus.repositories.total}\n`;
      responseText += `   • Active: ${systemStatus.repositories.active}\n`;
      if (systemStatus.repositories.ingesting > 0) {
        responseText += `   • Ingesting: ${systemStatus.repositories.ingesting}\n`;
      }
      if (systemStatus.repositories.error > 0) {
        responseText += `   • Errors: ${systemStatus.repositories.error}\n`;
      }

      // Overall system health
      const overallHealthy = systemStatus.mcpServer.status === 'healthy' && 
                           systemStatus.goBackend.status === 'healthy';
      const overallDegraded = systemStatus.mcpServer.status === 'degraded' || 
                            systemStatus.goBackend.status === 'degraded';
      
      let overallStatus = '❌ System Down';
      if (overallHealthy) {
        overallStatus = '✅ All Systems Operational';
      } else if (overallDegraded) {
        overallStatus = '⚠️ Degraded Performance';
      }

      responseText += `\n**Overall Status**: ${overallStatus}`;

      // Add circuit breaker status
      const circuitState = this.mcpCircuitBreaker.getState();
      if (circuitState.state !== 'closed') {
        responseText += `\n\n⚡ **Circuit Breaker**: ${circuitState.state} (${circuitState.failures} failures)`;
      }

      // Add timestamp
      responseText += `\n\n_Status checked at ${new Date().toLocaleTimeString()}_`;

      logger.info('Status command completed successfully', {
        userId: command.userId,
        mcpStatus: systemStatus.mcpServer.status,
        backendStatus: systemStatus.goBackend.status,
        totalRepos: systemStatus.repositories.total,
        circuitState: circuitState.state
      });

      return {
        text: responseText,
        responseType: 'ephemeral'
      };

    } catch (error) {
      logger.error('Status command failed', {
        userId: command.userId
      }, error as Error);

      const errorMessage = createSlackErrorMessage(error, 'system status check');
      const circuitState = this.mcpCircuitBreaker.getState();

      let statusText = `${errorMessage}\n\n`;
      
      if (circuitState.state === 'open') {
        statusText += '❌ **System Status: Circuit Breaker Open**\n\nThe system is currently protecting itself from cascading failures. Services will be retried automatically.';
      } else {
        statusText += '⚠️ **System Status: Degraded**\n\nUnable to retrieve complete system status at this time.';
      }

      return {
        text: `${statusText}\n\n_Status checked at ${new Date().toLocaleTimeString()}_\n\nNeed help? Contact your team administrator.`,
        responseType: 'ephemeral'
      };
    }
  }

  private createHelpResponse(command: string): SlackResponse {
    const helpText = `
❓ **Unknown command: ${command}**

**Available commands:**
• \`/context <query>\` - Query repository context with natural language
• \`/onboard [repository]\` - Get onboarding summary for a repository
• \`/recent [repository] [days]\` - Show recent activity for a repository
• \`/status\` - Check system status and health

**Examples:**
• \`/context What are the main components of this project?\`
• \`/onboard my-repo\`
• \`/recent 7\`
• \`/status\`

Need help? Contact your team administrator.
    `.trim();

    return {
      text: helpText,
      responseType: 'ephemeral'
    };
  }

  private verifySlackSignature(req: Request): boolean {
    try {
      const signature = req.headers['x-slack-signature'] as string;
      const timestamp = req.headers['x-slack-request-timestamp'] as string;
      const body = req.body;

      if (!signature || !timestamp) {
        logger.warn('Missing Slack signature or timestamp');
        return false;
      }

      // Check if request is too old (replay attack protection)
      const currentTime = Math.floor(Date.now() / 1000);
      const requestTime = parseInt(timestamp);
      
      if (Math.abs(currentTime - requestTime) > 300) { // 5 minutes
        logger.warn('Slack request too old', { 
          currentTime, 
          requestTime, 
          diff: Math.abs(currentTime - requestTime) 
        });
        return false;
      }

      // Create expected signature
      const sigBasestring = `v0:${timestamp}:${body}`;
      const expectedSignature = `v0=${createHmac('sha256', this.config.signingSecret)
        .update(sigBasestring)
        .digest('hex')}`;

      // Compare signatures using timing-safe comparison
      const signatureBuffer = Buffer.from(signature);
      const expectedBuffer = Buffer.from(expectedSignature);

      if (signatureBuffer.length !== expectedBuffer.length) {
        logger.warn('Slack signature length mismatch');
        return false;
      }

      return timingSafeEqual(signatureBuffer, expectedBuffer);

    } catch (error) {
      logger.error('Error verifying Slack signature', {}, error as Error);
      return false;
    }
  }

  isHealthy(): boolean {
    return this.isRunning;
  }

  getPort(): number {
    return this.config.port;
  }

  getApp(): Express {
    return this.app;
  }
}