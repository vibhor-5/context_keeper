/**
 * MCP Server implementation
 * Provides Model Context Protocol server functionality
 */

import express, { Express, Request, Response } from 'express';
import cors from 'cors';
import { Server } from 'http';
import { MCPServerConfig } from '../types/config';
import { MCPRequest, MCPResponse, MCPError } from '../types/mcp';
import { GoBackendClient } from '../services/goBackendClient';
import { ContextKeeperResourceProvider } from './resourceProvider';
import { ContextKeeperToolProvider } from './toolProvider';
import { DemoService, createDemoService } from '../demo/demoService';
import { logger } from '../utils/logger';

export class MCPServer {
  private config: MCPServerConfig;
  private app: Express;
  private server: Server | null = null;
  private isRunning: boolean = false;
  private goBackendClient: GoBackendClient;
  private resourceProvider: ContextKeeperResourceProvider;
  private toolProvider: ContextKeeperToolProvider;
  private demoService: DemoService;

  constructor(config: MCPServerConfig) {
    this.config = config;
    this.app = express();
    
    // Initialize Go backend client
    this.goBackendClient = new GoBackendClient({
      baseUrl: this.config.goBackendUrl,
      timeout: this.config.timeout,
      retryAttempts: 3,
      retryDelay: 1000
    });
    
    // Initialize demo service
    this.demoService = createDemoService({
      enabled: false, // Will be configured via environment
      fallbackOnError: true,
      predictableResponses: true,
      fallbackTimeout: this.config.timeout,
      maxFallbackAttempts: 2
    }, this.goBackendClient);
    
    // Initialize resource provider
    this.resourceProvider = new ContextKeeperResourceProvider(this.goBackendClient);
    
    // Initialize tool provider with demo service
    this.toolProvider = new ContextKeeperToolProvider(this.goBackendClient, this.demoService);
    
    this.setupMiddleware();
    this.setupRoutes();
  }

  private setupMiddleware(): void {
    // CORS configuration
    this.app.use(cors({
      origin: this.config.corsOrigins,
      methods: ['GET', 'POST', 'OPTIONS'],
      allowedHeaders: ['Content-Type', 'Authorization'],
      credentials: true
    }));

    // JSON parsing with size limit
    this.app.use(express.json({ limit: '10mb' }));

    // Request logging
    this.app.use((req, res, next) => {
      logger.debug('MCP HTTP request', {
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
        version: '1.0.0'
      });
    });

    // Main MCP endpoint
    this.app.post('/mcp', async (req: Request, res: Response) => {
      try {
        const mcpRequest: MCPRequest = req.body;
        
        // Validate MCP request format
        if (!mcpRequest.method) {
          const error: MCPError = {
            code: -32600,
            message: 'Invalid Request',
            data: { details: 'Missing required field: method' }
          };
          return res.status(400).json({ error, id: mcpRequest.id });
        }

        const response = await this.handleRequest(mcpRequest);
        res.json(response);
        
      } catch (error) {
        logger.error('Error processing MCP request', {}, error as Error);
        
        const mcpError: MCPError = {
          code: -32603,
          message: 'Internal error',
          data: { details: (error as Error).message }
        };
        
        res.status(500).json({ error: mcpError });
      }
    });

    // Handle unsupported methods
    this.app.all('*', (req: Request, res: Response) => {
      res.status(404).json({
        error: {
          code: -32601,
          message: 'Method not found',
          data: { details: `Endpoint ${req.path} not found` }
        }
      });
    });
  }

  async start(): Promise<void> {
    if (this.isRunning) {
      logger.warn('MCP Server is already running');
      return;
    }

    try {
      logger.info('Starting MCP Server...', {
        port: this.config.port,
        goBackendUrl: this.config.goBackendUrl
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
          logger.error('MCP Server error', {}, error);
          reject(error);
        });
      });
      
