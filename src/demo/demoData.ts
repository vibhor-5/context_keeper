/**
 * Demo data management for ContextKeeper
 * Provides sample repository data and predictable responses for demo scenarios
 */

import { logger } from '../utils/logger';

export interface DemoRepository {
  id: string;
  name: string;
  description: string;
  language: string;
  contributors: string[];
  lastActivity: string;
  status: 'active' | 'archived' | 'ingesting';
  structure: {
    directories: string[];
    keyFiles: string[];
    testCoverage: number;
  };
  recentActivity: DemoActivity[];
  contextSamples: DemoContextSample[];
  onboardingGuide: string;
}

export interface DemoActivity {
  timestamp: string;
  type: 'commit' | 'pr' | 'issue' | 'deployment' | 'review';
  author: string;
  title: string;
  description: string;
  impact: 'low' | 'medium' | 'high';
}

export interface DemoContextSample {
  query: string;
  response: string;
  confidence: number;
  sources: string[];
}

export interface DemoSystemStatus {
  mcpServer: {
    status: 'healthy' | 'degraded' | 'down';
    version: string;
    uptime: number;
  };
  goBackend: {
    status: 'healthy' | 'degraded' | 'down';
    responseTime: number;
    lastCheck: string;
  };
  repositories: {
    total: number;
    active: number;
    ingesting: number;
    error: number;
  };
}

/**
 * Sample repository data for demo purposes
 */
export const DEMO_REPOSITORIES: Record<string, DemoRepository> = {
  'demo-repo-1': {
    id: 'demo-repo-1',
    name: 'ContextKeeper Frontend',
    description: 'React-based frontend application for the ContextKeeper platform',
    language: 'TypeScript',
    contributors: ['alice.dev', 'bob.engineer', 'charlie.designer'],
    lastActivity: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(), // 2 hours ago
    status: 'active',
    structure: {
      directories: ['src', 'components', 'pages', 'hooks', 'utils', 'tests'],
      keyFiles: ['App.tsx', 'index.tsx', 'package.json', 'README.md'],
      testCoverage: 85
    },
    recentActivity: [
      {
        timestamp: new Date(Date.now() - 1 * 60 * 60 * 1000).toISOString(),
        type: 'commit',
        author: 'alice.dev',
        title: 'Add user authentication flow',
        description: 'Implemented OAuth2 integration with secure token handling',
        impact: 'high'
      },
      {
        timestamp: new Date(Date.now() - 3 * 60 * 60 * 1000).toISOString(),
        type: 'pr',
        author: 'bob.engineer',
        title: 'Fix responsive design issues',
        description: 'Updated CSS grid layouts for mobile compatibility',
        impact: 'medium'
      },
      {
        timestamp: new Date(Date.now() - 6 * 60 * 60 * 1000).toISOString(),
        type: 'deployment',
        author: 'system',
        title: 'Production deployment v2.1.0',
        description: 'Deployed latest features to production environment',
        impact: 'high'
      }
    ],
    contextSamples: [
      {
        query: 'How does authentication work?',
        response: 'The application uses OAuth2 for authentication with JWT tokens. Users authenticate through the /auth endpoint, receive a JWT token, and include it in subsequent requests via the Authorization header.',
        confidence: 0.95,
        sources: ['src/auth/AuthService.ts', 'src/components/LoginForm.tsx']
      },
      {
        query: 'What are the main components?',
        response: 'The main components include: App.tsx (root component), LoginForm.tsx (authentication), Dashboard.tsx (main interface), UserProfile.tsx (user management), and RepositoryList.tsx (repository display).',
        confidence: 0.92,
        sources: ['src/components/', 'src/App.tsx']
      }
    ],
    onboardingGuide: `# Welcome to ContextKeeper Frontend

## Getting Started
This is a React-based frontend application built with TypeScript. Here's what you need to know:

### Architecture
- **Framework**: React 18 with TypeScript
- **State Management**: Redux Toolkit
- **Styling**: Tailwind CSS
- **Testing**: Jest + React Testing Library

### Key Components
- \`App.tsx\`: Main application component
- \`components/\`: Reusable UI components
- \`pages/\`: Route-based page components
- \`hooks/\`: Custom React hooks
- \`utils/\`: Utility functions

### Development Workflow
1. Run \`npm install\` to install dependencies
2. Use \`npm run dev\` for development server
3. Run \`npm test\` for testing
4. Use \`npm run build\` for production builds

### Recent Changes
- Added OAuth2 authentication flow
- Improved responsive design
- Enhanced error handling

### Need Help?
- Check the README.md for detailed setup instructions
- Review component documentation in /docs
- Ask @alice.dev or @bob.engineer for technical questions`
  },

  'backend-service': {
    id: 'backend-service',
    name: 'ContextKeeper API',
    description: 'Go-based backend service providing REST APIs for context management',
    language: 'Go',
    contributors: ['david.backend', 'eve.devops', 'frank.architect'],
    lastActivity: new Date(Date.now() - 4 * 60 * 60 * 1000).toISOString(), // 4 hours ago
    status: 'active',
    structure: {
      directories: ['cmd', 'internal', 'pkg', 'api', 'migrations'],
      keyFiles: ['main.go', 'go.mod', 'Dockerfile', 'README.md'],
      testCoverage: 78
    },
    recentActivity: [
      {
        timestamp: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
        type: 'commit',
        author: 'david.backend',
        title: 'Optimize database queries',
        description: 'Added connection pooling and query optimization for better performance',
        impact: 'high'
      },
      {
        timestamp: new Date(Date.now() - 5 * 60 * 60 * 1000).toISOString(),
        type: 'issue',
        author: 'eve.devops',
        title: 'Memory usage monitoring',
        description: 'Added Prometheus metrics for memory and CPU monitoring',
        impact: 'medium'
      }
    ],
    contextSamples: [
      {
        query: 'How is the API structured?',
        response: 'The API follows REST principles with endpoints organized by resource: /api/repos for repository management, /api/context for context queries, and /api/auth for authentication. All responses use JSON format.',
        confidence: 0.93,
        sources: ['internal/handlers/', 'api/openapi.yaml']
      }
    ],
    onboardingGuide: `# Welcome to ContextKeeper API

## Overview
This is the backend service for ContextKeeper, built with Go and providing REST APIs.

### Architecture
- **Language**: Go 1.21+
- **Framework**: Gin HTTP framework
- **Database**: PostgreSQL with GORM
- **Authentication**: JWT tokens

### Key Directories
- \`cmd/\`: Application entry points
- \`internal/\`: Private application code
- \`pkg/\`: Public library code
- \`api/\`: API specifications

### Getting Started
1. Install Go 1.21+
2. Run \`go mod download\`
3. Set up PostgreSQL database
4. Use \`make run\` to start the server

### Recent Updates
- Optimized database performance
- Added monitoring metrics
- Enhanced error handling`
  }
};

