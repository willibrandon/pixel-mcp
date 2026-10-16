package tools

import (
	"context"
	"fmt"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/willibrandon/mtlog/core"
	"github.com/willibrandon/pixel-mcp/pkg/aseprite"
	"github.com/willibrandon/pixel-mcp/pkg/config"
)

// GetSpritePreviewInput defines parameters for getting a visual sprite preview.
type GetSpritePreviewInput struct {
	SpritePath  string `json:"sprite_path" jsonschema:"Path to the .aseprite file"`
	Scale       int    `json:"scale,omitempty" jsonschema:"Magnification scale factor (default: 4, min: 1, max: 32). Uses nearest-neighbor scaling for sharp pixels."`
	FrameNumber int    `json:"frame_number,omitempty" jsonschema:"Optional frame number (1-based). Defaults to 1."`
	LayerName   string `json:"layer_name,omitempty" jsonschema:"Optional layer name to isolate."`
}

// GetSpritePreviewOutput defines the result structure for sprite preview metadata.
type GetSpritePreviewOutput struct {
	Success bool   `json:"success" jsonschema:"Whether preview generation succeeded"`
	Scale   int    `json:"scale" jsonschema:"Scale factor applied to the sprite"`
	Message string `json:"message" jsonschema:"Summary message"`
}

// RegisterPreviewTools registers the get_sprite_preview tool with the MCP server.
func RegisterPreviewTools(server *mcp.Server, client *aseprite.Client, cfg *config.Config, logger core.Logger) {
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "get_sprite_preview",
			Description: "Render and return a crisp, scaled PNG visual preview of an Aseprite sprite directly in the MCP tool response. Enables multimodal models to view current pixel art, inspect anatomy, proportions, lighting, and palette placement.",
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint: true,
			},
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input GetSpritePreviewInput) (*mcp.CallToolResult, *GetSpritePreviewOutput, error) {
			opLogger := logger.WithContext(ctx)
			opLogger.Debug("get_sprite_preview called", "sprite", input.SpritePath, "scale", input.Scale)

			if input.SpritePath == "" {
				return ToolError[GetSpritePreviewOutput]("sprite_path cannot be empty", nil)
			}
			if _, err := os.Stat(input.SpritePath); os.IsNotExist(err) {
				return ToolError[GetSpritePreviewOutput](fmt.Sprintf("sprite file not found: %s", input.SpritePath), nil)
			}

			scale := input.Scale
			if scale <= 0 {
				scale = 4
			}
			if scale > 32 {
				scale = 32
			}

			frame := input.FrameNumber
			if frame <= 0 {
				frame = 1
			}

			imgBytes, err := client.ExportPreview(ctx, input.SpritePath, scale, frame, input.LayerName)
			if err != nil {
				opLogger.Error("Failed to render preview", "error", err)
				return ToolError[GetSpritePreviewOutput]("failed to render sprite preview", err)
			}

			out := &GetSpritePreviewOutput{
				Success: true,
				Scale:   scale,
				Message: fmt.Sprintf("Rendered %dx preview for %s", scale, input.SpritePath),
			}

			return ToolSuccessWithImage(out, imgBytes, "image/png", out.Message)
		},
	)
}
