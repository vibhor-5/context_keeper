# Slack Integration Installation Guide

This document describes the Slack workspace OAuth installation flow and integration management for the MCP Context Engine.

For detailed implementation documentation, see the main codebase in `internal/services/slack_integration.go` and `internal/handlers/slack_integration.go`.

## Quick Start

1. Create a Slack app in the [Slack API Portal](https://api.slack.com/apps)
2. Configure OAuth scopes
3. Install the app via OAuth: `POST /api/projects/{project_id}/integrations/slack/oauth/install`
4. Select channels to monitor

## API Endpoints

- `POST /api/projects/{project_id}/integrations/slack/oauth/install` - OAuth installation
- `GET /api/projects/{project_id}/integrations/slack/{integration_id}/channels` - List channels
- `POST /api/projects/{project_id}/integrations/slack/{integration_id}/channels/select` - Select channels
- `GET /api/projects/{project_id}/integrations/slack/{integration_id}/status` - Get status

For complete API documentation, see the implementation files.