/**
 * Demo system status data
 */
export const DEMO_SYSTEM_STATUS: DemoSystemStatus = {
  mcpServer: {
    status: 'healthy',
    version: '1.0.0',
    uptime: 86400 // 24 hours in seconds
  },
  goBackend: {
    status: 'healthy',
    responseTime: 45,
    lastCheck: new Date().toISOString()
  },
  repositories: {
    total: 12,
    active: 10,
    ingesting: 1,
    error: 1
  }
};

/**
 * Demo data manager class
 */
export class DemoDataManager {
  private repositories: Record<string, DemoRepository>;
  private systemStatus: DemoSystemStatus;
  private predictableMode: boolean;

  constructor(predictableMode: boolean = true) {
    this.repositories = { ...DEMO_REPOSITORIES };
    this.systemStatus = { ...DEMO_SYSTEM_STATUS };
    this.predictableMode = predictableMode;
    
    logger.info('Demo data manager initialized', {
      repositoryCount: Object.keys(this.repositories).length,
      predictableMode: this.predictableMode
    });
  }

  /**
   * Get repository by ID
   */
  getRepository(repositoryId: string): DemoRepository | null {
    const repo = this.repositories[repositoryId];
    if (!repo) {
      logger.warn('Demo repository not found', { repositoryId });
      return null;
    }
    
    logger.debug('Retrieved demo repository', { repositoryId, name: repo.name });
    return { ...repo }; // Return copy to prevent mutations
  }

  /**
   * Get all repositories
   */
  getAllRepositories(): DemoRepository[] {
    return Object.values(this.repositories).map(repo => ({ ...repo }));
  }

  /**
   * Query repository context with demo data
   */
  queryContext(query: string, repositoryId?: string): string {
    logger.debug('Querying demo context', { query: query.substring(0, 50), repositoryId });

    // If predictable mode is enabled, use predefined responses
    if (this.predictableMode) {
      const repo = repositoryId ? this.getRepository(repositoryId) : null;
      
      if (repo) {
        // Find matching context sample
        const sample = repo.contextSamples.find(s => 
          query.toLowerCase().includes(s.query.toLowerCase().split(' ')[0])
        );
        
        if (sample) {
          logger.debug('Found matching demo context sample', { 
            query: sample.query,
            confidence: sample.confidence
          });
          return sample.response;
        }
      }
      
      // Fallback to generic responses based on query keywords
      const lowerQuery = query.toLowerCase();
      
      if (lowerQuery.includes('auth') || lowerQuery.includes('login')) {
        return 'The system uses OAuth2 authentication with JWT tokens. Users authenticate through secure endpoints and receive tokens for subsequent API calls.';
      }
      
      if (lowerQuery.includes('component') || lowerQuery.includes('structure')) {
        return 'The application follows a modular architecture with clear separation of concerns. Main components include authentication, data management, and user interface layers.';
      }
      
      if (lowerQuery.includes('test') || lowerQuery.includes('testing')) {
        return 'The project uses comprehensive testing strategies including unit tests, integration tests, and end-to-end testing with good coverage metrics.';
      }
      
      if (lowerQuery.includes('deploy') || lowerQuery.includes('build')) {
        return 'The deployment process uses containerized builds with Docker, automated CI/CD pipelines, and environment-specific configurations.';
      }
    }

    // Generic fallback response
    return `Based on the repository analysis, here's what I found regarding "${query}": The codebase contains relevant information that addresses your query. For more specific details, please refine your search or specify a particular repository.`;
  }

