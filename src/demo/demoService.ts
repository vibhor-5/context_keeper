/**
 * Demo service for ContextKeeper
 * Provides demo mode functionality with fallback mechanisms
 */

import { DemoDataManager, demoDataManager } from './demoData';
import { GoBackendClient } from '../services/goBackendClient';
import { logger } from '../utils/logger';
import { 
  executeWithRetry, 
  executeWithTimeout, 
  handlePartialData,
  classifyError,
  ErrorType,
  DEFAULT_RETRY_CONFIG,
  DEFAULT_TIMEOUT_CONFIG
} from '../utils/errorHandling';

export interface DemoServiceConfig {
  enabled: boolean;
  fallbackOnError: boolean;
  predictableResponses: boolean;
  fallbackTimeout: number;
  maxFallbackAttempts: number;
}

export interface DemoServiceResult<T> {
  data: T;
  source: 'live' | 'demo' | 'fallback';
  isComplete: boolean;
  errors?: string[];
}

/**
 * Demo service that provides fallback functionality
 */
export class DemoService {
  private config: DemoServiceConfig;
  private demoDataManager: DemoDataManager;
  private goBackendClient?: GoBackendClient;

  constructor(config: DemoServiceConfig, goBackendClient?: GoBackendClient) {
    this.config = config;
    this.demoDataManager = demoDataManager;
    this.goBackendClient = goBackendClient;
    
    // Configure demo data manager
    this.demoDataManager.setPredictableMode(config.predictableResponses);
    
    logger.info('Demo service initialized', {
      enabled: config.enabled,
      fallbackOnError: config.fallbackOnError,
      predictableResponses: config.predictableResponses
    });
  }

  /**
   * Query repository context with demo fallback
   */
  async queryContext(query: string, repositoryId?: string): Promise<DemoServiceResult<string>> {
    logger.debug('Demo service context query', { 
      query: query.substring(0, 50), 
      repositoryId,
      demoMode: this.config.enabled
    });

    // If demo mode is enabled, use demo data directly
    if (this.config.enabled) {
      const demoResponse = this.demoDataManager.queryContext(query, repositoryId);
      
      logger.debug('Using demo data for context query', { 
        repositoryId,
        responseLength: demoResponse.length
      });
      
      return {
        data: demoResponse,
        source: 'demo',
        isComplete: true
      };
    }

    // Try live backend first
    if (this.goBackendClient && this.config.fallbackOnError) {
      try {
        const liveResponse = await executeWithTimeout(
          () => executeWithRetry(
            async () => {
              if (!this.goBackendClient) {
                throw new Error('Go backend client not available');
              }
              
              const result = await this.goBackendClient.queryContext({ 
                query, 
                repo_id: parseInt(repositoryId || '1') || 1 
              });
              const response = result.context || 'Context query completed successfully.';
              return typeof response === 'string' ? response : JSON.stringify(response);
            },
            {
              maxAttempts: this.config.maxFallbackAttempts,
              baseDelay: 1000,
              maxDelay: 5000
            },
            'live context query'
          ),
          this.config.fallbackTimeout,
          'live context query with timeout'
        );

        logger.info('Live context query successful', { 
          repositoryId,
          responseLength: liveResponse.length
        });

        return {
          data: liveResponse,
          source: 'live',
          isComplete: true
        };

      } catch (error) {
        const classifiedError = classifyError(error);
        
        logger.warn('Live context query failed, falling back to demo data', {
          repositoryId,
          errorType: classifiedError.type,
          message: classifiedError.message
        });

        // Fall back to demo data
        const demoResponse = this.demoDataManager.queryContext(query, repositoryId);
        
        return {
          data: demoResponse,
          source: 'fallback',
          isComplete: false,
          errors: [classifiedError.userFriendlyMessage]
        };
      }
    }

    // No backend available, use demo data
    const demoResponse = this.demoDataManager.queryContext(query, repositoryId);
    
    return {
      data: demoResponse,
      source: 'demo',
      isComplete: true
    };
  }

