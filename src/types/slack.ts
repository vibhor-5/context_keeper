/**
 * Core TypeScript interfaces for Slack Bot components
 */

export interface SlackBot {
  start(): Promise<void>;
  stop(): Promise<void>;
  handleCommand(command: SlackCommand): Promise<SlackResponse>;
}

export interface SlackCommand {
  command: string;
  text: string;
  userId: string;
  channelId: string;
  teamId: string;
  userName: string;
  responseUrl: string;
}

export interface SlackResponse {
  text: string;
  blocks?: SlackBlock[];
  responseType: 'ephemeral' | 'in_channel';
  replaceOriginal?: boolean;
  deleteOriginal?: boolean;
}

export interface SlackBlock {
  type: string;
  text?: SlackText;
  elements?: SlackElement[];
  fields?: SlackText[];
  accessory?: SlackElement;
}

export interface SlackText {
  type: 'plain_text' | 'mrkdwn';
  text: string;
  emoji?: boolean;
}

export interface SlackElement {
  type: string;
  text?: SlackText;
  value?: string;
  url?: string;
  action_id?: string;
}

// Command-specific interfaces
export interface ContextCommand {
  command: "/context";
  parameters: {
    query: string;
    repository?: string;
  };
}

export interface OnboardCommand {
  command: "/onboard";
  parameters: {
    repository?: string;
    focus?: string[];
  };
}

export interface RecentCommand {
  command: "/recent";
  parameters: {
    repository?: string;
    days?: number;
  };
}

export interface StatusCommand {
  command: "/status";
  parameters: {};
}

// Response Models
export interface SlackContextResponse {
  text: string;
  blocks: [
    {
      type: "section";
      text: {
        type: "mrkdwn";
        text: string; // Formatted context response
      };
    },
    {
      type: "context";
      elements: [
        {
          type: "mrkdwn";
          text: string; // Metadata (repository, query time, etc.)
        }
      ];
    }
  ];
  responseType: "ephemeral";
}

export interface SlackOnboardingResponse {
  text: string;
  blocks: SlackBlock[];
  responseType: "ephemeral";
}

export interface SlackRecentResponse {
  text: string;
  blocks: SlackBlock[];
  responseType: "ephemeral";
}

export interface SlackStatusResponse {
  text: string;
  blocks: SlackBlock[];
  responseType: "ephemeral";
}

// MCP Client interface for Slack Bot
export interface MCPClient {
  queryContext(query: string, repositoryId?: string): Promise<string>;
  getOnboardingSummary(repositoryId: string): Promise<string>;
  getRecentActivity(repositoryId: string, days?: number): Promise<string>;
  getSystemStatus(): Promise<SystemStatus>;
}

export interface SystemStatus {
  mcpServer: {
    status: 'healthy' | 'degraded' | 'down';
    uptime: number;
    version: string;
  };
  goBackend: {
    status: 'healthy' | 'degraded' | 'down';
    responseTime: number;
    lastCheck: string;
  };
  repositories: {
    total: number;
    active: number;
    ingesting: number;
    error: number;
  };
}