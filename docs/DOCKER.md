# Docker Deployment Guide

This document provides instructions for deploying the MCP Context Engine using Docker.

## Overview

The project uses a single `Dockerfile` that builds both Go and Node.js services in a multi-stage build. Docker Compose orchestrates all services with profile-based configuration.

## Quick Start

### Development

```bash
# Start backend + database
docker-compose up

# Start with MCP server
docker-compose --profile with-mcp up

# Start with Slack bot
docker-compose --profile with-slack up

# Full stack
docker-compose --profile with-mcp --profile with-slack up

# With Redis cache
docker-compose --profile with-cache up
```

### Production

```bash
# Build images
docker-compose build

# Start all services
docker-compose -f docker-compose.yml up -d

# Or with profiles
docker-compose --profile with-mcp --profile with-slack up -d
```

## Architecture

### Single Dockerfile

The `Dockerfile` uses multi-stage builds:

1. **go-builder**: Builds Go backend binary
2. **node-builder**: Builds TypeScript/Node.js services
3. **production**: Final minimal Alpine image with both runtimes

### Services

- **postgres**: PostgreSQL 15 database
- **redis**: Redis cache (optional, use `--profile with-cache`)
- **backend**: Go API server (port 8080)
- **mcp-server**: MCP server (port 3001, use `--profile with-mcp`)
- **slack-bot**: Slack bot (port 3002, use `--profile with-slack`)

## Configuration

### Environment Variables

Create a `.env` file in the project root:

```bash
# Database
POSTGRES_PASSWORD=your-secure-password

# GitHub OAuth
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret
GITHUB_REDIRECT_URL=http://localhost:8080/api/auth/github

# Google OAuth
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret

# Slack
SLACK_CLIENT_ID=your-slack-client-id
SLACK_CLIENT_SECRET=your-slack-client-secret
SLACK_SIGNING_SECRET=your-slack-signing-secret
SLACK_REDIRECT_URL=http://localhost:8080/api/auth/slack

# AI Service
AI_SERVICE_URL=http://ai-service:8000

# CORS
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080
```

### Secrets

For production, use Docker secrets:

```bash
# Create secret files
echo "your-postgres-password" > secrets/postgres_password.txt
echo "your-jwt-secret" > secrets/jwt_secret.txt
echo "your-github-client-id" > secrets/github_client_id.txt
echo "your-github-client-secret" > secrets/github_client_secret.txt
echo "your-google-client-secret" > secrets/google_client_secret.txt
echo "your-slack-client-secret" > secrets/slack_client_secret.txt
echo "your-slack-signing-secret" > secrets/slack_signing_secret.txt

# Set proper permissions
chmod 600 secrets/*.txt
```

## Development Mode

The `docker-compose.override.yml` file is automatically loaded in development and provides:

- Source code mounting for hot reload
- Debug logging
- Simple passwords (no secrets)
- Open CORS

### Hot Reload

```bash
# Backend with hot reload (requires air or similar)
docker-compose up

# Node services with hot reload
docker-compose --profile with-mcp up
```

## Production Deployment

### Build Production Images

```bash
# Build all images
docker-compose build

# Build specific service
docker-compose build backend

# Build with no cache
docker-compose build --no-cache
```

### Run Production Stack

```bash
# Start all services
docker-compose -f docker-compose.yml up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f backend

# Stop services
docker-compose down
```

### Health Checks

All services include health checks:

```bash
# Check backend health
curl http://localhost:8080/health

# Check MCP server health
curl http://localhost:3001/health

# Check Slack bot health
curl http://localhost:3002/health
```

## Profiles

Use profiles to control which services run:

- **default**: postgres + backend
- **with-mcp**: Add MCP server
- **with-slack**: Add Slack bot
- **with-cache**: Add Redis cache

```bash
# Multiple profiles
docker-compose \
  --profile with-mcp \
  --profile with-slack \
  --profile with-cache \
  up -d
```

## Volumes

Persistent data is stored in Docker volumes:

- `postgres_data`: Database data
- `redis_data`: Cache data
- `backend_logs`: Backend logs
- `mcp_logs`: MCP server logs
- `slack_logs`: Slack bot logs

```bash
# List volumes
docker volume ls

# Inspect volume
docker volume inspect mcp-context-engine_postgres_data

# Backup volume
docker run --rm -v mcp-context-engine_postgres_data:/data -v $(pwd):/backup alpine tar czf /backup/postgres-backup.tar.gz /data

# Restore volume
docker run --rm -v mcp-context-engine_postgres_data:/data -v $(pwd):/backup alpine tar xzf /backup/postgres-backup.tar.gz -C /
```

## Networking

All services communicate via the `mcp-network` bridge network:

```bash
# Inspect network
docker network inspect mcp-context-engine_mcp-network

# Service DNS names:
# - postgres
# - redis
# - backend
# - mcp-server
# - slack-bot
```

## Troubleshooting

### Container won't start

```bash
# Check logs
docker-compose logs backend

# Check container status
docker-compose ps

# Restart service
docker-compose restart backend
```

### Database connection issues

```bash
# Check postgres is running
docker-compose ps postgres

# Check postgres logs
docker-compose logs postgres

# Connect to postgres
docker-compose exec postgres psql -U contextkeeper -d contextkeeper
```

### Build issues

```bash
# Clean build
docker-compose down -v
docker-compose build --no-cache
docker-compose up
```

### Port conflicts

```bash
# Check what's using the port
lsof -i :8080

# Change port in docker-compose.yml
ports:
  - "8081:8080"  # Map to different host port
```

## Resource Limits

Add resource limits in production:

```yaml
services:
  backend:
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 1G
        reservations:
          cpus: '0.5'
          memory: 512M
```

## Monitoring

### Container Stats

```bash
# Real-time stats
docker stats

# Specific service
docker stats mcp-backend
```

### Logs

```bash
# Follow all logs
docker-compose logs -f

# Specific service
docker-compose logs -f backend

# Last 100 lines
docker-compose logs --tail=100 backend
```

## Cleanup

```bash
# Stop and remove containers
docker-compose down

# Remove volumes too
docker-compose down -v

# Remove images
docker-compose down --rmi all

# Full cleanup
docker system prune -a --volumes
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Docker Build

on:
  push:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Build images
        run: docker-compose build
      
      - name: Run tests
        run: docker-compose run backend go test ./...
      
      - name: Push to registry
        run: |
          docker tag mcp-backend:latest registry.example.com/mcp-backend:latest
          docker push registry.example.com/mcp-backend:latest
```

## Best Practices

1. **Use secrets** for sensitive data in production
2. **Set resource limits** to prevent resource exhaustion
3. **Enable health checks** for all services
4. **Use volumes** for persistent data
5. **Implement logging** with proper log rotation
6. **Monitor** container metrics and logs
7. **Backup** volumes regularly
8. **Update** base images regularly for security patches

## References

- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Docker Best Practices](https://docs.docker.com/develop/dev-best-practices/)
- [Multi-stage Builds](https://docs.docker.com/build/building/multi-stage/)
