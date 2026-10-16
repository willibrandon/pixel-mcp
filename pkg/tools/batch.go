package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/willibrandon/mtlog/core"
	"github.com/willibrandon/pixel-mcp/pkg/aseprite"
	"github.com/willibrandon/pixel-mcp/pkg/config"
)

// BatchOperationInput defines the parameters for a single drawing operation within a batch.
//
// Uses a flat object structure with a type discriminator. Each field's jsonschema
// description indicates which operation types use it.
type BatchOperationInput struct {
	Type        string       `json:"type" jsonschema:"Operation type: pixels, line, rectangle, circle, contour, or fill"`
	LayerName   string       `json:"layer_name,omitempty" jsonschema:"Optional layer name override for this operation (all types; default: batch layer_name)"`
	FrameNumber int          `json:"frame_number,omitempty" jsonschema:"Optional 1-based frame number override for this operation (all types; default: batch frame_number)"`
	Color       string       `json:"color,omitempty" jsonschema:"Hex color string in format #RRGGBB or #RRGGBBAA (required for line, rectangle, circle, contour, fill)"`
	Pixels      []PixelInput `json:"pixels,omitempty" jsonschema:"Array of pixels with coordinates and hex colors (used by: pixels)"`
	X1          int          `json:"x1,omitempty" jsonschema:"Starting X coordinate (used by: line)"`
	Y1          int          `json:"y1,omitempty" jsonschema:"Starting Y coordinate (used by: line)"`
	X2          int          `json:"x2,omitempty" jsonschema:"Ending X coordinate (used by: line)"`
	Y2          int          `json:"y2,omitempty" jsonschema:"Ending Y coordinate (used by: line)"`
	X           int          `json:"x,omitempty" jsonschema:"Top-left X coordinate or fill start X (used by: rectangle, fill)"`
	Y           int          `json:"y,omitempty" jsonschema:"Top-left Y coordinate or fill start Y (used by: rectangle, fill)"`
	Width       int          `json:"width,omitempty" jsonschema:"Rectangle width in pixels (used by: rectangle)"`
	Height      int          `json:"height,omitempty" jsonschema:"Rectangle height in pixels (used by: rectangle)"`
	CenterX     int          `json:"center_x,omitempty" jsonschema:"Circle center X coordinate (used by: circle)"`
	CenterY     int          `json:"center_y,omitempty" jsonschema:"Circle center Y coordinate (used by: circle)"`
	Radius      int          `json:"radius,omitempty" jsonschema:"Circle radius in pixels (used by: circle)"`
	Filled      bool         `json:"filled,omitempty" jsonschema:"Whether to fill interior (true) or draw outline only (false; used by: rectangle, circle)"`
	Points      []PointInput `json:"points,omitempty" jsonschema:"Array of vertex points (minimum 2 points; used by: contour)"`
	Closed      bool         `json:"closed,omitempty" jsonschema:"Whether to connect last point back to first (used by: contour)"`
	Thickness   *int         `json:"thickness,omitempty" jsonschema:"Line/stroke thickness in pixels (1-100, default: 1; used by: line, contour)"`
	Tolerance   int          `json:"tolerance,omitempty" jsonschema:"Color matching tolerance (0-255, default: 0; used by: fill)"`
}

// BatchDrawInput defines the input parameters for the batch_draw tool.
type BatchDrawInput struct {
	SpritePath    string                `json:"sprite_path" jsonschema:"Path to the Aseprite sprite file"`
	LayerName     string                `json:"layer_name" jsonschema:"Default layer name for operations"`
	FrameNumber   int                   `json:"frame_number" jsonschema:"Default frame number for operations (1-based)"`
	Operations    []BatchOperationInput `json:"operations" jsonschema:"Array of drawing operations to apply in order (1-1000 items)"`
	UsePalette    bool                  `json:"use_palette,omitempty" jsonschema:"Snap all colors to nearest palette color (default: false)"`
	ReturnPreview bool                  `json:"return_preview,omitempty" jsonschema:"If true, returns a 4x visual PNG preview of frame_number directly in the response"`
}

// BatchDrawOutput defines the output for the batch_draw tool.
type BatchDrawOutput struct {
	OperationsApplied int `json:"operations_applied" jsonschema:"Number of drawing operations successfully applied"`
	PixelsDrawn       int `json:"pixels_drawn" jsonschema:"Total number of pixels drawn across all pixels operations"`
}

