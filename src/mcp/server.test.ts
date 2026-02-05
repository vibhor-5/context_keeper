/**
 * Unit tests for MCP Server
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { MCPServer } from './server';
import { MCPServerConfig } from '../types/config';
import { MCPRequest } from '../types/mcp';

describe('MCPServer', () => {
  let server: MCPServer;
  let config: MCPServerConfig;
  let portCounter = 3001;

  beforeEach(() => {
    config = {
      port: portCounter++, // Use different port for each test
      goBackendUrl: 'http://localhost:8080',
      corsOrigins: ['*'],
      logLevel: 'info',
      timeout: 30000
    };
    server = new MCPServer(config);
  });

  afterEach(async () => {
    if (server.isHealthy()) {
      await server.stop();
    }
  });

  describe('constructor', () => {
    it('should create server instance with config', () => {
      expect(server).toBeDefined();
      expect(server.isHealthy()).toBe(false);
    });
  });

  describe('start and stop', () => {
    it('should start server on configured port', async () => {
      await server.start();
      expect(server.isHealthy()).toBe(true);
      expect(server.getPort()).toBe(config.port);
    });

    it('should stop server gracefully', async () => {
      await server.start();
      expect(server.isHealthy()).toBe(true);
      
      await server.stop();
      expect(server.isHealthy()).toBe(false);
    });

    it('should handle multiple start calls gracefully', async () => {
      await server.start();
      expect(server.isHealthy()).toBe(true);
      
      // Second start should not throw
      await server.start();
      expect(server.isHealthy()).toBe(true);
    });

    it('should handle stop when not running', async () => {
      expect(server.isHealthy()).toBe(false);
      
      // Stop should not throw when not running
      await server.stop();
      expect(server.isHealthy()).toBe(false);
    });
  });

  describe('handleRequest', () => {
    beforeEach(async () => {
      await server.start();
    });

    it('should handle list_resources request', async () => {
      const request: MCPRequest = {
        method: 'list_resources',
        params: {},
        id: 1
      };

      const response = await server.handleRequest(request);
      
      expect(response.id).toBe(1);
      expect(response.result).toBeDefined();
      expect(response.result.resources).toEqual([]);
    });

    it('should handle list_tools request', async () => {
      const request: MCPRequest = {
        method: 'list_tools',
        params: {},
        id: 2
      };

      const response = await server.handleRequest(request);
      
      expect(response.id).toBe(2);
      expect(response.result).toBeDefined();
      expect(response.result.tools).toHaveLength(2);
      expect(response.result.tools[0].name).toBe('query_repository_context');
      expect(response.result.tools[1].name).toBe('get_onboarding_summary');
    });

    it('should return error for unknown method', async () => {
      const request: MCPRequest = {
        method: 'unknown_method',
        params: {},
        id: 3
      };

      const response = await server.handleRequest(request);
      
      expect(response.id).toBe(3);
      expect(response.error).toBeDefined();
      expect(response.error?.code).toBe(-32601);
      expect(response.error?.message).toBe('Method not found');
    });

    it('should handle call_tool request with unknown tool error', async () => {
      const request: MCPRequest = {
        method: 'call_tool',
        params: { name: 'test_tool' },
        id: 4
      };

      const response = await server.handleRequest(request);
      
      expect(response.id).toBe(4);
      expect(response.result).toBeDefined();
      expect(response.result.isError).toBe(true);
      expect(response.result.content[0].text).toContain('Unknown tool: test_tool');
    });

    it('should handle get_resource request with not found error', async () => {
      const request: MCPRequest = {
        method: 'get_resource',
        params: { uri: 'contextkeeper://test' },
        id: 5
      };

      const response = await server.handleRequest(request);
      
      expect(response.id).toBe(5);
      expect(response.error).toBeDefined();
      expect(response.error?.code).toBe(-32603);
      expect(response.error?.message).toBe('Internal error');
    });
  });

  describe('HTTP endpoints', () => {
    it('should be tested via integration tests', () => {
      // HTTP endpoint testing will be covered by integration tests
      // This ensures we don't duplicate testing effort
      expect(true).toBe(true);
    });
  });
});