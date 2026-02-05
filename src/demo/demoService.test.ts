/**
 * Tests for demo service
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { DemoService, createDemoService } from './demoService';
import { GoBackendClient } from '../services/goBackendClient';

// Mock the Go backend client
vi.mock('../services/goBackendClient');

describe('DemoService', () => {
  let demoService: DemoService;
  let mockGoBackendClient: vi.Mocked<GoBackendClient>;

  beforeEach(() => {
    mockGoBackendClient = {
      queryContext: vi.fn(),
      getRepositoryStatus: vi.fn(),
      healthCheck: vi.fn()
    } as any;

    demoService = createDemoService({
      enabled: false,
      fallbackOnError: true,
      predictableResponses: true,
      fallbackTimeout: 5000,
      maxFallbackAttempts: 2
    }, mockGoBackendClient);
  });

  describe('Configuration', () => {
    it('should create service with default configuration', () => {
      const service = createDemoService();
      const config = service.getConfig();
      
      expect(config.enabled).toBe(false);
      expect(config.fallbackOnError).toBe(true);
      expect(config.predictableResponses).toBe(true);
    });

    it('should update configuration', () => {
      demoService.updateConfig({ enabled: true, predictableResponses: false });
      const config = demoService.getConfig();
      
      expect(config.enabled).toBe(true);
      expect(config.predictableResponses).toBe(false);
    });

    it('should check demo mode status', () => {
      expect(demoService.isDemoMode()).toBe(false);
      
      demoService.updateConfig({ enabled: true });
      expect(demoService.isDemoMode()).toBe(true);
    });

    it('should check fallback enabled status', () => {
      expect(demoService.isFallbackEnabled()).toBe(true);
      
      demoService.updateConfig({ fallbackOnError: false });
      expect(demoService.isFallbackEnabled()).toBe(false);
    });
  });

  describe('Context Queries', () => {
    it('should use demo data when demo mode is enabled', async () => {
      demoService.updateConfig({ enabled: true });
      
      const result = await demoService.queryContext('How does authentication work?', 'demo-repo-1');
      
      expect(result.source).toBe('demo');
      expect(result.isComplete).toBe(true);
      expect(result.data).toContain('OAuth2');
      expect(mockGoBackendClient.queryContext).not.toHaveBeenCalled();
    });

    it('should use live data when demo mode is disabled and backend is available', async () => {
      mockGoBackendClient.queryContext.mockResolvedValue({
        response: 'Live response from backend',
        confidence: 0.9,
        sources: ['live-source.ts']
      });

      const result = await demoService.queryContext('test query', 'demo-repo-1');
      
      expect(result.source).toBe('live');
      expect(result.isComplete).toBe(true);
      expect(result.data).toBe('Live response from backend');
      expect(mockGoBackendClient.queryContext).toHaveBeenCalledWith('test query', 'demo-repo-1');
    });

    it('should fallback to demo data when live backend fails', async () => {
      mockGoBackendClient.queryContext.mockRejectedValue(new Error('Backend unavailable'));

      const result = await demoService.queryContext('test query', 'demo-repo-1');
      
      expect(result.source).toBe('fallback');
      expect(result.isComplete).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.data).toContain('OAuth2'); // Demo data
    });

    it('should use demo data when no backend client is provided', async () => {
      const serviceWithoutBackend = createDemoService({ enabled: false });
      
      const result = await serviceWithoutBackend.queryContext('test query');
      
      expect(result.source).toBe('demo');
      expect(result.isComplete).toBe(true);
    });
  });

  describe('Onboarding Summaries', () => {
    it('should use demo data when demo mode is enabled', async () => {
      demoService.updateConfig({ enabled: true });
      
      const result = await demoService.getOnboardingSummary('demo-repo-1');
      
      expect(result.source).toBe('demo');
      expect(result.isComplete).toBe(true);
      expect(result.data).toContain('Welcome to ContextKeeper Frontend');
    });

    it('should use live data when demo mode is disabled and backend is available', async () => {
      mockGoBackendClient.getRepositoryStatus.mockResolvedValue({
        status: 'active',
        lastIngestion: new Date().toISOString(),
        fileCount: 150,
        errorCount: 0
      });

      const result = await demoService.getOnboardingSummary('test-repo');
      
      expect(result.source).toBe('live');
      expect(result.isComplete).toBe(true);
      expect(result.data).toContain('Welcome to test-repo');
      expect(result.data).toContain('Status: active');
    });

    it('should fallback to demo data when live backend fails', async () => {
      mockGoBackendClient.getRepositoryStatus.mockRejectedValue(new Error('Backend error'));

      const result = await demoService.getOnboardingSummary('demo-repo-1');
      
      expect(result.source).toBe('fallback');
      expect(result.isComplete).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.data).toContain('Welcome to ContextKeeper Frontend');
    });
  });

  describe('Recent Activity', () => {
    it('should use demo data when demo mode is enabled', async () => {
      demoService.updateConfig({ enabled: true });
      
      const result = await demoService.getRecentActivity('demo-repo-1', 7);
      
      expect(result.source).toBe('demo');
      expect(result.isComplete).toBe(true);
      expect(result.data).toContain('Recent activity');
      expect(result.data).toContain('alice.dev');
    });

    it('should use live data when demo mode is disabled and backend is available', async () => {
      mockGoBackendClient.getRepositoryStatus.mockResolvedValue({
        status: 'active',
        lastIngestion: new Date().toISOString(),
        fileCount: 200,
        errorCount: 0
      });

      const result = await demoService.getRecentActivity('test-repo', 14);
      
      expect(result.source).toBe('live');
      expect(result.isComplete).toBe(true);
      expect(result.data).toContain('Recent activity for test-repo');
      expect(result.data).toContain('Files Processed: 200');
    });

    it('should fallback to demo data when live backend fails', async () => {
      mockGoBackendClient.getRepositoryStatus.mockRejectedValue(new Error('Network error'));

      const result = await demoService.getRecentActivity('demo-repo-1', 7);
      
      expect(result.source).toBe('fallback');
      expect(result.isComplete).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.data).toContain('alice.dev'); // Demo data
    });
  });

  describe('System Status', () => {
    it('should use demo data when demo mode is enabled', async () => {
      demoService.updateConfig({ enabled: true });
      
      const result = await demoService.getSystemStatus();
      
      expect(result.source).toBe('demo');
      expect(result.isComplete).toBe(true);
      expect(result.data).toHaveProperty('mcpServer');
      expect(result.data).toHaveProperty('goBackend');
      expect(result.data).toHaveProperty('repositories');
    });

    it('should collect partial live data when backend is available', async () => {
      mockGoBackendClient.healthCheck.mockResolvedValue();

      const result = await demoService.getSystemStatus();
      
      expect(result.data.mcpServer.status).toBe('healthy');
      expect(result.data.goBackend.status).toBe('healthy');
      expect(result.data.goBackend.responseTime).toBeGreaterThan(0);
    });

    it('should handle backend failures gracefully', async () => {
      mockGoBackendClient.healthCheck.mockRejectedValue(new Error('Backend down'));

      const result = await demoService.getSystemStatus();
      
      expect(result.source).toBe('fallback');
      expect(result.data.mcpServer.status).toBe('healthy');
      expect(result.data.goBackend.status).toBe('down');
      expect(result.errors).toBeDefined();
    });

    it('should fallback completely when all operations fail', async () => {
      // Mock all operations to fail
      const serviceWithoutBackend = createDemoService({ enabled: false });
      
      const result = await serviceWithoutBackend.getSystemStatus();
      
      expect(result.source).toBe('fallback');
      expect(result.isComplete).toBe(false);
      expect(result.data.goBackend.status).toBe('down');
    });
  });

  describe('Demo Data Management', () => {
    it('should reset demo data', () => {
      const demoDataManager = demoService.getDemoDataManager();
      
      // Add some test data
      demoDataManager.addRepository({
        id: 'temp-repo',
        name: 'Temporary',
        description: 'Temp',
        language: 'Python',
        contributors: [],
        lastActivity: new Date().toISOString(),
        status: 'active',
        structure: { directories: [], keyFiles: [], testCoverage: 0 },
        recentActivity: [],
        contextSamples: [],
        onboardingGuide: 'Temp'
      });
      
      // Reset
      demoService.resetDemoData();
      
      // Check that temp data is gone
      const tempRepo = demoDataManager.getRepository('temp-repo');
      expect(tempRepo).toBeNull();
    });

    it('should provide access to demo data manager', () => {
      const demoDataManager = demoService.getDemoDataManager();
      expect(demoDataManager).toBeDefined();
      expect(typeof demoDataManager.getRepository).toBe('function');
    });
  });

  describe('Error Handling', () => {
    it('should handle timeout errors', async () => {
      demoService.updateConfig({ fallbackTimeout: 100 }); // Very short timeout
      
      mockGoBackendClient.queryContext.mockImplementation(() => 
        new Promise(resolve => setTimeout(resolve, 200)) // Longer than timeout
      );

      const result = await demoService.queryContext('test query');
      
      expect(result.source).toBe('fallback');
      expect(result.isComplete).toBe(false);
    });

    it('should respect max fallback attempts', async () => {
      demoService.updateConfig({ maxFallbackAttempts: 1 });
      
      mockGoBackendClient.queryContext.mockRejectedValue(new Error('Persistent error'));

      const result = await demoService.queryContext('test query');
      
      expect(result.source).toBe('fallback');
      expect(mockGoBackendClient.queryContext).toHaveBeenCalledTimes(1);
    });

    it('should handle network errors gracefully', async () => {
      const networkError = new Error('ECONNREFUSED');
      (networkError as any).code = 'ECONNREFUSED';
      
      mockGoBackendClient.queryContext.mockRejectedValue(networkError);

      const result = await demoService.queryContext('test query');
      
      expect(result.source).toBe('fallback');
      expect(result.errors?.[0]).toContain('Unable to connect');
    });
  });

  describe('Integration', () => {
    it('should work with different configuration combinations', async () => {
      // Test demo mode with fallback disabled
      demoService.updateConfig({ enabled: true, fallbackOnError: false });
      
      const result = await demoService.queryContext('test query');
      
      expect(result.source).toBe('demo');
      expect(mockGoBackendClient.queryContext).not.toHaveBeenCalled();
    });

    it('should handle predictable mode changes', async () => {
      const demoDataManager = demoService.getDemoDataManager();
      
      expect(demoDataManager.isPredictableMode()).toBe(true);
      
      demoService.updateConfig({ predictableResponses: false });
      expect(demoDataManager.isPredictableMode()).toBe(false);
    });
  });
});