/**
 * MCP Tool Provider
 * Provides MCP tools for repository context queries and onboarding summaries
 */

import { ToolProvider, Tool, ToolResult, JSONSchema } from '../types/mcp';
import { GoBackendClient } from '../services/goBackendClient';
import { DemoService } from '../demo/demoService';
import { logger } from '../utils/logger';

export class ContextKeeperToolProvider implements ToolProvider {
  private goBackendClient: GoBackendClient;
  private demoService?: DemoService;

  constructor(goBackendClient: GoBackendClient, demoService?: DemoService) {
    this.goBackendClient = goBackendClient;
    this.demoService = demoService;
  }

  async listTools(): Promise<Tool[]> {
    logger.debug('Listing available tools');

    const tools: Tool[] = [
      {
        name: 'query_repository_context',
        description: 'Query repository for specific context information using natural language',
        inputSchema: {
          type: 'object',
          properties: {
            query: {
              type: 'string',
              description: 'Natural language query about the repository (e.g., "What are the main components?", "Show me recent changes", "How does authentication work?")'
            },
            repositoryId: {
              type: 'string',
              description: 'Optional repository ID to query. If not provided, will search across available repositories.'
            },
            limit: {
              type: 'number',
              description: 'Maximum number of results to return',
              default: 10
            }
          },
          required: ['query'],
          additionalProperties: false
        }
      },
      {
        name: 'get_onboarding_summary',
        description: 'Generate comprehensive onboarding summary for new team members joining a repository',
        inputSchema: {
          type: 'object',
          properties: {
            repositoryId: {
              type: 'string',
              description: 'Repository ID to generate onboarding summary for'
            },
            focusAreas: {
              type: 'array',
              items: { type: 'string' },
              description: 'Optional focus areas for onboarding (e.g., ["architecture", "testing", "deployment", "contributing"])'
            }
          },
          required: ['repositoryId'],
          additionalProperties: false
        }
      }
    ];

    logger.info('Listed tools successfully', { toolCount: tools.length });
    return tools;
  }

  async callTool(name: string, args: Record<string, any>): Promise<ToolResult> {
    try {
      logger.debug('Calling tool', { name, args });

      switch (name) {
        case 'query_repository_context':
          return await this.queryRepositoryContext(args);
        
        case 'get_onboarding_summary':
          return await this.getOnboardingSummary(args);
        
        default:
          throw new Error(`Unknown tool: ${name}`);
      }

    } catch (error) {
      logger.error('Failed to call tool', { name, args }, error as Error);
      
      return {
        content: [{
          type: 'text',
          text: `Error calling tool ${name}: ${(error as Error).message}`
        }],
        isError: true
      };
    }
  }

