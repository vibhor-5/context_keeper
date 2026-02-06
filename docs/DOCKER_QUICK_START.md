# Docker Quick Start Guide

## TL;DR

```bash
# Development - Backend + Database
docker-compose up

# With MCP Server
docker-compose --profile with-mcp up

# With Slack Bot
docker-compose --profile with-slack up

# Full Stack
docker-compose --profile with-mcp --profile with-slack up

# Production
docker-compose -f docker-compose.yml up -d
```

## What Changed?

### Before (5 Docker files)
- `Dockerfile` - Main app
- `Dockerfile.mcp-slack` - Slack bot
- `docker-compose.yml` - Production
- `docker-compose.dev.yml` - Development
- `docker-compose.mcp-slack.yml` - Slack specific

### After (3 Docker files)
- `Dockerfile` - Single multi-stage build (Go + Node.js)
- `docker-compose.yml` - Main config with profiles
- `docker-compose.override.yml` - Dev settings (auto-loaded)

## Profiles Explained

Profiles let you enable optional services:

| Profile | Adds | Port |
|---------|------|------|
| (default) | Backend + Postgres | 8080, 5432 |
| `with-mcp` | MCP Server | 3001 |
| `with-slack` | Slack Bot | 3002 |
| `with-cache` | Redis | 6379 |

## Common Commands

```bash
# Start services
docker-compose up                    # Dev mode with hot reload
docker-compose up -d                 # Background
docker-compose --profile with-mcp up # With MCP server

# Stop services
docker-compose down                  # Stop and remove
docker-compose down -v               # Also remove volumes

# View logs
docker-compose logs -f               # All services
docker-compose logs -f backend       # Specific service

# Rebuild
docker-compose build                 # Rebuild images
docker-compose build --no-cache      # Clean rebuild

# Status
docker-compose ps                    # List containers
docker-compose top                   # Show processes
```

## Development vs Production

### Development (automatic)
When you run `docker-compose up`, it automatically:
- Loads `docker-compose.override.yml`
- Mounts source code for hot reload
- Uses debug logging
- Uses simple passwords (no secrets)
- Opens CORS

### Production (explicit)
Use `-f docker-compose.yml` to skip override:
```bash
docker-compose -f docker-compose.yml up -d
```

This:
- Uses Docker secrets
- No source mounting
- Production logging
- Strict CORS
- Optimized images

## Service URLs

| Service | URL | Description |
|---------|-----|-------------|
| Backend API | http://localhost:8080 | Go REST API |
| MCP Server | http://localhost:3001 | MCP tools |
| Slack Bot | http://localhost:3002 | Slack integration |
| Postgres | localhost:5432 | Database |
| Redis | localhost:6379 | Cache (optional) |

## Health Checks

```bash
# Backend
curl http://localhost:8080/health

# MCP Server
curl http://localhost:3001/health

# Slack Bot
curl http://localhost:3002/health
```

## Troubleshooting

### Port already in use
```bash
# Find what's using the port
lsof -i :8080

# Or change the port in docker-compose.yml
ports:
  - "8081:8080"
```

### Database connection failed
```bash
# Check postgres is running
docker-compose ps postgres

# View postgres logs
docker-compose logs postgres

# Restart postgres
docker-compose restart postgres
```

### Build failed
```bash
# Clean rebuild
docker-compose down -v
docker-compose build --no-cache
docker-compose up
```

### Hot reload not working
```bash
# Make sure you're in dev mode (not using -f flag)
docker-compose up

# Check volumes are mounted
docker-compose config
```

## Makefile Shortcuts

```bash
make docker-build    # Build images
make docker-up       # Start services
make docker-down     # Stop services
make docker-logs     # View logs
make docker-clean    # Clean everything
```

## Examples

### Full Development Stack
```bash
# Start everything with hot reload
docker-compose \
  --profile with-mcp \
  --profile with-slack \
  --profile with-cache \
  up
```

### Production Deployment
```bash
# Build
docker-compose -f docker-compose.yml build

# Start
docker-compose -f docker-compose.yml up -d

# Check status
docker-compose -f docker-compose.yml ps

# View logs
docker-compose -f docker-compose.yml logs -f
```

### Testing
```bash
# Start services
docker-compose up -d

# Run tests
docker-compose exec backend go test ./...

# Stop services
docker-compose down
```

## Need More Help?

See the full documentation:
- [docs/DOCKER.md](docs/DOCKER.md) - Comprehensive Docker guide
- [README.md](README.md) - Project overview
- [docs/SETUP_GUIDE.md](docs/SETUP_GUIDE.md) - Setup instructions
