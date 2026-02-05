/**
 * Configuration management system for ContextKeeper
 * Loads and validates configuration from environment variables and files
 */

import { config } from 'dotenv';
import { z } from 'zod';
import { readFileSync, existsSync } from 'fs';
import { join } from 'path';
import { SystemConfig, EnvironmentVariables, ConfigFile } from '../types/config';

// Load environment variables
config();

// Zod schemas for configuration validation
const MCPServerConfigSchema = z.object({
  port: z.number().int().min(1000).max(65535),
  goBackendUrl: z.string().url(),
  corsOrigins: z.array(z.string()),
  logLevel: z.enum(['debug', 'info', 'warn', 'error']),
  timeout: z.number().int().min(1000).max(60000)
});

const SlackBotConfigSchema = z.object({
  botToken: z.string().min(1),
  signingSecret: z.string().min(1),
  mcpServerUrl: z.string().url(),
  port: z.number().int().min(1000).max(65535),
  defaultRepository: z.string().optional(),
  maxResponseLength: z.number().int().min(100).max(10000),
  retryAttempts: z.number().int().min(1).max(10),
  retryDelay: z.number().int().min(100).max(10000)
});

const DemoConfigSchema = z.object({
  sampleRepositoryId: z.string().min(1),
  predictableResponses: z.boolean(),
  demoMode: z.boolean()
});

const SystemConfigSchema = z.object({
  mcp: MCPServerConfigSchema,
  slack: SlackBotConfigSchema,
  demo: DemoConfigSchema
});

/**
 * Load and validate system configuration
 */
export async function loadConfig(configFilePath?: string): Promise<SystemConfig> {
  const env = process.env as EnvironmentVariables;
  
  try {
    // Load configuration from file if provided
    let fileConfig: Partial<SystemConfig> = {};
    if (configFilePath && existsSync(configFilePath)) {
      try {
        const fileContent = readFileSync(configFilePath, 'utf-8');
        fileConfig = JSON.parse(fileContent) as any;
      } catch (error) {
        throw new Error(`Failed to parse configuration file ${configFilePath}: ${error}`);
      }
    }
    
    // Merge environment variables with file configuration (env takes precedence)
    const config: SystemConfig = {
      mcp: {
        port: parseInt(env.MCP_PORT || fileConfig.mcp?.port?.toString() || '3001'),
        goBackendUrl: env.GO_BACKEND_URL || fileConfig.mcp?.goBackendUrl || 'http://localhost:8080',
        corsOrigins: env.CORS_ORIGINS ? env.CORS_ORIGINS.split(',') : (fileConfig.mcp?.corsOrigins || ['*']),
        logLevel: (env.LOG_LEVEL as any) || fileConfig.mcp?.logLevel || 'info',
        timeout: parseInt(env.REQUEST_TIMEOUT || fileConfig.mcp?.timeout?.toString() || '30000')
      },
      slack: {
        botToken: env.SLACK_BOT_TOKEN || fileConfig.slack?.botToken || '',
        signingSecret: env.SLACK_SIGNING_SECRET || fileConfig.slack?.signingSecret || '',
        mcpServerUrl: env.MCP_SERVER_URL || fileConfig.slack?.mcpServerUrl || 'http://localhost:3001',
        port: parseInt(env.SLACK_PORT || fileConfig.slack?.port?.toString() || '3002'),
        defaultRepository: env.DEFAULT_REPOSITORY || fileConfig.slack?.defaultRepository,
        maxResponseLength: parseInt(env.MAX_RESPONSE_LENGTH || fileConfig.slack?.maxResponseLength?.toString() || '4000'),
        retryAttempts: parseInt(env.RETRY_ATTEMPTS || fileConfig.slack?.retryAttempts?.toString() || '3'),
        retryDelay: parseInt(env.RETRY_DELAY || fileConfig.slack?.retryDelay?.toString() || '1000')
      },
      demo: {
        sampleRepositoryId: env.SAMPLE_REPOSITORY_ID || fileConfig.demo?.sampleRepositoryId || 'demo-repo-1',
        predictableResponses: (env.PREDICTABLE_RESPONSES === 'true') || (fileConfig.demo?.predictableResponses ?? false),
        demoMode: (env.DEMO_MODE === 'true') || (fileConfig.demo?.demoMode ?? false)
      }
    };
    
    // Validate configuration
    const validatedConfig = SystemConfigSchema.parse(config);
    
    // Additional validation for required parameters
    validateRequiredParameters(validatedConfig);
    
    return validatedConfig;
    
  } catch (error) {
    if (error instanceof z.ZodError) {
      const errorMessages = error.errors.map(err => 
        `${err.path.join('.')}: ${err.message}`
      ).join('\n');
      
      throw new Error(`Configuration validation failed:\n${errorMessages}`);
    }
    
    throw error;
  }
}

