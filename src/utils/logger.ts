/**
 * Structured logging utility for ContextKeeper system
 * Provides secure logging without exposing sensitive information
 */

export type LogLevel = 'debug' | 'info' | 'warn' | 'error';

export interface LogEntry {
  timestamp: string;
  level: LogLevel;
  message: string;
  context?: Record<string, any>;
  error?: Error;
  requestId?: string;
  userId?: string;
  component?: string;
}

export interface LoggerConfig {
  level: LogLevel;
  enableRequestTracking: boolean;
  enableUserTracking: boolean;
  enableComponentTracking: boolean;
  maxContextLength: number;
  enableStackTraces: boolean;
  sensitiveKeys: string[];
}

class Logger {
  private config: LoggerConfig = {
    level: 'info',
    enableRequestTracking: true,
    enableUserTracking: true,
    enableComponentTracking: true,
    maxContextLength: 1000,
    enableStackTraces: false,
    sensitiveKeys: [
      'token', 'secret', 'password', 'key', 'auth', 'credential',
      'botToken', 'signingSecret', 'authorization', 'bearer',
      'apikey', 'api_key', 'access_token', 'refresh_token',
      'client_secret', 'private_key', 'passphrase', 'salt',
      'hash', 'signature', 'webhook_secret', 'session_id',
      'cookie', 'csrf_token', 'nonce'
    ]
  };

  private currentRequestId?: string;
  private currentUserId?: string;
  private currentComponent?: string;

  configure(config: Partial<LoggerConfig>): void {
    this.config = { ...this.config, ...config };
  }

  setLogLevel(level: LogLevel): void {
    this.config.level = level;
  }

  setRequestContext(requestId?: string, userId?: string, component?: string): void {
    if (this.config.enableRequestTracking) {
      this.currentRequestId = requestId;
    }
    if (this.config.enableUserTracking) {
      this.currentUserId = userId;
    }
    if (this.config.enableComponentTracking) {
      this.currentComponent = component;
    }
  }

  clearRequestContext(): void {
    this.currentRequestId = undefined;
    this.currentUserId = undefined;
    this.currentComponent = undefined;
  }

  private shouldLog(level: LogLevel): boolean {
    const levels: Record<LogLevel, number> = {
      debug: 0,
      info: 1,
      warn: 2,
      error: 3
    };
    
    return levels[level] >= levels[this.config.level];
  }

  private sanitizeValue(key: string, value: any, depth: number = 0): any {
    // Prevent infinite recursion
    if (depth > 3) {
      return '[MAX_DEPTH_REACHED]';
    }

    const lowerKey = key.toLowerCase();
    const isSensitive = this.config.sensitiveKeys.some(sensitiveKey => 
      lowerKey.includes(sensitiveKey)
    );
    
    if (isSensitive) {
      if (typeof value === 'string') {
        // Show partial information for debugging while keeping it secure
        if (value.length <= 4) {
          return '[REDACTED]';
        } else {
          return `${value.substring(0, 2)}***${value.substring(value.length - 2)}`;
        }
      }
      return '[REDACTED]';
    }

    if (typeof value === 'string') {
      // Truncate very long strings to prevent log bloat
      if (value.length > 500) {
        return value.substring(0, 500) + '...[TRUNCATED]';
      }
      return value;
    }

    if (Array.isArray(value)) {
      // Check if the array key itself is sensitive
      if (isSensitive) {
        return '[REDACTED]';
      }
      
      // Limit array size in logs
      if (value.length > 10) {
        const truncatedArray = value.slice(0, 10).map((item, index) => 
          this.sanitizeValue(`${key}[${index}]`, item, depth + 1)
        );
        truncatedArray.push(`...[${value.length - 10} more items]`);
        return truncatedArray;
      }
      return value.map((item, index) => this.sanitizeValue(`${key}[${index}]`, item, depth + 1));
    }

    if (typeof value === 'object' && value !== null) {
      // Handle circular references
      try {
        JSON.stringify(value);
      } catch (error) {
        return '[CIRCULAR_REFERENCE]';
      }

      const sanitized: Record<string, any> = {};
      for (const [objKey, objValue] of Object.entries(value)) {
        sanitized[objKey] = this.sanitizeValue(objKey, objValue, depth + 1);
      }
      return sanitized;
    }

    return value;
  }

  private sanitizeContext(context: Record<string, any>): Record<string, any> {
    const sanitized: Record<string, any> = {};
    
    for (const [key, value] of Object.entries(context)) {
      sanitized[key] = this.sanitizeValue(key, value);
    }
    
    return sanitized;
  }

