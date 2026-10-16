// Package server provides the MCP server implementation for Aseprite integration.
//
// This package orchestrates the MCP (Model Context Protocol) server lifecycle,
// connecting MCP tool requests to Aseprite operations through the aseprite and
// tools packages.
//
// Server Lifecycle:
//  1. Create server with New() using validated config
//  2. Tools are automatically registered during initialization
//  3. Run() starts the server with stdio transport
//  4. Server processes tool requests via MCP protocol
//  5. Context cancellation triggers graceful shutdown
//
// The server uses stdio transport for communication with MCP clients and
// supports all registered tool categories (canvas, drawing, animation, etc.).
package server

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/willibrandon/mtlog/core"
	"github.com/willibrandon/pixel-mcp/pkg/aseprite"
	"github.com/willibrandon/pixel-mcp/pkg/config"
	"github.com/willibrandon/pixel-mcp/pkg/tools"
)

// Server wraps the MCP server and provides Aseprite tool implementations.
//
// The server initializes all required components (MCP server, Aseprite client,
// Lua generator) and automatically registers all available tools. It handles
// the complete lifecycle of MCP tool request processing.
type Server struct {
	mcp    *mcp.Server
	client *aseprite.Client
	gen    *aseprite.LuaGenerator
	config *config.Config
	logger core.Logger
}

// New creates a new Aseprite MCP server with the given configuration.
//
// The configuration is validated before server creation. If validation fails,
// an error is returned immediately.
//
// During initialization, New:
//   - Creates an Aseprite client with configured executable path and timeout
//   - Initializes a Lua script generator
//   - Creates the MCP server with protocol implementation
//   - Automatically registers all available tools
//
// Returns an error if configuration validation fails.
func New(cfg *config.Config, logger core.Logger) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create Aseprite client
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, cfg.Timeout)

	// Create Lua generator
	gen := aseprite.NewLuaGenerator()

	// Create MCP server
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "pixel-mcp",
		Version: "0.1.0",
	}, nil)

	s := &Server{
		mcp:    mcpServer,
		client: client,
		gen:    gen,
		config: cfg,
		logger: logger,
	}

	// Register tools (will be implemented in next tasks)
	s.registerTools()

	return s, nil
}

// Run starts the MCP server with stdio transport.
//
// The server listens for MCP protocol messages on stdin and writes responses
// to stdout. Tool requests are processed synchronously in the order received.
//
// Run blocks until:
//   - The context is cancelled (graceful shutdown)
//   - The client closes the connection
//   - A fatal error occurs
//
// Returns an error if server startup or execution fails.
// Context cancellation triggers graceful shutdown and does not return an error.
func (s *Server) Run(ctx context.Context) error {
	s.logger.Information("Starting Aseprite MCP server")
	s.logger.Debug("Configuration: {@Config}", s.config)

	transport := &mcp.StdioTransport{}

	if err := s.mcp.Run(ctx, transport); err != nil {
		return fmt.Errorf("server failed: %w", err)
	}

	return nil
}

// registerTools registers all MCP tools with the server.
//
// Called automatically during server initialization. Registers tools from all
// categories in the following order:
//   - Canvas tools (sprite/layer/frame creation)
//   - Drawing tools (pixels, lines, shapes, fill)
//   - Selection tools (selection mask creation)
//   - Export tools (PNG, GIF, spritesheet export)
//   - Animation tools (frame timing, tags, linked cels)
//   - Inspection tools (pixel data reading)
//   - Transform tools (image downsampling)
//   - Analysis tools (palette extraction, edge detection)
//   - Dithering tools (gradient and texture patterns)
//   - Palette tools (color management and harmonies)
//   - Antialiasing tools (edge smoothing suggestions)
//
// This method is not intended for external use.
func (s *Server) registerTools() {
	s.logger.Debug("Registering MCP tools")

	// Register canvas management tools
	tools.RegisterCanvasTools(s.mcp, s.client, s.gen, s.config, s.logger)

	// Register drawing tools
	tools.RegisterDrawingTools(s.mcp, s.client, s.gen, s.config, s.logger)
	tools.RegisterBatchTools(s.mcp, s.client, s.gen, s.config, s.logger)

	// Register selection tools
	tools.RegisterSelectionTools(s.mcp, s.client, s.gen, s.config, s.logger)

	// Register export tools
	tools.RegisterExportTools(s.mcp, s.client, s.gen, s.config, s.logger)

	// Register animation tools
	tools.RegisterAnimationTools(s.mcp, s.client, s.gen, s.config, s.logger)

	// Register inspection tools
	tools.RegisterInspectionTools(s.mcp, s.client, s.gen, s.config, s.logger)

	// Register transform tools
	tools.RegisterTransformTools(s.mcp, s.client, s.gen, s.config, s.logger)

	// Register analysis tools
	tools.RegisterAnalysisTools(s.mcp, s.client, s.gen, s.config, s.logger)

	// Register dithering tools
	tools.RegisterDitheringTools(s.mcp, s.client, s.gen, s.config, s.logger)

	// Register palette tools
	tools.RegisterPaletteTools(s.mcp, s.client, s.gen, s.config, s.logger)

	// Register quantization tools
	tools.RegisterQuantizationTools(s.mcp, s.client, s.gen, s.config, s.logger)

	// Register auto-shading tools
	tools.RegisterAutoShadingTools(s.mcp, s.client, s.gen, s.config, s.logger)

	// Register antialiasing tools
	tools.RegisterAntialiasingTools(s.mcp, s.client, s.gen, s.config, s.logger)

	// Register quality engine tools (preview, color ramps, reference import)
	tools.RegisterPreviewTools(s.mcp, s.client, s.config, s.logger)
	tools.RegisterColorRampTools(s.mcp, s.client, s.gen, s.config, s.logger)
	tools.RegisterReferenceImportTools(s.mcp, s.client, s.config, s.logger)
}

// Client returns the underlying Aseprite client for testing.
//
// This method is primarily used in tests to access the client for
// verification and cleanup operations. Not intended for production use.
func (s *Server) Client() *aseprite.Client {
	return s.client
}
