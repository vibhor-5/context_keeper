# MCP Context Engine - Architecture

## Overview

The MCP Context Engine is a multi-tenant SaaS platform that aggregates developer context from multiple sources (GitHub, Slack, Discord) and provides intelligent context retrieval through MCP-compatible tools.

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Frontend Layer                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Landing Page │  │  Dashboard   │  │  Auth Pages  │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                      API Gateway Layer                       │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  HTTP Server (Go) - Authentication & Authorization   │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│   MCP Tools  │  │ Integration  │  │   Project    │
│   Service    │  │   Service    │  │  Management  │
└──────────────┘  └──────────────┘  └──────────────┘
        │                   │                   │
        └───────────────────┼───────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Business Logic Layer                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  Connector   │  │   Context    │  │  Knowledge   │      │
│  │   Manager    │  │  Processor   │  │    Graph     │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│         │                  │                  │              │
│  ┌──────┴──────┐          │                  │              │
│  │  GitHub     │          │                  │              │
│  │  Slack      │          │                  │              │
│  │  Discord    │          │                  │              │
│  └─────────────┘          │                  │              │
└────────────────────────────┼──────────────────┼──────────────┘
                            ▼                  ▼
┌─────────────────────────────────────────────────────────────┐
│                      Data Layer                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  PostgreSQL  │  │    Vector    │  │    Cache     │      │
│  │   Database   │  │  Embeddings  │  │    (Redis)   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Frontend Layer
- **Landing Page**: Marketing site with product information
- **Dashboard**: Project management and integration configuration
- **Auth Pages**: Login, signup, and OAuth flows

### 2. API Gateway
- **HTTP Server**: Go-based REST API
- **Authentication**: JWT-based auth with OAuth support
- **Authorization**: Role-based access control (RBAC)
- **Rate Limiting**: Per-user and per-project limits

### 3. MCP Tools Service
Provides 5 MCP-compatible tools:
- `search_project_knowledge`
- `get_context_for_file`
- `get_decision_history`
- `list_recent_architecture_discussions`
- `explain_why_code_exists`

### 4. Integration Service
Manages platform integrations:
- **GitHub Integration**: OAuth, repository selection, webhook handling
- **Slack Integration**: OAuth, channel selection, message ingestion
- **Discord Integration**: Bot token, server/channel selection

### 5. Connector Manager
Orchestrates data ingestion from platforms:
- **Platform Connectors**: GitHub, Slack, Discord
- **Ingestion Orchestrator**: Schedules and manages sync jobs
- **Deduplication**: Prevents duplicate event processing
- **Retry Logic**: Handles failures with exponential backoff

### 6. Context Processor
AI-powered context extraction:
- **Decision Extraction**: Identifies architectural decisions
- **Discussion Summarization**: Summarizes conversations
- **Relationship Identification**: Links files, features, contributors
- **Feature Context**: Builds feature development history

### 7. Knowledge Graph
Stores and queries project knowledge:
- **Entities**: Features, Files, Decisions, Discussions, Contributors
- **Relationships**: relates_to, introduced_by, modified_by, discussed_in
- **Vector Search**: Semantic search using embeddings
- **Graph Traversal**: Relationship exploration

### 8. Data Layer
- **PostgreSQL**: Primary data store
- **Vector Embeddings**: Semantic search (pgvector)
- **Cache**: Redis for frequently accessed data

## Data Flow

### Ingestion Flow
```
External Platform → Connector → Normalization → Context Processor
                                                        ↓
                                                Knowledge Graph
                                                        ↓
                                                   Database
```

### Query Flow
```
IDE/Client → MCP Tool → Knowledge Graph → Vector Search
                              ↓
                         Response Optimization
                              ↓
                         Formatted Result
```

## Multi-Tenancy

### Tenant Isolation
- **Database Level**: All tables include `project_id` for isolation
- **API Level**: All requests scoped to authenticated user's projects
- **Data Level**: Complete separation of project data

### Security Boundaries
- **Authentication**: JWT tokens with project claims
- **Authorization**: RBAC with project-level permissions
- **Encryption**: AES-256-GCM for sensitive data
- **Audit Logging**: All operations logged with user context

## Scalability

### Horizontal Scaling
- **Stateless API**: Can run multiple instances
- **Job Queue**: Distributed ingestion processing
- **Database Pooling**: Connection pool management
- **Caching**: Redis for hot data

### Performance Optimization
- **Batch Processing**: Efficient bulk operations
- **Incremental Sync**: Only fetch new data
- **Query Optimization**: Indexed queries, materialized views
- **Response Streaming**: Large results streamed to client

## Technology Stack

### Backend
- **Language**: Go 1.21+
- **Framework**: Standard library + Chi router
- **Database**: PostgreSQL 14+ with pgvector
- **Cache**: Redis (optional)

### Frontend
- **HTML/CSS/JavaScript**: Vanilla JS for landing page
- **Framework**: React/Vue (for dashboard - future)

### Infrastructure
- **Containerization**: Docker
- **Orchestration**: Kubernetes
- **CI/CD**: GitHub Actions
- **Monitoring**: Prometheus + Grafana

## Security Architecture

### Authentication
- **JWT Tokens**: Stateless authentication
- **OAuth 2.0**: GitHub, Google integration
- **Session Management**: Secure token storage

### Authorization
- **RBAC**: Role-based access control
- **Project Permissions**: Owner, Admin, Member, Viewer
- **API Scoping**: All endpoints project-scoped

### Data Protection
- **Encryption at Rest**: AES-256-GCM
- **Encryption in Transit**: TLS 1.3
- **Credential Storage**: Encrypted in database
- **Secrets Management**: Environment variables + secret files

## Deployment Architecture

### Development
```
Docker Compose → PostgreSQL + App + Redis
```

### Production
```
Kubernetes Cluster
├── API Pods (3+ replicas)
├── Worker Pods (2+ replicas)
├── PostgreSQL (managed service)
└── Redis (managed service)
```

## Future Enhancements

### Planned Features
- Real-time ingestion via webhooks
- Advanced analytics dashboard
- Custom MCP tool creation
- Plugin system for custom connectors
- GraphQL API
- Mobile app

### Scalability Improvements
- Event streaming (Kafka)
- Distributed caching (Redis Cluster)
- Read replicas for queries
- CDN for static assets
- Edge computing for global distribution

## References

- [MCP Specification](https://modelcontextprotocol.io/)
- [Go Best Practices](https://golang.org/doc/effective_go)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