  private truncateContext(context: Record<string, any>): Record<string, any> {
    const contextStr = JSON.stringify(context);
    if (contextStr.length <= this.config.maxContextLength) {
      return context;
    }

    // Try to keep the most important keys
    const importantKeys = ['userId', 'requestId', 'component', 'error', 'status', 'method', 'url'];
    const truncated: Record<string, any> = {};
    
    for (const key of importantKeys) {
      if (context[key] !== undefined) {
        truncated[key] = context[key];
      }
    }

    // Add remaining keys until we hit the limit
    let currentLength = JSON.stringify(truncated).length;
    for (const [key, value] of Object.entries(context)) {
      if (!importantKeys.includes(key)) {
        const keyValueStr = JSON.stringify({ [key]: value });
        if (currentLength + keyValueStr.length <= this.config.maxContextLength - 50) {
          truncated[key] = value;
          currentLength += keyValueStr.length;
        } else {
          truncated['_truncated'] = true;
          break;
        }
      }
    }

    return truncated;
  }

  private formatLogEntry(entry: LogEntry): string {
    const { timestamp, level, message, context, error, requestId, userId, component } = entry;
    
    let logMessage = `[${timestamp}] ${level.toUpperCase()}`;
    
    // Add context identifiers
    const identifiers = [];
    if (requestId) identifiers.push(`req:${requestId}`);
    if (userId) identifiers.push(`user:${userId}`);
    if (component) identifiers.push(`comp:${component}`);
    
    if (identifiers.length > 0) {
      logMessage += ` [${identifiers.join('|')}]`;
    }
    
    logMessage += `: ${message}`;
    
    if (context && Object.keys(context).length > 0) {
      const sanitizedContext = this.sanitizeContext(context);
      const truncatedContext = this.truncateContext(sanitizedContext);
      logMessage += ` | Context: ${JSON.stringify(truncatedContext)}`;
    }
    
    if (error) {
      logMessage += ` | Error: ${error.message}`;
      if ((this.config.level === 'debug' || this.config.enableStackTraces) && error.stack) {
        // Sanitize stack traces to remove potential sensitive file paths
        const sanitizedStack = error.stack
          .replace(/\/home\/[^\/]+/g, '/home/[USER]')
          .replace(/\/Users\/[^\/]+/g, '/Users/[USER]')
          .replace(/C:\\Users\\[^\\]+/g, 'C:\\Users\\[USER]');
        logMessage += `\nStack: ${sanitizedStack}`;
      }
    }
    
    return logMessage;
  }

  private log(level: LogLevel, message: string, context?: Record<string, any>, error?: Error): void {
    if (!this.shouldLog(level)) {
      return;
    }

    const entry: LogEntry = {
      timestamp: new Date().toISOString(),
      level,
      message,
      context,
      error,
      requestId: this.currentRequestId,
      userId: this.currentUserId,
      component: this.currentComponent
    };

    const formattedMessage = this.formatLogEntry(entry);
    
    // Output to appropriate stream
    if (level === 'error') {
      console.error(formattedMessage);
    } else {
      console.log(formattedMessage);
    }
  }

  debug(message: string, context?: Record<string, any>): void {
    this.log('debug', message, context);
  }

  info(message: string, context?: Record<string, any>): void {
    this.log('info', message, context);
  }

  warn(message: string, context?: Record<string, any>, error?: Error): void {
    this.log('warn', message, context, error);
  }

  error(message: string, context?: Record<string, any>, error?: Error): void {
    this.log('error', message, context, error);
  }

  // Convenience methods for common logging patterns
  httpRequest(method: string, url: string, statusCode?: number, responseTime?: number, context?: Record<string, any>): void {
    this.info('HTTP Request', {
      method,
      url,
      statusCode,
      responseTime,
      ...context
    });
  }

  httpResponse(method: string, url: string, statusCode: number, responseTime: number, context?: Record<string, any>): void {
    const level = statusCode >= 400 ? 'warn' : 'info';
    this.log(level, 'HTTP Response', {
      method,
      url,
      statusCode,
      responseTime,
      ...context
    });
  }

  security(event: string, context?: Record<string, any>): void {
    this.warn(`Security Event: ${event}`, context);
  }

  performance(operation: string, duration: number, context?: Record<string, any>): void {
    const level = duration > 5000 ? 'warn' : 'info'; // Warn if operation takes more than 5 seconds
    this.log(level, `Performance: ${operation}`, {
      duration,
      ...context
    });
  }

  audit(action: string, resource: string, context?: Record<string, any>): void {
    this.info(`Audit: ${action} on ${resource}`, context);
  }

  // Method to create a child logger with preset context
  child(context: Record<string, any>): Logger {
    const childLogger = new Logger();
    childLogger.config = { ...this.config };
    childLogger.currentRequestId = this.currentRequestId;
    childLogger.currentUserId = this.currentUserId;
    childLogger.currentComponent = this.currentComponent;
    
    // Override log method to include preset context
    const originalLog = childLogger.log.bind(childLogger);
    childLogger.log = (level: LogLevel, message: string, additionalContext?: Record<string, any>, error?: Error) => {
      const mergedContext = { ...context, ...additionalContext };
      originalLog(level, message, mergedContext, error);
    };
    
    return childLogger;
  }
}

export const logger = new Logger();