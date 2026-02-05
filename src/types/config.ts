/**
 * Configuration interfaces for the ContextKeeper system
 */

export interface SystemConfig {
  mcp: MCPServerConfig;
  slack: SlackBotConfig;
  demo: DemoConfig;
}

export interface MCPServerConfig {
  port: number;
  goBackendUrl: string;
  corsOrigins: string[];
  logLevel: "debug" | "info" | "warn" | "error";
  timeout: number; // Request timeout in milliseconds
}

export interface SlackBotConfig {
  botToken: string;
  signingSecret: string;
  mcpServerUrl: string;
  port: number;
  defaultRepository?: string;
  maxResponseLength: number;
  retryAttempts: number;
  retryDelay: number;
}

export interface DemoConfig {
  sampleRepositoryId: string;
  predictableResponses: boolean;
  demoMode: boolean;
}

// Configuration file format (JSON)
export interface ConfigFile {
  mcp?: Partial<MCPServerConfig>;
  slack?: Partial<SlackBotConfig>;
  demo?: Partial<DemoConfig>;
}

// Environment variable mappings
export interface EnvironmentVariables {
  // MCP Server
  MCP_PORT?: string;
  GO_BACKEND_URL?: string;
  CORS_ORIGINS?: string;
  LOG_LEVEL?: string;
  REQUEST_TIMEOUT?: string;
  
  // Slack Bot
  SLACK_BOT_TOKEN?: string;
  SLACK_SIGNING_SECRET?: string;
  SLACK_PORT?: string;
  MCP_SERVER_URL?: string;
  DEFAULT_REPOSITORY?: string;
  MAX_RESPONSE_LENGTH?: string;
  RETRY_ATTEMPTS?: string;
  RETRY_DELAY?: string;
  
  // Demo
  SAMPLE_REPOSITORY_ID?: string;
  PREDICTABLE_RESPONSES?: string;
  DEMO_MODE?: string;
}