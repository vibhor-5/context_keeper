/**
 * Property-based tests for Slack Bot Command Processing
 * Feature: contextkeeper-mcp-slack-bot, Property 5: Slack Command Processing
 * Validates: Requirements 2.1, 2.2, 2.3, 2.4
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import fc from 'fast-check';
import request from 'supertest';
import { createHmac } from 'crypto';
import { ContextKeeperSlackBot } from './bot';
import { SlackMCPClient } from './mcpClient';

// Mock the MCP client
vi.mock('./mcpClient');
const MockedSlackMCPClient = vi.mocked(SlackMCPClient);

// Helper function to create valid Slack signatures for testing
function createSlackSignature(body: string, timestamp: string, signingSecret: string): string {
  const sigBasestring = `v0:${timestamp}:${body}`;
  return `v0=${createHmac('sha256', signingSecret).update(sigBasestring).digest('hex')}`;
}

describe('Slack Bot Command Processing Properties', () => {
  let bot: ContextKeeperSlackBot;
  let mockMCPClient: any;

  beforeEach(() => {
    vi.clearAllMocks();

    mockMCPClient = {
      queryContext: vi.fn(),
      getOnboardingSummary: vi.fn(),
      getRecentActivity: vi.fn(),
      getSystemStatus: vi.fn(),
      healthCheck: vi.fn()
    };

    MockedSlackMCPClient.mockImplementation(() => mockMCPClient);

    bot = new ContextKeeperSlackBot({
      botToken: 'xoxb-test-token',
      signingSecret: 'test-signing-secret',
      mcpServerUrl: 'http://localhost:3001',
      port: 3002,
      maxResponseLength: 4000,
      retryAttempts: 3,
      retryDelay: 1000
    });
  });

  /**
   * Property 5: Slack Command Processing
   * For any valid Slack command (/context, /onboard, /recent, /status), 
   * the Slack bot should query the MCP server and return appropriately formatted responses.
   * Validates: Requirements 2.1, 2.2, 2.3, 2.4
   */
  it('should process all valid Slack commands correctly', async () => {
    await fc.assert(fc.asyncProperty(
      fc.record({
        command: fc.constantFrom('/context', '/onboard', '/recent', '/status'),
        text: fc.string({ minLength: 0, maxLength: 100 }),
        userId: fc.string({ minLength: 5, maxLength: 20 }).map(s => `U${s}`),
        channelId: fc.string({ minLength: 5, maxLength: 20 }).map(s => `C${s}`),
        teamId: fc.string({ minLength: 5, maxLength: 20 }).map(s => `T${s}`),
        timestamp: fc.float({ min: 1600000000, max: 2000000000 }).map(t => t.toString())
      }),
      async (slackCommand) => {
        // Setup mock responses based on command type
        switch (slackCommand.command) {
          case '/context':
            mockMCPClient.queryContext.mockResolvedValue('Mock context response');
            break;
          case '/onboard':
            mockMCPClient.getOnboardingSummary.mockResolvedValue('Mock onboarding response');
            break;
          case '/recent':
            mockMCPClient.getRecentActivity.mockResolvedValue('Mock recent activity response');
            break;
          case '/status':
            mockMCPClient.getSystemStatus.mockResolvedValue({
              mcpServer: { status: 'healthy', uptime: 12345, version: '1.0.0' },
              goBackend: { status: 'healthy', responseTime: 150, lastCheck: new Date().toISOString() },
              repositories: { total: 5, active: 5, ingesting: 0, error: 0 }
            });
            break;
        }

        // Create valid Slack request body
        const requestBody = new URLSearchParams({
          token: 'verification-token',
          team_id: slackCommand.teamId,
          team_domain: 'test-team',
          channel_id: slackCommand.channelId,
          channel_name: 'general',
          user_id: slackCommand.userId,
          user_name: 'testuser',
          command: slackCommand.command,
          text: slackCommand.text,
          response_url: 'https://hooks.slack.com/commands/test',
          trigger_id: 'test-trigger-id'
        }).toString();

        // Create valid Slack signature
        const mockTimestamp = Math.floor(Date.now() / 1000).toString();
        const mockSignature = createSlackSignature(requestBody, mockTimestamp, 'test-signing-secret');

        const response = await request(bot.getApp())
          .post('/slack/commands')
          .set('Content-Type', 'application/x-www-form-urlencoded')
          .set('X-Slack-Request-Timestamp', mockTimestamp)
          .set('X-Slack-Signature', mockSignature)
          .send(requestBody);

        // Property: All valid commands should return 200 status
        expect(response.status).toBe(200);

        // Property: All responses should have proper Slack response format
        expect(response.body).toHaveProperty('text');
        expect(typeof response.body.text).toBe('string');
        expect(response.body.text.length).toBeGreaterThan(0);

        // Property: Response should be ephemeral (private to user)
        expect(response.body.responseType).toBe('ephemeral');

        // Property: Response should not exceed maximum length
        expect(response.body.text.length).toBeLessThanOrEqual(4000);

        // Property: MCP client should be called for each command
        switch (slackCommand.command) {
          case '/context':
            expect(mockMCPClient.queryContext).toHaveBeenCalled();
            break;
          case '/onboard':
            expect(mockMCPClient.getOnboardingSummary).toHaveBeenCalled();
            break;
          case '/recent':
            expect(mockMCPClient.getRecentActivity).toHaveBeenCalled();
            break;
          case '/status':
            expect(mockMCPClient.getSystemStatus).toHaveBeenCalled();
            break;
        }
      }
    ), { numRuns: 100 });
  });

  it('should handle command parameter variations correctly', async () => {
    await fc.assert(fc.asyncProperty(
      fc.record({
        command: fc.constantFrom('/context', '/onboard', '/recent'),
        parameters: fc.record({
          query: fc.option(fc.string({ minLength: 1, maxLength: 200 })),
          repository: fc.option(fc.string({ minLength: 1, maxLength: 50 })),
          days: fc.option(fc.integer({ min: 1, max: 30 }))
        })
      }),
      async (commandData) => {
        // Build text parameter based on command type
        let textParam = '';
        if (commandData.command === '/context' && commandData.parameters.query) {
          textParam = commandData.parameters.query;
          if (commandData.parameters.repository) {
            textParam += ` repo:${commandData.parameters.repository}`;
          }
        } else if (commandData.command === '/onboard' && commandData.parameters.repository) {
          textParam = commandData.parameters.repository;
        } else if (commandData.command === '/recent') {
          if (commandData.parameters.repository) {
            textParam = commandData.parameters.repository;
          }
          if (commandData.parameters.days) {
            textParam += ` days:${commandData.parameters.days}`;
          }
        }

        // Setup appropriate mock responses
        mockMCPClient.queryContext.mockResolvedValue('Mock context response');
        mockMCPClient.getOnboardingSummary.mockResolvedValue('Mock onboarding response');
        mockMCPClient.getRecentActivity.mockResolvedValue('Mock recent activity response');

        const requestBody = new URLSearchParams({
          token: 'verification-token',
          team_id: 'T12345',
          team_domain: 'test-team',
          channel_id: 'C12345',
          channel_name: 'general',
          user_id: 'U12345',
          user_name: 'testuser',
          command: commandData.command,
          text: textParam,
          response_url: 'https://hooks.slack.com/commands/test',
          trigger_id: 'test-trigger-id'
        }).toString();

        const mockTimestamp = Math.floor(Date.now() / 1000).toString();
        const mockSignature = createSlackSignature(requestBody, mockTimestamp, 'test-signing-secret');

        const response = await request(bot.getApp())
          .post('/slack/commands')
          .set('Content-Type', 'application/x-www-form-urlencoded')
          .set('X-Slack-Request-Timestamp', mockTimestamp)
          .set('X-Slack-Signature', mockSignature)
          .send(requestBody);

        // Property: Parameter parsing should not cause errors
        expect(response.status).toBe(200);
        expect(response.body).toHaveProperty('text');
        expect(typeof response.body.text).toBe('string');

        // Property: Commands with parameters should pass them to MCP client
        if (commandData.command === '/context' && commandData.parameters.query) {
          expect(mockMCPClient.queryContext).toHaveBeenCalledWith(
            expect.any(String),
            commandData.parameters.repository || undefined
          );
        }

        if (commandData.command === '/onboard' && commandData.parameters.repository) {
          expect(mockMCPClient.getOnboardingSummary).toHaveBeenCalledWith(
            commandData.parameters.repository
          );
        }

        if (commandData.command === '/recent') {
          expect(mockMCPClient.getRecentActivity).toHaveBeenCalledWith(
            commandData.parameters.repository || undefined,
            commandData.parameters.days || 7
          );
        }
      }
    ), { numRuns: 50 });
  });

  it('should handle MCP client errors gracefully', async () => {
    await fc.assert(fc.asyncProperty(
      fc.record({
        command: fc.constantFrom('/context', '/onboard', '/recent', '/status'),
        errorType: fc.constantFrom('network', 'timeout', 'invalid_response', 'server_error')
      }),
      async (testCase) => {
        // Setup error responses based on error type
        const errorMessage = `Mock ${testCase.errorType} error`;
        const error = new Error(errorMessage);

        switch (testCase.command) {
          case '/context':
            mockMCPClient.queryContext.mockRejectedValue(error);
            break;
          case '/onboard':
            mockMCPClient.getOnboardingSummary.mockRejectedValue(error);
            break;
          case '/recent':
            mockMCPClient.getRecentActivity.mockRejectedValue(error);
            break;
          case '/status':
            mockMCPClient.getSystemStatus.mockRejectedValue(error);
            break;
        }

        const requestBody = new URLSearchParams({
          token: 'verification-token',
          team_id: 'T12345',
          team_domain: 'test-team',
          channel_id: 'C12345',
          channel_name: 'general',
          user_id: 'U12345',
          user_name: 'testuser',
          command: testCase.command,
          text: 'test query',
          response_url: 'https://hooks.slack.com/commands/test',
          trigger_id: 'test-trigger-id'
        }).toString();

        const mockTimestamp = Math.floor(Date.now() / 1000).toString();
        const mockSignature = createSlackSignature(requestBody, mockTimestamp, 'test-signing-secret');

        const response = await request(bot.getApp())
          .post('/slack/commands')
          .set('Content-Type', 'application/x-www-form-urlencoded')
          .set('X-Slack-Request-Timestamp', mockTimestamp)
          .set('X-Slack-Signature', mockSignature)
          .send(requestBody);

        // Property: Errors should not crash the bot
        expect(response.status).toBe(200);

        // Property: Error responses should be user-friendly
        expect(response.body).toHaveProperty('text');
        expect(response.body.text).toContain('⚠️');
        expect(response.body.text.length).toBeGreaterThan(0);

        // Property: Error responses should still be ephemeral
        expect(response.body.responseType).toBe('ephemeral');

        // Property: Error messages should not expose internal details
        expect(response.body.text).not.toContain('Error:');
        expect(response.body.text).not.toContain('Stack trace');
        expect(response.body.text).not.toContain('at ');
      }
    ), { numRuns: 50 });
  });

  it('should validate command format and structure', async () => {
    await fc.assert(fc.asyncProperty(
      fc.record({
        validCommand: fc.boolean(),
        command: fc.oneof(
          fc.constantFrom('/context', '/onboard', '/recent', '/status'), // Valid commands
          fc.string({ minLength: 1, maxLength: 20 }).map(s => `/${s}`) // Invalid commands
        ),
        hasRequiredFields: fc.boolean()
      }),
      async (testCase) => {
        const baseRequestBody = {
          token: 'verification-token',
          team_id: 'T12345',
          team_domain: 'test-team',
          channel_id: 'C12345',
          channel_name: 'general',
          user_id: 'U12345',
          user_name: 'testuser',
          command: testCase.command,
          text: 'test',
          response_url: 'https://hooks.slack.com/commands/test',
          trigger_id: 'test-trigger-id'
        };

        // Conditionally remove required fields to test validation
        const requestBody = testCase.hasRequiredFields 
          ? baseRequestBody 
          : { ...baseRequestBody, user_id: undefined, channel_id: undefined };

        // Setup mock responses for valid commands
        if (['/context', '/onboard', '/recent', '/status'].includes(testCase.command)) {
          mockMCPClient.queryContext.mockResolvedValue('Mock response');
          mockMCPClient.getOnboardingSummary.mockResolvedValue('Mock response');
          mockMCPClient.getRecentActivity.mockResolvedValue('Mock response');
          mockMCPClient.getSystemStatus.mockResolvedValue({
            mcpServer: { status: 'healthy', uptime: 12345, version: '1.0.0' },
            goBackend: { status: 'healthy', responseTime: 150, lastCheck: new Date().toISOString() },
            repositories: { total: 5, active: 5, ingesting: 0, error: 0 }
          });
        }

        const requestBodyString = new URLSearchParams(
          Object.fromEntries(
            Object.entries(requestBody).filter(([_, value]) => value !== undefined)
          )
        ).toString();

        const mockTimestamp = Math.floor(Date.now() / 1000).toString();
        const mockSignature = createSlackSignature(requestBodyString, mockTimestamp, 'test-signing-secret');

        const response = await request(bot.getApp())
          .post('/slack/commands')
          .set('Content-Type', 'application/x-www-form-urlencoded')
          .set('X-Slack-Request-Timestamp', mockTimestamp)
          .set('X-Slack-Signature', mockSignature)
          .send(requestBodyString);

        const isValidCommand = ['/context', '/onboard', '/recent', '/status'].includes(testCase.command);

        if (isValidCommand && testCase.hasRequiredFields) {
          // Property: Valid commands with required fields should succeed
          expect(response.status).toBe(200);
          expect(response.body).toHaveProperty('text');
          expect(response.body.responseType).toBe('ephemeral');
        } else {
          // Property: Invalid commands or missing fields should return appropriate responses
          expect(response.status).toBe(200);
          expect(response.body).toHaveProperty('text');
          
          if (!isValidCommand) {
            // Should contain help or error message for unknown commands
            expect(response.body.text).toMatch(/unknown|help|available|commands/i);
          }
        }
      }
    ), { numRuns: 75 });
  });
});