# Project Structure

This document describes the organization of the MCP Context Engine codebase.

## Directory Structure

```
context_keeper/
├── .git/                       # Git repository data
├── .kiro/                      # Kiro IDE configuration
│   └── specs/                  # Feature specifications
│       └── mcp-context-engine/ # Main spec
├── .vscode/                    # VS Code settings
├── bin/                        # Compiled binaries (gitignored)
├── cmd/                        # Application entrypoints
│   └── server/                 # Main server application
│       └── main.go             # Server entry point
├── deployment/                 # Deployment configurations
│   ├── kubernetes/             # Kubernetes manifests
│   └── README.md               # Deployment guide
├── docs/                       # Documentation
│   ├── ARCHITECTURE.md         # System architecture
│   ├── DISCORD_INTEGRATION.md  # Discord setup guide
│   ├── DOCKER.md               # Docker deployment
│   ├── PROJECT_STRUCTURE.md    # This file
│   └── SLACK_INTEGRATION.md    # Slack setup guide
├── internal/                   # Private application code
│   ├── config/                 # Configuration management
│   │   ├── config.go           # Config struct and loading
│   │   └── config_test.go      # Config tests
│   ├── database/               # Database operations
│   │   └── migrations.go       # Database migrations
│   ├── handlers/               # HTTP request handlers
│   │   ├── handlers.go         # Core handlers
│   │   ├── project_handlers.go # Project management
│   │   ├── github_integration.go # GitHub integration handlers
│   │   ├── slack_integration.go  # Slack integration handlers
│   │   └── discord_integration.go # Discord integration handlers
│   ├── logger/                 # Logging utilities
│   │   ├── logger.go           # Logger interface
│   │   └── logger_test.go      # Logger tests
│   ├── middleware/             # HTTP middleware
│   │   └── auth.go             # Authentication middleware
│   ├── models/                 # Data models
│   │   ├── models.go           # Core models
│   │   └── models_test.go      # Model tests
│   ├── repository/             # Data access layer
│   │   ├── repository.go       # Repository interface
│   │   └── repository_test.go  # Repository tests
│   ├── server/                 # Server setup
│   │   ├── server.go           # HTTP server
│   │   └── server_test.go      # Server tests
│   └── services/               # Business logic
│       ├── auth.go             # Authentication service
│       ├── context.go          # Context service
│       ├── context_processor.go # Context processing
│       ├── encryption.go       # Encryption service
│       ├── github.go           # GitHub service
│       ├── github_integration.go # GitHub integration
│       ├── slack_integration.go  # Slack integration
│       ├── discord_integration.go # Discord integration
│       ├── ingestion_orchestrator.go # Ingestion orchestration
│       ├── interfaces.go       # Service interfaces
│       ├── job.go              # Job service
│       ├── jwt.go              # JWT service
│       ├── knowledge_graph.go  # Knowledge graph service
│       ├── mcp_server.go       # MCP server
│       ├── mcp_tools.go        # MCP tools
│       ├── password.go         # Password service
│       ├── permission.go       # Permission service
│       ├── project_workspace.go # Project workspace service
│       └── connectors/         # Platform connectors
│           ├── base.go         # Base connector
│           ├── config.go       # Connector config
│           ├── discord.go      # Discord connector
│           ├── github.go       # GitHub connector
│           ├── interfaces.go   # Connector interfaces
│           ├── registry.go     # Connector registry
│           └── slack.go        # Slack connector
├── scripts/                    # Utility scripts
│   ├── test-go-backend.sh      # Go backend tests
│   └── test_knowledge_graph.sh # Knowledge graph tests
├── secrets/                    # Secret files (gitignored)
│   ├── .gitkeep                # Keep directory in git
│   ├── github_client_secret.txt.example
│   ├── jwt_secret.txt.example
│   └── postgres_password.txt.example
├── src/                        # TypeScript/Node.js code
│   ├── config/                 # Node configuration
│   ├── demo/                   # Demo data and services
│   ├── mcp/                    # MCP server (Node)
│   ├── services/               # Node services
│   ├── slack/                  # Slack bot
│   ├── types/                  # TypeScript types
│   ├── utils/                  # Utilities
│   └── index.ts                # Node entry point
├── test/                       # Integration tests
│   └── system_test.go          # System-level tests
├── web/                        # Frontend web application
│   ├── auth.css                # Auth page styles
│   ├── auth.js                 # Auth JavaScript
│   ├── index.html              # Landing page
│   ├── login.html              # Login page
│   ├── nginx.conf.example      # Nginx config
│   ├── README.md               # Web docs
│   ├── script.js               # Main JavaScript
│   ├── server.js               # Dev server
│   ├── signup.html             # Signup page
│   └── styles.css              # Main styles
├── .env.example                # Environment variables template
├── .env.test                   # Test environment
├── .eslintrc.js                # ESLint configuration
├── .gitignore                  # Git ignore rules
├── docker-compose.dev.yml      # Dev Docker Compose
├── docker-compose.mcp-slack.yml # Slack bot Docker Compose
├── docker-compose.yml          # Production Docker Compose
├── Dockerfile                  # Main Dockerfile
├── Dockerfile.mcp-slack        # Slack bot Dockerfile
├── go.mod                      # Go module definition
├── go.sum                      # Go dependencies
├── Makefile                    # Build automation
├── package.json                # Node dependencies
├── package-lock.json           # Node lock file
├── README.md                   # Project README
├── SECURITY.md                 # Security policies
├── SETUP_GUIDE.md              # Setup instructions
├── tsconfig.json               # TypeScript config
└── vitest.config.ts            # Vitest config
```

