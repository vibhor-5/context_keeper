# Discord Integration Installation Guide

This document describes the Discord bot installation flow and integration management for the MCP Context Engine.

For detailed implementation documentation, see the main codebase in `internal/services/discord_integration.go` and `internal/handlers/discord_integration.go`.

## Quick Start

1. Create a Discord bot in the [Discord Developer Portal](https://discord.com/developers/applications)
2. Copy the bot token
3. Install the bot via the API: `POST /api/projects/{project_id}/integrations/discord/bot/install`
4. Select servers and channels to monitor

## API Endpoints

- `POST /api/projects/{project_id}/integrations/discord/bot/install` - Install bot
- `GET /api/projects/{project_id}/integrations/discord/{integration_id}/servers` - List servers
- `GET /api/projects/{project_id}/integrations/discord/{integration_id}/servers/{guild_id}/channels` - List channels
- `POST /api/projects/{project_id}/integrations/discord/{integration_id}/channels/select` - Select channels
- `GET /api/projects/{project_id}/integrations/discord/{integration_id}/status` - Get status

For complete API documentation, see the implementation files.
