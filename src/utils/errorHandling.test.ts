/**
 * Error handling utilities tests
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  classifyError,
  executeWithRetry,
  executeWithTimeout,
  CircuitBreaker,
  handlePartialData,
  createSlackErrorMessage,
  ErrorType,
  DEFAULT_RETRY_CONFIG
} from './errorHandling';

describe('Error Handling Utilities', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('classifyError', () => {
    it('should classify network errors correctly', () => {
      const networkError = { code: 'ECONNREFUSED', message: 'Connection refused' };
      const classified = classifyError(networkError);

      expect(classified.type).toBe(ErrorType.NETWORK);
      expect(classified.isRetryable).toBe(true);
      expect(classified.userFriendlyMessage).toContain('Unable to connect');
    });

    it('should classify timeout errors correctly', () => {
      const timeoutError = { code: 'ECONNABORTED', message: 'Request aborted' };
      const classified = classifyError(timeoutError);

      expect(classified.type).toBe(ErrorType.TIMEOUT);
      expect(classified.isRetryable).toBe(true);
      expect(classified.userFriendlyMessage).toContain('timed out');
    });

    it('should classify HTTP 401 errors correctly', () => {
      const authError = { 
        response: { status: 401 }, 
        message: 'Unauthorized' 
      };
      const classified = classifyError(authError);

      expect(classified.type).toBe(ErrorType.AUTHENTICATION);
      expect(classified.isRetryable).toBe(false);
      expect(classified.statusCode).toBe(401);
      expect(classified.userFriendlyMessage).toContain('Authentication required');
    });

    it('should classify HTTP 404 errors correctly', () => {
      const notFoundError = { 
        response: { status: 404 }, 
        message: 'Not found' 
      };
      const classified = classifyError(notFoundError);

      expect(classified.type).toBe(ErrorType.NOT_FOUND);
      expect(classified.isRetryable).toBe(false);
      expect(classified.statusCode).toBe(404);
    });

    it('should classify HTTP 500 errors as retryable', () => {
      const serverError = { 
        response: { status: 500 }, 
        message: 'Internal server error' 
      };
      const classified = classifyError(serverError);

      expect(classified.type).toBe(ErrorType.SERVER_ERROR);
      expect(classified.isRetryable).toBe(true);
      expect(classified.statusCode).toBe(500);
    });

    it('should classify HTTP 503 errors as service unavailable', () => {
      const serviceError = { 
        response: { status: 503 }, 
        message: 'Service unavailable' 
      };
      const classified = classifyError(serviceError);

      expect(classified.type).toBe(ErrorType.SERVICE_UNAVAILABLE);
      expect(classified.isRetryable).toBe(true);
      expect(classified.statusCode).toBe(503);
    });

    it('should classify unknown errors', () => {
      const unknownError = { message: 'Something went wrong' };
      const classified = classifyError(unknownError);

      expect(classified.type).toBe(ErrorType.UNKNOWN);
      expect(classified.isRetryable).toBe(false);
      expect(classified.userFriendlyMessage).toContain('unexpected error');
    });
  });

  describe('executeWithRetry', () => {
    it('should succeed on first attempt', async () => {
      const operation = vi.fn().mockResolvedValue('success');

      const result = await executeWithRetry(operation, {}, 'test-operation');

      expect(result).toBe('success');
      expect(operation).toHaveBeenCalledTimes(1);
    });

    it('should retry on retryable errors', async () => {
      const operation = vi.fn()
        .mockRejectedValueOnce({ response: { status: 500 } })
        .mockRejectedValueOnce({ response: { status: 503 } })
        .mockResolvedValue('success');

      const result = await executeWithRetry(
        operation,
        { maxAttempts: 3, baseDelay: 10 },
        'test-operation'
      );

      expect(result).toBe('success');
      expect(operation).toHaveBeenCalledTimes(3);
    });

    it('should not retry on non-retryable errors', async () => {
      const operation = vi.fn().mockRejectedValue({ response: { status: 400 } });

      await expect(executeWithRetry(
        operation,
        { maxAttempts: 3, baseDelay: 10 },
        'test-operation'
      )).rejects.toThrow();

      expect(operation).toHaveBeenCalledTimes(1);
    });

    it('should respect max attempts', async () => {
      const operation = vi.fn().mockRejectedValue({ response: { status: 500 } });

      await expect(executeWithRetry(
        operation,
        { maxAttempts: 2, baseDelay: 10 },
        'test-operation'
      )).rejects.toThrow();

      expect(operation).toHaveBeenCalledTimes(2);
    });

    it('should use exponential backoff', async () => {
      const operation = vi.fn()
        .mockRejectedValueOnce({ response: { status: 500 } })
        .mockRejectedValueOnce({ response: { status: 500 } })
        .mockResolvedValue('success');

      const startTime = Date.now();
      await executeWithRetry(
        operation,
        { maxAttempts: 3, baseDelay: 100, backoffMultiplier: 2, jitter: false },
        'test-operation'
      );
      const endTime = Date.now();

      // Should have waited at least 100ms + 200ms = 300ms
      expect(endTime - startTime).toBeGreaterThan(250);
      expect(operation).toHaveBeenCalledTimes(3);
    });
  });

  describe('executeWithTimeout', () => {
    it('should complete within timeout', async () => {
      const operation = vi.fn().mockResolvedValue('success');

      const result = await executeWithTimeout(operation, 1000, 'test-operation');

      expect(result).toBe('success');
      expect(operation).toHaveBeenCalledTimes(1);
    });

    it('should timeout if operation takes too long', async () => {
      const operation = vi.fn().mockImplementation(() => 
        new Promise(resolve => setTimeout(() => resolve('success'), 200))
      );

      await expect(executeWithTimeout(operation, 100, 'test-operation'))
        .rejects.toThrow('test-operation timed out after 100ms');

      expect(operation).toHaveBeenCalledTimes(1);
    });
  });

  describe('CircuitBreaker', () => {
    it('should allow requests when closed', async () => {
      const circuitBreaker = new CircuitBreaker({ failureThreshold: 2 }, 'test-service');
      const operation = vi.fn().mockResolvedValue('success');

      const result = await circuitBreaker.execute(operation);

      expect(result).toBe('success');
      expect(operation).toHaveBeenCalledTimes(1);
      expect(circuitBreaker.getState().state).toBe('closed');
    });

    it('should open after threshold failures', async () => {
      const circuitBreaker = new CircuitBreaker({ failureThreshold: 2 }, 'test-service');
      const operation = vi.fn().mockRejectedValue(new Error('Service error'));

      // First failure
      await expect(circuitBreaker.execute(operation)).rejects.toThrow();
      expect(circuitBreaker.getState().state).toBe('closed');

      // Second failure - should open circuit
      await expect(circuitBreaker.execute(operation)).rejects.toThrow();
      expect(circuitBreaker.getState().state).toBe('open');

      // Third attempt should be rejected without calling operation
      await expect(circuitBreaker.execute(operation)).rejects.toThrow('Circuit breaker is open');
      expect(operation).toHaveBeenCalledTimes(2); // Not called on third attempt
    });

    it('should move to half-open after reset timeout', async () => {
      const circuitBreaker = new CircuitBreaker({ 
        failureThreshold: 1, 
        resetTimeout: 50 
      }, 'test-service');
      const operation = vi.fn()
        .mockRejectedValueOnce(new Error('Service error'))
        .mockResolvedValue('success');

      // Trigger circuit open
      await expect(circuitBreaker.execute(operation)).rejects.toThrow();
      expect(circuitBreaker.getState().state).toBe('open');

      // Wait for reset timeout
      await new Promise(resolve => setTimeout(resolve, 60));

      // Should now allow one request (half-open)
      const result = await circuitBreaker.execute(operation);
      expect(result).toBe('success');
      expect(circuitBreaker.getState().state).toBe('closed');
    });
  });

  describe('handlePartialData', () => {
    it('should handle all successful operations', async () => {
      const operations = [
        { name: 'op1', operation: () => Promise.resolve('data1'), required: true },
        { name: 'op2', operation: () => Promise.resolve('data2'), required: false },
        { name: 'op3', operation: () => Promise.resolve('data3'), required: true }
      ];

      const result = await handlePartialData(operations, 'test-context');

      expect(result.isComplete).toBe(true);
      expect(result.completionPercentage).toBe(100);
      expect(result.errors).toHaveLength(0);
      expect(result.data).toEqual({
        op1: 'data1',
        op2: 'data2',
        op3: 'data3'
      });
    });

    it('should handle partial failures gracefully', async () => {
      const operations = [
        { name: 'op1', operation: () => Promise.resolve('data1'), required: true },
        { name: 'op2', operation: () => Promise.reject(new Error('Failed')), required: false },
        { name: 'op3', operation: () => Promise.resolve('data3'), required: true }
      ];

      const result = await handlePartialData(operations, 'test-context');

      expect(result.isComplete).toBe(true); // All required operations succeeded
      expect(result.completionPercentage).toBe(67); // 2 out of 3 succeeded
      expect(result.errors).toHaveLength(1);
      expect(result.data).toEqual({
        op1: 'data1',
        op3: 'data3'
      });
    });

    it('should mark as incomplete when required operations fail', async () => {
      const operations = [
        { name: 'op1', operation: () => Promise.resolve('data1'), required: true },
        { name: 'op2', operation: () => Promise.reject(new Error('Failed')), required: true },
        { name: 'op3', operation: () => Promise.resolve('data3'), required: false }
      ];

      const result = await handlePartialData(operations, 'test-context');

      expect(result.isComplete).toBe(false); // Required operation failed
      expect(result.completionPercentage).toBe(67); // 2 out of 3 succeeded
      expect(result.errors).toHaveLength(1);
      expect(result.data).toEqual({
        op1: 'data1',
        op3: 'data3'
      });
    });
  });

  describe('createSlackErrorMessage', () => {
    it('should create appropriate message for network errors', () => {
      const error = { code: 'ECONNREFUSED', message: 'Connection refused' };
      const message = createSlackErrorMessage(error, 'test context');

      expect(message).toContain('🔌');
      expect(message).toContain('Unable to connect');
    });

    it('should create appropriate message for timeout errors', () => {
      const error = { message: 'Request timeout' };
      const message = createSlackErrorMessage(error, 'test context');

      expect(message).toContain('⏱️');
      expect(message).toContain('timed out');
    });

    it('should create appropriate message for authentication errors', () => {
      const error = { response: { status: 401 }, message: 'Unauthorized' };
      const message = createSlackErrorMessage(error, 'test context');

      expect(message).toContain('🔒');
      expect(message).toContain('Authentication required');
    });

    it('should create appropriate message for not found errors', () => {
      const error = { response: { status: 404 }, message: 'Not found' };
      const message = createSlackErrorMessage(error, 'test context');

      expect(message).toContain('🔍');
      expect(message).toContain('not found');
    });

    it('should create appropriate message for server errors', () => {
      const error = { response: { status: 500 }, message: 'Internal error' };
      const message = createSlackErrorMessage(error, 'test context');

      expect(message).toContain('🚨');
      expect(message).toContain('temporarily unavailable');
    });
  });
});