/**
 * Unit tests for Slack MCP Client
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import axios from 'axios';
import { SlackMCPClient } from './mcpClient';
import { MCPResponse } from '../types/mcp';

// Mock axios
vi.mock('axios');
const mockedAxios = vi.mocked(axios);

describe('SlackMCPClient', () => {
  let client: SlackMCPClient;
  let mockAxiosInstance: any;

  beforeEach(() => {
    vi.clearAllMocks();

    mockAxiosInstance = {
      post: vi.fn(),
      get: vi.fn(),
      defaults: { timeout: 30000 },
      interceptors: {
        request: { use: vi.fn() },
        response: { use: vi.fn() }
      }
    };

    mockedAxios.create.mockReturnValue(mockAxiosInstance);

    client = new SlackMCPClient({
      mcpServerUrl: 'http://localhost:3001',
      timeout: 30000,
      retryAttempts: 2,
      retryDelay: 100
    });
  });

  describe('queryContext', () => {
    it('should query context successfully', async () => {
      const mockResponse: MCPResponse = {
        result: {
          content: [{
            type: 'text',
            text: 'This is the repository context information.'
          }],
          isError: false
        },
        id: 'test-123'
      };

      mockAxiosInstance.post.mockResolvedValue({
        data: mockResponse
      });

      const result = await client.queryContext('What is this repository about?', '1');

      expect(result).toBe('This is the repository context information.');
      expect(mockAxiosInstance.post).toHaveBeenCalledWith('/mcp', {
        method: 'call_tool',
        params: {
          name: 'query_repository_context',
          arguments: {
            query: 'What is this repository about?',
            repositoryId: '1',
            limit: 10
          }
        },
        id: expect.any(String)
      });
    });

    it('should handle MCP error response', async () => {
      const mockResponse: MCPResponse = {
        error: {
          code: -32603,
          message: 'Internal error',
          data: { details: 'Backend unavailable' }
        },
        id: 'test-123'
      };

      mockAxiosInstance.post.mockResolvedValue({
        data: mockResponse
      });

      await expect(client.queryContext('test query')).rejects.toThrow('MCP Error: Internal error');
    });

    it('should handle missing content in response', async () => {
      const mockResponse: MCPResponse = {
        result: {},
        id: 'test-123'
      };

      mockAxiosInstance.post.mockResolvedValue({
        data: mockResponse
      });

      await expect(client.queryContext('test query')).rejects.toThrow('Invalid MCP response: missing content');
    });

    it('should query without repository ID', async () => {
      const mockResponse: MCPResponse = {
        result: {
          content: [{
            type: 'text',
            text: 'General context information.'
          }],
          isError: false
        },
        id: 'test-123'
      };

      mockAxiosInstance.post.mockResolvedValue({
        data: mockResponse
      });

      const result = await client.queryContext('What is this about?');

      expect(result).toBe('General context information.');
      expect(mockAxiosInstance.post).toHaveBeenCalledWith('/mcp', {
        method: 'call_tool',
        params: {
          name: 'query_repository_context',
          arguments: {
            query: 'What is this about?',
            repositoryId: undefined,
            limit: 10
          }
        },
        id: expect.any(String)
      });
    });
  });

  describe('getOnboardingSummary', () => {
    it('should get onboarding summary successfully', async () => {
      const mockResponse: MCPResponse = {
        result: {
          content: [{
            type: 'text',
            text: '# Onboarding Guide\n\nWelcome to the repository!'
          }],
          isError: false
        },
        id: 'test-123'
      };

      mockAxiosInstance.post.mockResolvedValue({
        data: mockResponse
      });

      const result = await client.getOnboardingSummary('1');

      expect(result).toBe('# Onboarding Guide\n\nWelcome to the repository!');
      expect(mockAxiosInstance.post).toHaveBeenCalledWith('/mcp', {
        method: 'call_tool',
        params: {
          name: 'get_onboarding_summary',
          arguments: {
            repositoryId: '1',
            focusAreas: ['overview', 'architecture', 'getting-started', 'contributing']
          }
        },
        id: expect.any(String)
      });
    });

    it('should handle onboarding summary error', async () => {
      const mockResponse: MCPResponse = {
        error: {
          code: -32602,
          message: 'Invalid params',
          data: { details: 'Repository not found' }
        },
        id: 'test-123'
      };

      mockAxiosInstance.post.mockResolvedValue({
        data: mockResponse
      });

      await expect(client.getOnboardingSummary('999')).rejects.toThrow('MCP Error: Invalid params');
    });
  });

  describe('getRecentActivity', () => {
    it('should get recent activity successfully', async () => {
      const mockResponse: MCPResponse = {
        result: {
          content: [{
            type: 'text',
            text: 'Recent activity:\n- PR #123 merged\n- Issue #456 opened'
          }],
          isError: false
        },
        id: 'test-123'
      };

      mockAxiosInstance.post.mockResolvedValue({
        data: mockResponse
      });

      const result = await client.getRecentActivity('1', 7);

      expect(result).toBe('Recent activity:\n- PR #123 merged\n- Issue #456 opened');
      expect(mockAxiosInstance.post).toHaveBeenCalledWith('/mcp', {
        method: 'call_tool',
        params: {
          name: 'query_repository_context',
          arguments: {
            query: 'Show me the recent activity and changes in this repository from the last 7 days',
            repositoryId: '1',
            limit: 20
          }
        },
        id: expect.any(String)
      });
    });

    it('should use default days parameter', async () => {
      const mockResponse: MCPResponse = {
        result: {
          content: [{
            type: 'text',
            text: 'Recent activity for 7 days'
          }],
          isError: false
        },
        id: 'test-123'
      };

      mockAxiosInstance.post.mockResolvedValue({
        data: mockResponse
      });

      await client.getRecentActivity('1');

      expect(mockAxiosInstance.post).toHaveBeenCalledWith('/mcp', expect.objectContaining({
        params: expect.objectContaining({
          arguments: expect.objectContaining({
            query: expect.stringContaining('last 7 days')
          })
        })
      }));
    });
  });

  describe('getSystemStatus', () => {
    it('should get system status successfully', async () => {
      // Mock health check
      mockAxiosInstance.get.mockResolvedValueOnce({
        data: { status: 'healthy', version: '1.0.0' }
      });

      // Mock resources list
      const mockResourcesResponse: MCPResponse = {
        result: {
          resources: [
            { uri: 'contextkeeper://repository/1', name: 'repo1' },
            { uri: 'contextkeeper://repository/2', name: 'repo2' },
            { uri: 'contextkeeper://context/1', name: 'context1' }
          ]
        },
        id: 'test-123'
      };

      mockAxiosInstance.post.mockResolvedValue({
        data: mockResourcesResponse
      });

      const result = await client.getSystemStatus();

      expect(result.mcpServer.status).toBe('healthy');
      expect(result.mcpServer.version).toBe('1.0.0');
      expect(result.goBackend.status).toBe('healthy');
      expect(result.repositories.total).toBe(2); // Only repository resources
      expect(result.repositories.active).toBe(2);
    });

    it('should handle system status errors gracefully', async () => {
      mockAxiosInstance.get.mockRejectedValue(new Error('Connection failed'));

      const result = await client.getSystemStatus();

      expect(result.mcpServer.status).toBe('down');
      expect(result.goBackend.status).toBe('down');
      expect(result.repositories.total).toBe(0);
    });

    it('should handle backend degraded status', async () => {
      // Mock health check success
      mockAxiosInstance.get.mockResolvedValueOnce({
        data: { status: 'healthy', version: '1.0.0' }
      });

      // Mock resources list with error
      const mockResourcesResponse: MCPResponse = {
        error: {
          code: -32603,
          message: 'Backend unavailable'
        },
        id: 'test-123'
      };

      mockAxiosInstance.post.mockResolvedValue({
        data: mockResourcesResponse
      });

      const result = await client.getSystemStatus();

      expect(result.mcpServer.status).toBe('healthy');
      expect(result.goBackend.status).toBe('degraded');
    });
  });

  describe('healthCheck', () => {
    it('should return true when MCP server is healthy', async () => {
      mockAxiosInstance.get.mockResolvedValue({
        data: { status: 'healthy' }
      });

      const result = await client.healthCheck();

      expect(result).toBe(true);
      expect(mockAxiosInstance.get).toHaveBeenCalledWith('/health', { timeout: 5000 });
    });

    it('should return false when MCP server is unhealthy', async () => {
      mockAxiosInstance.get.mockRejectedValue(new Error('Connection failed'));

      const result = await client.healthCheck();

      expect(result).toBe(false);
    });
  });

  describe('configuration', () => {
    it('should return client configuration', () => {
      const config = client.getConfig();

      expect(config.mcpServerUrl).toBe('http://localhost:3001');
      expect(config.timeout).toBe(30000);
      expect(config.retryAttempts).toBe(2);
      expect(config.retryDelay).toBe(100);
    });

    it('should update timeout', () => {
      client.setTimeout(60000);

      expect(mockAxiosInstance.defaults.timeout).toBe(60000);
    });
  });
});