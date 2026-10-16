// Package tools implements MCP tool handlers for Aseprite sprite manipulation.
//
// This package provides the bridge between MCP (Model Context Protocol) tool requests
// and Aseprite operations. Tools are organized by category:
//
//   - Canvas tools (canvas.go): Sprite/layer/frame creation and management
//   - Drawing tools (drawing.go): Pixel, line, shape rendering with palette support
//   - Animation tools (animation.go): Frame timing, tags, and linked cels
//   - Inspection tools (inspection.go): Reading pixel data and sprite metadata
//   - Analysis tools (analysis.go): Reference image palette extraction and edge detection
//   - Dithering tools (dithering.go): 15 dithering patterns for gradients and textures
//   - Palette tools (palette_tools.go): Palette management and color harmony analysis
//   - Transform tools (transform.go): Image downsampling for pixel art conversion
//   - Export tools (export.go): Sprite export to PNG, GIF, and other formats
//   - Selection tools (selection.go): Selection mask creation and manipulation
//   - Antialiasing tools (antialiasing.go): Edge detection and smoothing suggestions
//
// All tools follow a common pattern:
//  1. Input struct defines tool parameters with JSON schema annotations
//  2. Output struct defines tool results
//  3. Register function creates MCP tool descriptor with schema
//  4. Handler function executes the Aseprite operation via Client
//
// Tools use Aseprite's Lua API through generated scripts for all operations.
// No GUI automation or screen scraping is used.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/willibrandon/mtlog/core"
	"github.com/willibrandon/pixel-mcp/pkg/aseprite"
	"github.com/willibrandon/pixel-mcp/pkg/config"
)

// CreateCanvasInput defines the input parameters for the create_canvas tool.
//
// Creates a new sprite file with the specified dimensions and color mode.
// The sprite is saved as a .aseprite file in the configured temp directory or custom output path.
type CreateCanvasInput struct {
	Width      int    `json:"width" jsonschema:"Canvas width in pixels (1-65535)"`
	Height     int    `json:"height" jsonschema:"Canvas height in pixels (1-65535)"`
	ColorMode  string `json:"color_mode" jsonschema:"Color mode: rgb, grayscale, or indexed"`
	OutputPath string `json:"output_path,omitempty" jsonschema:"Optional custom destination path for the created .aseprite file. If omitted, saves to temp directory."`
}

// CreateCanvasOutput defines the output for the create_canvas tool.
//
// Returns the absolute path to the newly created sprite file.
type CreateCanvasOutput struct {
	FilePath string `json:"file_path" jsonschema:"Absolute path to the created Aseprite file"` // Absolute path to the created .aseprite file
}

// AddLayerInput defines the input parameters for the add_layer tool.
//
// Adds a new layer to an existing sprite. The layer is created above all existing layers.
type AddLayerInput struct {
	SpritePath string `json:"sprite_path" jsonschema:"Path to the Aseprite sprite file"` // Path to the sprite file to modify
	LayerName  string `json:"layer_name" jsonschema:"Name for the new layer"`            // Name for the new layer
}

// AddLayerOutput defines the output for the add_layer tool.
//
// Indicates whether the layer was successfully added.
type AddLayerOutput struct {
	Success bool `json:"success" jsonschema:"Whether the layer was added successfully"` // True if the layer was added successfully
}

// AddFrameInput defines the input parameters for the add_frame tool.
//
// Adds a new animation frame to the sprite with the specified duration.
type AddFrameInput struct {
	SpritePath string `json:"sprite_path" jsonschema:"Path to the Aseprite sprite file"`         // Path to the sprite file to modify
	DurationMs int    `json:"duration_ms" jsonschema:"Frame duration in milliseconds (1-65535)"` // Frame duration in milliseconds (1-65535)
}

// AddFrameOutput defines the output for the add_frame tool.
//
// Returns the 1-based index of the newly created frame.
type AddFrameOutput struct {
	FrameNumber int `json:"frame_number" jsonschema:"Index of the created frame (1-based)"` // 1-based index of the created frame
}

// GetSpriteInfoInput defines the input parameters for the get_sprite_info tool.
//
// Retrieves metadata about an existing sprite file.
type GetSpriteInfoInput struct {
	SpritePath string `json:"sprite_path" jsonschema:"Path to the Aseprite sprite file"` // Path to the sprite file to inspect
}

