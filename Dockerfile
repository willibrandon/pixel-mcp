# Multi-stage build for pixel-mcp server

# Stage 1: Build the Go binary
FROM golang:1.24.1-alpine AS builder

WORKDIR /build

# Cache dependency layer
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build statically-linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o pixel-mcp ./cmd/pixel-mcp

# Stage 2: Runtime image with Aseprite dependencies
FROM ubuntu:22.04

LABEL org.opencontainers.image.source=https://github.com/willibrandon/pixel-mcp
LABEL org.opencontainers.image.description="MCP server for Aseprite pixel art and animation capabilities"
LABEL org.opencontainers.image.licenses=MIT
LABEL io.modelcontextprotocol.server.name="io.github.willibrandon/pixel-mcp"

# Install minimal runtime dependencies for headless Aseprite execution
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    libx11-6 \
    libxcursor1 \
    libxi6 \
    libxrandr2 \
    libgl1 \
    libfontconfig1 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Create unprivileged user
RUN groupadd -g 10001 appgroup && \
    useradd -u 10001 -g appgroup -m -s /bin/bash appuser

# Copy the Go binary from builder
COPY --from=builder /build/pixel-mcp /usr/local/bin/pixel-mcp

# Create directories and set ownership
RUN mkdir -p /tmp/pixel-mcp /home/appuser/.config/pixel-mcp && \
    chown -R appuser:appgroup /tmp/pixel-mcp /home/appuser

# Default configuration via environment variables
ENV PIXEL_MCP_ASEPRITE_PATH=/usr/local/bin/aseprite \
    PIXEL_MCP_TEMP_DIR=/tmp/pixel-mcp \
    PIXEL_MCP_TIMEOUT=30 \
    PIXEL_MCP_LOG_LEVEL=info

USER appuser
WORKDIR /home/appuser

# MCP server communicates via stdio
ENTRYPOINT ["/usr/local/bin/pixel-mcp"]
