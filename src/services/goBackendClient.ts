/**
 * Go Backend REST API Client
 * Handles communication with the existing Go backend service
 */

import axios, { AxiosInstance, AxiosError, AxiosRequestConfig } from 'axios';
import { 
  ContextQuery, 
  ContextResponse, 
  RepositoryStatus, 
  ErrorResponse,
  RepositoryListResponse,
  IngestRepoRequest,
  IngestRepoResponse
} from '../types/backend';
import { logger } from '../utils/logger';

export interface GoBackendClientConfig {
  baseUrl: string;
  timeout: number;
  retryAttempts?: number;
  retryDelay?: number;
}

export class GoBackendClient {
  private client: AxiosInstance;
  private config: GoBackendClientConfig;

  constructor(config: GoBackendClientConfig) {
    this.config = {
      retryAttempts: 3,
      retryDelay: 1000,
      ...config
    };

    this.client = axios.create({
      baseURL: this.config.baseUrl,
      timeout: this.config.timeout,
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json'
      }
    });

    // Add request interceptor for logging
    this.client.interceptors.request.use(
      (config) => {
        logger.debug('Go Backend API request', {
          method: config.method?.toUpperCase(),
          url: config.url,
          baseURL: config.baseURL
        });
        return config;
      },
      (error) => {
        logger.error('Go Backend API request error', {}, error);
        return Promise.reject(error);
      }
    );

