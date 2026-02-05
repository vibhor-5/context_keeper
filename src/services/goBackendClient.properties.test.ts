/**
 * Property-based tests for Go Backend Client
 * Feature: contextkeeper-mcp-slack-bot
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import * as fc from 'fast-check';
import axios from 'axios';
import { GoBackendClient } from './goBackendClient';
import { ContextQuery, ContextResponse, RepositoryStatus } from '../types/backend';

// Mock axios
vi.mock('axios');
const mockedAxios = vi.mocked(axios);

describe('Go Backend Client Property-Based Tests', () => {
  let client: GoBackendClient;
  let mockAxiosInstance: any;

  beforeEach(() => {
    vi.clearAllMocks();

    mockAxiosInstance = {
      post: vi.fn(),
      get: vi.fn(),
      defaults: { timeout: 30000 },
      interceptors: {
        request: { use: vi.fn() },
        response: { 
          use: vi.fn((successHandler, errorHandler) => {
            // Store the error handler so we can call it in tests
            mockAxiosInstance._errorHandler = errorHandler;
          })
        }
      },
      _errorHandler: null // Will be set by the interceptor
    };

    mockedAxios.create.mockReturnValue(mockAxiosInstance);

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

  // Feature: contextkeeper-mcp-slack-bot, Property 3: Go Backend API Integration
  // **Validates: Requirements 1.3**
  it('Property 3: Go Backend API Integration - client should make correct REST API calls and return properly formatted responses', async () => {
    await fc.assert(fc.asyncProperty(
      // Generate various context queries
      fc.record({
        repo_id: fc.integer({ min: 1, max: 999999 }),
        query: fc.string({ minLength: 1, maxLength: 1000 }),
        mode: fc.option(fc.constantFrom('query', 'restore', 'clarify'))
      }),
      // Generate various repository IDs
      fc.integer({ min: 1, max: 999999 }),
      async (contextQuery, repoId) => {
        // Property: For any valid context query, the client should make the correct 
        // REST API calls to the Go backend and return properly formatted responses

        // Test context query
        const mockContextResponse: ContextResponse = {
          context: { test: 'data' },
          clarified_goal: 'Test goal',
          tasks: [{ title: 'Test task', acceptance: 'Test acceptance' }],
          questions: ['Test question?']
        };

        mockAxiosInstance.post.mockResolvedValue({
          data: mockContextResponse
        });

        const contextResult = await client.queryContext(contextQuery);

        // Verify correct API call was made
        expect(mockAxiosInstance.post).toHaveBeenCalledWith('/api/context/query', contextQuery);
        
        // Verify response format is correct
        expect(contextResult).toEqual(mockContextResponse);
        expect(typeof contextResult).toBe('object');
        
        // If context exists, it should be an object
        if (contextResult.context) {
          expect(typeof contextResult.context).toBe('object');
        }
        
        // If tasks exist, they should be an array with proper structure
        if (contextResult.tasks) {
          expect(Array.isArray(contextResult.tasks)).toBe(true);
          contextResult.tasks.forEach(task => {
            expect(typeof task.title).toBe('string');
            expect(typeof task.acceptance).toBe('string');
          });
        }

        // Test repository status query
        const mockStatusResponse: RepositoryStatus = {
          id: Math.floor(Math.random() * 1000),
          repo_id: repoId,
          status: fc.sample(fc.constantFrom('pending', 'running', 'completed', 'partial', 'failed'), 1)[0],
          started_at: new Date().toISOString(),
          finished_at: Math.random() > 0.5 ? new Date().toISOString() : undefined
        };

        mockAxiosInstance.get.mockResolvedValue({
          data: mockStatusResponse
        });

        const statusResult = await client.getRepositoryStatus(repoId);

        // Verify correct API call was made
        expect(mockAxiosInstance.get).toHaveBeenCalledWith(`/api/repos/${repoId}/status`);
        
        // Verify response format is correct
        expect(statusResult).toEqual(mockStatusResponse);
        expect(typeof statusResult.id).toBe('number');
        expect(statusResult.repo_id).toBe(repoId);
        expect(['pending', 'running', 'completed', 'partial', 'failed']).toContain(statusResult.status);
        
        // If timestamps exist, they should be strings
        if (statusResult.started_at) {
          expect(typeof statusResult.started_at).toBe('string');
        }
        if (statusResult.finished_at) {
          expect(typeof statusResult.finished_at).toBe('string');
        }
      }
    ), { numRuns: 100 });
  });

  // Additional property test for error handling
  it('Property 3a: Go Backend Error Handling - client should handle various error conditions gracefully', async () => {
    await fc.assert(fc.asyncProperty(
      fc.record({
        repo_id: fc.integer({ min: 1, max: 999999 }),
        query: fc.string({ minLength: 1, maxLength: 100 }),
        mode: fc.option(fc.constantFrom('query', 'restore', 'clarify'))
      }),
      fc.integer({ min: 400, max: 599 }), // HTTP error status codes
      fc.string({ minLength: 1, maxLength: 100 }), // Error message
      async (contextQuery, errorStatus, errorMessage) => {
        // Property: For any error condition, the client should handle it gracefully
        // and provide meaningful error messages

        const mockError = new Error('Request failed');
        (mockError as any).response = {
          status: errorStatus,
          data: {
            error: 'test_error',
            message: errorMessage,
            code: errorStatus
          }
        };

        // Mock the axios call to reject with the error, then simulate the interceptor
        mockAxiosInstance.post.mockImplementation(() => {
          const axiosError = mockError as any;
          // Simulate the response interceptor calling handleApiError
          if (mockAxiosInstance._errorHandler) {
            return Promise.reject(mockAxiosInstance._errorHandler(axiosError));
          }
          return Promise.reject(axiosError);
        });

        try {
          await client.queryContext(contextQuery);
          // Should not reach here
          expect(true).toBe(false);
        } catch (error) {
          // Verify error is properly handled and formatted
          expect(error).toBeInstanceOf(Error);
          expect((error as Error).message).toBeTruthy();
          expect(typeof (error as Error).message).toBe('string');
          
          // Error message should contain meaningful information
          const errorMsg = (error as Error).message;
          expect(errorMsg.length).toBeGreaterThan(0);
          
          // Different status codes should produce appropriate error messages
          if (errorStatus >= 400 && errorStatus < 500) {
            // Client errors should indicate client-side issues
            expect(errorMsg.toLowerCase()).toMatch(/invalid|unauthorized|forbidden|not found|bad request/);
          } else if (errorStatus >= 500) {
            // Server errors should indicate server-side issues
            expect(errorMsg.toLowerCase()).toMatch(/server|internal|unavailable|timeout/);
          }
        }
      }
    ), { numRuns: 100 });
  });

  // Property test for configuration handling
  it('Property 3b: Configuration Management - client should handle various configuration parameters correctly', async () => {
    await fc.assert(fc.property(
      fc.record({
        baseUrl: fc.webUrl(),
        timeout: fc.integer({ min: 1000, max: 120000 }),
        retryAttempts: fc.integer({ min: 1, max: 10 }),
        retryDelay: fc.integer({ min: 100, max: 5000 })
      }),
      (config) => {
        // Property: For any valid configuration, the client should be created successfully
        // and maintain the configuration correctly

        const testClient = new GoBackendClient(config);
        
        // Verify configuration is stored correctly
        const storedConfig = testClient.getConfig();
        expect(storedConfig.baseUrl).toBe(config.baseUrl);
        expect(storedConfig.timeout).toBe(config.timeout);
        expect(storedConfig.retryAttempts).toBe(config.retryAttempts);
        expect(storedConfig.retryDelay).toBe(config.retryDelay);

        // Verify axios instance was created with correct base URL and timeout
        expect(mockedAxios.create).toHaveBeenCalledWith(
          expect.objectContaining({
            baseURL: config.baseUrl,
            timeout: config.timeout
          })
        );

        // Test timeout update
        const newTimeout = fc.sample(fc.integer({ min: 5000, max: 60000 }), 1)[0];
        testClient.setTimeout(newTimeout);
        
        // Verify timeout was updated
        expect(mockAxiosInstance.defaults.timeout).toBe(newTimeout);
      }
    ), { numRuns: 100 });
  });

  // Feature: contextkeeper-mcp-slack-bot, Property 6: Error Message Formatting
  // **Validates: Requirements 1.6, 2.5, 7.1, 7.4**
  it('Property 6: Error Message Formatting - system should return user-friendly error messages for any error condition', async () => {
    await fc.assert(fc.asyncProperty(
      // Generate various error scenarios
      fc.record({
        repo_id: fc.integer({ min: 1, max: 999999 }),
        query: fc.string({ minLength: 1, max: 100 }),
        mode: fc.option(fc.constantFrom('query', 'restore', 'clarify'))
      }),
      fc.oneof(
        // Network errors (no response)
        fc.record({
          type: fc.constant('network'),
          message: fc.string({ minLength: 1, max: 100 }),
          code: fc.constantFrom('ECONNREFUSED', 'ENOTFOUND', 'ETIMEDOUT', 'ECONNRESET')
        }),
        // HTTP errors (with response)
        fc.record({
          type: fc.constant('http'),
          status: fc.integer({ min: 400, max: 599 }),
          message: fc.string({ minLength: 1, max: 100 }),
          data: fc.option(fc.record({
            error: fc.string({ minLength: 1, max: 50 }),
            message: fc.string({ minLength: 1, max: 100 }),
            code: fc.integer({ min: 400, max: 599 })
          }))
        })
      ),
      async (contextQuery, errorScenario) => {
        // Property: For any error condition, the system should return user-friendly 
        // error messages appropriate for the interface

        let mockError: any;
        
        if (errorScenario.type === 'network') {
          // Network error - no response
          mockError = new Error(errorScenario.message);
          (mockError as any).code = errorScenario.code;
        } else {
          // HTTP error - with response
          mockError = new Error(errorScenario.message);
          (mockError as any).response = {
            status: errorScenario.status,
            data: errorScenario.data || {
              error: 'test_error',
              message: errorScenario.message,
              code: errorScenario.status
            }
          };
        }

        mockAxiosInstance.post.mockImplementation(() => {
          const axiosError = mockError as any;
          // Simulate the response interceptor calling handleApiError
          if (mockAxiosInstance._errorHandler) {
            return Promise.reject(mockAxiosInstance._errorHandler(axiosError));
          }
          return Promise.reject(axiosError);
        });
        
        mockAxiosInstance.get.mockImplementation(() => {
          const axiosError = mockError as any;
          // Simulate the response interceptor calling handleApiError
          if (mockAxiosInstance._errorHandler) {
            return Promise.reject(mockAxiosInstance._errorHandler(axiosError));
          }
          return Promise.reject(axiosError);
        });

        try {
          await client.queryContext(contextQuery);
          // Should not reach here
          expect(true).toBe(false);
        } catch (error) {
          // Verify error is properly handled and formatted
          expect(error).toBeInstanceOf(Error);
          const errorMsg = (error as Error).message;
          
          // Error message should be non-empty and meaningful
          expect(errorMsg).toBeTruthy();
          expect(typeof errorMsg).toBe('string');
          expect(errorMsg.length).toBeGreaterThan(0);
          
          // Error messages should be user-friendly (not technical stack traces)
          expect(errorMsg).not.toMatch(/at\s+\w+\s+\(/); // No stack trace
          expect(errorMsg).not.toMatch(/node_modules/); // No internal paths
          expect(errorMsg).not.toMatch(/Error:\s*Error:/); // No double error prefixes
          
          // Error messages should provide context about the failure
          if (errorScenario.type === 'network') {
            // Network errors should indicate connection issues
            expect(errorMsg.toLowerCase()).toMatch(/connection|network|unavailable|failed/);
          } else {
            // HTTP errors should provide appropriate context based on status
            const status = errorScenario.status;
            
            if (status >= 400 && status < 500) {
              // Client errors should indicate client-side issues
              expect(errorMsg.toLowerCase()).toMatch(/invalid|unauthorized|forbidden|not found|bad request/);
            } else if (status >= 500) {
              // Server errors should indicate server-side issues
              expect(errorMsg.toLowerCase()).toMatch(/server|internal|unavailable|timeout/);
            }
            
            // Specific status code handling
            if (status === 401) {
              expect(errorMsg.toLowerCase()).toMatch(/authentication|unauthorized/);
            } else if (status === 403) {
              expect(errorMsg.toLowerCase()).toMatch(/access|forbidden/);
            } else if (status === 404) {
              expect(errorMsg.toLowerCase()).toMatch(/not found|resource/);
            } else if (status === 408 || status === 504) {
              expect(errorMsg.toLowerCase()).toMatch(/timeout/);
            } else if (status === 429) {
              expect(errorMsg.toLowerCase()).toMatch(/rate limit|too many/);
            } else if (status === 500) {
              expect(errorMsg.toLowerCase()).toMatch(/internal.*error|server.*error/);
            } else if (status === 502) {
              expect(errorMsg.toLowerCase()).toMatch(/unavailable|bad gateway/);
            } else if (status === 503) {
              expect(errorMsg.toLowerCase()).toMatch(/service.*unavailable|unavailable/);
            }
          }
          
          // Error messages should not expose sensitive information
          expect(errorMsg).not.toMatch(/password|token|secret|key|auth/i);
          
          // Error messages should be reasonably concise (not too verbose)
          expect(errorMsg.length).toBeLessThan(500);
        }

        // Test repository status error handling as well
        try {
          await client.getRepositoryStatus(contextQuery.repo_id);
          // Should not reach here
          expect(true).toBe(false);
        } catch (error) {
          // Same validation for repository status errors
          expect(error).toBeInstanceOf(Error);
          const errorMsg = (error as Error).message;
          expect(errorMsg).toBeTruthy();
          expect(typeof errorMsg).toBe('string');
          expect(errorMsg.length).toBeGreaterThan(0);
          expect(errorMsg).not.toMatch(/at\s+\w+\s+\(/);
          expect(errorMsg).not.toMatch(/node_modules/);
          expect(errorMsg).not.toMatch(/password|token|secret|key|auth/i);
        }
      }
    ), { numRuns: 100 });
  });
});