  /**
   * Get onboarding summary with demo fallback
   */
  async getOnboardingSummary(repositoryId: string): Promise<DemoServiceResult<string>> {
    logger.debug('Demo service onboarding summary', { 
      repositoryId,
      demoMode: this.config.enabled
    });

    // If demo mode is enabled, use demo data directly
    if (this.config.enabled) {
      const demoResponse = this.demoDataManager.getOnboardingSummary(repositoryId);
      
      return {
        data: demoResponse,
        source: 'demo',
        isComplete: true
      };
    }

    // Try live backend first
    if (this.goBackendClient && this.config.fallbackOnError) {
      try {
        const liveResponse = await executeWithTimeout(
          () => executeWithRetry(
            async () => {
              if (!this.goBackendClient) {
                throw new Error('Go backend client not available');
              }
              
              // For now, generate onboarding from repository status and context
              const repoStatus = await this.goBackendClient.getRepositoryStatus(parseInt(repositoryId));
              
              // Create basic onboarding summary from available data
              let summary = `# Welcome to ${repositoryId}\n\n`;
              summary += `**Status**: ${repoStatus.status}\n`;
              summary += `**Last Updated**: ${(repoStatus as any).lastIngestion || 'Never'}\n\n`;
              summary += `This repository is actively maintained and available for development.\n`;
              summary += `Contact your team lead for specific onboarding instructions.`;
              
              return summary;
            },
            {
              maxAttempts: this.config.maxFallbackAttempts,
              baseDelay: 1000,
              maxDelay: 5000
            },
            'live onboarding query'
          ),
          this.config.fallbackTimeout,
          'live onboarding query with timeout'
        );

        logger.info('Live onboarding query successful', { 
          repositoryId,
          responseLength: liveResponse.length
        });

        return {
          data: liveResponse,
          source: 'live',
          isComplete: true
        };

      } catch (error) {
        const classifiedError = classifyError(error);
        
        logger.warn('Live onboarding query failed, falling back to demo data', {
          repositoryId,
          errorType: classifiedError.type,
          message: classifiedError.message
        });

        // Fall back to demo data
        const demoResponse = this.demoDataManager.getOnboardingSummary(repositoryId);
        
        return {
          data: demoResponse,
          source: 'fallback',
          isComplete: false,
          errors: [classifiedError.userFriendlyMessage]
        };
      }
    }

    // No backend available, use demo data
    const demoResponse = this.demoDataManager.getOnboardingSummary(repositoryId);
    
    return {
      data: demoResponse,
      source: 'demo',
      isComplete: true
    };
  }

  /**
   * Get recent activity with demo fallback
   */
  async getRecentActivity(repositoryId: string, days: number = 7): Promise<DemoServiceResult<string>> {
    logger.debug('Demo service recent activity', { 
      repositoryId,
      days,
      demoMode: this.config.enabled
    });

    // If demo mode is enabled, use demo data directly
    if (this.config.enabled) {
      const demoResponse = this.demoDataManager.getRecentActivity(repositoryId, days);
      
      return {
        data: demoResponse,
        source: 'demo',
        isComplete: true
      };
    }

    // Try live backend first
    if (this.goBackendClient && this.config.fallbackOnError) {
      try {
        const liveResponse = await executeWithTimeout(
          () => executeWithRetry(
            async () => {
              if (!this.goBackendClient) {
                throw new Error('Go backend client not available');
              }
              
              // Get repository status which includes some activity info
              const repoStatus = await this.goBackendClient.getRepositoryStatus(parseInt(repositoryId));
              
              // Create basic activity summary from available data
              let activity = `Recent activity for ${repositoryId}:\n\n`;
              activity += `• **Status**: ${repoStatus.status}\n`;
              activity += `• **Last Ingestion**: ${(repoStatus as any).lastIngestion || 'Never'}\n`;
              activity += `• **Files Processed**: ${(repoStatus as any).fileCount || 'Unknown'}\n\n`;
              activity += `For detailed activity history, check your version control system.`;
              
              return activity;
            },
            {
              maxAttempts: this.config.maxFallbackAttempts,
              baseDelay: 1000,
              maxDelay: 5000
            },
            'live activity query'
          ),
          this.config.fallbackTimeout,
          'live activity query with timeout'
        );

        logger.info('Live activity query successful', { 
          repositoryId,
          days,
          responseLength: liveResponse.length
        });

        return {
          data: liveResponse,
          source: 'live',
          isComplete: true
        };

      } catch (error) {
        const classifiedError = classifyError(error);
        
        logger.warn('Live activity query failed, falling back to demo data', {
          repositoryId,
          days,
          errorType: classifiedError.type,
          message: classifiedError.message
        });

        // Fall back to demo data
        const demoResponse = this.demoDataManager.getRecentActivity(repositoryId, days);
        
        return {
          data: demoResponse,
          source: 'fallback',
          isComplete: false,
          errors: [classifiedError.userFriendlyMessage]
        };
      }
    }

    // No backend available, use demo data
    const demoResponse = this.demoDataManager.getRecentActivity(repositoryId, days);
    
    return {
      data: demoResponse,
      source: 'demo',
      isComplete: true
    };
  }

