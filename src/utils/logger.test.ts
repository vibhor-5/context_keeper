/**
 * Logger tests
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { logger, LogLevel } from './logger';

describe('Logger', () => {
  let consoleSpy: any;
  let consoleErrorSpy: any;

  beforeEach(() => {
    consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    logger.clearRequestContext();
    // Don't reset configuration here - let each test configure as needed
  });

  afterEach(() => {
    consoleSpy.mockRestore();
    consoleErrorSpy.mockRestore();
  });

  describe('basic logging', () => {
    it('should log debug messages when level is debug', () => {
      logger.setLogLevel('debug');
      logger.debug('Test debug message');

      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('DEBUG: Test debug message')
      );
    });

    it('should not log debug messages when level is info', () => {
      logger.setLogLevel('info');
      logger.debug('Test debug message');

      expect(consoleSpy).not.toHaveBeenCalled();
    });

    it('should log error messages to stderr', () => {
      logger.error('Test error message');

      expect(consoleErrorSpy).toHaveBeenCalledWith(
        expect.stringContaining('ERROR: Test error message')
      );
    });

    it('should include timestamp in log messages', () => {
      logger.info('Test message');

      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringMatching(/\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z\]/)
      );
    });
  });

  describe('context sanitization', () => {
    beforeEach(() => {
      logger.configure({ maxContextLength: 2000 }); // Increase limit for these tests
    });

    it('should redact sensitive information', () => {
      logger.info('Test message', {
        username: 'testuser',
        token: 'secret-token-123',
        password: 'my-password',
        apiKey: 'api-key-456'
      });

      const logCall = consoleSpy.mock.calls[0][0];
      expect(logCall).toContain('username":"testuser"');
      expect(logCall).toContain('token":"se***23"'); // Partial redaction
      expect(logCall).toContain('password":"my***rd"'); // Partial redaction for longer values
      expect(logCall).toContain('apiKey":"ap***56"'); // Partial redaction
    });

    it('should sanitize nested objects', () => {
      logger.info('Test message', {
        user: {
          name: 'testuser',
          token: 'secret-token-123',
          refreshToken: 'refresh-token-456'
        }
      });

      const logCall = consoleSpy.mock.calls[0][0];
      expect(logCall).toContain('name":"testuser"');
      expect(logCall).toContain('token":"se***23"');
      expect(logCall).toContain('refreshToken":"re***56"');
    });

    it('should handle arrays with sensitive data', () => {
      logger.info('Test message', {
        tokens: ['token1', 'token2', 'token3'],
        users: ['user1', 'user2']
      });

      const logCall = consoleSpy.mock.calls[0][0];
      expect(logCall).toContain('tokens":"[REDACTED]"'); // Entire array redacted when key is sensitive
      expect(logCall).toContain('users":["user1","user2"]');
    });

    it('should truncate very long strings', () => {
      const longString = 'a'.repeat(600);
      logger.info('Test message', { longData: longString });

      const logCall = consoleSpy.mock.calls[0][0];
      expect(logCall).toContain('...[TRUNCATED]');
      expect(logCall).not.toContain('a'.repeat(600));
    });

    it('should limit array size in logs', () => {
      const largeArray = Array.from({ length: 15 }, (_, i) => `item${i}`);
      logger.info('Test message', { items: largeArray });

      const logCall = consoleSpy.mock.calls[0][0];
      expect(logCall).toContain('...[5 more items]');
    });
  });

  describe('request context tracking', () => {
    it('should include request context in logs', () => {
      logger.setRequestContext('req-123', 'user-456', 'slack-bot');
      logger.info('Test message');

      const logCall = consoleSpy.mock.calls[0][0];
      expect(logCall).toContain('[req:req-123|user:user-456|comp:slack-bot]');
    });

    it('should clear request context', () => {
      logger.setRequestContext('req-123', 'user-456', 'slack-bot');
      logger.clearRequestContext();
      logger.info('Test message');

      const logCall = consoleSpy.mock.calls[0][0];
      expect(logCall).not.toContain('[req:');
    });

    it('should handle partial context', () => {
      logger.setRequestContext('req-123', undefined, 'slack-bot');
      logger.info('Test message');

      const logCall = consoleSpy.mock.calls[0][0];
      expect(logCall).toContain('[req:req-123|comp:slack-bot]');
      expect(logCall).not.toContain('user:');
    });
  });

  describe('error handling', () => {
    it('should include error message in logs', () => {
      const error = new Error('Test error');
      logger.error('Something went wrong', {}, error);

      const logCall = consoleErrorSpy.mock.calls[0][0];
      expect(logCall).toContain('Error: Test error');
    });

    it('should include stack trace in debug mode', () => {
      logger.configure({ enableStackTraces: true });
      const error = new Error('Test error');
      logger.error('Something went wrong', {}, error);

      const logCall = consoleErrorSpy.mock.calls[0][0];
      expect(logCall).toContain('Stack:');
    });

    it('should sanitize file paths in stack traces', () => {
      logger.configure({ enableStackTraces: true });
      const error = new Error('Test error');
      error.stack = 'Error: Test error\n    at /home/username/project/file.js:10:5';
      logger.error('Something went wrong', {}, error);

      const logCall = consoleErrorSpy.mock.calls[0][0];
      expect(logCall).toContain('/home/[USER]/project/file.js');
    });
  });

  describe('convenience methods', () => {
    it('should log HTTP requests', () => {
      logger.httpRequest('GET', '/api/test', 200, 150);

      const logCall = consoleSpy.mock.calls[0][0];
      expect(logCall).toContain('HTTP Request');
      expect(logCall).toContain('method":"GET"');
      expect(logCall).toContain('url":"/api/test"');
      expect(logCall).toContain('statusCode":200');
      expect(logCall).toContain('responseTime":150');
    });

    it('should log HTTP responses with appropriate level', () => {
      // Success response
      logger.httpResponse('GET', '/api/test', 200, 150);
      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('INFO')
      );

      // Error response
      logger.httpResponse('GET', '/api/test', 500, 150);
      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('WARN')
      );
    });

    it('should log security events', () => {
      logger.security('Invalid signature detected', { ip: '192.168.1.1' });

      const logCall = consoleSpy.mock.calls[0][0];
      expect(logCall).toContain('WARN');
      expect(logCall).toContain('Security Event: Invalid signature detected');
      expect(logCall).toContain('ip":"192.168.1.1"');
    });

    it('should log performance metrics', () => {
      // Fast operation
      logger.performance('Database query', 100);
      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('INFO')
      );

      // Slow operation
      logger.performance('Database query', 6000);
      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('WARN')
      );
    });

    it('should log audit events', () => {
      logger.audit('CREATE', 'user', { userId: 'user-123' });

      const logCall = consoleSpy.mock.calls[0][0];
      expect(logCall).toContain('Audit: CREATE on user');
      expect(logCall).toContain('userId":"user-123"');
    });
  });

  describe('child logger', () => {
    it('should create child logger with preset context', () => {
      const childLogger = logger.child({ component: 'test-component' });
      childLogger.info('Test message', { additional: 'data' });

      const logCall = consoleSpy.mock.calls[0][0];
      expect(logCall).toContain('component":"test-component"');
      expect(logCall).toContain('additional":"data"');
    });

    it('should inherit parent configuration', () => {
      logger.configure({ level: 'warn' });
      const childLogger = logger.child({ component: 'test' });
      
      childLogger.info('This should not appear');
      expect(consoleSpy).not.toHaveBeenCalled();

      childLogger.warn('This should appear');
      expect(consoleSpy).toHaveBeenCalled();
    });
  });

  describe('configuration', () => {
    it('should respect maxContextLength setting', () => {
      logger.configure({ maxContextLength: 50, level: 'info' });
      
      const largeContext = {
        field1: 'value1',
        field2: 'value2',
        field3: 'value3',
        field4: 'a very long value that should cause truncation'
      };
      
      logger.info('Test message', largeContext);

      const logCall = consoleSpy.mock.calls[0][0];
      expect(logCall).toContain('_truncated":true');
    });

    it('should disable request tracking when configured', () => {
      logger.configure({ enableRequestTracking: false, level: 'info' });
      logger.setRequestContext('req-123', 'user-456', 'component');
      logger.info('Test message');

      const logCall = consoleSpy.mock.calls[0][0];
      expect(logCall).not.toContain('[req:');
    });

    it('should use custom sensitive keys', () => {
      // Clear console spy to avoid interference from previous tests
      consoleSpy.mockClear();
      
      // Configure with ONLY custom keys (no default keys)
      logger.configure({ 
        sensitiveKeys: ['customSecret', 'internalKey'],
        maxContextLength: 2000,
        level: 'info'
      });
      
      logger.info('Test message', {
        customSecret: 'redacted',
        internalKey: 'redacted',
        token: 'should-not-be-redacted', // Not in custom list
        password: 'should-also-not-be-redacted' // Not in custom list
      });

      const logCall = consoleSpy.mock.calls[0][0];
      // With custom keys, default sensitive keys should not be redacted
      expect(logCall).toContain('token":"should-not-be-redacted"');
      expect(logCall).toContain('password":"should-also-not-be-redacted"');
      expect(logCall).toContain('customSecret":"[REDACTED]"'); // Should be redacted
      expect(logCall).toContain('internalKey":"[REDACTED]"'); // Should be redacted
    });
  });

  describe('edge cases', () => {
    beforeEach(() => {
      logger.configure({ maxContextLength: 2000 }); // Increase limit for these tests
    });

    it('should handle circular references', () => {
      const obj: any = { name: 'test' };
      obj.self = obj;
      
      logger.info('Test message', { circular: obj });

      // Should not throw and should handle gracefully
      expect(consoleSpy).toHaveBeenCalled();
      const logCall = consoleSpy.mock.calls[0][0];
      expect(logCall).toContain('[CIRCULAR_REFERENCE]');
    });

    it('should handle null and undefined values', () => {
      logger.info('Test message', {
        nullValue: null,
        undefinedValue: undefined,
        emptyString: '',
        zero: 0,
        false: false
      });

      const logCall = consoleSpy.mock.calls[0][0];
      expect(logCall).toContain('nullValue":null');
      expect(logCall).toContain('emptyString":""');
      expect(logCall).toContain('zero":0');
      expect(logCall).toContain('false":false');
    });

    it('should handle very deep nested objects', () => {
      const deepObj = { level1: { level2: { level3: { level4: { level5: 'deep' } } } } };
      
      logger.info('Test message', deepObj);

      const logCall = consoleSpy.mock.calls[0][0];
      expect(logCall).toContain('[MAX_DEPTH_REACHED]');
    });
  });
});