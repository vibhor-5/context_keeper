/**
 * Tests for main application entry point and graceful restart functionality
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { gracefulRestart, startSystem, stopSystem } from './index';
import { MCPServer } from './mcp/server';
import { ContextKeeperSlackBot } from './slack/bot';
import { loadConfig } from './config/config';

// Mock dependencies
vi.mock('./mcp/server');
vi.mock('./slack/bot');
vi.mock('./config/config');
vi.mock('./utils/logger', () => ({
  logger: {
    info: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
    debug: vi.fn()
  }
}));

const mockMCPServer = vi.mocked(MCPServer);
const mockSlackBot = vi.mocked(ContextKeeperSlackBot);
const mockLoadConfig = vi.mocked(loadConfig);

describe('Main Application', () => {
  const mockConfig = {
    mcp: {
      port: 3001,
      goBackendUrl: 'http://localhost:3000',
      timeout: 30000,
      corsOrigins: ['*']
    },
    slack: {
      port: 3002,
      signingSecret: 'test-secret',
      mcpServerUrl: 'http://localhost:3001',
      maxResponseLength: 4000,
      retryAttempts: 3,
      retryDelay: 1000
    }
  };

  let mockMcpServerInstance: any;
  let mockSlackBotInstance: any;

  beforeEach(() => {
    vi.clearAllMocks();
    
    // Create mock instances
    mockMcpServerInstance = {
      start: vi.fn().mockResolvedValue(undefined),
      stop: vi.fn().mockResolvedValue(undefined),
      isHealthy: vi.fn().mockReturnValue(true)
    };
    
    mockSlackBotInstance = {
      start: vi.fn().mockResolvedValue(undefined),
      stop: vi.fn().mockResolvedValue(undefined),
      isHealthy: vi.fn().mockReturnValue(true)
    };
    
    mockMCPServer.mockImplementation(() => mockMcpServerInstance);
    mockSlackBot.mockImplementation(() => mockSlackBotInstance);
    mockLoadConfig.mockResolvedValue(mockConfig);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('startSystem', () => {
    it('should start both MCP server and Slack bot', async () => {
      const components = await startSystem();
      
      expect(mockLoadConfig).toHaveBeenCalledOnce();
      expect(mockMCPServer).toHaveBeenCalledWith(mockConfig.mcp);
      expect(mockSlackBot).toHaveBeenCalledWith(mockConfig.slack);
      expect(mockMcpServerInstance.start).toHaveBeenCalledOnce();
      expect(mockSlackBotInstance.start).toHaveBeenCalledOnce();
      
      expect(components).toEqual({
        mcpServer: mockMcpServerInstance,
        slackBot: mockSlackBotInstance,
        config: mockConfig
      });
    });

    it('should handle MCP server startup failure', async () => {
      const error = new Error('MCP server failed to start');
      mockMcpServerInstance.start.mockRejectedValue(error);
      
      await expect(startSystem()).rejects.toThrow('MCP server failed to start');
      expect(mockMcpServerInstance.start).toHaveBeenCalledOnce();
      expect(mockSlackBotInstance.start).not.toHaveBeenCalled();
    });

    it('should handle Slack bot startup failure', async () => {
      const error = new Error('Slack bot failed to start');
      mockSlackBotInstance.start.mockRejectedValue(error);
      
      await expect(startSystem()).rejects.toThrow('Slack bot failed to start');
      expect(mockMcpServerInstance.start).toHaveBeenCalledOnce();
      expect(mockSlackBotInstance.start).toHaveBeenCalledOnce();
    });
  });

  describe('stopSystem', () => {
    it('should stop both services in reverse order', async () => {
      const components = {
        mcpServer: mockMcpServerInstance,
        slackBot: mockSlackBotInstance,
        config: mockConfig
      };
      
      await stopSystem(components);
      
      expect(mockSlackBotInstance.stop).toHaveBeenCalledOnce();
      expect(mockMcpServerInstance.stop).toHaveBeenCalledOnce();
      
      // Verify order: Slack bot stops before MCP server
      const slackStopCall = mockSlackBotInstance.stop.mock.invocationCallOrder[0];
      const mcpStopCall = mockMcpServerInstance.stop.mock.invocationCallOrder[0];
      expect(slackStopCall).toBeLessThan(mcpStopCall);
    });

    it('should handle Slack bot stop failure', async () => {
      const error = new Error('Slack bot failed to stop');
      mockSlackBotInstance.stop.mockRejectedValue(error);
      
      const components = {
        mcpServer: mockMcpServerInstance,
        slackBot: mockSlackBotInstance,
        config: mockConfig
      };
      
      await expect(stopSystem(components)).rejects.toThrow('Slack bot failed to stop');
      expect(mockSlackBotInstance.stop).toHaveBeenCalledOnce();
      expect(mockMcpServerInstance.stop).not.toHaveBeenCalled();
    });

    it('should handle MCP server stop failure', async () => {
      const error = new Error('MCP server failed to stop');
      mockMcpServerInstance.stop.mockRejectedValue(error);
      
      const components = {
        mcpServer: mockMcpServerInstance,
        slackBot: mockSlackBotInstance,
        config: mockConfig
      };
      
      await expect(stopSystem(components)).rejects.toThrow('MCP server failed to stop');
      expect(mockSlackBotInstance.stop).toHaveBeenCalledOnce();
      expect(mockMcpServerInstance.stop).toHaveBeenCalledOnce();
    });
  });

  describe('gracefulRestart', () => {
    it('should restart system with fresh configuration', async () => {
      // Mock the module-level currentComponents
      const originalStartSystem = startSystem;
      const originalStopSystem = stopSystem;
      
      // First call to start initial system
      const initialComponents = await originalStartSystem();
      
      // Mock new configuration for restart
      const newConfig = {
        ...mockConfig,
        mcp: { ...mockConfig.mcp, port: 3003 }
      };
      mockLoadConfig.mockResolvedValueOnce(newConfig);
      
      // Create new mock instances for restart
      const newMcpInstance = {
        start: vi.fn().mockResolvedValue(undefined),
        stop: vi.fn().mockResolvedValue(undefined),
        isHealthy: vi.fn().mockReturnValue(true)
      };
      const newSlackInstance = {
        start: vi.fn().mockResolvedValue(undefined),
        stop: vi.fn().mockResolvedValue(undefined),
        isHealthy: vi.fn().mockReturnValue(true)
      };
      
      mockMCPServer.mockImplementationOnce(() => newMcpInstance);
      mockSlackBot.mockImplementationOnce(() => newSlackInstance);
      
      await gracefulRestart();
      
      // Verify old system was stopped
      expect(initialComponents.slackBot.stop).toHaveBeenCalled();
      expect(initialComponents.mcpServer.stop).toHaveBeenCalled();
      
      // Verify new system was started with new configuration
      expect(newMcpInstance.start).toHaveBeenCalled();
      expect(newSlackInstance.start).toHaveBeenCalled();
    });

    it('should handle concurrent restart requests', async () => {
      // Start multiple restart requests concurrently
      const restartPromises = [
        gracefulRestart(),
        gracefulRestart(),
        gracefulRestart()
      ];
      
      await Promise.all(restartPromises);
      
      // Should only perform one actual restart
      expect(mockLoadConfig).toHaveBeenCalledTimes(1);
    });

    it('should handle restart failure and attempt fallback', async () => {
      // Mock restart failure
      mockMcpServerInstance.start.mockRejectedValueOnce(new Error('Restart failed'));
      
      // Mock successful fallback
      const fallbackMcpInstance = {
        start: vi.fn().mockResolvedValue(undefined),
        stop: vi.fn().mockResolvedValue(undefined),
        isHealthy: vi.fn().mockReturnValue(true)
      };
      const fallbackSlackInstance = {
        start: vi.fn().mockResolvedValue(undefined),
        stop: vi.fn().mockResolvedValue(undefined),
        isHealthy: vi.fn().mockReturnValue(true)
      };
      
      mockMCPServer.mockImplementationOnce(() => fallbackMcpInstance);
      mockSlackBot.mockImplementationOnce(() => fallbackSlackInstance);
      
      await expect(gracefulRestart()).rejects.toThrow('Restart failed');
      
      // Verify fallback was attempted
      expect(fallbackMcpInstance.start).toHaveBeenCalled();
      expect(fallbackSlackInstance.start).toHaveBeenCalled();
    });
  });
});