      this.isRunning = true;
      logger.info('MCP Server started successfully', { port: this.config.port });
      
    } catch (error) {
      logger.error('Failed to start MCP Server', {}, error as Error);
      throw error;
    }
  }

  async stop(): Promise<void> {
    if (!this.isRunning) {
      logger.warn('MCP Server is not running');
      return;
    }

    try {
      logger.info('Stopping MCP Server...');
      
      if (this.server) {
        // Set a timeout for graceful shutdown
        const shutdownTimeout = 10000; // 10 seconds
        
        await new Promise<void>((resolve, reject) => {
          const timeout = setTimeout(() => {
            logger.warn('MCP Server shutdown timeout, forcing close');
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
      logger.info('MCP Server stopped successfully');
      
    } catch (error) {
      logger.error('Failed to stop MCP Server', {}, error as Error);
      throw error;
    }
  }

  async handleRequest(request: MCPRequest): Promise<MCPResponse> {
    logger.debug('Handling MCP request', { method: request.method });
    
    try {
      // Route MCP methods to appropriate handlers
      switch (request.method) {
        case 'list_resources':
          return await this.handleListResources(request);
        
        case 'list_tools':
          return await this.handleListTools(request);
        
        case 'call_tool':
          return await this.handleCallTool(request);
        
        case 'get_resource':
          return await this.handleGetResource(request);
        
        default:
          return {
            error: {
              code: -32601,
              message: 'Method not found',
              data: { details: `Unknown method: ${request.method}` }
            },
            id: request.id
          };
      }
      
    } catch (error) {
      logger.error('Error handling MCP request', { method: request.method }, error as Error);
      
      return {
        error: {
          code: -32603,
          message: 'Internal error',
          data: { details: (error as Error).message }
        },
        id: request.id
      };
    }
  }

  private async handleListResources(request: MCPRequest): Promise<MCPResponse> {
    logger.debug('Handling list_resources request');
    
    try {
      const resources = await this.resourceProvider.listResources();
      
      return {
        result: {
          resources
        },
        id: request.id
      };
    } catch (error) {
      logger.error('Failed to list resources', {}, error as Error);
      
      return {
        error: {
          code: -32603,
          message: 'Internal error',
          data: { details: `Failed to list resources: ${(error as Error).message}` }
        },
        id: request.id
      };
    }
  }

  private async handleListTools(request: MCPRequest): Promise<MCPResponse> {
    logger.debug('Handling list_tools request');
    
    try {
      const tools = await this.toolProvider.listTools();
      
      return {
        result: {
          tools
        },
        id: request.id
      };
    } catch (error) {
      logger.error('Failed to list tools', {}, error as Error);
      
      return {
        error: {
          code: -32603,
          message: 'Internal error',
          data: { details: `Failed to list tools: ${(error as Error).message}` }
        },
        id: request.id
      };
    }
  }

  private async handleCallTool(request: MCPRequest): Promise<MCPResponse> {
    logger.debug('Handling call_tool request', { toolName: request.params?.name });
    
    try {
      if (!request.params?.name) {
        return {
          error: {
            code: -32602,
            message: 'Invalid params',
            data: { details: 'Missing required parameter: name' }
          },
          id: request.id
        };
      }

      const toolName = request.params.name;
      const toolArgs = request.params.arguments || {};

      const result = await this.toolProvider.callTool(toolName, toolArgs);
      
      return {
        result,
        id: request.id
      };
    } catch (error) {
      logger.error('Failed to call tool', { toolName: request.params?.name }, error as Error);
      
      return {
        error: {
          code: -32603,
          message: 'Internal error',
          data: { details: `Failed to call tool: ${(error as Error).message}` }
        },
        id: request.id
      };
    }
  }

  private async handleGetResource(request: MCPRequest): Promise<MCPResponse> {
    logger.debug('Handling get_resource request', { uri: request.params?.uri });
    
    try {
      if (!request.params?.uri) {
        return {
          error: {
            code: -32602,
            message: 'Invalid params',
            data: { details: 'Missing required parameter: uri' }
          },
          id: request.id
        };
      }

      const resourceContent = await this.resourceProvider.getResource(request.params.uri);
      
      return {
        result: {
          contents: [resourceContent]
        },
        id: request.id
      };
    } catch (error) {
      logger.error('Failed to get resource', { uri: request.params?.uri }, error as Error);
      
      return {
        error: {
          code: -32603,
          message: 'Internal error',
          data: { details: `Failed to get resource: ${(error as Error).message}` }
        },
        id: request.id
      };
    }
  }

  isHealthy(): boolean {
    return this.isRunning;
  }

  getPort(): number {
    return this.config.port;
  }

  /**
   * Configure demo mode
   */
  configureDemoMode(enabled: boolean, predictableResponses: boolean = true): void {
    this.demoService.updateConfig({
      enabled,
      predictableResponses,
      fallbackOnError: !enabled // Disable fallback when demo mode is explicitly enabled
    });
    
    logger.info('Demo mode configured', { enabled, predictableResponses });
  }

  /**
   * Get demo service (for testing and configuration)
   */
  getDemoService(): DemoService {
    return this.demoService;
  }
}