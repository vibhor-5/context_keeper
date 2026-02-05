/**
 * MCP Resource Provider
 * Provides repository resources (metadata, context, timeline) for MCP clients
 */

import { ResourceProvider, Resource, ResourceContent, RepositoryMetadata, TimelineEvent } from '../types/mcp';
import { GoBackendClient } from '../services/goBackendClient';
import { logger } from '../utils/logger';

export class ContextKeeperResourceProvider implements ResourceProvider {
  private goBackendClient: GoBackendClient;

  constructor(goBackendClient: GoBackendClient) {
    this.goBackendClient = goBackendClient;
  }

  async listResources(): Promise<Resource[]> {
    try {
      logger.debug('Listing available resources');

      // Get available repositories from Go backend
      const repositoriesResponse = await this.goBackendClient.getRepositories();
      const repositories = repositoriesResponse.repositories;

      const resources: Resource[] = [];

      // Create resources for each repository
      for (const repo of repositories) {
        // Repository metadata resource
        resources.push({
          uri: `contextkeeper://repository/${repo.id}`,
          name: `${repo.name} - Repository Metadata`,
          description: `Repository metadata for ${repo.full_name}`,
          mimeType: 'application/json'
        });

        // Repository context resource
        resources.push({
          uri: `contextkeeper://context/${repo.id}`,
          name: `${repo.name} - Repository Context`,
          description: `AI-powered context and insights for ${repo.full_name}`,
          mimeType: 'text/plain'
        });

        // Repository timeline resource
        resources.push({
          uri: `contextkeeper://timeline/${repo.id}`,
          name: `${repo.name} - Repository Timeline`,
          description: `Recent activity timeline for ${repo.full_name}`,
          mimeType: 'application/json'
        });
      }

      logger.info('Listed resources successfully', { 
        repositoryCount: repositories.length,
        resourceCount: resources.length 
      });

      return resources;

    } catch (error) {
      logger.error('Failed to list resources', {}, error as Error);
      
      // Return empty list on error to maintain MCP protocol compliance
      return [];
    }
  }

  async getResource(uri: string): Promise<ResourceContent> {
    try {
      logger.debug('Getting resource', { uri });

      const parsedUri = this.parseResourceUri(uri);
      if (!parsedUri) {
        throw new Error(`Invalid resource URI format: ${uri}`);
      }

      const { type, repositoryId } = parsedUri;

      switch (type) {
        case 'repository':
          return await this.getRepositoryMetadataResource(repositoryId, uri);
        
        case 'context':
          return await this.getRepositoryContextResource(repositoryId, uri);
        
        case 'timeline':
          return await this.getRepositoryTimelineResource(repositoryId, uri);
        
        default:
          throw new Error(`Unknown resource type: ${type}`);
      }

    } catch (error) {
      logger.error('Failed to get resource', { uri }, error as Error);
      throw error;
    }
  }

  private parseResourceUri(uri: string): { type: string; repositoryId: string } | null {
    // Expected format: contextkeeper://repository/{id}, contextkeeper://context/{id}, contextkeeper://timeline/{id}
    const match = uri.match(/^contextkeeper:\/\/([^\/]+)\/(.+)$/);
    if (!match) {
      return null;
    }

    return {
      type: match[1],
      repositoryId: match[2]
    };
  }