    // Add response interceptor for logging and error handling
    this.client.interceptors.response.use(
      (response) => {
        logger.debug('Go Backend API response', {
          status: response.status,
          url: response.config.url
        });
        return response;
      },
      (error: AxiosError) => {
        logger.error('Go Backend API response error', {
          status: error.response?.status,
          url: error.config?.url,
          message: error.message
        });
        return Promise.reject(this.handleApiError(error));
      }
    );
  }

  /**
   * Query repository context using the Go backend AI service
   */
  async queryContext(query: ContextQuery): Promise<ContextResponse> {
    try {
      logger.info('Querying repository context', {
        repoId: query.repo_id,
        mode: query.mode,
        queryLength: query.query.length
      });

      const response = await this.executeWithRetry(() =>
        this.client.post<ContextResponse>('/api/context/query', query)
      );

      logger.info('Context query successful', {
        repoId: query.repo_id,
        hasContext: !!response.data.context,
        hasTasks: !!response.data.tasks?.length,
        hasQuestions: !!response.data.questions?.length
      });

      return response.data;
    } catch (error) {
      logger.error('Context query failed', { repoId: query.repo_id }, error as Error);
      throw error;
    }
  }

  /**
   * Get repository ingestion status
   */
  async getRepositoryStatus(repoId: number): Promise<RepositoryStatus> {
    try {
      logger.info('Getting repository status', { repoId });

      const response = await this.executeWithRetry(() =>
        this.client.get<RepositoryStatus>(`/api/repos/${repoId}/status`)
      );

      logger.info('Repository status retrieved', {
        repoId,
        status: response.data.status,
        jobId: response.data.id
      });

      return response.data;
    } catch (error) {
      logger.error('Failed to get repository status', { repoId }, error as Error);
      throw error;
    }
  }

  /**
   * Get list of repositories for authenticated user
   */
  async getRepositories(authToken?: string): Promise<RepositoryListResponse> {
    try {
      logger.info('Getting repositories list');

      const config: AxiosRequestConfig = {};
      if (authToken) {
        config.headers = { Authorization: `Bearer ${authToken}` };
      }

      const response = await this.executeWithRetry(() =>
        this.client.get<RepositoryListResponse>('/api/repos', config)
      );

      logger.info('Repositories list retrieved', {
        count: response.data.repositories.length
      });

      return response.data;
    } catch (error) {
      logger.error('Failed to get repositories', {}, error as Error);
      throw error;
    }
  }

  /**
   * Trigger repository ingestion
   */
  async ingestRepository(request: IngestRepoRequest, authToken?: string): Promise<IngestRepoResponse> {
    try {
      logger.info('Triggering repository ingestion', { repoId: request.repo_id });

      const config: AxiosRequestConfig = {};
      if (authToken) {
        config.headers = { Authorization: `Bearer ${authToken}` };
      }

      const response = await this.executeWithRetry(() =>
        this.client.post<IngestRepoResponse>('/api/repos/ingest', request, config)
      );

      logger.info('Repository ingestion triggered', {
        repoId: request.repo_id,
        jobId: response.data.job_id,
        status: response.data.status
      });

      return response.data;
    } catch (error) {
      logger.error('Failed to trigger repository ingestion', { repoId: request.repo_id }, error as Error);
      throw error;
    }
  }

  /**
   * Check if the Go backend is healthy
   */
  async healthCheck(): Promise<boolean> {
    try {
      logger.debug('Performing Go backend health check');

      // Try a simple request to check if the service is available
      await this.client.get('/api/repos', { timeout: 5000 });
      
      logger.debug('Go backend health check passed');
      return true;
    } catch (error) {
      logger.warn('Go backend health check failed', {}, error as Error);
      return false;
    }
  }

  /**
   * Execute a request with retry logic
   */
  private async executeWithRetry<T>(
    operation: () => Promise<T>,
    attempt: number = 1
  ): Promise<T> {
    try {
      return await operation();
    } catch (error) {
      if (attempt >= (this.config.retryAttempts || 3)) {
        throw error;
      }

      const isRetryableError = this.isRetryableError(error as AxiosError);
      if (!isRetryableError) {
        throw error;
      }

      const delay = this.config.retryDelay! * Math.pow(2, attempt - 1); // Exponential backoff
      logger.warn(`Go Backend API request failed, retrying in ${delay}ms`, {
        attempt,
        maxAttempts: this.config.retryAttempts,
        error: (error as Error).message
      });

      await new Promise(resolve => setTimeout(resolve, delay));
      return this.executeWithRetry(operation, attempt + 1);
    }
  }

  /**
   * Determine if an error is retryable
   */
  private isRetryableError(error: AxiosError): boolean {
    // Retry on network errors
    if (!error.response) {
      return true;
    }

    // Retry on server errors (5xx) but not client errors (4xx)
    const status = error.response.status;
    return status >= 500 || status === 408 || status === 429;
  }

  /**
   * Handle and transform API errors
   */
  private handleApiError(error: AxiosError): Error {
    if (!error.response) {
      // Network error
      return new Error(`Go Backend connection failed: ${error.message}`);
    }

    const status = error.response.status;
    const errorData = error.response.data as ErrorResponse;

    // Create descriptive error messages based on status codes
    switch (status) {
      case 400:
        return new Error(`Invalid request: ${errorData?.message || error.message}`);
      case 401:
        return new Error(`Authentication required: ${errorData?.message || 'Unauthorized'}`);
      case 403:
        return new Error(`Access forbidden: ${errorData?.message || 'Forbidden'}`);
      case 404:
        return new Error(`Resource not found: ${errorData?.message || 'Not found'}`);
      case 408:
        return new Error(`Request timeout: ${errorData?.message || 'Request timed out'}`);
      case 429:
        return new Error(`Rate limit exceeded: ${errorData?.message || 'Too many requests'}`);
      case 500:
        return new Error(`Go Backend internal error: ${errorData?.message || 'Internal server error'}`);
      case 502:
        return new Error(`Go Backend unavailable: ${errorData?.message || 'Bad gateway'}`);
      case 503:
        return new Error(`Go Backend service unavailable: ${errorData?.message || 'Service unavailable'}`);
      case 504:
        return new Error(`Go Backend timeout: ${errorData?.message || 'Gateway timeout'}`);
      default:
        return new Error(`Go Backend error (${status}): ${errorData?.message || error.message}`);
    }
  }

  /**
   * Get client configuration
   */
  getConfig(): GoBackendClientConfig {
    return { ...this.config };
  }

  /**
   * Update client timeout
   */
  setTimeout(timeout: number): void {
    this.config.timeout = timeout;
    this.client.defaults.timeout = timeout;
  }
}