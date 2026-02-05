/**
 * Integration tests for MCP Server HTTP endpoints
 */

import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { MCPServer } from './server';
import { MCPServerConfig } from '../types/config';

describe('MCPServer HTTP Integration', () => {
  let server: MCPServer;
  let config: MCPServerConfig;
  let baseUrl: string;

  beforeEach(async () => {
    config = {
      port: 3100 + Math.floor(Math.random() * 100), // Random port to avoid conflicts
      goBackendUrl: 'http://localhost:8080',
      corsOrigins: ['*'],
      logLevel: 'info',
      timeout: 30000
    };
    server = new MCPServer(config);
    baseUrl = `http://localhost:${config.port}`;
    await server.start();
  });

  afterEach(async () => {
    if (server.isHealthy()) {
      await server.stop();
    }
  });

  describe('HTTP endpoints', () => {
    it('should respond to health check', async () => {
      const response = await fetch(`${baseUrl}/health`);
      expect(response.status).toBe(200);
      
      const data = await response.json();
      expect(data.status).toBe('healthy');
      expect(data.timestamp).toBeDefined();
      expect(data.version).toBe('1.0.0');
    });

    it('should handle MCP list_resources request', async () => {
      const mcpRequest = {
        method: 'list_resources',
        params: {},
        id: 1
      };

      const response = await fetch(`${baseUrl}/mcp`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(mcpRequest)
      });

      expect(response.status).toBe(200);
      
      const data = await response.json();
      expect(data.id).toBe(1);
      expect(data.result).toBeDefined();
      expect(data.result.resources).toEqual([]);
    });

    it('should handle MCP list_tools request', async () => {
      const mcpRequest = {
        method: 'list_tools',
        params: {},
        id: 2
      };

      const response = await fetch(`${baseUrl}/mcp`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(mcpRequest)
      });

      expect(response.status).toBe(200);
      
      const data = await response.json();
      expect(data.id).toBe(2);
      expect(data.result).toBeDefined();
      expect(data.result.tools).toEqual([]);
    });

    it('should return error for invalid MCP request', async () => {
      const invalidRequest = {
        params: {},
        id: 3
        // Missing method field
      };

      const response = await fetch(`${baseUrl}/mcp`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(invalidRequest)
      });

      expect(response.status).toBe(400);
      
      const data = await response.json();
      expect(data.error).toBeDefined();
      expect(data.error.code).toBe(-32600);
      expect(data.error.message).toBe('Invalid Request');
    });

    it('should return 404 for unknown endpoints', async () => {
      const response = await fetch(`${baseUrl}/unknown`);
      expect(response.status).toBe(404);
      
      const data = await response.json();
      expect(data.error).toBeDefined();
      expect(data.error.code).toBe(-32601);
      expect(data.error.message).toBe('Method not found');
    });

    it('should handle CORS preflight requests', async () => {
      const response = await fetch(`${baseUrl}/mcp`, {
        method: 'OPTIONS',
        headers: {
          'Origin': 'http://localhost:3000',
          'Access-Control-Request-Method': 'POST',
          'Access-Control-Request-Headers': 'Content-Type'
        }
      });

      // CORS middleware should handle OPTIONS requests successfully
      expect(response.status).toBe(204);
    });
  });
});