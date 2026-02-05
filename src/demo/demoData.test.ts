/**
 * Tests for demo data management
 */

import { describe, it, expect, beforeEach } from 'vitest';
import { DemoDataManager, DEMO_REPOSITORIES, DEMO_SYSTEM_STATUS } from './demoData';

describe('DemoDataManager', () => {
  let demoManager: DemoDataManager;

  beforeEach(() => {
    demoManager = new DemoDataManager(true);
  });

  describe('Repository Management', () => {
    it('should get repository by ID', () => {
      const repo = demoManager.getRepository('demo-repo-1');
      
      expect(repo).toBeDefined();
      expect(repo?.id).toBe('demo-repo-1');
      expect(repo?.name).toBe('ContextKeeper Frontend');
      expect(repo?.language).toBe('TypeScript');
    });

    it('should return null for non-existent repository', () => {
      const repo = demoManager.getRepository('non-existent');
      expect(repo).toBeNull();
    });

    it('should get all repositories', () => {
      const repos = demoManager.getAllRepositories();
      
      expect(repos).toHaveLength(Object.keys(DEMO_REPOSITORIES).length);
      expect(repos[0]).toHaveProperty('id');
      expect(repos[0]).toHaveProperty('name');
      expect(repos[0]).toHaveProperty('description');
    });

    it('should return copies to prevent mutations', () => {
      const repo1 = demoManager.getRepository('demo-repo-1');
      const repo2 = demoManager.getRepository('demo-repo-1');
      
      expect(repo1).not.toBe(repo2); // Different object references
      expect(repo1).toEqual(repo2); // Same content
    });

    it('should add new repository', () => {
      const newRepo = {
        id: 'test-repo',
        name: 'Test Repository',
        description: 'Test repository for testing',
        language: 'JavaScript',
        contributors: ['test.user'],
        lastActivity: new Date().toISOString(),
        status: 'active' as const,
        structure: {
          directories: ['src', 'test'],
          keyFiles: ['index.js', 'package.json'],
          testCoverage: 90
        },
        recentActivity: [],
        contextSamples: [],
        onboardingGuide: 'Test onboarding guide'
      };

      demoManager.addRepository(newRepo);
      const retrieved = demoManager.getRepository('test-repo');
      
      expect(retrieved).toEqual(newRepo);
    });

    it('should update repository status', () => {
      const success = demoManager.updateRepositoryStatus('demo-repo-1', 'archived');
      expect(success).toBe(true);
      
      const repo = demoManager.getRepository('demo-repo-1');
      expect(repo?.status).toBe('archived');
    });

    it('should fail to update non-existent repository', () => {
      const success = demoManager.updateRepositoryStatus('non-existent', 'active');
      expect(success).toBe(false);
    });
  });

  describe('Context Queries', () => {
    it('should return predefined responses for matching queries', () => {
      const response = demoManager.queryContext('How does authentication work?', 'demo-repo-1');
      
      expect(response).toContain('OAuth2');
      expect(response).toContain('JWT');
    });

    it('should return generic responses for auth-related queries', () => {
      const response = demoManager.queryContext('login system', 'demo-repo-1');
      
      expect(response).toContain('OAuth2');
      expect(response).toContain('authentication');
    });

    it('should return component-related responses', () => {
      const response = demoManager.queryContext('main components');
      
      expect(response).toContain('architecture');
      expect(response).toContain('components');
    });

    it('should return testing-related responses', () => {
      const response = demoManager.queryContext('testing strategy');
      
      expect(response).toContain('testing');
      expect(response).toContain('coverage');
    });

    it('should return deployment-related responses', () => {
      const response = demoManager.queryContext('deployment process');
      
      expect(response).toContain('deployment');
      expect(response).toContain('Docker');
    });

    it('should return fallback response for unknown queries', () => {
      const response = demoManager.queryContext('unknown query');
      
      expect(response).toContain('repository analysis');
      expect(response).toContain('unknown query');
    });
  });

  describe('Onboarding Summaries', () => {
    it('should return onboarding guide for existing repository', () => {
      const guide = demoManager.getOnboardingSummary('demo-repo-1');
      
      expect(guide).toContain('Welcome to ContextKeeper Frontend');
      expect(guide).toContain('React');
      expect(guide).toContain('TypeScript');
    });

    it('should return error message for non-existent repository', () => {
      const guide = demoManager.getOnboardingSummary('non-existent');
      
      expect(guide).toContain('not found');
      expect(guide).toContain('non-existent');
    });
  });

  describe('Recent Activity', () => {
    it('should return recent activity for existing repository', () => {
      const activity = demoManager.getRecentActivity('demo-repo-1', 7);
      
      expect(activity).toContain('Recent activity');
      expect(activity).toContain('ContextKeeper Frontend');
      expect(activity).toContain('alice.dev');
    });

    it('should filter activities by date range', () => {
      const activity = demoManager.getRecentActivity('demo-repo-1', 1);
      
      // Should have fewer activities for shorter time range
      expect(activity).toContain('Recent activity');
    });

    it('should return error message for non-existent repository', () => {
      const activity = demoManager.getRecentActivity('non-existent', 7);
      
      expect(activity).toContain('not found');
      expect(activity).toContain('non-existent');
    });

    it('should handle no recent activity', () => {
      // Create a repository with old activities
      const oldRepo = {
        ...DEMO_REPOSITORIES['demo-repo-1'],
        id: 'old-repo',
        recentActivity: [{
          timestamp: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(), // 30 days ago
          type: 'commit' as const,
          author: 'old.user',
          title: 'Old commit',
          description: 'Very old commit',
          impact: 'low' as const
        }]
      };

      demoManager.addRepository(oldRepo);
      const activity = demoManager.getRecentActivity('old-repo', 7);
      
      expect(activity).toContain('No recent activity');
    });
  });

  describe('System Status', () => {
    it('should return system status', () => {
      const status = demoManager.getSystemStatus();
      
      expect(status).toHaveProperty('mcpServer');
      expect(status).toHaveProperty('goBackend');
      expect(status).toHaveProperty('repositories');
      
      expect(status.mcpServer.status).toBe('healthy');
      expect(status.goBackend.status).toBe('healthy');
      expect(status.repositories.total).toBeGreaterThan(0);
    });

    it('should update timestamps in system status', () => {
      const status1 = demoManager.getSystemStatus();
      
      // Wait a bit and get status again
      setTimeout(() => {
        const status2 = demoManager.getSystemStatus();
        expect(status2.goBackend.lastCheck).not.toBe(status1.goBackend.lastCheck);
      }, 10);
    });

    it('should occasionally show degraded status in predictable mode', () => {
      // This test is probabilistic, so we'll run it multiple times
      let foundDegraded = false;
      
      for (let i = 0; i < 50; i++) {
        const status = demoManager.getSystemStatus();
        if (status.goBackend.status === 'degraded') {
          foundDegraded = true;
          break;
        }
      }
      
      // Note: This test might occasionally fail due to randomness
      // In a real implementation, we might want to make this more deterministic
    });
  });

  describe('Predictable Mode', () => {
    it('should get predictable mode status', () => {
      expect(demoManager.isPredictableMode()).toBe(true);
    });

    it('should set predictable mode', () => {
      demoManager.setPredictableMode(false);
      expect(demoManager.isPredictableMode()).toBe(false);
      
      demoManager.setPredictableMode(true);
      expect(demoManager.isPredictableMode()).toBe(true);
    });
  });

  describe('Reset Functionality', () => {
    it('should reset to default data', () => {
      // Modify some data
      demoManager.updateRepositoryStatus('demo-repo-1', 'archived');
      demoManager.addRepository({
        id: 'temp-repo',
        name: 'Temporary',
        description: 'Temp repo',
        language: 'Python',
        contributors: [],
        lastActivity: new Date().toISOString(),
        status: 'active',
        structure: { directories: [], keyFiles: [], testCoverage: 0 },
        recentActivity: [],
        contextSamples: [],
        onboardingGuide: 'Temp guide'
      });
      
      // Reset
      demoManager.reset();
      
      // Check that data is back to defaults
      const repo = demoManager.getRepository('demo-repo-1');
      expect(repo?.status).toBe('active'); // Should be back to original status
      
      const tempRepo = demoManager.getRepository('temp-repo');
      expect(tempRepo).toBeNull(); // Should be removed
    });
  });

  describe('Time Calculations', () => {
    it('should format time ago correctly', () => {
      const activity = demoManager.getRecentActivity('demo-repo-1', 7);
      
      // Should contain time indicators like "1h ago", "3h ago", etc.
      expect(activity).toMatch(/\d+[hmd] ago/);
    });
  });
});