  private async getRepositoryMetadataResource(repositoryId: string, uri: string): Promise<ResourceContent> {
    try {
      // Get repository status and basic info
      const repoStatus = await this.goBackendClient.getRepositoryStatus(parseInt(repositoryId));
      
      // Get repositories list to find the specific repository
      const repositoriesResponse = await this.goBackendClient.getRepositories();
      const repository = repositoriesResponse.repositories.find(repo => repo.id.toString() === repositoryId);

      if (!repository) {
        throw new Error(`Repository not found: ${repositoryId}`);
      }

      // Create repository metadata
      const metadata: RepositoryMetadata = {
        id: repository.id.toString(),
        name: repository.name,
        fullName: repository.full_name,
        description: '', // Not available in current backend interface
        url: `https://github.com/${repository.full_name}`, // Construct from full_name
        defaultBranch: 'main', // Default value
        language: 'Unknown', // Not available in current backend interface
        topics: [], // Not available in current backend interface
        createdAt: repository.created_at,
        updatedAt: repository.updated_at,
        lastIngestionAt: repoStatus.started_at || '',
        status: this.mapRepositoryStatus(repoStatus.status)
      };

      logger.debug('Retrieved repository metadata', { repositoryId, name: repository.name });

      return {
        uri,
        mimeType: 'application/json',
        text: JSON.stringify(metadata, null, 2)
      };

    } catch (error) {
      logger.error('Failed to get repository metadata resource', { repositoryId }, error as Error);
      throw error;
    }
  }

  private async getRepositoryContextResource(repositoryId: string, uri: string): Promise<ResourceContent> {
    try {
      // Get general repository context using a broad query
      const contextQuery = {
        repo_id: parseInt(repositoryId),
        query: 'What is this repository about? What are the main components and recent changes?',
        mode: 'query' as const
      };

      const contextResponse = await this.goBackendClient.queryContext(contextQuery);

      // Format the context as readable text
      let contextText = '';

      if (contextResponse.clarified_goal) {
        contextText += `## Repository Purpose\n${contextResponse.clarified_goal}\n\n`;
      }

      if (contextResponse.context && typeof contextResponse.context === 'object') {
        contextText += `## Repository Context\n`;
        
        // Extract meaningful information from the context object
        const contextObj = contextResponse.context as any;
        if (contextObj.summary) {
          contextText += `${contextObj.summary}\n\n`;
        } else {
          // If no summary, provide a general description
          contextText += `This repository contains code and documentation. Context data is available through the query tools.\n\n`;
        }
      }

      if (contextResponse.tasks && contextResponse.tasks.length > 0) {
        contextText += `## Key Areas\n`;
        contextResponse.tasks.forEach((task, index) => {
          contextText += `${index + 1}. **${task.title}**\n   ${task.acceptance}\n\n`;
        });
      }

      if (contextResponse.questions && contextResponse.questions.length > 0) {
        contextText += `## Common Questions\n`;
        contextResponse.questions.forEach((question, index) => {
          contextText += `${index + 1}. ${question}\n`;
        });
        contextText += '\n';
      }

      if (!contextText.trim()) {
        contextText = `Repository context for ID ${repositoryId}. Use the query_repository_context tool for specific information.`;
      }

      logger.debug('Retrieved repository context', { repositoryId, contextLength: contextText.length });

      return {
        uri,
        mimeType: 'text/plain',
        text: contextText
      };

    } catch (error) {
      logger.error('Failed to get repository context resource', { repositoryId }, error as Error);
      
      // Return a fallback context on error
      return {
        uri,
        mimeType: 'text/plain',
        text: `Repository context for ID ${repositoryId} is currently unavailable. Please try using the query_repository_context tool for specific information.`
      };
    }
  }