// validateBatchInput validates input parameters for batch_draw before running Aseprite.
func validateBatchInput(input BatchDrawInput) error {
	if input.SpritePath == "" {
		return fmt.Errorf("sprite_path cannot be empty")
	}
	if input.LayerName == "" {
		return fmt.Errorf("layer_name cannot be empty")
	}
	if input.FrameNumber < 1 {
		return fmt.Errorf("frame_number must be at least 1, got %d", input.FrameNumber)
	}
	if len(input.Operations) < 1 || len(input.Operations) > 1000 {
		return fmt.Errorf("operations must contain between 1 and 1000 items, got %d", len(input.Operations))
	}

	totalPixels := 0
	for i, op := range input.Operations {
		switch op.Type {
		case "pixels", "line", "rectangle", "circle", "contour", "fill":
			// Valid type
		default:
			return fmt.Errorf("operations[%d]: unknown operation type %q, valid types are: pixels, line, rectangle, circle, contour, fill", i, op.Type)
		}

		if op.FrameNumber < 0 {
			return fmt.Errorf("operations[%d] (%s): frame_number must be at least 1, got %d", i, op.Type, op.FrameNumber)
		}

		if op.Type != "pixels" {
			if op.Color == "" {
				return fmt.Errorf("operations[%d] (%s): color is required", i, op.Type)
			}
			var c aseprite.Color
			if err := c.FromHex(op.Color); err != nil {
				return fmt.Errorf("operations[%d] (%s): invalid color format: %w", i, op.Type, err)
			}
		}

		switch op.Type {
		case "pixels":
			if len(op.Pixels) == 0 {
				return fmt.Errorf("operations[%d] (pixels): pixels array cannot be empty", i)
			}
			totalPixels += len(op.Pixels)
			if totalPixels > 65536 {
				return fmt.Errorf("total pixels across all operations cannot exceed 65536, got %d", totalPixels)
			}
			for pIdx, p := range op.Pixels {
				var pc aseprite.Color
				if err := pc.FromHex(p.Color); err != nil {
					return fmt.Errorf("operations[%d] (pixels): invalid color format for pixel %d: %w", i, pIdx, err)
				}
			}

		case "line":
			if op.Thickness != nil {
				if *op.Thickness < 1 || *op.Thickness > 100 {
					return fmt.Errorf("operations[%d] (line): thickness must be between 1 and 100, got %d", i, *op.Thickness)
				}
			}

		case "rectangle":
			if op.Width < 1 || op.Height < 1 {
				return fmt.Errorf("operations[%d] (rectangle): width and height must be at least 1, got width=%d height=%d", i, op.Width, op.Height)
			}

		case "circle":
			if op.Radius < 1 {
				return fmt.Errorf("operations[%d] (circle): radius must be at least 1, got %d", i, op.Radius)
			}

		case "contour":
			if len(op.Points) < 2 {
				return fmt.Errorf("operations[%d] (contour): at least 2 points are required, got %d", i, len(op.Points))
			}
			if op.Thickness != nil {
				if *op.Thickness < 1 || *op.Thickness > 100 {
					return fmt.Errorf("operations[%d] (contour): thickness must be between 1 and 100, got %d", i, *op.Thickness)
				}
			}

		case "fill":
			if op.Tolerance < 0 || op.Tolerance > 255 {
				return fmt.Errorf("operations[%d] (fill): tolerance must be between 0 and 255, got %d", i, op.Tolerance)
			}
		}
	}

	return nil
}

