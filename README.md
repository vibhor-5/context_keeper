# ContextKeeper MCP + Slack Bot

AI-powered repository context and team onboarding through a Model Context Protocol (MCP) server and Slack bot integration.

## Overview

ContextKeeper transforms the manual process of hunting for repository context (taking hours) into instant AI-powered responses (taking seconds). The system consists of two main components:

- **MCP Server**: Bridges AI assistants with the existing Go backend via Model Context Protocol
- **Slack Bot**: Provides team interaction surface for context queries through slash commands

## Architecture

```
AI Assistants (Claude, GPT-4) → MCP Server → Go Backend → PostgreSQL
Team Members → Slack Bot → MCP Server → Go Backend → PostgreSQL
```

## Features

- **AI Assistant Integration**: Query repository context through MCP protocol
- **Slack Commands**: `/context`, `/onboard`, `/recent`, `/status`
- **Repository Context**: Access PRs, issues, commits, and timeline data
- **Team Onboarding**: AI-generated summaries for new team members
- **Demo Mode**: Predictable responses for hackathon demonstrations

## Quick Start

### Prerequisites

- Node.js 18+
- Existing ContextKeeper Go backend running
- Slack workspace with bot permissions

### Installation

1. Clone and install dependencies:
```bash
npm install
```

2. Copy environment configuration:
```bash
cp .env.example .env
```

3. Configure your environment variables in `.env`:
   - Set your Slack bot token and signing secret
   - Configure Go backend URL
   - Adjust ports as needed

4. Build the project:
```bash
npm run build
```

5. Start the system:
```bash
npm start
```

For development with hot reload:
```bash
npm run dev
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `MCP_PORT` | MCP server port | 3001 |
| `GO_BACKEND_URL` | Go backend URL | http://localhost:8080 |
| `SLACK_BOT_TOKEN` | Slack bot token | Required |
| `SLACK_SIGNING_SECRET` | Slack signing secret | Required |
| `SLACK_PORT` | Slack bot port | 3002 |
| `DEMO_MODE` | Enable demo mode | false |

See `.env.example` for complete configuration options.

## Usage

### Slack Commands

- `/context <query>` - Query repository context
- `/onboard [repository]` - Get onboarding summary
- `/recent [repository] [days]` - View recent activity
- `/status` - Check system status

### MCP Integration

AI assistants can access repository resources and tools:

- **Resources**: Repository metadata, context, timeline
- **Tools**: `query_repository_context`, `get_onboarding_summary`

## Development

### Scripts

- `npm run build` - Build TypeScript to JavaScript
- `npm run dev` - Start with hot reload
- `npm test` - Run tests
- `npm run test:watch` - Run tests in watch mode
- `npm run lint` - Lint code
- `npm run clean` - Clean build directory

### Project Structure

```
src/
├── index.ts              # Main entry point
├── types/                # TypeScript interfaces
│   ├── mcp.ts           # MCP protocol types
│   ├── slack.ts         # Slack bot types
│   ├── backend.ts       # Go backend types
│   └── config.ts        # Configuration types
├── config/              # Configuration management
├── mcp/                 # MCP server implementation
├── slack/               # Slack bot implementation
└── utils/               # Utilities (logger, etc.)
```

## Testing

Run the test suite:
```bash
npm test
```

The project uses Vitest for testing with both unit tests and property-based tests using fast-check.

## License

MIT License - see LICENSE file for details.