  private async getRepositoryTimelineResource(repositoryId: string, uri: string): Promise<ResourceContent> {
    try {
      // Get recent activity by querying for timeline information
      const timelineQuery = {
        repo_id: parseInt(repositoryId),
        query: 'Show me the recent activity, pull requests, issues, and commits in chronological order',
        mode: 'query' as const
      };

      const timelineResponse = await this.goBackendClient.queryContext(timelineQuery);

      // Create timeline events from the response
      const timelineEvents: TimelineEvent[] = [];

      // If we have context data, try to extract timeline information
      if (timelineResponse.context && typeof timelineResponse.context === 'object') {
        const contextObj = timelineResponse.context as any;
        
        // Try to extract timeline data from various possible formats
        if (contextObj.recent_activity) {
          this.extractTimelineFromActivity(contextObj.recent_activity, timelineEvents);
        } else if (contextObj.timeline) {
          this.extractTimelineFromTimeline(contextObj.timeline, timelineEvents);
        } else if (contextObj.events) {
          this.extractTimelineFromEvents(contextObj.events, timelineEvents);
        }
      }

      // If no timeline events found, create a placeholder
      if (timelineEvents.length === 0) {
        timelineEvents.push({
          id: `repo-${repositoryId}-placeholder`,
          type: 'commit',
          title: 'Repository Timeline',
          body: 'Timeline data is available through the query_repository_context tool',
          author: 'System',
          timestamp: new Date().toISOString(),
          url: `#repository-${repositoryId}`
        });
      }

      // Sort timeline events by timestamp (most recent first)
      timelineEvents.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());

      logger.debug('Retrieved repository timeline', { repositoryId, eventCount: timelineEvents.length });

      return {
        uri,
        mimeType: 'application/json',
        text: JSON.stringify(timelineEvents, null, 2)
      };

    } catch (error) {
      logger.error('Failed to get repository timeline resource', { repositoryId }, error as Error);
      
      // Return a fallback timeline on error
      const fallbackTimeline: TimelineEvent[] = [{
        id: `repo-${repositoryId}-error`,
        type: 'commit',
        title: 'Timeline Unavailable',
        body: 'Repository timeline is currently unavailable. Please try using the query_repository_context tool.',
        author: 'System',
        timestamp: new Date().toISOString(),
        url: `#repository-${repositoryId}`
      }];

      return {
        uri,
        mimeType: 'application/json',
        text: JSON.stringify(fallbackTimeline, null, 2)
      };
    }
  }

  private extractTimelineFromActivity(activity: any, events: TimelineEvent[]): void {
    if (Array.isArray(activity)) {
      activity.forEach((item, index) => {
        if (item && typeof item === 'object') {
          events.push(this.createTimelineEventFromObject(item, `activity-${index}`));
        }
      });
    }
  }

  private extractTimelineFromTimeline(timeline: any, events: TimelineEvent[]): void {
    if (Array.isArray(timeline)) {
      timeline.forEach((item, index) => {
        if (item && typeof item === 'object') {
          events.push(this.createTimelineEventFromObject(item, `timeline-${index}`));
        }
      });
    }
  }

  private extractTimelineFromEvents(eventsList: any, events: TimelineEvent[]): void {
    if (Array.isArray(eventsList)) {
      eventsList.forEach((item, index) => {
        if (item && typeof item === 'object') {
          events.push(this.createTimelineEventFromObject(item, `event-${index}`));
        }
      });
    }
  }

  private createTimelineEventFromObject(obj: any, fallbackId: string): TimelineEvent {
    return {
      id: obj.id?.toString() || fallbackId,
      type: this.determineEventType(obj),
      title: obj.title || obj.name || obj.subject || 'Repository Activity',
      body: obj.body || obj.description || obj.message || '',
      author: obj.author || obj.user || obj.creator || 'Unknown',
      timestamp: obj.timestamp || obj.created_at || obj.updated_at || new Date().toISOString(),
      url: obj.url || obj.html_url || `#${fallbackId}`,
      labels: obj.labels || [],
      filesChanged: obj.files_changed || obj.changed_files || []
    };
  }

  private determineEventType(obj: any): 'pr' | 'issue' | 'commit' {
    if (obj.type) {
      const type = obj.type.toLowerCase();
      if (type.includes('pull') || type.includes('pr')) return 'pr';
      if (type.includes('issue')) return 'issue';
      if (type.includes('commit')) return 'commit';
    }
    
    if (obj.pull_request || obj.pr) return 'pr';
    if (obj.issue_number || obj.issue) return 'issue';
    if (obj.sha || obj.commit) return 'commit';
    
    // Default to commit
    return 'commit';
  }

  private mapRepositoryStatus(status: string): "active" | "ingesting" | "error" {
    switch (status.toLowerCase()) {
      case 'completed':
        return 'active';
      case 'running':
      case 'pending':
        return 'ingesting';
      case 'failed':
      case 'partial':
        return 'error';
      default:
        return 'active';
    }
  }
}