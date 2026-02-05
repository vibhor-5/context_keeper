/**
 * Comprehensive error handling utilities
 * Provides retry logic, timeout handling, and error classification
 */

import { logger } from './logger';

export interface RetryConfig {
  maxAttempts: number;
  baseDelay: number;
  maxDelay: number;
  backoffMultiplier: number;
  jitter: boolean;
}

export interface TimeoutConfig {
  operation: number;
  connection: number;
  idle: number;
}

export interface CircuitBreakerConfig {
  failureThreshold: number;
  resetTimeout: number;
  monitoringPeriod: number;
}

export enum ErrorType {
  NETWORK = 'network',
  TIMEOUT = 'timeout',
  AUTHENTICATION = 'authentication',
  AUTHORIZATION = 'authorization',
  VALIDATION = 'validation',
  NOT_FOUND = 'not_found',
  RATE_LIMIT = 'rate_limit',
  SERVER_ERROR = 'server_error',
  SERVICE_UNAVAILABLE = 'service_unavailable',
  UNKNOWN = 'unknown'
}

export interface ClassifiedError {
  type: ErrorType;
  message: string;
  originalError: Error;
  isRetryable: boolean;
  userFriendlyMessage: string;
  statusCode?: number;
}

/**
 * Default retry configuration
 */
export const DEFAULT_RETRY_CONFIG: RetryConfig = {
  maxAttempts: 3,
  baseDelay: 1000,
  maxDelay: 30000,
  backoffMultiplier: 2,
  jitter: true
};

/**
 * Default timeout configuration
 */
export const DEFAULT_TIMEOUT_CONFIG: TimeoutConfig = {
  operation: 30000,
  connection: 10000,
  idle: 60000
};

/**
 * Default circuit breaker configuration
 */
export const DEFAULT_CIRCUIT_BREAKER_CONFIG: CircuitBreakerConfig = {
  failureThreshold: 5,
  resetTimeout: 60000,
  monitoringPeriod: 300000
};

/**
 * Classify an error based on its characteristics
 */
export function classifyError(error: any): ClassifiedError {
  let type = ErrorType.UNKNOWN;
  let isRetryable = false;
  let userFriendlyMessage = 'An unexpected error occurred. Please try again later.';
  let statusCode: number | undefined;

  // Handle axios errors
  if (error.response) {
    statusCode = error.response.status;
    
    switch (statusCode) {
      case 400:
        type = ErrorType.VALIDATION;
        userFriendlyMessage = 'Invalid request. Please check your input and try again.';
        break;
      case 401:
        type = ErrorType.AUTHENTICATION;
        userFriendlyMessage = 'Authentication required. Please check your credentials.';
        break;
      case 403:
        type = ErrorType.AUTHORIZATION;
        userFriendlyMessage = 'Access denied. You don\'t have permission to perform this action.';
        break;
      case 404:
        type = ErrorType.NOT_FOUND;
        userFriendlyMessage = 'The requested resource was not found.';
        break;
      case 408:
        type = ErrorType.TIMEOUT;
        isRetryable = true;
        userFriendlyMessage = 'Request timed out. Please try again.';
        break;
      case 429:
        type = ErrorType.RATE_LIMIT;
        isRetryable = true;
        userFriendlyMessage = 'Too many requests. Please wait a moment and try again.';
        break;
      case 500:
      case 502:
      case 503:
      case 504:
        type = statusCode === 503 ? ErrorType.SERVICE_UNAVAILABLE : ErrorType.SERVER_ERROR;
        isRetryable = true;
        userFriendlyMessage = 'Service is temporarily unavailable. Please try again in a few moments.';
        break;
    }
  } else if (error.code) {
    // Handle network errors
    switch (error.code) {
      case 'ECONNREFUSED':
      case 'ENOTFOUND':
      case 'ECONNRESET':
      case 'ETIMEDOUT':
        type = ErrorType.NETWORK;
        isRetryable = true;
        userFriendlyMessage = 'Unable to connect to the service. Please check your connection and try again.';
        break;
      case 'ECONNABORTED':
        type = ErrorType.TIMEOUT;
        isRetryable = true;
        userFriendlyMessage = 'Request timed out. Please try again.';
        break;
    }
  } else if (error.message) {
    // Handle timeout errors and other message-based classification
    const message = error.message.toLowerCase();
    if (message.includes('timeout')) {
      type = ErrorType.TIMEOUT;
      isRetryable = true;
      userFriendlyMessage = 'Request timed out. Please try again.';
    } else if (message.includes('econnrefused') || message.includes('econnreset') || 
               message.includes('socket hang up') || message.includes('read econnreset')) {
      type = ErrorType.NETWORK;
      isRetryable = true;
      userFriendlyMessage = 'Unable to connect to the service. Please check your connection and try again.';
    } else if (message.includes('network error') || message.includes('connection failed')) {
      type = ErrorType.NETWORK;
      isRetryable = true;
      userFriendlyMessage = 'Unable to connect to the service. Please check your connection and try again.';
    }
  }

  return {
    type,
    message: error.message || 'Unknown error',
    originalError: error,
    isRetryable,
    userFriendlyMessage,
    statusCode
  };
}