  /**
   * Get onboarding summary for repository
   */
  getOnboardingSummary(repositoryId: string): string {
    const repo = this.getRepository(repositoryId);
    
    if (!repo) {
      logger.warn('Repository not found for onboarding', { repositoryId });
      return `Repository "${repositoryId}" not found. Please check the repository name and try again.`;
    }
    
    logger.debug('Generated demo onboarding summary', { repositoryId, name: repo.name });
    return repo.onboardingGuide;
  }

  /**
   * Get recent activity for repository
   */
  getRecentActivity(repositoryId: string, days: number = 7): string {
    const repo = this.getRepository(repositoryId);
    
    if (!repo) {
      logger.warn('Repository not found for recent activity', { repositoryId });
      return `Repository "${repositoryId}" not found. Please check the repository name and try again.`;
    }

    // Filter activities by date range
    const cutoffDate = new Date(Date.now() - days * 24 * 60 * 60 * 1000);
    const recentActivities = repo.recentActivity.filter(activity => 
      new Date(activity.timestamp) > cutoffDate
    );

    if (recentActivities.length === 0) {
      return `No recent activity found for ${repo.name} in the last ${days} days.`;
    }

    // Format activities for display
    let activityText = `Recent activity for **${repo.name}**:\n\n`;
    
    recentActivities.forEach(activity => {
      const timeAgo = this.getTimeAgo(activity.timestamp);
      const impactEmoji = activity.impact === 'high' ? '🔥' : 
                         activity.impact === 'medium' ? '⚡' : '📝';
      
      activityText += `${impactEmoji} **${activity.title}** (${timeAgo})\n`;
      activityText += `   ${activity.description}\n`;
      activityText += `   _by ${activity.author}_\n\n`;
    });

    logger.debug('Generated demo recent activity', { 
      repositoryId, 
      days, 
      activityCount: recentActivities.length 
    });
    
    return activityText.trim();
  }

  /**
   * Get system status
   */
  getSystemStatus(): DemoSystemStatus {
    // Update timestamps for realism
    const updatedStatus = {
      ...this.systemStatus,
      goBackend: {
        ...this.systemStatus.goBackend,
        lastCheck: new Date().toISOString(),
        responseTime: this.predictableMode ? 45 : Math.floor(Math.random() * 100) + 20
      }
    };

    // In predictable mode, occasionally show degraded status for demo variety
    if (this.predictableMode && Math.random() < 0.1) {
      updatedStatus.goBackend.status = 'degraded';
      updatedStatus.goBackend.responseTime = 150;
    }

    logger.debug('Retrieved demo system status', { 
      mcpStatus: updatedStatus.mcpServer.status,
      backendStatus: updatedStatus.goBackend.status
    });
    
    return updatedStatus;
  }

  /**
   * Add new repository (for testing)
   */
  addRepository(repository: DemoRepository): void {
    this.repositories[repository.id] = { ...repository };
    logger.info('Added demo repository', { repositoryId: repository.id, name: repository.name });
  }

  /**
   * Update repository status
   */
  updateRepositoryStatus(repositoryId: string, status: 'active' | 'archived' | 'ingesting'): boolean {
    const repo = this.repositories[repositoryId];
    if (!repo) {
      return false;
    }
    
    repo.status = status;
    repo.lastActivity = new Date().toISOString();
    
    logger.debug('Updated demo repository status', { repositoryId, status });
    return true;
  }

  /**
   * Get predictable mode status
   */
  isPredictableMode(): boolean {
    return this.predictableMode;
  }

  /**
   * Set predictable mode
   */
  setPredictableMode(enabled: boolean): void {
    this.predictableMode = enabled;
    logger.info('Demo predictable mode updated', { enabled });
  }

  /**
   * Helper method to calculate time ago
   */
  private getTimeAgo(timestamp: string): string {
    const now = new Date();
    const past = new Date(timestamp);
    const diffMs = now.getTime() - past.getTime();
    
    const diffMinutes = Math.floor(diffMs / (1000 * 60));
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
    
    if (diffMinutes < 60) {
      return `${diffMinutes}m ago`;
    } else if (diffHours < 24) {
      return `${diffHours}h ago`;
    } else {
      return `${diffDays}d ago`;
    }
  }

  /**
   * Reset to default demo data
   */
  reset(): void {
    this.repositories = { ...DEMO_REPOSITORIES };
    this.systemStatus = { ...DEMO_SYSTEM_STATUS };
    logger.info('Demo data manager reset to defaults');
  }
}

// Export singleton instance
export const demoDataManager = new DemoDataManager();