/**
 * Unit tests for Go Backend Client
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import axios from 'axios';
import { GoBackendClient } from './goBackendClient';
import { ContextQuery, ContextResponse, RepositoryStatus } from '../types/backend';

// Mock axios
vi.mock('axios');
const mockedAxios = vi.mocked(axios);

describe('GoBackendClient', () => {
  let client: GoBackendClient;
  let mockAxiosInstance: any;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Create mock axios instance
    mockAxiosInstance = {
      post: vi.fn(),
      get: vi.fn(),
      defaults: { timeout: 30000 },
      interceptors: {
        request: { use: vi.fn() },
        response: { use: vi.fn() }
      }
    };

    // Mock axios.create to return our mock instance
    mockedAxios.create.mockReturnValue(mockAxiosInstance);

    // Create client instance
    client = new GoBackendClient({
      baseUrl: 'http://localhost:8080',
      timeout: 30000,
      retryAttempts: 2,
      retryDelay: 100
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('constructor', () => {
    it('should create axios instance with correct config', () => {
      expect(mockedAxios.create).toHaveBeenCalledWith({
        baseURL: 'http://localhost:8080',
        timeout: 30000,
        headers: {
          'Content-Type': 'application/json',
          'Accept': 'application/json'
        }
      });
    });

    it('should set up request and response interceptors', () => {
      expect(mockAxiosInstance.interceptors.request.use).toHaveBeenCalled();
      expect(mockAxiosInstance.interceptors.response.use).toHaveBeenCalled();
    });
  });

  describe('queryContext', () => {
    it('should successfully query context', async () => {
      const query: ContextQuery = {
        repo_id: 123,
        query: 'test query',
        mode: 'query'
      };

      const expectedResponse: ContextResponse = {
        context: { test: 'data' },
        clarified_goal: 'Test goal'
      };

      mockAxiosInstance.post.mockResolvedValue({
        data: expectedResponse
      });

      const result = await client.queryContext(query);

      expect(mockAxiosInstance.post).toHaveBeenCalledWith('/api/context/query', query);
      expect(result).toEqual(expectedResponse);
    });

    it('should handle context query errors', async () => {
      const query: ContextQuery = {
        repo_id: 123,
        query: 'test query'
      };

      const error = new Error('Network error');
      mockAxiosInstance.post.mockRejectedValue(error);

      await expect(client.queryContext(query)).rejects.toThrow('Network error');
    });
  });

  describe('getRepositoryStatus', () => {
    it('should successfully get repository status', async () => {
      const repoId = 123;
      const expectedStatus: RepositoryStatus = {
        id: 1,
        repo_id: repoId,
        status: 'completed',
        started_at: '2023-01-01T00:00:00Z',
        finished_at: '2023-01-01T01:00:00Z'
      };

      mockAxiosInstance.get.mockResolvedValue({
        data: expectedStatus
      });

      const result = await client.getRepositoryStatus(repoId);

      expect(mockAxiosInstance.get).toHaveBeenCalledWith(`/api/repos/${repoId}/status`);
      expect(result).toEqual(expectedStatus);
    });

    it('should handle repository status errors', async () => {
      const repoId = 123;
      const error = new Error('Repository not found');
      mockAxiosInstance.get.mockRejectedValue(error);

      await expect(client.getRepositoryStatus(repoId)).rejects.toThrow('Repository not found');
    });
  });

  describe('getRepositories', () => {
    it('should successfully get repositories without auth token', async () => {
      const expectedResponse = {
        repositories: [
          { id: 1, name: 'test-repo', full_name: 'user/test-repo', owner: 'user' }
        ]
      };

      mockAxiosInstance.get.mockResolvedValue({
        data: expectedResponse
      });

      const result = await client.getRepositories();

      expect(mockAxiosInstance.get).toHaveBeenCalledWith('/api/repos', {});
      expect(result).toEqual(expectedResponse);
    });

    it('should successfully get repositories with auth token', async () => {
      const authToken = 'test-token';
      const expectedResponse = {
        repositories: [
          { id: 1, name: 'test-repo', full_name: 'user/test-repo', owner: 'user' }
        ]
      };

      mockAxiosInstance.get.mockResolvedValue({
        data: expectedResponse
      });

      const result = await client.getRepositories(authToken);

      expect(mockAxiosInstance.get).toHaveBeenCalledWith('/api/repos', {
        headers: { Authorization: `Bearer ${authToken}` }
      });
      expect(result).toEqual(expectedResponse);
    });
  });

  describe('ingestRepository', () => {
    it('should successfully trigger repository ingestion', async () => {
      const request = { repo_id: 123 };
      const expectedResponse = {
        job_id: 456,
        status: 'pending' as const
      };

      mockAxiosInstance.post.mockResolvedValue({
        data: expectedResponse
      });

      const result = await client.ingestRepository(request);

      expect(mockAxiosInstance.post).toHaveBeenCalledWith('/api/repos/ingest', request, {});
      expect(result).toEqual(expectedResponse);
    });

    it('should include auth token when provided', async () => {
      const request = { repo_id: 123 };
      const authToken = 'test-token';
      const expectedResponse = {
        job_id: 456,
        status: 'pending' as const
      };

      mockAxiosInstance.post.mockResolvedValue({
        data: expectedResponse
      });

      const result = await client.ingestRepository(request, authToken);

      expect(mockAxiosInstance.post).toHaveBeenCalledWith('/api/repos/ingest', request, {
        headers: { Authorization: `Bearer ${authToken}` }
      });
      expect(result).toEqual(expectedResponse);
    });
  });

  describe('healthCheck', () => {
    it('should return true when backend is healthy', async () => {
      mockAxiosInstance.get.mockResolvedValue({ data: {} });

      const result = await client.healthCheck();

      expect(result).toBe(true);
      expect(mockAxiosInstance.get).toHaveBeenCalledWith('/api/repos', { timeout: 5000 });
    });

    it('should return false when backend is unhealthy', async () => {
      mockAxiosInstance.get.mockRejectedValue(new Error('Connection failed'));

      const result = await client.healthCheck();

      expect(result).toBe(false);
    });
  });

  describe('configuration', () => {
    it('should return current configuration', () => {
      const config = client.getConfig();

      expect(config).toEqual({
        baseUrl: 'http://localhost:8080',
        timeout: 30000,
        retryAttempts: 2,
        retryDelay: 100
      });
    });

    it('should update timeout', () => {
      client.setTimeout(60000);

      expect(mockAxiosInstance.defaults.timeout).toBe(60000);
    });
  });

  describe('retry logic', () => {
    it('should retry on network errors', async () => {
      const query: ContextQuery = {
        repo_id: 123,
        query: 'test query'
      };

      // First call fails with network error, second succeeds
      mockAxiosInstance.post
        .mockRejectedValueOnce(new Error('Network error'))
        .mockResolvedValueOnce({ data: { context: {} } });

      const result = await client.queryContext(query);

      expect(mockAxiosInstance.post).toHaveBeenCalledTimes(2);
      expect(result).toEqual({ context: {} });
    });

    it('should not retry on client errors (4xx)', async () => {
      const query: ContextQuery = {
        repo_id: 123,
        query: 'test query'
      };

      const error = {
        response: { status: 400, data: { message: 'Bad request' } },
        message: 'Request failed'
      };

      mockAxiosInstance.post.mockRejectedValue(error);

      await expect(client.queryContext(query)).rejects.toThrow();
      expect(mockAxiosInstance.post).toHaveBeenCalledTimes(1);
    });
  });
});