  private async queryRepositoryContext(args: Record<string, any>): Promise<ToolResult> {
    // Validate required parameters
    if (!args.query || typeof args.query !== 'string') {
      throw new Error('Missing or invalid required parameter: query');
    }

    const query = args.query.trim();
    if (query.length === 0) {
      throw new Error('Query cannot be empty');
    }

    const repositoryId = args.repositoryId ? parseInt(args.repositoryId) : undefined;
    const limit = args.limit || 10;

    logger.info('Querying repository context', { 
      query: query.substring(0, 100), 
      repositoryId, 
      limit 
    });

    try {
      // If no repository ID provided, we need to pick a default or search across repositories
      let targetRepoId = repositoryId;
      
      if (!targetRepoId) {
        // Get available repositories and use the first one as default
        const repositories = await this.goBackendClient.getRepositories();
        if (repositories.repositories.length === 0) {
          return {
            content: [{
              type: 'text',
              text: 'No repositories are currently available. Please ensure repositories have been ingested into the system.'
            }],
            isError: false
          };
        }
        targetRepoId = repositories.repositories[0].id;
        logger.debug('Using default repository', { repositoryId: targetRepoId });
      }

      // Query the Go backend for context
      const contextQuery = {
        repo_id: targetRepoId,
        query: query,
        mode: 'query' as const
      };

      const contextResponse = await this.goBackendClient.queryContext(contextQuery);

      // Format the response for MCP consumption
      let responseText = '';

      // Add clarified goal if available
      if (contextResponse.clarified_goal) {
        responseText += `## Query Understanding\n${contextResponse.clarified_goal}\n\n`;
      }

      // Add main context information
      if (contextResponse.context) {
        responseText += `## Context Information\n`;
        
        if (typeof contextResponse.context === 'object') {
          // Try to extract meaningful information from the context object
          const contextObj = contextResponse.context as any;
          
          if (contextObj.summary) {
            responseText += `${contextObj.summary}\n\n`;
          } else if (contextObj.description) {
            responseText += `${contextObj.description}\n\n`;
          } else {
            // If it's a complex object, provide a structured view
            responseText += this.formatContextObject(contextObj) + '\n\n';
          }
        } else {
          responseText += `${contextResponse.context}\n\n`;
        }
      }

      // Add tasks/action items if available
      if (contextResponse.tasks && contextResponse.tasks.length > 0) {
        responseText += `## Key Areas & Action Items\n`;
        contextResponse.tasks.slice(0, limit).forEach((task, index) => {
          responseText += `${index + 1}. **${task.title}**\n`;
          if (task.acceptance) {
            responseText += `   ${task.acceptance}\n`;
          }
          responseText += '\n';
        });
      }

      // Add follow-up questions if available
      if (contextResponse.questions && contextResponse.questions.length > 0) {
        responseText += `## Related Questions\n`;
        contextResponse.questions.slice(0, 5).forEach((question, index) => {
          responseText += `${index + 1}. ${question}\n`;
        });
        responseText += '\n';
      }

      // Add PR scaffold information if available
      if (contextResponse.pr_scaffold) {
        responseText += `## Suggested Implementation\n`;
        responseText += `**Branch:** ${contextResponse.pr_scaffold.branch}\n`;
        responseText += `**Title:** ${contextResponse.pr_scaffold.title}\n`;
        if (contextResponse.pr_scaffold.body) {
          responseText += `**Description:** ${contextResponse.pr_scaffold.body}\n`;
        }
        responseText += '\n';
      }

      // If no meaningful content, provide a helpful message
      if (!responseText.trim()) {
        responseText = `I found information related to your query "${query}" but it may require more specific questions to get detailed results. Try asking about specific components, features, or recent changes in the repository.`;
      }

      logger.info('Context query completed successfully', { 
        repositoryId: targetRepoId,
        responseLength: responseText.length,
        hasContext: !!contextResponse.context,
        hasTasks: !!contextResponse.tasks?.length,
        hasQuestions: !!contextResponse.questions?.length
      });

      return {
        content: [{
          type: 'text',
          text: responseText
        }],
        isError: false
      };

    } catch (error) {
      logger.error('Context query failed', { query, repositoryId }, error as Error);
      throw error;
    }
  }

