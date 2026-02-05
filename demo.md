# ContextKeeper MCP + Slack Bot Demo

## Overview

This demo showcases the ContextKeeper MCP + Slack Bot system, which provides:

1. **MCP Server**: Model Context Protocol server for AI assistant integration
2. **Slack Bot**: Team collaboration commands for repository context queries
3. **Health Monitoring**: Built-in health checks and system monitoring
4. **Graceful Operations**: Restart and configuration reload capabilities

## Demo Flow

### 1. System Startup

```bash
# Start the system
npm run build
npm start

# Or using the demo verification script
node demo-verification.js
```

The system will start three services:
- **Health Server** (port 8080): Monitoring and operational endpoints
- **MCP Server** (port 3001): AI assistant integration
- **Slack Bot** (port 3002): Team collaboration commands

### 2. Health Check Verification

```bash
# Check overall system health
curl http://localhost:8080/health

# Expected response:
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00.000Z",
  "version": "1.0.0",
  "uptime": 120,
  "components": {
    "mcpServer": {
      "status": "healthy",
      "port": 3001
    },
    "slackBot": {
      "status": "healthy",
      "port": 3002
    }
  }
}
```

### 3. MCP Server Integration

The MCP server provides resources and tools for AI assistants:

```bash
# List available tools
curl -X POST http://localhost:3001/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "method": "list_tools",
    "id": "demo-1"
  }'

# Expected response:
{
  "result": {
    "tools": [
      {
        "name": "query_repository_context",
        "description": "Query repository context with natural language",
        "inputSchema": {
          "type": "object",
          "properties": {
            "query": {"type": "string"},
            "repositoryId": {"type": "string"}
          }
        }
      },
      {
        "name": "get_onboarding_summary",
        "description": "Generate onboarding summary for repositories",
        "inputSchema": {
          "type": "object",
          "properties": {
            "repositoryId": {"type": "string"}
          }
        }
      }
    ]
  },
  "id": "demo-1"
}
```

### 4. Slack Bot Commands

The Slack bot handles team collaboration commands:

```bash
# Simulate a Slack /context command
curl -X POST http://localhost:3002/slack/commands \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "command=/context&text=What are the main components?&user_id=U123&channel_id=C123&team_id=T123"

# Expected response:
{
  "text": "🔍 **Context Search Results**\n\nBased on the repository structure, the main components include:\n\n• **MCP Server**: Provides Model Context Protocol integration\n• **Slack Bot**: Handles team collaboration commands\n• **Configuration System**: Manages environment-specific settings\n• **Demo Service**: Provides predictable responses for demonstrations\n\n_Repository: demo-repo • Query: \"What are the main components?\"_",
  "responseType": "ephemeral"
}
```

### 5. System Status Monitoring

```bash
# Get detailed system status
curl http://localhost:8080/status

# Expected response includes:
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00.000Z",
  "version": "1.0.0",
  "uptime": 300,
  "components": {...},
  "restart": {
    "isRestarting": false,
    "lastRestart": 1705312200000
  },
  "environment": {
    "nodeVersion": "v18.17.0",
    "platform": "linux",
    "arch": "x64",
    "pid": 12345
  }
}
```

### 6. Graceful Restart

```bash
# Trigger graceful restart
curl -X POST http://localhost:8080/restart

# Expected response:
{
  "message": "Graceful restart initiated",
  "timestamp": "2024-01-15T10:30:00.000Z"
}

# System will restart without dropping connections
# Health endpoint will show temporary "not ready" status during restart
```

### 7. Demo Mode Features

The system includes demo mode for reliable presentations:

- **Predictable Responses**: Consistent outputs for demo scenarios
- **Fallback Handling**: Graceful degradation when backend is unavailable
- **Sample Data**: Pre-configured repository data for demonstrations

## Key Demo Points

### 1. **Dual Integration**
- Works with AI assistants via MCP protocol
- Provides team collaboration via Slack commands
- Single system serving both use cases

### 2. **Production Ready**
- Health monitoring and observability
- Graceful restart and configuration reload
- Proper error handling and circuit breakers
- Docker and Kubernetes deployment configurations

### 3. **Developer Experience**
- TypeScript with full type safety
- Comprehensive testing (unit + property-based)
- Clear configuration management
- Structured logging

### 4. **Operational Excellence**
- Zero-downtime restarts
- Health checks for container orchestration
- Resource limits and security best practices
- Monitoring and alerting ready

## Demo Script

1. **Start**: Show system startup and health checks
2. **MCP Integration**: Demonstrate AI assistant tool usage
3. **Slack Commands**: Show team collaboration features
4. **Monitoring**: Display health and status endpoints
5. **Operations**: Demonstrate graceful restart
6. **Scaling**: Show Docker/Kubernetes deployment

## Troubleshooting

### Common Issues

1. **Port conflicts**: Ensure ports 3001, 3002, 8080 are available
2. **Build failures**: Run `npm install` and check Node.js version (18+)
3. **Health checks failing**: Check if all components started successfully
4. **Demo mode not working**: Verify DEMO_MODE=true environment variable

### Debug Commands

```bash
# Check application logs
npm run dev  # Development mode with detailed logging

# Verify configuration
curl http://localhost:8080/status

# Test individual components
curl http://localhost:3001/health  # MCP Server
curl http://localhost:3002/health  # Slack Bot
```

## Next Steps

After the demo:

1. **Backend Integration**: Connect to actual ContextKeeper Go Backend
2. **Slack App Setup**: Configure real Slack application
3. **Production Deployment**: Use provided Docker/Kubernetes configs
4. **Monitoring Setup**: Integrate with your observability stack
5. **Customization**: Adapt commands and responses for your team

The system is designed to be production-ready while maintaining simplicity for development and demonstration purposes.