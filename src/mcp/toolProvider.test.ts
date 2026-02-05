/**
 * Unit tests for MCP Tool Provider
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { ContextKeeperToolProvider } from './toolProvider';
import { GoBackendClient } from '../services/goBackendClient';
import { RepositoryListResponse, ContextResponse } from '../types/backend';

// Mock the GoBackendClient
vi.mock('../services/goBackendClient');

describe('ContextKeeperToolProvider', () => {
  let toolProvider: ContextKeeperToolProvider;
  let mockGoBackendClient: vi.Mocked<GoBackendClient>;

  beforeEach(() => {
    mockGoBackendClient = {
      getRepositories: vi.fn(),
      queryContext: vi.fn(),
    } as any;

    toolProvider = new ContextKeeperToolProvider(mockGoBackendClient);
  });

  describe('listTools', () => {
    it('should return available tools', async () => {
      const tools = await toolProvider.listTools();

      expect(tools).toHaveLength(2);
      
      // Check query_repository_context tool
      const queryTool = tools.find(t => t.name === 'query_repository_context');
      expect(queryTool).toBeDefined();
      expect(queryTool?.description).toContain('Query repository for specific context');
      expect(queryTool?.inputSchema.properties).toHaveProperty('query');
      expect(queryTool?.inputSchema.properties).toHaveProperty('repositoryId');
      expect(queryTool?.inputSchema.required).toContain('query');

      // Check get_onboarding_summary tool
      const onboardingTool = tools.find(t => t.name === 'get_onboarding_summary');
      expect(onboardingTool).toBeDefined();
      expect(onboardingTool?.description).toContain('Generate comprehensive onboarding summary');
      expect(onboardingTool?.inputSchema.properties).toHaveProperty('repositoryId');
      expect(onboardingTool?.inputSchema.properties).toHaveProperty('focusAreas');
      expect(onboardingTool?.inputSchema.required).toContain('repositoryId');
    });
  });

  describe('callTool', () => {
    describe('query_repository_context', () => {
      it('should query repository context successfully', async () => {
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

        const mockContextResponse: ContextResponse = {
          clarified_goal: 'Understanding the test repository',
          context: { summary: 'This is a test repository for demonstration' },
          tasks: [{ title: 'Main component', acceptance: 'Core functionality' }],
          questions: ['How does it work?']
        };

        mockGoBackendClient.getRepositories.mockResolvedValue(mockRepositories);
        mockGoBackendClient.queryContext.mockResolvedValue(mockContextResponse);

        const result = await toolProvider.callTool('query_repository_context', {
          query: 'What is this repository about?',
          repositoryId: '1'
        });

        expect(result.isError).toBe(false);
        expect(result.content).toHaveLength(1);
        expect(result.content[0].type).toBe('text');
        expect(result.content[0].text).toContain('Query Understanding');
        expect(result.content[0].text).toContain('Understanding the test repository');
        expect(result.content[0].text).toContain('Context Information');
        expect(result.content[0].text).toContain('Key Areas & Action Items');
        expect(result.content[0].text).toContain('Related Questions');
      });

      it('should use default repository when none specified', async () => {
        const mockRepositories: RepositoryListResponse = {
          repositories: [
            {
              id: 1,
              name: 'default-repo',
              full_name: 'owner/default-repo',
              owner: 'owner',
              created_at: '2023-01-01T00:00:00Z',
              updated_at: '2023-01-02T00:00:00Z'
            }
          ]
        };

        const mockContextResponse: ContextResponse = {
          context: { summary: 'Default repository context' }
        };

        mockGoBackendClient.getRepositories.mockResolvedValue(mockRepositories);
        mockGoBackendClient.queryContext.mockResolvedValue(mockContextResponse);

        const result = await toolProvider.callTool('query_repository_context', {
          query: 'What is this about?'
        });

        expect(result.isError).toBe(false);
        expect(mockGoBackendClient.queryContext).toHaveBeenCalledWith({
          repo_id: 1,
          query: 'What is this about?',
          mode: 'query'
        });
      });

      it('should handle no repositories available', async () => {
        const mockRepositories: RepositoryListResponse = {
          repositories: []
        };

        mockGoBackendClient.getRepositories.mockResolvedValue(mockRepositories);

        const result = await toolProvider.callTool('query_repository_context', {
          query: 'What is this about?'
        });

        expect(result.isError).toBe(false);
        expect(result.content[0].text).toContain('No repositories are currently available');
      });

      it('should validate required parameters', async () => {
        const result = await toolProvider.callTool('query_repository_context', {});

        expect(result.isError).toBe(true);
        expect(result.content[0].text).toContain('Missing or invalid required parameter: query');
      });

      it('should handle empty query', async () => {
        const result = await toolProvider.callTool('query_repository_context', {
          query: '   '
        });

        expect(result.isError).toBe(true);
        expect(result.content[0].text).toContain('Query cannot be empty');
      });
    });

    describe('get_onboarding_summary', () => {
      it('should generate onboarding summary successfully', async () => {
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

        const mockContextResponse: ContextResponse = {
          clarified_goal: 'This repository provides testing functionality',
          context: { summary: 'A comprehensive testing framework' }
        };

        mockGoBackendClient.getRepositories.mockResolvedValue(mockRepositories);
        mockGoBackendClient.queryContext.mockResolvedValue(mockContextResponse);

        const result = await toolProvider.callTool('get_onboarding_summary', {
          repositoryId: '1'
        });

        expect(result.isError).toBe(false);
        expect(result.content).toHaveLength(1);
        expect(result.content[0].type).toBe('text');
        expect(result.content[0].text).toContain('Onboarding Guide: test-repo');
        expect(result.content[0].text).toContain('**Repository:** owner/test-repo');
        expect(result.content[0].text).toContain('Getting Started Checklist');
        expect(result.content[0].text).toContain('Need More Information?');
      });

      it('should validate required parameters', async () => {
        const result = await toolProvider.callTool('get_onboarding_summary', {});

        expect(result.isError).toBe(true);
        expect(result.content[0].text).toContain('Missing required parameter: repositoryId');
      });

      it('should handle repository not found', async () => {
        const mockRepositories: RepositoryListResponse = {
          repositories: []
        };

        mockGoBackendClient.getRepositories.mockResolvedValue(mockRepositories);

        const result = await toolProvider.callTool('get_onboarding_summary', {
          repositoryId: '999'
        });

        expect(result.isError).toBe(true);
        expect(result.content[0].text).toContain('Repository not found: 999');
      });
    });

    it('should handle unknown tool', async () => {
      const result = await toolProvider.callTool('unknown_tool', {});

      expect(result.isError).toBe(true);
      expect(result.content[0].text).toContain('Unknown tool: unknown_tool');
    });
  });
});