# ContextKeeper Go Backend

A GitHub repository memory system that preserves engineering context from PRs, issues, and commits.

## Architecture

- **Go Backend**: API orchestration, GitHub integration, data storage
- **Python AI Service**: Context processing and requirement clarification  
- **Frontend**: Web dashboard for context queries

## Features

- GitHub OAuth authentication
- Repository data ingestion (PRs, issues, commits)
- Background job processing
- AI-powered context restoration
- Requirement clarification

## Development

### Prerequisites

- Go 1.21+
- PostgreSQL
- Docker (optional)

### Environment Variables

```bash
PORT=8080
DATABASE_URL=postgres://localhost/contextkeeper?sslmode=disable
JWT_SECRET=your-secret-key
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret
GITHUB_REDIRECT_URL=http://localhost:8080/api/auth/github
AI_SERVICE_URL=http://localhost:8000
AI_SERVICE_TIMEOUT=30
```

### Running Locally

```bash
# Install dependencies
go mod tidy

# Run migrations
go run cmd/server/main.go

# Or with Docker
docker-compose up
```

### API Endpoints

- `POST /api/auth/github` - GitHub OAuth callback
- `GET /api/repos` - List ingested repositories
- `POST /api/repos/ingest` - Trigger repository ingestion
- `GET /api/repos/{id}/status` - Get ingestion status
- `POST /api/context/query` - Query repository context

## Project Structure

```
cmd/
  server/           # Application entry point
internal/
  config/           # Configuration management
  database/         # Database migrations
  handlers/         # HTTP handlers
  middleware/       # HTTP middleware
  models/           # Data models
  repository/       # Database operations
  server/           # HTTP server setup
  services/         # Business logic interfaces
```

## Implementation Status

- [x] Project structure and core interfaces
- [ ] Database schema and repository layer
- [ ] GitHub OAuth authentication
- [ ] GitHub API client
- [ ] Background job processing
- [ ] AI service integration
- [ ] HTTP API endpoints
- [ ] Integration and deployment