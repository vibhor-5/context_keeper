/**
 * MCP Client for Slack Bot
 * Handles communication with the MCP server using HTTP transport
 */

import axios, { AxiosInstance, AxiosError } from 'axios';
import { MCPClient, SystemStatus } from '../types/slack';
import { MCPRequest, MCPResponse } from '../types/mcp';
import { logger } from '../utils/logger';
import { 
  executeWithRetry, 
  executeWithTimeout, 
  classifyError,
  DEFAULT_RETRY_CONFIG,
  DEFAULT_TIMEOUT_CONFIG
} from '../utils/errorHandling';

export interface MCPClientConfig {
  mcpServerUrl: string;
  timeout: number;
  retryAttempts?: number;
  retryDelay?: number;
}

export class SlackMCPClient implements MCPClient {
  private client: AxiosInstance;
  private config: MCPClientConfig;

  constructor(config: MCPClientConfig) {
    this.config = {
      retryAttempts: DEFAULT_RETRY_CONFIG.maxAttempts,
      retryDelay: DEFAULT_RETRY_CONFIG.baseDelay,
      ...config
    };

    this.client = axios.create({
      baseURL: this.config.mcpServerUrl,
      timeout: this.config.timeout,
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json'
      }
    });

    // Add request interceptor for logging
    this.client.interceptors.request.use(
      (config) => {
        logger.debug('MCP Client request', {
          method: config.method?.toUpperCase(),
          url: config.url,
          baseURL: config.baseURL
        });
        return config;
      },
      (error) => {
        logger.error('MCP Client request error', {}, error);
        return Promise.reject(error);
      }
    );