## Key Directories

### `/cmd`
Application entry points. Each subdirectory is a separate executable.

### `/internal`
Private application code that cannot be imported by other projects.

### `/internal/services`
Business logic layer. Contains all service implementations.

### `/internal/services/connectors`
Platform connector implementations (GitHub, Slack, Discord).

### `/internal/handlers`
HTTP request handlers. Maps HTTP requests to service calls.

### `/internal/models`
Data models and database schemas.

### `/internal/repository`
Data access layer. Abstracts database operations.

### `/src`
TypeScript/Node.js code for MCP server and Slack bot.

### `/web`
Frontend web application (landing page, dashboard).

### `/docs`
Project documentation.

### `/deployment`
Deployment configurations (Kubernetes, Docker).

### `/scripts`
Utility scripts for testing and development.

### `/test`
Integration and system tests.

## File Naming Conventions

### Go Files
- `*_test.go` - Test files
- `*_properties_test.go` - Property-based test files
- `*_integration_test.go` - Integration test files

### TypeScript Files
- `*.test.ts` - Unit test files
- `*.integration.test.ts` - Integration test files
- `*.properties.test.ts` - Property-based test files

### Documentation
- `*.md` - Markdown documentation
- `README.md` - Directory-specific documentation

## Configuration Files

### Root Level
- `.env.example` - Environment variable template
- `.gitignore` - Git ignore rules
- `Makefile` - Build automation
- `go.mod` - Go module definition
- `package.json` - Node dependencies
- `tsconfig.json` - TypeScript configuration
- `vitest.config.ts` - Test configuration

### Docker
- `Dockerfile` - Main application container
- `Dockerfile.mcp-slack` - Slack bot container
- `docker-compose.yml` - Production compose
- `docker-compose.dev.yml` - Development compose
- `docker-compose.mcp-slack.yml` - Slack bot compose

## Build Artifacts (Gitignored)

- `/bin/` - Compiled Go binaries
- `/dist/` - Distribution builds
- `/coverage/` - Test coverage reports
- `/node_modules/` - Node dependencies
- `*.log` - Log files
- `*.db` - Database files

## Best Practices

### Code Organization
1. Keep related code together
2. Use clear, descriptive names
3. Separate concerns (handlers, services, models)
4. Write tests alongside code

### File Structure
1. One primary type per file
2. Group related functions
3. Keep files focused and small
4. Use subdirectories for logical grouping

### Documentation
1. Document public APIs
2. Include examples in docs
3. Keep README files updated
4. Use godoc comments for Go code

### Testing
1. Test files next to source files
2. Integration tests in `/test`
3. Property tests with `*_properties_test.go`
4. Maintain high test coverage

## Adding New Features

When adding a new feature:

1. **Create spec** in `.kiro/specs/`
2. **Add models** in `internal/models/`
3. **Implement service** in `internal/services/`
4. **Add handlers** in `internal/handlers/`
5. **Write tests** alongside code
6. **Update docs** in `/docs/`
7. **Add to README** if user-facing

## References

- [Go Project Layout](https://github.com/golang-standards/project-layout)
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Effective Go](https://golang.org/doc/effective_go)
