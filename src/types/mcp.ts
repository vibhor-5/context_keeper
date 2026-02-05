/**
 * Core TypeScript interfaces for MCP (Model Context Protocol) components
 */

export interface MCPRequest {
  method: string;
  params: Record<string, any>;
  id?: string | number;
}

export interface MCPResponse {
  result?: any;
  error?: MCPError;
  id?: string | number;
}

export interface MCPError {
  code: number;
  message: string;
  data?: any;
}

export interface MCPServer {
  start(port: number): Promise<void>;
  stop(): Promise<void>;
  handleRequest(request: MCPRequest): Promise<MCPResponse>;
}

export interface Resource {
  uri: string;
  name: string;
  description: string;
  mimeType: string;
}

export interface ResourceContent {
  uri: string;
  mimeType: string;
  text?: string;
  blob?: Uint8Array;
}

export interface ResourceProvider {
  listResources(): Promise<Resource[]>;
  getResource(uri: string): Promise<ResourceContent>;
}

export interface Tool {
  name: string;
  description: string;
  inputSchema: JSONSchema;
}

export interface ToolResult {
  content: Array<{
    type: 'text' | 'image' | 'resource';
    text?: string;
    data?: string;
    resource?: {
      uri: string;
      mimeType: string;
    };
  }>;
  isError?: boolean;
}

export interface ToolProvider {
  listTools(): Promise<Tool[]>;
  callTool(name: string, args: Record<string, any>): Promise<ToolResult>;
}

export interface JSONSchema {
  type: string;
  properties?: Record<string, any>;
  required?: string[];
  items?: JSONSchema;
  additionalProperties?: boolean;
  description?: string;
  default?: any;
}

// MCP Resource Models
export interface RepositoryResource {
  uri: string; // "contextkeeper://repository/{id}"
  name: string; // Repository name
  description: string; // Repository description
  mimeType: "application/json";
  content: RepositoryMetadata;
}

export interface ContextResource {
  uri: string; // "contextkeeper://context/{id}"
  name: string; // "Repository Context"
  description: string; // Context description
  mimeType: "text/plain";
  content: string; // Formatted context text
}

export interface TimelineResource {
  uri: string; // "contextkeeper://timeline/{id}"
  name: string; // "Repository Timeline"
  description: string; // Timeline description
  mimeType: "application/json";
  content: TimelineEvent[];
}

// MCP Tool Schemas
export interface QueryRepositoryContextTool {
  name: "query_repository_context";
  description: "Query repository for specific context information";
  inputSchema: {
    type: "object";
    properties: {
      query: { type: "string"; description: "Natural language query" };
      repositoryId: { type: "string"; description: "Optional repository ID" };
      limit: { type: "number"; description: "Maximum results"; default: 10 };
    };
    required: ["query"];
  };
}

export interface GetOnboardingSummaryTool {
  name: "get_onboarding_summary";
  description: "Generate onboarding summary for new team members";
  inputSchema: {
    type: "object";
    properties: {
      repositoryId: { type: "string"; description: "Repository ID" };
      focusAreas: { 
        type: "array"; 
        items: { type: "string" };
        description: "Optional focus areas for onboarding";
      };
    };
    required: ["repositoryId"];
  };
}

// Repository Data Models
export interface RepositoryMetadata {
  id: string;
  name: string;
  fullName: string;
  description: string;
  url: string;
  defaultBranch: string;
  language: string;
  topics: string[];
  createdAt: string;
  updatedAt: string;
  lastIngestionAt: string;
  status: "active" | "ingesting" | "error";
}

export interface TimelineEvent {
  id: string;
  type: 'pr' | 'issue' | 'commit';
  title: string;
  body: string;
  author: string;
  timestamp: string;
  url: string;
  labels?: Label[];
  filesChanged?: string[];
}

export interface PullRequest {
  id: number;
  title: string;
  body: string;
  state: "open" | "closed" | "merged";
  labels: Label[];
  filesChanged: string[];
  createdAt: string;
  updatedAt: string;
  author: string;
  url: string;
}

export interface Issue {
  id: number;
  title: string;
  body: string;
  state: "open" | "closed";
  labels: Label[];
  createdAt: string;
  updatedAt: string;
  author: string;
  url: string;
}

export interface Commit {
  sha: string;
  title: string;
  body: string;
  author: string;
  timestamp: string;
  filesChanged: string[];
  url: string;
}

export interface Label {
  name: string;
  color: string;
  description: string;
}