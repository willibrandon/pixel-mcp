package tools

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/willibrandon/mtlog/core"
	"github.com/willibrandon/pixel-mcp/pkg/aseprite"
	"github.com/willibrandon/pixel-mcp/pkg/config"
)

// ApplyAutoShadingInput defines the input parameters for the apply_auto_shading tool.
type ApplyAutoShadingInput struct {
	SpritePath     string  `json:"sprite_path" jsonschema:"Path to .aseprite file"`
	LayerName      string  `json:"layer_name" jsonschema:"Layer to apply shading to"`
	FrameNumber    int     `json:"frame_number" jsonschema:"Frame number (1-based)"`
	LightDirection string  `json:"light_direction" jsonschema:"Light direction: top_left, top, top_right, left, right, bottom_left, bottom, bottom_right"`
	Intensity      float64 `json:"intensity" jsonschema:"Shading intensity (0.0-1.0)"`
	Style          string  `json:"style" jsonschema:"Shading style: cell, smooth, or soft"`
	HueShift       bool    `json:"hue_shift" jsonschema:"Apply hue shifting (shadows→cool, highlights→warm) default: true"`
}

// ApplyAutoShadingOutput defines the output for the apply_auto_shading tool.
type ApplyAutoShadingOutput struct {
	Success       bool     `json:"success" jsonschema:"Whether the operation succeeded"`
	ColorsAdded   int      `json:"colors_added" jsonschema:"Number of colors added to palette"`
	Palette       []string `json:"palette,omitempty" jsonschema:"Final palette after shading"`
	RegionsShaded int      `json:"regions_shaded" jsonschema:"Number of regions shaded"`
}