  private async getOnboardingSummary(args: Record<string, any>): Promise<ToolResult> {
    // Validate required parameters
    if (!args.repositoryId) {
      throw new Error('Missing required parameter: repositoryId');
    }

    const repositoryId = parseInt(args.repositoryId);
    const focusAreas = args.focusAreas || ['overview', 'architecture', 'getting-started', 'contributing'];

    logger.info('Generating onboarding summary', { repositoryId, focusAreas });

    try {
      // Get repository metadata first
      const repositories = await this.goBackendClient.getRepositories();
      const repository = repositories.repositories.find(repo => repo.id === repositoryId);
      
      if (!repository) {
        throw new Error(`Repository not found: ${repositoryId}`);
      }

      // Generate comprehensive onboarding queries
      const onboardingQueries = [
        'What is this repository about and what problem does it solve?',
        'What is the overall architecture and main components?',
        'How do I set up the development environment and get started?',
        'What are the key files and directories I should know about?',
        'What are the coding standards and contribution guidelines?',
        'What are the main dependencies and technologies used?',
        'How do I run tests and what testing practices are followed?',
        'What is the deployment process and CI/CD setup?'
      ];

      let summaryText = `# Onboarding Guide: ${repository.name}\n\n`;
      summaryText += `**Repository:** ${repository.full_name}\n`;
      summaryText += `**Owner:** ${repository.owner}\n`;
      summaryText += `**Created:** ${new Date(repository.created_at).toLocaleDateString()}\n`;
      summaryText += `**Last Updated:** ${new Date(repository.updated_at).toLocaleDateString()}\n\n`;

      // Query each onboarding topic
      for (const query of onboardingQueries) {
        try {
          const contextQuery = {
            repo_id: repositoryId,
            query: query,
            mode: 'query' as const
          };

          const response = await this.goBackendClient.queryContext(contextQuery);
          
          // Extract section title from query
          const sectionTitle = this.extractSectionTitle(query);
          summaryText += `## ${sectionTitle}\n`;

          if (response.clarified_goal) {
            summaryText += `${response.clarified_goal}\n\n`;
          } else if (response.context) {
            if (typeof response.context === 'object') {
              summaryText += this.formatContextObject(response.context as any) + '\n\n';
            } else {
              summaryText += `${response.context}\n\n`;
            }
          } else {
            summaryText += `Information about ${sectionTitle.toLowerCase()} is available through specific queries.\n\n`;
          }

          // Add tasks if relevant
          if (response.tasks && response.tasks.length > 0) {
            summaryText += `### Key Points:\n`;
            response.tasks.slice(0, 3).forEach((task, index) => {
              summaryText += `- ${task.title}\n`;
            });
            summaryText += '\n';
          }

        } catch (queryError) {
          logger.warn('Failed to get onboarding section', { query }, queryError as Error);
          // Continue with other sections
        }
      }

      // Add getting started section
      summaryText += `## Getting Started Checklist\n`;
      summaryText += `- [ ] Clone the repository\n`;
      summaryText += `- [ ] Set up development environment\n`;
      summaryText += `- [ ] Install dependencies\n`;
      summaryText += `- [ ] Run tests to verify setup\n`;
      summaryText += `- [ ] Read contributing guidelines\n`;
      summaryText += `- [ ] Explore the codebase structure\n`;
      summaryText += `- [ ] Try making a small change\n\n`;

      summaryText += `## Need More Information?\n`;
      summaryText += `Use the \`query_repository_context\` tool with specific questions about:\n`;
      summaryText += `- Specific components or features\n`;
      summaryText += `- Recent changes or development activity\n`;
      summaryText += `- Technical implementation details\n`;
      summaryText += `- Troubleshooting and common issues\n`;

      logger.info('Onboarding summary generated successfully', { 
        repositoryId,
        summaryLength: summaryText.length
      });

      return {
        content: [{
          type: 'text',
          text: summaryText
        }],
        isError: false
      };

    } catch (error) {
      logger.error('Onboarding summary generation failed', { repositoryId }, error as Error);
      throw error;
    }
  }

  private formatContextObject(obj: any): string {
    if (!obj || typeof obj !== 'object') {
      return String(obj || '');
    }

    let formatted = '';
    
    // Handle common context object structures
    if (obj.summary) {
      formatted += obj.summary;
    } else if (obj.description) {
      formatted += obj.description;
    } else if (obj.content) {
      formatted += obj.content;
    } else {
      // For other objects, try to extract meaningful information
      const keys = Object.keys(obj);
      if (keys.length > 0) {
        formatted += 'Available information:\n';
        keys.slice(0, 5).forEach(key => {
          const value = obj[key];
          if (typeof value === 'string' && value.length > 0) {
            formatted += `- ${key}: ${value.substring(0, 100)}${value.length > 100 ? '...' : ''}\n`;
          } else if (Array.isArray(value) && value.length > 0) {
            formatted += `- ${key}: ${value.length} items\n`;
          } else if (typeof value === 'object' && value !== null) {
            formatted += `- ${key}: [object]\n`;
          }
        });
      }
    }

    return formatted || 'Context information is available but requires specific queries to access.';
  }

  private extractSectionTitle(query: string): string {
    // Extract meaningful section titles from queries
    if (query.includes('architecture')) return 'Architecture & Components';
    if (query.includes('setup') || query.includes('environment')) return 'Development Setup';
    if (query.includes('files') || query.includes('directories')) return 'Project Structure';
    if (query.includes('standards') || query.includes('guidelines')) return 'Coding Standards';
    if (query.includes('dependencies') || query.includes('technologies')) return 'Technologies & Dependencies';
    if (query.includes('tests') || query.includes('testing')) return 'Testing';
    if (query.includes('deployment') || query.includes('CI/CD')) return 'Deployment & CI/CD';
    if (query.includes('what is') || query.includes('about')) return 'Project Overview';
    
    return 'Information';
  }
}