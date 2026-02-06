# Multi-stage Dockerfile for MCP Context Engine
# Supports both Go backend and Node.js services

# Stage 1: Build Go backend
FROM golang:1.21-alpine AS go-builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make

# Copy Go module files
COPY go.mod go.sum ./
RUN go mod download

# Copy Go source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/bin/server ./cmd/server

# Stage 2: Build Node.js services
FROM node:18-alpine AS node-builder

WORKDIR /app

# Copy package files
COPY package*.json tsconfig.json ./

# Install dependencies
RUN npm ci

# Copy TypeScript source
COPY src/ ./src/

# Build TypeScript
RUN npm run build

# Stage 3: Production image
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    nodejs \
    npm \
    dumb-init \
    && rm -rf /var/cache/apk/*

# Create app user
RUN addgroup -g 1001 -S appuser && \
    adduser -S appuser -u 1001 -G appuser

WORKDIR /app

# Copy Go binary
COPY --from=go-builder /app/bin/server /app/bin/server

# Copy Node.js built files
COPY --from=node-builder /app/dist /app/dist
COPY --from=node-builder /app/node_modules /app/node_modules
COPY package.json /app/

# Copy web assets
COPY web/ /app/web/

# Create necessary directories
RUN mkdir -p /app/logs /app/data && \
    chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose ports
# 8080: Go backend API
# 3001: MCP server
# 3002: Slack bot
EXPOSE 8080 3001 3002

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Use dumb-init for proper signal handling
ENTRYPOINT ["dumb-init", "--"]

# Default command (can be overridden)
CMD ["/app/bin/server"]
