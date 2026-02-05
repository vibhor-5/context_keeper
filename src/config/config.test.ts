/**
 * Configuration tests
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { writeFileSync, unlinkSync, existsSync } from 'fs';
import { join } from 'path';
import { loadConfig, loadConfigFromFile, saveConfigToFile, getDefaultConfig, getEnvironmentConfig } from './config';

describe('Configuration', () => {
  let originalEnv: NodeJS.ProcessEnv;
  const testConfigPath = join(__dirname, 'test-config.json');

  beforeEach(() => {
    originalEnv = { ...process.env };
    // Clean up any existing test config file
    if (existsSync(testConfigPath)) {
      unlinkSync(testConfigPath);
    }
  });

  afterEach(() => {
    process.env = originalEnv;
    // Clean up test config file
    if (existsSync(testConfigPath)) {
      unlinkSync(testConfigPath);
    }
  });

  describe('getDefaultConfig', () => {
    it('should return default configuration', () => {
      const config = getDefaultConfig();
      
      expect(config).toBeDefined();
      expect(config.mcp.port).toBe(3001);
      expect(config.slack.port).toBe(3002);
      expect(config.demo.demoMode).toBe(false);
    });

    it('should have valid MCP server configuration', () => {
      const config = getDefaultConfig();
      
      expect(config.mcp.port).toBeGreaterThan(1000);
      expect(config.mcp.port).toBeLessThan(65536);
      expect(config.mcp.goBackendUrl).toMatch(/^https?:\/\//);
      expect(config.mcp.timeout).toBeGreaterThan(0);
    });

    it('should have valid Slack bot configuration', () => {
      const config = getDefaultConfig();
      
      expect(config.slack.port).toBeGreaterThan(1000);
      expect(config.slack.port).toBeLessThan(65536);
      expect(config.slack.mcpServerUrl).toMatch(/^https?:\/\//);
      expect(config.slack.maxResponseLength).toBeGreaterThan(0);
      expect(config.slack.retryAttempts).toBeGreaterThan(0);
    });
  });

  describe('loadConfig', () => {
    it('should load configuration from environment variables', async () => {
      process.env.MCP_PORT = '4001';
      process.env.SLACK_PORT = '4002';
      process.env.SLACK_BOT_TOKEN = 'xoxb-test-token';
      process.env.SLACK_SIGNING_SECRET = 'test-secret';
      process.env.LOG_LEVEL = 'debug';

      const config = await loadConfig();

      expect(config.mcp.port).toBe(4001);
      expect(config.slack.port).toBe(4002);
      expect(config.slack.botToken).toBe('xoxb-test-token');
      expect(config.slack.signingSecret).toBe('test-secret');
      expect(config.mcp.logLevel).toBe('debug');
    });

    it('should load configuration from file', async () => {
      // Clear environment variables that might interfere
      delete process.env.LOG_LEVEL;
      delete process.env.SLACK_BOT_TOKEN;
      delete process.env.SLACK_SIGNING_SECRET;
      
      const fileConfig = {
        mcp: {
          port: 5001,
          logLevel: 'warn' as const
        },
        slack: {
          port: 5002,
          botToken: 'xoxb-file-token',
          signingSecret: 'file-secret'
        }
      };

      writeFileSync(testConfigPath, JSON.stringify(fileConfig));

      const config = await loadConfig(testConfigPath);

      expect(config.mcp.port).toBe(5001);
      expect(config.mcp.logLevel).toBe('warn');
      expect(config.slack.port).toBe(5002);
      expect(config.slack.botToken).toBe('xoxb-file-token');
      expect(config.slack.signingSecret).toBe('file-secret');
    });

    it('should prioritize environment variables over file configuration', async () => {
      const fileConfig = {
        mcp: { port: 6001 },
        slack: { 
          port: 6002,
          botToken: 'xoxb-file-token',
          signingSecret: 'file-secret'
        }
      };

      writeFileSync(testConfigPath, JSON.stringify(fileConfig));

      process.env.MCP_PORT = '7001';
      process.env.SLACK_BOT_TOKEN = 'xoxb-env-token';

      const config = await loadConfig(testConfigPath);

      expect(config.mcp.port).toBe(7001); // From env
      expect(config.slack.port).toBe(6002); // From file
      expect(config.slack.botToken).toBe('xoxb-env-token'); // From env
      expect(config.slack.signingSecret).toBe('file-secret'); // From file
    });

    it('should throw error for missing required parameters', async () => {
      process.env.SLACK_BOT_TOKEN = '';
      process.env.SLACK_SIGNING_SECRET = '';

      await expect(loadConfig()).rejects.toThrow('Configuration validation failed');
    });

    it('should throw error for invalid bot token format', async () => {
      process.env.SLACK_BOT_TOKEN = 'invalid-token';
      process.env.SLACK_SIGNING_SECRET = 'test-secret';

      await expect(loadConfig()).rejects.toThrow('must be a valid Slack bot token');
    });

    it('should throw error for port conflicts', async () => {
      process.env.MCP_PORT = '8000';
      process.env.SLACK_PORT = '8000';
      process.env.SLACK_BOT_TOKEN = 'xoxb-test-token';
      process.env.SLACK_SIGNING_SECRET = 'test-secret';

      await expect(loadConfig()).rejects.toThrow('cannot use the same port');
    });

    it('should throw error for invalid URLs', async () => {
      process.env.GO_BACKEND_URL = 'invalid-url';
      process.env.SLACK_BOT_TOKEN = 'xoxb-test-token';
      process.env.SLACK_SIGNING_SECRET = 'test-secret';

      await expect(loadConfig()).rejects.toThrow('Invalid url');
    });

    it('should handle malformed JSON config file', async () => {
      writeFileSync(testConfigPath, '{ invalid json }');

      await expect(loadConfig(testConfigPath)).rejects.toThrow('Failed to parse configuration file');
    });
  });

  describe('loadConfigFromFile', () => {
    it('should load configuration from specified file', async () => {
      const fileConfig = {
        mcp: { port: 9001 },
        slack: { 
          port: 9002,
          botToken: 'xoxb-file-token',
          signingSecret: 'file-secret'
        }
      };

      writeFileSync(testConfigPath, JSON.stringify(fileConfig));

      const config = await loadConfigFromFile(testConfigPath);

      expect(config.mcp.port).toBe(9001);
      expect(config.slack.port).toBe(9002);
    });

    it('should throw error for non-existent file', async () => {
      await expect(loadConfigFromFile('non-existent.json')).rejects.toThrow('Configuration file not found');
    });
  });

  describe('saveConfigToFile', () => {
    it('should save configuration to file', async () => {
      const config = getDefaultConfig();
      config.mcp.port = 10001;
      config.slack.port = 10002;

      await saveConfigToFile(config, testConfigPath);

      expect(existsSync(testConfigPath)).toBe(true);

      const savedConfig = await loadConfigFromFile(testConfigPath);
      expect(savedConfig.mcp.port).toBe(10001);
      expect(savedConfig.slack.port).toBe(10002);
    });
  });

  describe('getEnvironmentConfig', () => {
    it('should return development configuration', () => {
      const config = getEnvironmentConfig('development');

      expect(config.mcp?.logLevel).toBe('debug');
      expect(config.demo?.demoMode).toBe(true);
      expect(config.mcp?.corsOrigins).toEqual(['*']);
    });

    it('should return production configuration', () => {
      const config = getEnvironmentConfig('production');

      expect(config.mcp?.logLevel).toBe('info');
      expect(config.demo?.demoMode).toBe(false);
      expect(config.mcp?.corsOrigins).toEqual([]);
    });

    it('should return test configuration', () => {
      const config = getEnvironmentConfig('test');

      expect(config.mcp?.logLevel).toBe('warn');
      expect(config.mcp?.port).toBe(0);
      expect(config.slack?.port).toBe(0);
      expect(config.slack?.botToken).toBe('xoxb-test-token');
    });

    it('should return empty object for unknown environment', () => {
      const config = getEnvironmentConfig('unknown' as any);

      expect(config).toEqual({});
    });
  });
});