/**
 * Execute an operation with retry logic and exponential backoff
 */
export async function executeWithRetry<T>(
  operation: () => Promise<T>,
  config: Partial<RetryConfig> = {},
  context: string = 'operation'
): Promise<T> {
  const retryConfig = { ...DEFAULT_RETRY_CONFIG, ...config };
  let lastError: Error;

  for (let attempt = 1; attempt <= retryConfig.maxAttempts; attempt++) {
    try {
      logger.debug(`Executing ${context}`, { attempt, maxAttempts: retryConfig.maxAttempts });
      
      const result = await operation();
      
      if (attempt > 1) {
        logger.info(`${context} succeeded after ${attempt} attempts`);
      }
      
      return result;
    } catch (error) {
      lastError = error as Error;
      const classifiedError = classifyError(error);

      logger.warn(`${context} failed on attempt ${attempt}`, {
        attempt,
        maxAttempts: retryConfig.maxAttempts,
        errorType: classifiedError.type,
        isRetryable: classifiedError.isRetryable,
        message: classifiedError.message
      });

      // Don't retry if this is the last attempt or error is not retryable
      if (attempt >= retryConfig.maxAttempts || !classifiedError.isRetryable) {
        break;
      }

      // Calculate delay with exponential backoff and optional jitter
      let delay = Math.min(
        retryConfig.baseDelay * Math.pow(retryConfig.backoffMultiplier, attempt - 1),
        retryConfig.maxDelay
      );

      if (retryConfig.jitter) {
        // Add random jitter to prevent thundering herd
        delay = delay * (0.5 + Math.random() * 0.5);
      }

      logger.debug(`Retrying ${context} in ${Math.round(delay)}ms`, { attempt, delay });
      await new Promise(resolve => setTimeout(resolve, delay));
    }
  }

  // All attempts failed
  const classifiedError = classifyError(lastError!);
  logger.error(`${context} failed after ${retryConfig.maxAttempts} attempts`, {
    errorType: classifiedError.type,
    finalError: classifiedError.message
  });

  throw lastError!;
}

/**
 * Execute an operation with timeout
 */
export async function executeWithTimeout<T>(
  operation: () => Promise<T>,
  timeoutMs: number,
  context: string = 'operation'
): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    const timeoutId = setTimeout(() => {
      const timeoutError = new Error(`${context} timed out after ${timeoutMs}ms`);
      logger.warn(`${context} timed out`, { timeoutMs });
      reject(timeoutError);
    }, timeoutMs);

    operation()
      .then(result => {
        clearTimeout(timeoutId);
        resolve(result);
      })
      .catch(error => {
        clearTimeout(timeoutId);
        reject(error);
      });
  });
}

/**
 * Circuit breaker implementation
 */
export class CircuitBreaker {
  private failures: number = 0;
  private lastFailureTime: number = 0;
  private state: 'closed' | 'open' | 'half-open' = 'closed';
  private config: CircuitBreakerConfig;
  private context: string;

  constructor(config: Partial<CircuitBreakerConfig> = {}, context: string = 'service') {
    this.config = { ...DEFAULT_CIRCUIT_BREAKER_CONFIG, ...config };
    this.context = context;
  }