// buildBatchOps converts validated BatchDrawInput into domain aseprite.BatchOp structs.
func buildBatchOps(input BatchDrawInput) ([]aseprite.BatchOp, int) {
	ops := make([]aseprite.BatchOp, len(input.Operations))
	totalPixels := 0

	for i, opInput := range input.Operations {
		layerName := opInput.LayerName
		if layerName == "" {
			layerName = input.LayerName
		}

		frameNumber := opInput.FrameNumber
		if frameNumber <= 0 {
			frameNumber = input.FrameNumber
		}

		op := aseprite.BatchOp{
			Type:        aseprite.BatchOpType(opInput.Type),
			LayerName:   layerName,
			FrameNumber: frameNumber,
		}

		if opInput.Type != "pixels" {
			var color aseprite.Color
			_ = color.FromHex(opInput.Color)
			op.Color = color
		}

		switch opInput.Type {
		case "pixels":
			pixels := make([]aseprite.Pixel, len(opInput.Pixels))
			for j, p := range opInput.Pixels {
				var c aseprite.Color
				_ = c.FromHex(p.Color)
				pixels[j] = aseprite.Pixel{
					Point: aseprite.Point{X: p.X, Y: p.Y},
					Color: c,
				}
			}
			op.Pixels = pixels
			totalPixels += len(pixels)

		case "line":
			op.X1 = opInput.X1
			op.Y1 = opInput.Y1
			op.X2 = opInput.X2
			op.Y2 = opInput.Y2
			thickness := 1
			if opInput.Thickness != nil {
				thickness = *opInput.Thickness
			}
			op.Thickness = thickness

		case "rectangle":
			op.X = opInput.X
			op.Y = opInput.Y
			op.Width = opInput.Width
			op.Height = opInput.Height
			op.Filled = opInput.Filled

		case "circle":
			op.CenterX = opInput.CenterX
			op.CenterY = opInput.CenterY
			op.Radius = opInput.Radius
			op.Filled = opInput.Filled

		case "contour":
			pts := make([]aseprite.Point, len(opInput.Points))
			for j, pt := range opInput.Points {
				pts[j] = aseprite.Point{X: pt.X, Y: pt.Y}
			}
			op.Points = pts
			op.Closed = opInput.Closed
			thickness := 1
			if opInput.Thickness != nil {
				thickness = *opInput.Thickness
			}
			op.Thickness = thickness

		case "fill":
			op.X = opInput.X
			op.Y = opInput.Y
			op.Tolerance = opInput.Tolerance
		}

		ops[i] = op
	}

	return ops, totalPixels
}

// RegisterBatchTools registers the batch_draw tool with the MCP server.
func RegisterBatchTools(server *mcp.Server, client *aseprite.Client, gen *aseprite.LuaGenerator, cfg *config.Config, logger core.Logger) {
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "batch_draw",
			Description: "Apply many drawing operations (pixels, line, rectangle, circle, contour, fill) to a sprite in one call: one Aseprite run, one atomic transaction, one save. Prefer this over repeated single draw_* calls when drawing more than one shape. If any operation fails, nothing is applied.",
		},
		maybeWrapWithTiming("batch_draw", logger, cfg.EnableTiming, func(ctx context.Context, _ *mcp.CallToolRequest, input BatchDrawInput) (*mcp.CallToolResult, *BatchDrawOutput, error) {
			opLogger := logger.WithContext(ctx)
			opLogger.Debug("batch_draw tool called", "sprite_path", input.SpritePath, "layer_name", input.LayerName, "frame_number", input.FrameNumber, "operations_count", len(input.Operations))

			// Validate inputs
			if err := validateBatchInput(input); err != nil {
				return ToolError[BatchDrawOutput](err.Error(), nil)
			}

			// Convert inputs to domain BatchOp structs
			ops, totalPixels := buildBatchOps(input)

			// Generate Lua script
			script := gen.BatchDraw(ops, input.UsePalette)

			// Execute Lua script with the sprite
			output, err := client.ExecuteLua(ctx, script, input.SpritePath)
			if err != nil {
				opLogger.Error("Failed to execute batch draw", "error", err)
				return ToolError[BatchDrawOutput]("failed to execute batch draw", err)
			}

			// Check for success message
			if !strings.Contains(output, "Batch completed:") {
				opLogger.Warning("Unexpected output from batch_draw", "output", output)
			}

			opLogger.Information("Batch drawing completed successfully", "sprite", input.SpritePath, "operations_applied", len(ops), "pixels_drawn", totalPixels)

			out := &BatchDrawOutput{
				OperationsApplied: len(ops),
				PixelsDrawn:       totalPixels,
			}

			if input.ReturnPreview {
				imgBytes, err := client.ExportPreview(ctx, input.SpritePath, 4, input.FrameNumber, "")
				if err == nil && len(imgBytes) > 0 {
					return ToolSuccessWithImage(out, imgBytes, "image/png", fmt.Sprintf("Successfully applied %d operations (with 4x preview)", len(ops)))
				}
			}

			return ToolSuccess(out)
		}),
	)
}
