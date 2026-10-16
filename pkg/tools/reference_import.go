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

// ImportReferenceLayerInput defines parameters for importing a reference image as a tracing guide.
type ImportReferenceLayerInput struct {
	SpritePath  string `json:"sprite_path" jsonschema:"Path to the .aseprite file"`
	ImagePath   string `json:"image_path" jsonschema:"Path to the reference image (PNG, JPG, BMP)"`
	LayerName   string `json:"layer_name,omitempty" jsonschema:"Name for the reference guide layer (default: Reference)"`
	Opacity     int    `json:"opacity,omitempty" jsonschema:"Layer opacity 0-255 (default: 160 for semi-transparent tracing)"`
	FrameNumber int    `json:"frame_number,omitempty" jsonschema:"Target frame number (default: 1)"`
}

// ImportReferenceLayerOutput defines the result structure for importing a reference guide.
type ImportReferenceLayerOutput struct {
	Success   bool   `json:"success" jsonschema:"Whether import succeeded"`
	LayerName string `json:"layer_name" jsonschema:"Created guide layer name"`
	Message   string `json:"message" jsonschema:"Summary message"`
}

// RegisterReferenceImportTools registers the import_reference_layer tool with the MCP server.
func RegisterReferenceImportTools(server *mcp.Server, client *aseprite.Client, cfg *config.Config, logger core.Logger) {
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "import_reference_layer",
			Description: "Import an external reference image (character artwork, photo, or sketch) and place it on a semi-transparent guide layer scaled to canvas bounds. Allows tracing accurate anatomy, silhouettes, and proportions.",
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint: false,
			},
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input ImportReferenceLayerInput) (*mcp.CallToolResult, *ImportReferenceLayerOutput, error) {
			opLogger := logger.WithContext(ctx)
			opLogger.Debug("import_reference_layer called", "sprite", input.SpritePath, "image", input.ImagePath)

			if input.SpritePath == "" {
				return ToolError[ImportReferenceLayerOutput]("sprite_path cannot be empty", nil)
			}
			if _, err := os.Stat(input.SpritePath); os.IsNotExist(err) {
				return ToolError[ImportReferenceLayerOutput](fmt.Sprintf("sprite not found: %s", input.SpritePath), nil)
			}
			if input.ImagePath == "" {
				return ToolError[ImportReferenceLayerOutput]("image_path cannot be empty", nil)
			}
			if _, err := os.Stat(input.ImagePath); os.IsNotExist(err) {
				return ToolError[ImportReferenceLayerOutput](fmt.Sprintf("reference image not found: %s", input.ImagePath), nil)
			}

			layerName := input.LayerName
			if layerName == "" {
				layerName = "Reference"
			}

			opacity := input.Opacity
			if opacity <= 0 {
				opacity = 160
			}
			if opacity > 255 {
				opacity = 255
			}

			frame := input.FrameNumber
			if frame <= 0 {
				frame = 1
			}

			script := fmt.Sprintf(`local spr = app.activeSprite
if not spr then error("No active sprite") end

local refImg = Image{fromFile=%q}
if not refImg then error("Failed to load reference image") end

-- Scale reference image to fit canvas proportionally
local scaleX = spr.width / refImg.width
local scaleY = spr.height / refImg.height
local scale = math.min(scaleX, scaleY)
local newW = math.max(1, math.floor(refImg.width * scale))
local newH = math.max(1, math.floor(refImg.height * scale))

local scaledImg = Image(spr.width, spr.height, spr.colorMode)
-- Center image
local offsetX = math.floor((spr.width - newW) / 2)
local offsetY = math.floor((spr.height - newH) / 2)

-- Find or create reference layer
local refLayer = nil
for _, l in ipairs(spr.layers) do
	if l.name == %q then
		refLayer = l
		break
	end
end

if not refLayer then
	refLayer = spr:newLayer()
	refLayer.name = %q
end

refLayer.opacity = %d

app.transaction(function()
	local tgtFrame = spr.frames[%d]
	if not tgtFrame then
		error("Frame not found: %d")
	end
	local cel = refLayer:cel(tgtFrame)
	if cel then
		spr:deleteCel(cel)
	end
	-- Draw resized reference
	scaledImg:drawSprite(refImg, Point(offsetX, offsetY))
	spr:newCel(refLayer, tgtFrame, scaledImg, Point(0, 0))
end)

spr:saveAs(spr.filename)
print("Reference imported successfully")`,
				input.ImagePath,
				aseprite.EscapeString(layerName),
				aseprite.EscapeString(layerName),
				opacity,
				frame, frame)

			if _, err := client.ExecuteLua(ctx, script, input.SpritePath); err != nil {
				opLogger.Error("Failed to import reference layer", "error", err)
				return ToolError[ImportReferenceLayerOutput]("failed to import reference layer", err)
			}

			out := &ImportReferenceLayerOutput{
				Success:   true,
				LayerName: layerName,
				Message:   fmt.Sprintf("Imported reference image as guide layer %q with opacity %d", layerName, opacity),
			}
			return ToolSuccess(out)
		},
	)
}
