/**
 * Main entry point for ContextKeeper MCP + Slack Bot system
 * Starts both MCP server and Slack bot components with graceful restart support
 */

import { MCPServer } from './mcp/server';
import { ContextKeeperSlackBot } from './slack/bot';
import { loadConfig } from './config/config';
import { logger } from './utils/logger';
import { SystemConfig } from './types/config';
import express, { Express, Request, Response } from 'express';
import { Server } from 'http';

interface SystemComponents {
  mcpServer: MCPServer;
  slackBot: ContextKeeperSlackBot;
  config: SystemConfig;
}

interface SystemHealth {
  status: 'healthy' | 'degraded' | 'unhealthy';
  timestamp: string;
  version: string;
  uptime: number;
  components: {
    mcpServer: {
      status: 'healthy' | 'unhealthy';
      port: number;
    };
    slackBot: {
      status: 'healthy' | 'unhealthy';
      port: number;
    };
  };
}

let currentComponents: SystemComponents | null = null;
let isRestarting = false;
let restartPromise: Promise<void> | null = null;
let healthServer: Server | null = null;
let systemStartTime = Date.now();

async function startHealthServer(port: number = 8080): Promise<void> {
  const app: Express = express();
  
  app.use(express.json());
  
  // System health endpoint
  app.get('/health', (req: Request, res: Response) => {
    const health = getSystemHealth();
    const statusCode = health.status === 'healthy' ? 200 : 
                      health.status === 'degraded' ? 200 : 503;
    
    res.status(statusCode).json(health);
  });
  
  // Detailed system status
  app.get('/status', (req: Request, res: Response) => {
    const health = getSystemHealth();
    const detailedStatus = {
      ...health,
      restart: {
        isRestarting,
        lastRestart: systemStartTime
      },
      environment: {
        nodeVersion: process.version,
        platform: process.platform,
        arch: process.arch,
        pid: process.pid
      }
    };
    
    res.json(detailedStatus);
  });
  
  // Graceful restart endpoint (for operational use)
  app.post('/restart', async (req: Request, res: Response) => {
    try {
      logger.info('Restart requested via HTTP endpoint');
      
      // Start restart in background
      gracefulRestart().catch(error => {
        logger.error('Background restart failed', {}, error as Error);
      });
      
      res.json({
        message: 'Graceful restart initiated',
        timestamp: new Date().toISOString()
      });
      
    } catch (error) {
      logger.error('Failed to initiate restart', {}, error as Error);
      res.status(500).json({
        error: 'Failed to initiate restart',
        message: (error as Error).message
      });
    }
  });
  
  // Ready endpoint for container orchestration
  app.get('/ready', (req: Request, res: Response) => {
    const isReady = currentComponents && 
                   currentComponents.mcpServer.isHealthy() && 
                   currentComponents.slackBot.isHealthy() &&
                   !isRestarting;
    
    if (isReady) {
      res.json({ status: 'ready', timestamp: new Date().toISOString() });
    } else {
      res.status(503).json({ 
        status: 'not ready', 
        timestamp: new Date().toISOString(),
        reason: isRestarting ? 'restarting' : 'components not healthy'
      });
    }
  });
  
  await new Promise<void>((resolve, reject) => {
    healthServer = app.listen(port, (error?: Error) => {
      if (error) {
        reject(error);
      } else {
        logger.info(`Health server started on port ${port}`);
        resolve();
      }
    });
  });
}

async function stopHealthServer(): Promise<void> {
  if (healthServer) {
    await new Promise<void>((resolve, reject) => {
      healthServer!.close((error?: Error) => {
        if (error) {
          reject(error);
        } else {
          resolve();
        }
      });
    });
    healthServer = null;
    logger.info('Health server stopped');
  }
}

function getSystemHealth(): SystemHealth {
  const uptime = Math.floor((Date.now() - systemStartTime) / 1000);
  
  let mcpStatus: 'healthy' | 'unhealthy' = 'unhealthy';
  let slackStatus: 'healthy' | 'unhealthy' = 'unhealthy';
  let mcpPort = 0;
  let slackPort = 0;
  
  if (currentComponents) {
    mcpStatus = currentComponents.mcpServer.isHealthy() ? 'healthy' : 'unhealthy';
    slackStatus = currentComponents.slackBot.isHealthy() ? 'healthy' : 'unhealthy';
    mcpPort = currentComponents.mcpServer.getPort();
    slackPort = currentComponents.slackBot.getPort();
  }
  
  let overallStatus: 'healthy' | 'degraded' | 'unhealthy' = 'unhealthy';
  if (mcpStatus === 'healthy' && slackStatus === 'healthy' && !isRestarting) {
    overallStatus = 'healthy';
  } else if ((mcpStatus === 'healthy' || slackStatus === 'healthy') && !isRestarting) {
    overallStatus = 'degraded';
  }
  
  return {
    status: overallStatus,
    timestamp: new Date().toISOString(),
    version: '1.0.0',
    uptime,
    components: {
      mcpServer: {
        status: mcpStatus,
        port: mcpPort
      },
      slackBot: {
        status: slackStatus,
        port: slackPort
      }
    }
  };
}

