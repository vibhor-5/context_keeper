/**
 * Unit tests for MCP Resource Provider
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { ContextKeeperResourceProvider } from './resourceProvider';
import { GoBackendClient } from '../services/goBackendClient';
import { RepositoryListResponse, RepositoryStatus } from '../types/backend';

// Mock the GoBackendClient
vi.mock('../services/goBackendClient');

describe('ContextKeeperResourceProvider', () => {
  let resourceProvider: ContextKeeperResourceProvider;
  let mockGoBackendClient: vi.Mocked<GoBackendClient>;

  beforeEach(() => {
    mockGoBackendClient = {
      getRepositories: vi.fn(),
      getRepositoryStatus: vi.fn(),
      queryContext: vi.fn(),
    } as any;

    resourceProvider = new ContextKeeperResourceProvider(mockGoBackendClient);
  });

  describe('listResources', () => {
    it('should return resources for available repositories', async () => {
      const mockRepositories: RepositoryListResponse = {
        repositories: [
          {
            id: 1,
            name: 'test-repo',
            full_name: 'owner/test-repo',
            owner: 'owner',
            created_at: '2023-01-01T00:00:00Z',
            updated_at: '2023-01-02T00:00:00Z'
          }
        ]
      };

      mockGoBackendClient.getRepositories.mockResolvedValue(mockRepositories);

      const resources = await resourceProvider.listResources();

      expect(resources).toHaveLength(3); // metadata, context, timeline
      expect(resources[0]).toEqual({
        uri: 'contextkeeper://repository/1',
        name: 'test-repo - Repository Metadata',
        description: 'Repository metadata for owner/test-repo',
        mimeType: 'application/json'
      });
      expect(resources[1]).toEqual({
        uri: 'contextkeeper://context/1',
        name: 'test-repo - Repository Context',
        description: 'AI-powered context and insights for owner/test-repo',
        mimeType: 'text/plain'
      });
      expect(resources[2]).toEqual({
        uri: 'contextkeeper://timeline/1',
        name: 'test-repo - Repository Timeline',
        description: 'Recent activity timeline for owner/test-repo',
        mimeType: 'application/json'
      });
    });

    it('should return empty array on error', async () => {
      mockGoBackendClient.getRepositories.mockRejectedValue(new Error('Backend error'));

      const resources = await resourceProvider.listResources();

      expect(resources).toEqual([]);
    });
  });

  describe('getResource', () => {
    it('should return repository metadata resource', async () => {
      const mockRepositories: RepositoryListResponse = {
        repositories: [
          {
            id: 1,
            name: 'test-repo',
            full_name: 'owner/test-repo',
            owner: 'owner',
            created_at: '2023-01-01T00:00:00Z',
            updated_at: '2023-01-02T00:00:00Z'
          }
        ]
      };

      const mockStatus: RepositoryStatus = {
        id: 1,
        repo_id: 1,
        status: 'completed',
        started_at: '2023-01-01T00:00:00Z'
      };

      mockGoBackendClient.getRepositories.mockResolvedValue(mockRepositories);
      mockGoBackendClient.getRepositoryStatus.mockResolvedValue(mockStatus);

      const resource = await resourceProvider.getResource('contextkeeper://repository/1');

      expect(resource.uri).toBe('contextkeeper://repository/1');
      expect(resource.mimeType).toBe('application/json');
      expect(resource.text).toBeTruthy();

      const metadata = JSON.parse(resource.text!);
      expect(metadata.id).toBe('1');
      expect(metadata.name).toBe('test-repo');
      expect(metadata.fullName).toBe('owner/test-repo');
      expect(metadata.status).toBe('active');
    });

    it('should return context resource', async () => {
      const mockContextResponse = {
        clarified_goal: 'Test repository purpose',
        context: { summary: 'This is a test repository' },
        tasks: [{ title: 'Test task', acceptance: 'Test acceptance' }],
        questions: ['What does this do?']
      };

      mockGoBackendClient.queryContext.mockResolvedValue(mockContextResponse);

      const resource = await resourceProvider.getResource('contextkeeper://context/1');

      expect(resource.uri).toBe('contextkeeper://context/1');
      expect(resource.mimeType).toBe('text/plain');
      expect(resource.text).toContain('Repository Purpose');
      expect(resource.text).toContain('Test repository purpose');
      expect(resource.text).toContain('Key Areas');
      expect(resource.text).toContain('Common Questions');
    });

    it('should return timeline resource', async () => {
      const mockTimelineResponse = {
        context: {
          recent_activity: [
            {
              id: 'test-1',
              type: 'commit',
              title: 'Test commit',
              author: 'testuser',
              timestamp: '2023-01-01T00:00:00Z',
              url: 'https://github.com/test'
            }
          ]
        }
      };

      mockGoBackendClient.queryContext.mockResolvedValue(mockTimelineResponse);

      const resource = await resourceProvider.getResource('contextkeeper://timeline/1');

      expect(resource.uri).toBe('contextkeeper://timeline/1');
      expect(resource.mimeType).toBe('application/json');
      expect(resource.text).toBeTruthy();

      const timeline = JSON.parse(resource.text!);
      expect(Array.isArray(timeline)).toBe(true);
      expect(timeline.length).toBeGreaterThan(0);
      expect(timeline[0]).toHaveProperty('id');
      expect(timeline[0]).toHaveProperty('type');
      expect(timeline[0]).toHaveProperty('title');
    });

    it('should throw error for invalid URI', async () => {
      await expect(resourceProvider.getResource('invalid://uri')).rejects.toThrow('Invalid resource URI format');
    });

    it('should throw error for unknown resource type', async () => {
      await expect(resourceProvider.getResource('contextkeeper://unknown/1')).rejects.toThrow('Unknown resource type');
    });
  });
});