// GetSpriteInfoOutput defines the output for the get_sprite_info tool.
//
// Contains complete metadata about a sprite including dimensions, color mode,
// and animation/layer information.
type GetSpriteInfoOutput struct {
	Width      int      `json:"width" jsonschema:"Sprite width in pixels"`                       // Sprite width in pixels
	Height     int      `json:"height" jsonschema:"Sprite height in pixels"`                     // Sprite height in pixels
	ColorMode  string   `json:"color_mode" jsonschema:"Color mode (rgb, grayscale, or indexed)"` // Color mode: "rgb", "grayscale", or "indexed"
	FrameCount int      `json:"frame_count" jsonschema:"Number of frames in the sprite"`         // Total number of animation frames
	LayerCount int      `json:"layer_count" jsonschema:"Number of layers in the sprite"`         // Total number of layers
	Layers     []string `json:"layers" jsonschema:"Names of all layers in the sprite"`           // Names of all layers from bottom to top
}

// RegisterCanvasTools registers all canvas management tools with the MCP server.
//
// Registers the following tools:
//   - create_canvas: Create new sprites with specified dimensions and color mode
//   - add_layer: Add layers to existing sprites
//   - add_frame: Add animation frames with duration control
//   - get_sprite_info: Retrieve sprite metadata and structure
//
// All tools operate on sprite files saved to disk. Canvas creation generates
// a new .aseprite file in the configured temp directory.
func RegisterCanvasTools(server *mcp.Server, client *aseprite.Client, gen *aseprite.LuaGenerator, cfg *config.Config, logger core.Logger) {
	// Register create_canvas tool
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "create_canvas",
			Description: "Create a new Aseprite sprite with specified dimensions and color mode. Returns the path to the created .aseprite file.",
		},
		maybeWrapWithTiming("create_canvas", logger, cfg.EnableTiming, func(ctx context.Context, req *mcp.CallToolRequest, input CreateCanvasInput) (*mcp.CallToolResult, *CreateCanvasOutput, error) {
			// Use logger with context for request tracking
			opLogger := logger.WithContext(ctx)
			opLogger.Debug("create_canvas parameters", "width", input.Width, "height", input.Height, "color_mode", input.ColorMode)

			// Validate color mode
			var colorMode aseprite.ColorMode
			switch input.ColorMode {
			case "rgb":
				colorMode = aseprite.ColorModeRGB
			case "grayscale":
				colorMode = aseprite.ColorModeGrayscale
			case "indexed":
				colorMode = aseprite.ColorModeIndexed
			default:
				return ToolError[CreateCanvasOutput](fmt.Sprintf("invalid color mode: %s (must be rgb, grayscale, or indexed)", input.ColorMode), nil)
			}

			// Generate filename
			filename := input.OutputPath
			if filename == "" {
				filename = filepath.Join(cfg.TempDir, fmt.Sprintf("sprite-%d.aseprite", generateTimestamp()))
			} else {
				if dir := filepath.Dir(filename); dir != "" {
					_ = os.MkdirAll(dir, 0755)
				}
			}

			// Generate Lua script
			script := gen.CreateCanvas(input.Width, input.Height, colorMode, filename)

			// Execute Lua script with Aseprite
			output, err := client.ExecuteLua(ctx, script, "")
			if err != nil {
				opLogger.Error("Failed to create canvas", "error", err)
				return ToolError[CreateCanvasOutput]("failed to create canvas", err)
			}

			// Parse output to get file path
			filePath := strings.TrimSpace(output)

			opLogger.Debug("Canvas created successfully", "path", filePath, "width", input.Width, "height", input.Height)

			return ToolSuccess(&CreateCanvasOutput{FilePath: filePath})
		}),
	)

	// Register add_layer tool
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "add_layer",
			Description: "Add a new layer to an existing Aseprite sprite. Returns success status.",
		},
		maybeWrapWithTiming("add_layer", logger, cfg.EnableTiming, func(ctx context.Context, req *mcp.CallToolRequest, input AddLayerInput) (*mcp.CallToolResult, *AddLayerOutput, error) {
			opLogger := logger.WithContext(ctx)
			opLogger.Debug("add_layer tool called", "sprite_path", input.SpritePath, "layer_name", input.LayerName)

			// Validate layer name
			if input.LayerName == "" {
				return nil, nil, fmt.Errorf("layer_name cannot be empty")
			}

			// Generate Lua script
			script := gen.AddLayer(input.LayerName)

			// Execute Lua script with the sprite
			_, err := client.ExecuteLua(ctx, script, input.SpritePath)
			if err != nil {
				opLogger.Error("Failed to add layer", "error", err)
				return nil, nil, fmt.Errorf("failed to add layer: %w", err)
			}

			opLogger.Debug("Layer added successfully", "sprite", input.SpritePath, "layer", input.LayerName)

			return nil, &AddLayerOutput{Success: true}, nil
		}),
	)

	// Register add_frame tool
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "add_frame",
			Description: "Add a new frame to an existing Aseprite sprite. Returns the frame number (1-based index).",
		},
		maybeWrapWithTiming("add_frame", logger, cfg.EnableTiming, func(ctx context.Context, req *mcp.CallToolRequest, input AddFrameInput) (*mcp.CallToolResult, *AddFrameOutput, error) {
			opLogger := logger.WithContext(ctx)
			opLogger.Debug("add_frame tool called", "sprite_path", input.SpritePath, "duration_ms", input.DurationMs)

			// Validate duration
			if input.DurationMs < 1 || input.DurationMs > 65535 {
				return nil, nil, fmt.Errorf("duration_ms must be between 1 and 65535, got %d", input.DurationMs)
			}

			// Generate Lua script
			script := gen.AddFrame(input.DurationMs)

			// Execute Lua script with the sprite
			output, err := client.ExecuteLua(ctx, script, input.SpritePath)
			if err != nil {
				opLogger.Error("Failed to add frame", "error", err)
				return nil, nil, fmt.Errorf("failed to add frame: %w", err)
			}

			// Parse frame number from output
			var frameNumber int
			_, err = fmt.Sscanf(strings.TrimSpace(output), "%d", &frameNumber)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse frame number from output: %w", err)
			}

			opLogger.Debug("Frame added successfully", "sprite", input.SpritePath, "frame_number", frameNumber)

			return nil, &AddFrameOutput{FrameNumber: frameNumber}, nil
		}),
	)

	// Register get_sprite_info tool
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "get_sprite_info",
			Description: "Retrieve metadata about an existing Aseprite sprite including dimensions, color mode, frame count, layer count, and layer names.",
		},
		maybeWrapWithTiming("get_sprite_info", logger, cfg.EnableTiming, func(ctx context.Context, req *mcp.CallToolRequest, input GetSpriteInfoInput) (*mcp.CallToolResult, *GetSpriteInfoOutput, error) {
			opLogger := logger.WithContext(ctx)
			opLogger.Debug("get_sprite_info tool called", "sprite_path", input.SpritePath)

			// Generate Lua script
			script := gen.GetSpriteInfo()

			// Execute Lua script with the sprite
			output, err := client.ExecuteLua(ctx, script, input.SpritePath)
			if err != nil {
				opLogger.Error("Failed to get sprite info", "error", err)
				return nil, nil, fmt.Errorf("failed to get sprite info: %w", err)
			}

			// Parse JSON output
			var info GetSpriteInfoOutput
			if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &info); err != nil {
				opLogger.Error("Failed to parse sprite info", "error", err, "output", output)
				return nil, nil, fmt.Errorf("failed to parse sprite info: %w", err)
			}

			opLogger.Debug("Sprite info retrieved successfully", "sprite", input.SpritePath, "width", info.Width, "height", info.Height)

			return nil, &info, nil
		}),
	)

	// Register delete_layer tool
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "delete_layer",
			Description: "Delete a layer from an existing sprite. Cannot delete the last remaining layer.",
		},
		maybeWrapWithTiming("delete_layer", logger, cfg.EnableTiming, func(ctx context.Context, req *mcp.CallToolRequest, input DeleteLayerInput) (*mcp.CallToolResult, *DeleteLayerOutput, error) {
			opLogger := logger.WithContext(ctx)
			opLogger.Debug("delete_layer tool called", "sprite_path", input.SpritePath, "layer_name", input.LayerName)

			// Generate Lua script
			script := gen.DeleteLayer(input.LayerName)

			// Execute Lua script with the sprite
			_, err := client.ExecuteLua(ctx, script, input.SpritePath)
			if err != nil {
				opLogger.Error("Failed to delete layer", "error", err)
				return nil, nil, fmt.Errorf("failed to delete layer: %w", err)
			}

			opLogger.Information("Layer deleted successfully", "sprite", input.SpritePath, "layer", input.LayerName)

			return nil, &DeleteLayerOutput{Success: true}, nil
		}),
	)

	// Register delete_frame tool
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "delete_frame",
			Description: "Delete a frame from an existing sprite. Cannot delete the last remaining frame.",
		},
		maybeWrapWithTiming("delete_frame", logger, cfg.EnableTiming, func(ctx context.Context, req *mcp.CallToolRequest, input DeleteFrameInput) (*mcp.CallToolResult, *DeleteFrameOutput, error) {
			opLogger := logger.WithContext(ctx)
			opLogger.Debug("delete_frame tool called", "sprite_path", input.SpritePath, "frame_number", input.FrameNumber)

			// Generate Lua script
			script := gen.DeleteFrame(input.FrameNumber)

			// Execute Lua script with the sprite
			_, err := client.ExecuteLua(ctx, script, input.SpritePath)
			if err != nil {
				opLogger.Error("Failed to delete frame", "error", err)
				return nil, nil, fmt.Errorf("failed to delete frame: %w", err)
			}

			opLogger.Information("Frame deleted successfully", "sprite", input.SpritePath, "frame_number", input.FrameNumber)

			return nil, &DeleteFrameOutput{Success: true}, nil
		}),
	)

	// Register flatten_layers tool
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "flatten_layers",
			Description: "Flatten all layers in a sprite into a single layer.",
		},
		maybeWrapWithTiming("flatten_layers", logger, cfg.EnableTiming, func(ctx context.Context, req *mcp.CallToolRequest, input FlattenLayersInput) (*mcp.CallToolResult, *FlattenLayersOutput, error) {
			opLogger := logger.WithContext(ctx)
			opLogger.Debug("flatten_layers tool called", "sprite_path", input.SpritePath)

			// Generate Lua script
			script := gen.FlattenLayers()

			// Execute Lua script with the sprite
			_, err := client.ExecuteLua(ctx, script, input.SpritePath)
			if err != nil {
				opLogger.Error("Failed to flatten layers", "error", err)
				return nil, nil, fmt.Errorf("failed to flatten layers: %w", err)
			}

			opLogger.Information("Layers flattened successfully", "sprite", input.SpritePath)

			return nil, &FlattenLayersOutput{Success: true}, nil
		}),
	)
}