  async execute<T>(operation: () => Promise<T>): Promise<T> {
    if (this.state === 'open') {
      if (Date.now() - this.lastFailureTime > this.config.resetTimeout) {
        this.state = 'half-open';
        logger.info(`Circuit breaker for ${this.context} moved to half-open state`);
      } else {
        const error = new Error(`Circuit breaker is open for ${this.context}`);
        logger.warn(`Circuit breaker rejected request for ${this.context}`, {
          state: this.state,
          failures: this.failures
        });
        throw error;
      }
    }

    try {
      const result = await operation();
      
      if (this.state === 'half-open') {
        this.reset();
        logger.info(`Circuit breaker for ${this.context} reset to closed state`);
      }
      
      return result;
    } catch (error) {
      this.recordFailure();
      throw error;
    }
  }

  private recordFailure(): void {
    this.failures++;
    this.lastFailureTime = Date.now();

    if (this.failures >= this.config.failureThreshold) {
      this.state = 'open';
      logger.warn(`Circuit breaker for ${this.context} opened`, {
        failures: this.failures,
        threshold: this.config.failureThreshold
      });
    }
  }

  private reset(): void {
    this.failures = 0;
    this.state = 'closed';
    this.lastFailureTime = 0;
  }

  getState(): { state: string; failures: number; lastFailureTime: number } {
    return {
      state: this.state,
      failures: this.failures,
      lastFailureTime: this.lastFailureTime
    };
  }
}

/**
 * Handle partial data scenarios gracefully
 */
export interface PartialDataResult<T> {
  data: Partial<T>;
  errors: string[];
  isComplete: boolean;
  completionPercentage: number;
}

export async function handlePartialData<T>(
  operations: Array<{
    name: string;
    operation: () => Promise<any>;
    required: boolean;
  }>,
  context: string = 'data collection'
): Promise<PartialDataResult<T>> {
  const results: any = {};
  const errors: string[] = [];
  let completedOperations = 0;
  let requiredOperations = 0;
  let completedRequiredOperations = 0;

  // Count required operations
  operations.forEach(op => {
    if (op.required) requiredOperations++;
  });

  logger.info(`Starting ${context} with ${operations.length} operations (${requiredOperations} required)`);

  // Execute all operations concurrently
  const promises = operations.map(async ({ name, operation, required }) => {
    try {
      logger.debug(`Executing ${name} for ${context}`);
      const result = await operation();
      results[name] = result;
      completedOperations++;
      if (required) completedRequiredOperations++;
      logger.debug(`Completed ${name} for ${context}`);
    } catch (error) {
      const classifiedError = classifyError(error);
      const errorMessage = `${name}: ${classifiedError.userFriendlyMessage}`;
      errors.push(errorMessage);
      
      logger.warn(`Failed to execute ${name} for ${context}`, {
        required,
        errorType: classifiedError.type,
        message: classifiedError.message
      });

      if (required) {
        logger.error(`Required operation ${name} failed for ${context}`, {
          error: classifiedError.message
        });
      }
    }
  });

  // Wait for all operations to complete
  await Promise.allSettled(promises);

  const isComplete = completedRequiredOperations === requiredOperations;
  const completionPercentage = Math.round((completedOperations / operations.length) * 100);

  logger.info(`${context} completed`, {
    completedOperations,
    totalOperations: operations.length,
    completedRequired: completedRequiredOperations,
    totalRequired: requiredOperations,
    isComplete,
    completionPercentage,
    errorCount: errors.length
  });

  return {
    data: results as Partial<T>,
    errors,
    isComplete,
    completionPercentage
  };
}

/**
 * Create a user-friendly error message for Slack responses
 */
export function createSlackErrorMessage(error: any, context: string): string {
  const classifiedError = classifyError(error);
  
  let emoji = '⚠️';
  switch (classifiedError.type) {
    case ErrorType.NETWORK:
    case ErrorType.SERVICE_UNAVAILABLE:
      emoji = '🔌';
      break;
    case ErrorType.TIMEOUT:
      emoji = '⏱️';
      break;
    case ErrorType.AUTHENTICATION:
    case ErrorType.AUTHORIZATION:
      emoji = '🔒';
      break;
    case ErrorType.NOT_FOUND:
      emoji = '🔍';
      break;
    case ErrorType.RATE_LIMIT:
      emoji = '🚦';
      break;
    case ErrorType.SERVER_ERROR:
      emoji = '🚨';
      break;
  }

  return `${emoji} ${classifiedError.userFriendlyMessage}`;
}