  /**
   * Get system status with demo fallback
   */
  async getSystemStatus(): Promise<DemoServiceResult<any>> {
    logger.debug('Demo service system status', { demoMode: this.config.enabled });

    // If demo mode is enabled, use demo data directly
    if (this.config.enabled) {
      const demoStatus = this.demoDataManager.getSystemStatus();
      
      return {
        data: {
          mcpServer: demoStatus.mcpServer,
          goBackend: demoStatus.goBackend,
          repositories: demoStatus.repositories
        },
        source: 'demo',
        isComplete: true
      };
    }

    // Try to get partial live data with fallbacks
    const operations = [
      {
        name: 'mcpServer',
        operation: async () => ({
          status: 'healthy' as const,
          version: '1.0.0',
          uptime: process.uptime()
        }),
        required: true
      },
      {
        name: 'goBackend',
        operation: async () => {
          if (!this.goBackendClient) {
            throw new Error('Go backend client not available');
          }
          
          const startTime = Date.now();
          await this.goBackendClient.healthCheck();
          const responseTime = Date.now() - startTime;
          
          return {
            status: 'healthy' as const,
            responseTime,
            lastCheck: new Date().toISOString()
          };
        },
        required: false
      },
      {
        name: 'repositories',
        operation: async () => {
          // For now, return demo data for repositories
          // In a real implementation, this would query the backend
          const demoStatus = this.demoDataManager.getSystemStatus();
          return demoStatus.repositories;
        },
        required: false
      }
    ];

    try {
      const result = await handlePartialData(operations, 'system status collection');
      
      // Fill in missing data with demo fallbacks
      const demoStatus = this.demoDataManager.getSystemStatus();
      const finalStatus = {
        mcpServer: (result.data as any)?.mcpServer || demoStatus.mcpServer,
        goBackend: (result.data as any)?.goBackend || {
          status: 'down' as const,
          responseTime: 0,
          lastCheck: new Date().toISOString()
        },
        repositories: (result.data as any)?.repositories || demoStatus.repositories
      };

      logger.info('System status collected', {
        isComplete: result.isComplete,
        completionPercentage: result.completionPercentage,
        errorCount: result.errors.length
      });

      return {
        data: finalStatus,
        source: result.isComplete ? 'live' : 'fallback',
        isComplete: result.isComplete,
        errors: result.errors
      };

    } catch (error) {
      logger.error('System status collection failed completely', {}, error as Error);
      
      // Complete fallback to demo data
      const demoStatus = this.demoDataManager.getSystemStatus();
      
      return {
        data: {
          mcpServer: demoStatus.mcpServer,
          goBackend: { ...demoStatus.goBackend, status: 'down' as const },
          repositories: demoStatus.repositories
        },
        source: 'fallback',
        isComplete: false,
        errors: ['System status unavailable - using demo data']
      };
    }
  }

  /**
   * Check if demo mode is enabled
   */
  isDemoMode(): boolean {
    return this.config.enabled;
  }

  /**
   * Check if fallback is enabled
   */
  isFallbackEnabled(): boolean {
    return this.config.fallbackOnError;
  }

  /**
   * Update demo service configuration
   */
  updateConfig(newConfig: Partial<DemoServiceConfig>): void {
    this.config = { ...this.config, ...newConfig };
    
    // Update demo data manager if predictable mode changed
    if (newConfig.predictableResponses !== undefined) {
      this.demoDataManager.setPredictableMode(newConfig.predictableResponses);
    }
    
    logger.info('Demo service configuration updated', this.config);
  }

  /**
   * Get current configuration
   */
  getConfig(): DemoServiceConfig {
    return { ...this.config };
  }

  /**
   * Reset demo data to defaults
   */
  resetDemoData(): void {
    this.demoDataManager.reset();
    logger.info('Demo data reset to defaults');
  }

  /**
   * Get demo data manager (for testing)
   */
  getDemoDataManager(): DemoDataManager {
    return this.demoDataManager;
  }
}

/**
 * Create demo service with default configuration
 */
export function createDemoService(
  config: Partial<DemoServiceConfig> = {},
  goBackendClient?: GoBackendClient
): DemoService {
  const defaultConfig: DemoServiceConfig = {
    enabled: false,
    fallbackOnError: true,
    predictableResponses: true,
    fallbackTimeout: 10000,
    maxFallbackAttempts: 2
  };

  return new DemoService({ ...defaultConfig, ...config }, goBackendClient);
}