    // Add response interceptor for logging and error handling
    this.client.interceptors.response.use(
      (response) => {
        logger.debug('MCP Client response', {
          status: response.status,
          url: response.config.url
        });
        return response;
      },
      (error: AxiosError) => {
        logger.error('MCP Client response error', {
          status: error.response?.status,
          url: error.config?.url,
          message: error.message
        });
        return Promise.reject(this.handleMCPError(error));
      }
    );
  }

  async queryContext(query: string, repositoryId?: string): Promise<string> {
    try {
      logger.info('Querying context via MCP', {
        query: query.substring(0, 100),
        repositoryId,
        queryLength: query.length
      });

      const mcpRequest: MCPRequest = {
        method: 'call_tool',
        params: {
          name: 'query_repository_context',
          arguments: {
            query,
            repositoryId,
            limit: 10
          }
        },
        id: this.generateRequestId()
      };

      const response = await executeWithRetry(
        () => this.client.post<MCPResponse>('/mcp', mcpRequest),
        {
          maxAttempts: this.config.retryAttempts || DEFAULT_RETRY_CONFIG.maxAttempts,
          baseDelay: this.config.retryDelay || DEFAULT_RETRY_CONFIG.baseDelay,
          maxDelay: DEFAULT_RETRY_CONFIG.maxDelay,
          backoffMultiplier: DEFAULT_RETRY_CONFIG.backoffMultiplier,
          jitter: true
        },
        'MCP context query'
      );

      const mcpResponse = response.data;

      if (mcpResponse.error) {
        throw new Error(`MCP Error: ${mcpResponse.error.message}`);
      }

      if (!mcpResponse.result || !mcpResponse.result.content) {
        throw new Error('Invalid MCP response: missing content');
      }

      // Extract text content from MCP tool result
      const content = mcpResponse.result.content;
      if (Array.isArray(content) && content.length > 0 && content[0].text) {
        const resultText = content[0].text;
        
        logger.info('Context query completed successfully', {
          repositoryId,
          responseLength: resultText.length,
          isError: mcpResponse.result.isError || false
        });

        return resultText;
      }

      throw new Error('Invalid MCP response: no text content found');

    } catch (error) {
      logger.error('Context query failed', { query, repositoryId }, error as Error);
      throw error;
    }
  }

  async getOnboardingSummary(repositoryId: string): Promise<string> {
    try {
      logger.info('Getting onboarding summary via MCP', { repositoryId });

      const mcpRequest: MCPRequest = {
        method: 'call_tool',
        params: {
          name: 'get_onboarding_summary',
          arguments: {
            repositoryId,
            focusAreas: ['overview', 'architecture', 'getting-started', 'contributing']
          }
        },
        id: this.generateRequestId()
      };

      const response = await executeWithRetry(
        () => this.client.post<MCPResponse>('/mcp', mcpRequest),
        {
          maxAttempts: this.config.retryAttempts || DEFAULT_RETRY_CONFIG.maxAttempts,
          baseDelay: this.config.retryDelay || DEFAULT_RETRY_CONFIG.baseDelay,
          maxDelay: DEFAULT_RETRY_CONFIG.maxDelay,
          backoffMultiplier: DEFAULT_RETRY_CONFIG.backoffMultiplier,
          jitter: true
        },
        'MCP onboarding summary'
      );

      const mcpResponse = response.data;

      if (mcpResponse.error) {
        throw new Error(`MCP Error: ${mcpResponse.error.message}`);
      }

      if (!mcpResponse.result || !mcpResponse.result.content) {
        throw new Error('Invalid MCP response: missing content');
      }

      // Extract text content from MCP tool result
      const content = mcpResponse.result.content;
      if (Array.isArray(content) && content.length > 0 && content[0].text) {
        const resultText = content[0].text;
        
        logger.info('Onboarding summary completed successfully', {
          repositoryId,
          responseLength: resultText.length,
          isError: mcpResponse.result.isError || false
        });

        return resultText;
      }

      throw new Error('Invalid MCP response: no text content found');

    } catch (error) {
      logger.error('Onboarding summary failed', { repositoryId }, error as Error);
      throw error;
    }
  }

  async getRecentActivity(repositoryId: string, days: number = 7): Promise<string> {
    try {
      logger.info('Getting recent activity via MCP', { repositoryId, days });

      // Use context query to get recent activity
      const query = `Show me the recent activity and changes in this repository from the last ${days} days`;
      
      const mcpRequest: MCPRequest = {
        method: 'call_tool',
        params: {
          name: 'query_repository_context',
          arguments: {
            query,
            repositoryId,
            limit: 20
          }
        },
        id: this.generateRequestId()
      };

      const response = await executeWithRetry(
        () => this.client.post<MCPResponse>('/mcp', mcpRequest),
        {
          maxAttempts: this.config.retryAttempts || DEFAULT_RETRY_CONFIG.maxAttempts,
          baseDelay: this.config.retryDelay || DEFAULT_RETRY_CONFIG.baseDelay,
          maxDelay: DEFAULT_RETRY_CONFIG.maxDelay,
          backoffMultiplier: DEFAULT_RETRY_CONFIG.backoffMultiplier,
          jitter: true
        },
        'MCP recent activity query'
      );

      const mcpResponse = response.data;

      if (mcpResponse.error) {
        throw new Error(`MCP Error: ${mcpResponse.error.message}`);
      }

      if (!mcpResponse.result || !mcpResponse.result.content) {
        throw new Error('Invalid MCP response: missing content');
      }

      // Extract text content from MCP tool result
      const content = mcpResponse.result.content;
      if (Array.isArray(content) && content.length > 0 && content[0].text) {
        const resultText = content[0].text;
        
        logger.info('Recent activity query completed successfully', {
          repositoryId,
          days,
          responseLength: resultText.length,
          isError: mcpResponse.result.isError || false
        });

        return resultText;
      }

      throw new Error('Invalid MCP response: no text content found');

    } catch (error) {
      logger.error('Recent activity query failed', { repositoryId, days }, error as Error);
      throw error;
    }
  }

  async getSystemStatus(): Promise<SystemStatus> {
    try {
      logger.info('Getting system status via MCP');

      // Check MCP server health
      const healthResponse = await executeWithRetry(
        () => this.client.get('/health'),
        {
          maxAttempts: 2, // Limited retries for health checks
          baseDelay: 500,
          maxDelay: 2000
        },
        'MCP health check'
      );

      const mcpServerStatus = {
        status: 'healthy' as const,
        uptime: 0, // Would need to be calculated from server start time
        version: healthResponse.data.version || '1.0.0'
      };

      // Try to get repository list to check Go backend status
      let goBackendStatus;
      let repositoriesStatus;

      try {
        const startTime = Date.now();
        
        const mcpRequest: MCPRequest = {
          method: 'list_resources',
          params: {},
          id: this.generateRequestId()
        };

        const resourcesResponse = await executeWithRetry(
          () => this.client.post<MCPResponse>('/mcp', mcpRequest),
          {
            maxAttempts: 2,
            baseDelay: 500,
            maxDelay: 2000
          },
          'MCP resources list'
        );

        const responseTime = Date.now() - startTime;

        if (resourcesResponse.data.error) {
          goBackendStatus = {
            status: 'degraded' as const,
            responseTime,
            lastCheck: new Date().toISOString()
          };

          repositoriesStatus = {
            total: 0,
            active: 0,
            ingesting: 0,
            error: 0
          };
        } else {
          goBackendStatus = {
            status: 'healthy' as const,
            responseTime,
            lastCheck: new Date().toISOString()
          };

          // Count repositories from resources
          const resources = resourcesResponse.data.result?.resources || [];
          const repositoryResources = resources.filter((r: any) => 
            r.uri && r.uri.startsWith('contextkeeper://repository/')
          );

          repositoriesStatus = {
            total: repositoryResources.length,
            active: repositoryResources.length, // Assume all are active for now
            ingesting: 0, // Would need more detailed status
            error: 0
          };
        }

      } catch (error) {
        logger.warn('Failed to get Go backend status', {}, error as Error);
        
        goBackendStatus = {
          status: 'down' as const,
          responseTime: 0,
          lastCheck: new Date().toISOString()
        };

        repositoriesStatus = {
          total: 0,
          active: 0,
          ingesting: 0,
          error: 0
        };
      }

      const systemStatus: SystemStatus = {
        mcpServer: mcpServerStatus,
        goBackend: goBackendStatus,
        repositories: repositoriesStatus
      };

      logger.info('System status retrieved successfully', {
        mcpStatus: mcpServerStatus.status,
        backendStatus: goBackendStatus.status,
        totalRepos: repositoriesStatus.total
      });

      return systemStatus;

    } catch (error) {
      logger.error('System status query failed', {}, error as Error);
      
      // Return degraded status on error
      return {
        mcpServer: {
          status: 'down',
          uptime: 0,
          version: 'unknown'
        },
        goBackend: {
          status: 'down',
          responseTime: 0,
          lastCheck: new Date().toISOString()
        },
        repositories: {
          total: 0,
          active: 0,
          ingesting: 0,
          error: 0
        }
      };
    }
  }

  private handleMCPError(error: AxiosError): Error {
    // Don't transform the error here - let the error handling utilities classify it properly
    return error as Error;
  }

  private generateRequestId(): string {
    return `slack-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }

  async healthCheck(): Promise<boolean> {
    try {
      logger.debug('Performing MCP server health check');

      await executeWithTimeout(
        () => this.client.get('/health'),
        5000,
        'MCP health check'
      );
      
      logger.debug('MCP server health check passed');
      return true;
    } catch (error) {
      logger.warn('MCP server health check failed', {}, error as Error);
      return false;
    }
  }

  getConfig(): MCPClientConfig {
    return { ...this.config };
  }

  setTimeout(timeout: number): void {
    this.config.timeout = timeout;
    this.client.defaults.timeout = timeout;
  }
}