async function startSystem(): Promise<SystemComponents> {
  logger.info('Starting ContextKeeper MCP + Slack Bot system...');
  
  const config = await loadConfig();
  
  // Initialize components
  const mcpServer = new MCPServer(config.mcp);
  const slackBot = new ContextKeeperSlackBot(config.slack);
  
  // Start services
  await mcpServer.start();
  logger.info(`MCP Server started on port ${config.mcp.port}`);
  
  await slackBot.start();
  logger.info('Slack Bot started successfully');
  
  return { mcpServer, slackBot, config };
}

async function stopSystem(components: SystemComponents): Promise<void> {
  logger.info('Stopping ContextKeeper system...');
  
  try {
    // Stop services in reverse order
    await components.slackBot.stop();
    logger.info('Slack Bot stopped');
    
    await components.mcpServer.stop();
    logger.info('MCP Server stopped');
    
  } catch (error) {
    logger.error('Error during system shutdown', {}, error as Error);
    throw error;
  }
}

async function gracefulRestart(): Promise<void> {
  if (isRestarting) {
    logger.info('Restart already in progress, waiting...');
    if (restartPromise) {
      await restartPromise;
    }
    return;
  }

  isRestarting = true;
  restartPromise = performRestart();
  
  try {
    await restartPromise;
  } finally {
    isRestarting = false;
    restartPromise = null;
  }
}

async function performRestart(): Promise<void> {
  logger.info('Starting graceful restart...');
  
  try {
    // Stop current system if running
    if (currentComponents) {
      await stopSystem(currentComponents);
      currentComponents = null;
    }
    
    // Start new system with fresh configuration
    currentComponents = await startSystem();
    systemStartTime = Date.now(); // Reset uptime counter
    
    logger.info('Graceful restart completed successfully');
    
  } catch (error) {
    logger.error('Failed during graceful restart', {}, error as Error);
    
    // If restart fails, try to start with previous configuration
    if (!currentComponents) {
      try {
        logger.info('Attempting to start system with default configuration...');
        currentComponents = await startSystem();
        systemStartTime = Date.now();
        logger.info('System started with fallback configuration');
      } catch (fallbackError) {
        logger.error('Failed to start system with fallback configuration', {}, fallbackError as Error);
        throw fallbackError;
      }
    }
    
    throw error;
  }
}

async function gracefulShutdown(signal: string): Promise<void> {
  logger.info(`Received ${signal}, shutting down gracefully...`);
  
  try {
    // If restart is in progress, wait for it to complete first
    if (isRestarting && restartPromise) {
      logger.info('Waiting for restart to complete before shutdown...');
      await restartPromise;
    }
    
    if (currentComponents) {
      await stopSystem(currentComponents);
      currentComponents = null;
    }
    
    await stopHealthServer();
    
    logger.info('Graceful shutdown completed');
    process.exit(0);
    
  } catch (error) {
    logger.error('Error during graceful shutdown', {}, error as Error);
    process.exit(1);
  }
}

async function main() {
  try {
    // Start the health server first
    const healthPort = parseInt(process.env.HEALTH_PORT || '8080');
    await startHealthServer(healthPort);
    
    // Start the main system
    currentComponents = await startSystem();
    
    // Set up signal handlers for graceful shutdown
    process.on('SIGINT', () => gracefulShutdown('SIGINT'));
    process.on('SIGTERM', () => gracefulShutdown('SIGTERM'));
    
    // Set up signal handler for graceful restart
    process.on('SIGUSR1', async () => {
      logger.info('Received SIGUSR1, performing graceful restart...');
      try {
        await gracefulRestart();
        logger.info('Graceful restart completed via SIGUSR1');
      } catch (error) {
        logger.error('Graceful restart failed via SIGUSR1', {}, error as Error);
      }
    });
    
    // Set up signal handler for configuration reload
    process.on('SIGUSR2', async () => {
      logger.info('Received SIGUSR2, reloading configuration...');
      try {
        await gracefulRestart();
        logger.info('Configuration reloaded via SIGUSR2');
      } catch (error) {
        logger.error('Configuration reload failed via SIGUSR2', {}, error as Error);
      }
    });
    
    logger.info('ContextKeeper system is running');
    logger.info(`Health endpoints available on port ${healthPort}`);
    logger.info('Send SIGUSR1 or SIGUSR2 to gracefully restart/reload configuration');
    logger.info('POST /restart endpoint available for operational restarts');
    
  } catch (error) {
    logger.error('Failed to start ContextKeeper system:', {}, error as Error);
    process.exit(1);
  }
}

if (require.main === module) {
  main();
}

// Export functions for testing and external control
export { 
  main, 
  gracefulRestart, 
  gracefulShutdown,
  startSystem,
  stopSystem,
  getSystemHealth
};