// RegisterAutoShadingTools registers the apply_auto_shading tool with the MCP server.
func RegisterAutoShadingTools(server *mcp.Server, client *aseprite.Client, gen *aseprite.LuaGenerator, cfg *config.Config, logger core.Logger) {
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "apply_auto_shading",
			Description: "Automatically add shading to sprite based on light direction. Analyzes sprite geometry to identify surfaces/regions, determines which surfaces face toward/away from light, generates shadow and highlight colors for each base color (with optional hue shifting), and applies shading pixels with smooth transitions. Supports three styles: cell (hard-edged 2-3 bands), smooth (gradient with dithering), soft (subtle gradient). Essential for adding depth and dimension to pixel art automatically.",
		},
		maybeWrapWithTiming("apply_auto_shading", logger, cfg.EnableTiming, func(ctx context.Context, req *mcp.CallToolRequest, input ApplyAutoShadingInput) (*mcp.CallToolResult, *ApplyAutoShadingOutput, error) {
			opLogger := logger.WithContext(ctx)
			opLogger.Debug("apply_auto_shading tool called",
				"sprite", input.SpritePath,
				"layer", input.LayerName,
				"frame", input.FrameNumber,
				"light_direction", input.LightDirection,
				"intensity", input.Intensity,
				"style", input.Style)

			// Validate inputs
			if input.Intensity < 0.0 || input.Intensity > 1.0 {
				return ToolError[ApplyAutoShadingOutput](fmt.Sprintf("intensity must be between 0.0 and 1.0, got %f", input.Intensity), nil)
			}

			validDirections := map[string]bool{
				"top_left": true, "top": true, "top_right": true,
				"left": true, "right": true,
				"bottom_left": true, "bottom": true, "bottom_right": true,
			}
			if !validDirections[input.LightDirection] {
				return ToolError[ApplyAutoShadingOutput](fmt.Sprintf("invalid light direction: %s", input.LightDirection), nil)
			}

			validStyles := map[string]bool{
				"cell": true, "smooth": true, "soft": true,
			}
			if !validStyles[input.Style] {
				return ToolError[ApplyAutoShadingOutput](fmt.Sprintf("invalid style: %s (must be cell, smooth, or soft)", input.Style), nil)
			}

			// Check sprite file exists
			if _, err := os.Stat(input.SpritePath); os.IsNotExist(err) {
				return ToolError[ApplyAutoShadingOutput](fmt.Sprintf("sprite file not found: %s", input.SpritePath), nil)
			}

			// Step 1: Export layer/frame to temporary PNG
			tempDir, err := os.MkdirTemp("", "pixel-mcp-shading-*")
			if err != nil {
				return ToolError[ApplyAutoShadingOutput]("failed to create temp directory", err)
			}
			defer os.RemoveAll(tempDir)

			tempPNG := filepath.Join(tempDir, "layer.png")

			// Export specific layer and frame to PNG
			exportScript := fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

-- Find target layer
local targetLayer = nil
for _, layer in ipairs(spr.layers) do
	if layer.name == %q then
		targetLayer = layer
		break
	end
end

if not targetLayer then
	error("Layer not found: " .. %q)
end

-- Get cel at specified frame
local cel = targetLayer:cel(%d)
if not cel then
	error("No cel found at frame " .. %d)
end

-- Export cel image
cel.image:saveAs(%q)
print("Exported successfully")`,
				input.LayerName, input.LayerName,
				input.FrameNumber, input.FrameNumber,
				tempPNG)

			_, err = client.ExecuteLua(ctx, exportScript, input.SpritePath)
			if err != nil {
				return ToolError[ApplyAutoShadingOutput]("failed to export layer to PNG", err)
			}

			// Step 2: Load PNG and apply auto-shading in Go
			imgFile, err := os.Open(tempPNG)
			if err != nil {
				return ToolError[ApplyAutoShadingOutput]("failed to open exported PNG", err)
			}
			defer imgFile.Close()

			img, err := png.Decode(imgFile)
			if err != nil {
				// Try to decode as any image format
				imgFile.Seek(0, 0)
				img, _, err = image.Decode(imgFile)
				if err != nil {
					return ToolError[ApplyAutoShadingOutput]("failed to decode image", err)
				}
			}

			// Apply auto-shading
			shadedImg, generatedColors, regionsShadedCount, err := aseprite.ApplyAutoShading(
				img,
				input.LightDirection,
				input.Intensity,
				input.Style,
				input.HueShift,
			)
			if err != nil {
				return ToolError[ApplyAutoShadingOutput]("auto-shading failed", err)
			}

			opLogger.Information("Auto-shading completed",
				"colors_added", len(generatedColors),
				"regions_shaded", regionsShadedCount)

			// Save shaded image to temp file
			shadedPNG := filepath.Join(tempDir, "shaded.png")
			shadedFile, err := os.Create(shadedPNG)
			if err != nil {
				return ToolError[ApplyAutoShadingOutput]("failed to create shaded PNG", err)
			}
			defer shadedFile.Close()

			if err := png.Encode(shadedFile, shadedImg); err != nil {
				return ToolError[ApplyAutoShadingOutput]("failed to encode shaded PNG", err)
			}

			// Step 3: Generate and execute Lua script to apply shaded result
			applyScript := gen.ApplyAutoShadingResult(
				shadedPNG,
				input.LayerName,
				input.FrameNumber,
				generatedColors,
				regionsShadedCount,
			)

			output, err := client.ExecuteLua(ctx, applyScript, input.SpritePath)
			if err != nil {
				opLogger.Error("Failed to apply shaded result", "error", err)
				return ToolError[ApplyAutoShadingOutput]("failed to apply shaded result", err)
			}

			// Parse JSON output from Lua
			var result ApplyAutoShadingOutput
			if err := parseJSON(output, &result); err != nil {
				return ToolError[ApplyAutoShadingOutput]("failed to parse shading output", err)
			}

			opLogger.Information("Auto-shading applied successfully",
				"sprite", input.SpritePath,
				"layer", input.LayerName,
				"colors_added", result.ColorsAdded,
				"regions_shaded", result.RegionsShaded)

			return ToolSuccess(&result)
		}),
	)
}