/**
 * Validate required configuration parameters
 */
function validateRequiredParameters(config: SystemConfig): void {
  const errors: string[] = [];
  
  // Validate required Slack parameters
  if (!config.slack.botToken || config.slack.botToken === '') {
    errors.push('slack.botToken is required. Set SLACK_BOT_TOKEN environment variable or provide in config file.');
  }
  
  if (!config.slack.signingSecret || config.slack.signingSecret === '') {
    errors.push('slack.signingSecret is required. Set SLACK_SIGNING_SECRET environment variable or provide in config file.');
  }
  
  // Validate bot token format
  if (config.slack.botToken && !config.slack.botToken.startsWith('xoxb-')) {
    errors.push('slack.botToken must be a valid Slack bot token (starts with "xoxb-").');
  }
  
  // Validate URLs
  try {
    new URL(config.mcp.goBackendUrl);
  } catch {
    errors.push('mcp.goBackendUrl must be a valid URL.');
  }
  
  try {
    new URL(config.slack.mcpServerUrl);
  } catch {
    errors.push('slack.mcpServerUrl must be a valid URL.');
  }
  
  // Validate port conflicts
  if (config.mcp.port === config.slack.port) {
    errors.push('MCP server and Slack bot cannot use the same port.');
  }
  
  if (errors.length > 0) {
    throw new Error(`Configuration validation failed:\n${errors.join('\n')}`);
  }
}

/**
 * Load configuration from file
 */
export async function loadConfigFromFile(filePath: string): Promise<SystemConfig> {
  if (!existsSync(filePath)) {
    throw new Error(`Configuration file not found: ${filePath}`);
  }
  
  return loadConfig(filePath);
}

/**
 * Save configuration to file
 */
export async function saveConfigToFile(config: SystemConfig, filePath: string): Promise<void> {
  try {
    const configJson = JSON.stringify(config, null, 2);
    const fs = await import('fs/promises');
    await fs.writeFile(filePath, configJson, 'utf-8');
  } catch (error) {
    throw new Error(`Failed to save configuration to ${filePath}: ${error}`);
  }
}

/**
 * Get configuration for specific environment
 */
export function getEnvironmentConfig(environment: 'development' | 'production' | 'test'): Partial<SystemConfig> {
  const baseConfig = {
    development: {
      mcp: {
        logLevel: 'debug' as const,
        corsOrigins: ['*']
      },
      demo: {
        demoMode: true,
        predictableResponses: true
      }
    },
    production: {
      mcp: {
        logLevel: 'info' as const,
        corsOrigins: []
      },
      demo: {
        demoMode: false,
        predictableResponses: false
      }
    },
    test: {
      mcp: {
        logLevel: 'warn' as const,
        port: 0, // Use random port for tests
        corsOrigins: ['*']
      },
      slack: {
        port: 0, // Use random port for tests
        botToken: 'xoxb-test-token',
        signingSecret: 'test-signing-secret'
      },
      demo: {
        demoMode: true,
        predictableResponses: true
      }
    }
  };
  
  return (baseConfig as any)[environment] || {} as any;
}

/**
 * Get default configuration for development
 */
export function getDefaultConfig(): SystemConfig {
  return {
    mcp: {
      port: 3001,
      goBackendUrl: 'http://localhost:8080',
      corsOrigins: ['*'],
      logLevel: 'info',
      timeout: 30000
    },
    slack: {
      botToken: 'xoxb-your-bot-token',
      signingSecret: 'your-signing-secret',
      mcpServerUrl: 'http://localhost:3001',
      port: 3002,
      maxResponseLength: 4000,
      retryAttempts: 3,
      retryDelay: 1000
    },
    demo: {
      sampleRepositoryId: 'demo-repo-1',
      predictableResponses: true,
      demoMode: false
    }
  };
}