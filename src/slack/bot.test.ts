/**
 * Unit tests for Slack Bot
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { ContextKeeperSlackBot } from './bot';
import { SlackBotConfig } from '../types/config';
import { SlackCommand } from '../types/slack';

describe('ContextKeeperSlackBot', () => {
  let bot: ContextKeeperSlackBot;
  let config: SlackBotConfig;

  beforeEach(() => {
    config = {
      botToken: 'xoxb-test-token',
      signingSecret: 'test-signing-secret',
      mcpServerUrl: 'http://localhost:3001',
      port: 3100,
      maxResponseLength: 4000,
      retryAttempts: 3,
      retryDelay: 1000
    };

    bot = new ContextKeeperSlackBot(config);
  });

  afterEach(async () => {
    if (bot.isHealthy()) {
      await bot.stop();
    }
  });

  describe('constructor', () => {
    it('should create bot instance with config', () => {
      expect(bot).toBeDefined();
      expect(bot.getPort()).toBe(3100);
      expect(bot.isHealthy()).toBe(false);
    });
  });

  describe('start and stop', () => {
    it('should start bot on configured port', async () => {
      await bot.start();
      
      expect(bot.isHealthy()).toBe(true);
      expect(bot.getPort()).toBe(3100);
      
      await bot.stop();
      expect(bot.isHealthy()).toBe(false);
    });

    it('should stop bot gracefully', async () => {
      await bot.start();
      expect(bot.isHealthy()).toBe(true);
      
      await bot.stop();
      expect(bot.isHealthy()).toBe(false);
    });

    it('should handle multiple start calls gracefully', async () => {
      await bot.start();
      expect(bot.isHealthy()).toBe(true);
      
      // Second start should not throw
      await bot.start();
      expect(bot.isHealthy()).toBe(true);
      
      await bot.stop();
    });

    it('should handle stop when not running', async () => {
      expect(bot.isHealthy()).toBe(false);
      
      // Should not throw
      await bot.stop();
      expect(bot.isHealthy()).toBe(false);
    });
  });

  describe('handleCommand', () => {
    const createTestCommand = (command: string, text: string = ''): SlackCommand => ({
      command,
      text,
      userId: 'U123456',
      channelId: 'C123456',
      teamId: 'T123456',
      userName: 'testuser',
      responseUrl: 'https://hooks.slack.com/commands/123/456'
    });

    it('should handle /context command', async () => {
      const command = createTestCommand('/context', 'What is this repository about?');
      const response = await bot.handleCommand(command);

      expect(response.responseType).toBe('ephemeral');
      // Since MCP server isn't running in tests, expect error response
      expect(response.text).toContain('🔌'); // Network error emoji
      expect(response.text).toContain('Unable to connect');
    });

    it('should handle /onboard command', async () => {
      const command = createTestCommand('/onboard', 'my-repo');
      const response = await bot.handleCommand(command);

      expect(response.responseType).toBe('ephemeral');
      // Since MCP server isn't running in tests, expect error response
      expect(response.text).toContain('🔌'); // Network error emoji
      expect(response.text).toContain('Unable to connect');
    });

    it('should handle /recent command', async () => {
      const command = createTestCommand('/recent', '7');
      const response = await bot.handleCommand(command);

      expect(response.responseType).toBe('ephemeral');
      // Since MCP server isn't running in tests, expect error response
      expect(response.text).toContain('🔌'); // Network error emoji
      expect(response.text).toContain('Unable to connect');
    });

    it('should handle /status command', async () => {
      const command = createTestCommand('/status');
      const response = await bot.handleCommand(command);

      expect(response.responseType).toBe('ephemeral');
      // Status command returns status even when MCP server is down
      expect(response.text).toContain('📊');
      expect(response.text).toContain('System Status');
      expect(response.text).toContain('❌'); // Should show down status
    });

    it('should handle unknown command with help response', async () => {
      const command = createTestCommand('/unknown');
      const response = await bot.handleCommand(command);

      expect(response.responseType).toBe('ephemeral');
      expect(response.text).toContain('Unknown command: /unknown');
      expect(response.text).toContain('Available commands:');
      expect(response.text).toContain('/context');
      expect(response.text).toContain('/onboard');
      expect(response.text).toContain('/recent');
      expect(response.text).toContain('/status');
    });

    it('should handle empty command text', async () => {
      const command = createTestCommand('/context', '');
      const response = await bot.handleCommand(command);

      expect(response.responseType).toBe('ephemeral');
      expect(response.text).toContain('❓ Please provide a query');
      expect(response.text).toContain('Usage:');
    });
  });

  describe('HTTP endpoints', () => {
    it('should be tested via integration tests', () => {
      // HTTP endpoint testing would require more complex setup
      // with request mocking and signature verification
      // This is a placeholder for integration tests
      expect(true).toBe(true);
    });
  });
});