// DeleteLayerInput defines the input parameters for the delete_layer tool.
type DeleteLayerInput struct {
	SpritePath string `json:"sprite_path" jsonschema:"Path to the sprite file"`
	LayerName  string `json:"layer_name" jsonschema:"Name of the layer to delete"`
}

// DeleteLayerOutput defines the output for the delete_layer tool.
type DeleteLayerOutput struct {
	Success bool `json:"success" jsonschema:"Whether the layer was deleted successfully"`
}

// DeleteFrameInput defines the input parameters for the delete_frame tool.
type DeleteFrameInput struct {
	SpritePath  string `json:"sprite_path" jsonschema:"Path to the sprite file"`
	FrameNumber int    `json:"frame_number" jsonschema:"Frame number to delete (1-based)"`
}

// DeleteFrameOutput defines the output for the delete_frame tool.
type DeleteFrameOutput struct {
	Success bool `json:"success" jsonschema:"Whether the frame was deleted successfully"`
}

// FlattenLayersInput defines the input parameters for the flatten_layers tool.
//
// Flattens all layers in a sprite into a single layer.
type FlattenLayersInput struct {
	SpritePath string `json:"sprite_path" jsonschema:"Path to the Aseprite sprite file"` // Path to the sprite file to modify
}

// FlattenLayersOutput defines the output for the flatten_layers tool.
//
// Indicates whether the layers were successfully flattened.
type FlattenLayersOutput struct {
	Success bool `json:"success" jsonschema:"Whether the layers were flattened successfully"` // True if the layers were flattened successfully
}

// generateTimestamp returns a Unix timestamp in nanoseconds suitable for unique filenames.
func generateTimestamp() int64 {
	return time.Now().UnixNano()
}
