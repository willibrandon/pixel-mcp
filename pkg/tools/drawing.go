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

// PixelInput represents a single pixel to be drawn.
//
// Used by draw_pixels to specify individual pixel positions and colors.
type PixelInput struct {
	X     int    `json:"x" jsonschema:"X coordinate of the pixel"`                           // X coordinate in sprite pixel space
	Y     int    `json:"y" jsonschema:"Y coordinate of the pixel"`                           // Y coordinate in sprite pixel space
	Color string `json:"color" jsonschema:"Hex color string in format #RRGGBB or #RRGGBBAA"` // Color in hex format (#RRGGBB or #RRGGBBAA)
}

// DrawPixelsInput defines the input parameters for the draw_pixels tool.
//
// Draws individual pixels at specified coordinates. Supports batch drawing
// for efficient bulk pixel operations. When UsePalette is true, colors are
// snapped to the nearest palette color using LAB color space distance.
type DrawPixelsInput struct {
	SpritePath    string       `json:"sprite_path" jsonschema:"Path to the Aseprite sprite file"`
	LayerName     string       `json:"layer_name" jsonschema:"Name of the layer to draw on"`
	FrameNumber   int          `json:"frame_number" jsonschema:"Frame number to draw on (1-based)"`
	Pixels        []PixelInput `json:"pixels" jsonschema:"Array of pixels to draw"`
	MirrorX       *int         `json:"mirror_x,omitempty" jsonschema:"Optional X coordinate for bilateral symmetry. When set, each pixel at (x,y) is also drawn at (2*mirror_x - x, y)."`
	ReturnPreview bool         `json:"return_preview,omitempty" jsonschema:"If true, returns a 4x visual PNG preview directly in the response"`
	UsePalette    bool         `json:"use_palette,omitempty" jsonschema:"Snap colors to nearest palette color (default: false)"`
}

// DrawPixelsOutput defines the output for the draw_pixels tool.
//
// Reports the number of pixels successfully drawn.
type DrawPixelsOutput struct {
	PixelsDrawn int `json:"pixels_drawn" jsonschema:"Number of pixels successfully drawn"` // Count of pixels drawn
}

// DrawLineInput defines the input parameters for the draw_line tool.
type DrawLineInput struct {
	SpritePath  string `json:"sprite_path" jsonschema:"Path to the Aseprite sprite file"`
	LayerName   string `json:"layer_name" jsonschema:"Name of the layer to draw on"`
	FrameNumber int    `json:"frame_number" jsonschema:"Frame number to draw on (1-based)"`
	X1          int    `json:"x1" jsonschema:"X coordinate of line start point"`
	Y1          int    `json:"y1" jsonschema:"Y coordinate of line start point"`
	X2          int    `json:"x2" jsonschema:"X coordinate of line end point"`
	Y2          int    `json:"y2" jsonschema:"Y coordinate of line end point"`
	Color       string `json:"color" jsonschema:"Hex color string in format #RRGGBB or #RRGGBBAA"`
	Thickness   int    `json:"thickness" jsonschema:"Line thickness in pixels (1-100)"`
	UsePalette  bool   `json:"use_palette,omitempty" jsonschema:"Snap colors to nearest palette color (default: false)"`
}

// DrawLineOutput defines the output for the draw_line tool.
type DrawLineOutput struct {
	Success bool `json:"success" jsonschema:"Whether the line was drawn successfully"`
}

// PointInput represents a point in the contour.
type PointInput struct {
	X int `json:"x" jsonschema:"X coordinate of the point"`
	Y int `json:"y" jsonschema:"Y coordinate of the point"`
}

// DrawContourInput defines the input parameters for the draw_contour tool.
type DrawContourInput struct {
	SpritePath  string       `json:"sprite_path" jsonschema:"Path to the Aseprite sprite file"`
	LayerName   string       `json:"layer_name" jsonschema:"Name of the layer to draw on"`
	FrameNumber int          `json:"frame_number" jsonschema:"Frame number to draw on (1-based)"`
	Points      []PointInput `json:"points" jsonschema:"Array of points to connect (minimum 2 points)"`
	Color       string       `json:"color" jsonschema:"Hex color string in format #RRGGBB or #RRGGBBAA"`
	Thickness   int          `json:"thickness" jsonschema:"Line thickness in pixels (1-100)"`
	Closed      bool         `json:"closed" jsonschema:"Connect last point to first to form a closed polygon (default: false)"`
	UsePalette  bool         `json:"use_palette,omitempty" jsonschema:"Snap colors to nearest palette color (default: false)"`
}

// DrawContourOutput defines the output for the draw_contour tool.
type DrawContourOutput struct {
	Success bool `json:"success" jsonschema:"Whether the contour was drawn successfully"`
}

// DrawRectangleInput defines the input parameters for the draw_rectangle tool.
type DrawRectangleInput struct {
	SpritePath  string `json:"sprite_path" jsonschema:"Path to the Aseprite sprite file"`
	LayerName   string `json:"layer_name" jsonschema:"Name of the layer to draw on"`
	FrameNumber int    `json:"frame_number" jsonschema:"Frame number to draw on (1-based)"`
	X           int    `json:"x" jsonschema:"X coordinate of rectangle top-left corner"`
	Y           int    `json:"y" jsonschema:"Y coordinate of rectangle top-left corner"`
	Width       int    `json:"width" jsonschema:"Width of rectangle in pixels"`
	Height      int    `json:"height" jsonschema:"Height of rectangle in pixels"`
	Color       string `json:"color" jsonschema:"Hex color string in format #RRGGBB or #RRGGBBAA"`
	Filled      bool   `json:"filled" jsonschema:"Fill interior (true) or draw outline only (false)"`
	UsePalette  bool   `json:"use_palette,omitempty" jsonschema:"Snap colors to nearest palette color (default: false)"`
}

// DrawRectangleOutput defines the output for the draw_rectangle tool.
type DrawRectangleOutput struct {
	Success bool `json:"success" jsonschema:"Whether the rectangle was drawn successfully"`
}

// DrawCircleInput defines the input parameters for the draw_circle tool.
type DrawCircleInput struct {
	SpritePath  string `json:"sprite_path" jsonschema:"Path to the Aseprite sprite file"`
	LayerName   string `json:"layer_name" jsonschema:"Name of the layer to draw on"`
	FrameNumber int    `json:"frame_number" jsonschema:"Frame number to draw on (1-based)"`
	CenterX     int    `json:"center_x" jsonschema:"X coordinate of circle center"`
	CenterY     int    `json:"center_y" jsonschema:"Y coordinate of circle center"`
	Radius      int    `json:"radius" jsonschema:"Radius of circle in pixels"`
	Color       string `json:"color" jsonschema:"Hex color string in format #RRGGBB or #RRGGBBAA"`
	Filled      bool   `json:"filled" jsonschema:"Fill interior (true) or draw outline only (false)"`
	UsePalette  bool   `json:"use_palette,omitempty" jsonschema:"Snap colors to nearest palette color (default: false)"`
}

// DrawCircleOutput defines the output for the draw_circle tool.
type DrawCircleOutput struct {
	Success bool `json:"success" jsonschema:"Whether the circle was drawn successfully"`
}

// FillAreaInput defines the input parameters for the fill_area tool.
type FillAreaInput struct {
	SpritePath  string `json:"sprite_path" jsonschema:"Path to the Aseprite sprite file"`
	LayerName   string `json:"layer_name" jsonschema:"Name of the layer to draw on"`
	FrameNumber int    `json:"frame_number" jsonschema:"Frame number to draw on (1-based)"`
	X           int    `json:"x" jsonschema:"X coordinate of starting point"`
	Y           int    `json:"y" jsonschema:"Y coordinate of starting point"`
	Color       string `json:"color" jsonschema:"Hex color string in format #RRGGBB or #RRGGBBAA"`
	Tolerance   int    `json:"tolerance" jsonschema:"Color matching tolerance (0-255, default 0)"`
	UsePalette  bool   `json:"use_palette,omitempty" jsonschema:"Snap colors to nearest palette color (default: false)"`
}

// FillAreaOutput defines the output for the fill_area tool.
type FillAreaOutput struct {
	Success bool `json:"success" jsonschema:"Whether the fill operation was successful"`
}

// RegisterDrawingTools registers all drawing tools with the MCP server.
//
// Registers the following drawing primitives:
//   - draw_pixels: Draw individual pixels (bulk operations)
//   - draw_line: Draw straight lines with thickness control
//   - draw_contour: Draw multi-segment polylines (open or closed)
//   - draw_rectangle: Draw rectangles (filled or outline)
//   - draw_ellipse: Draw ellipses and circles (filled or outline)
//   - fill: Flood fill with color replacement
//
// All drawing tools support palette-aware color snapping via the UsePalette flag,
// which snaps arbitrary colors to the nearest palette color using LAB color space.
func RegisterDrawingTools(server *mcp.Server, client *aseprite.Client, gen *aseprite.LuaGenerator, cfg *config.Config, logger core.Logger) {
	// Register draw_pixels tool
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "draw_pixels",
			Description: "Draw individual pixels at specified coordinates with colors. Supports batch operations for efficiency.",
		},
		maybeWrapWithTiming("draw_pixels", logger, cfg.EnableTiming, func(ctx context.Context, req *mcp.CallToolRequest, input DrawPixelsInput) (*mcp.CallToolResult, *DrawPixelsOutput, error) {
			opLogger := logger.WithContext(ctx)
			opLogger.Debug("draw_pixels tool called", "sprite_path", input.SpritePath, "layer_name", input.LayerName, "frame_number", input.FrameNumber, "pixel_count", len(input.Pixels))

			// Validate inputs
			if input.LayerName == "" {
				return ToolError[DrawPixelsOutput]("layer_name cannot be empty", nil)
			}

			if input.FrameNumber < 1 {
				return ToolError[DrawPixelsOutput](fmt.Sprintf("frame_number must be at least 1, got %d", input.FrameNumber), nil)
			}

			if len(input.Pixels) == 0 {
				return ToolError[DrawPixelsOutput]("pixels array cannot be empty", nil)
			}

			// Convert pixel inputs to aseprite.Pixel types with optional bilateral symmetry
			var pixels []aseprite.Pixel
			for i, p := range input.Pixels {
				var color aseprite.Color
				if err := color.FromHex(p.Color); err != nil {
					return ToolError[DrawPixelsOutput](fmt.Sprintf("invalid color format for pixel %d", i), err)
				}

				pixels = append(pixels, aseprite.Pixel{
					Point: aseprite.Point{X: p.X, Y: p.Y},
					Color: color,
				})

				if input.MirrorX != nil {
					mirroredX := 2*(*input.MirrorX) - p.X
					if mirroredX != p.X {
						pixels = append(pixels, aseprite.Pixel{
							Point: aseprite.Point{X: mirroredX, Y: p.Y},
							Color: color,
						})
					}
				}
			}

			// Generate Lua script
			script := gen.DrawPixels(input.LayerName, input.FrameNumber, pixels, input.UsePalette)

			// Execute Lua script with the sprite
			output, err := client.ExecuteLua(ctx, script, input.SpritePath)
			if err != nil {
				opLogger.Error("Failed to draw pixels", "error", err)
				return ToolError[DrawPixelsOutput]("failed to draw pixels", err)
			}

			// Check for success message
			if !strings.Contains(output, "Pixels drawn successfully") {
				opLogger.Warning("Unexpected output from draw_pixels", "output", output)
			}

			opLogger.Information("Pixels drawn successfully", "sprite", input.SpritePath, "layer", input.LayerName, "frame", input.FrameNumber, "count", len(pixels))

			out := &DrawPixelsOutput{PixelsDrawn: len(pixels)}

			if input.ReturnPreview {
				imgBytes, err := client.ExportPreview(ctx, input.SpritePath, 4, input.FrameNumber, "")
				if err == nil && len(imgBytes) > 0 {
					return ToolSuccessWithImage(out, imgBytes, "image/png", fmt.Sprintf("Successfully drew %d pixels (with 4x preview)", len(pixels)))
				}
			}

			return ToolSuccess(out)
		}),
	)

	// Register draw_line tool
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "draw_line",
			Description: "Draw a line between two points with specified color and thickness.",
		},
		maybeWrapWithTiming("draw_line", logger, cfg.EnableTiming, func(ctx context.Context, req *mcp.CallToolRequest, input DrawLineInput) (*mcp.CallToolResult, *DrawLineOutput, error) {
			opLogger := logger.WithContext(ctx)
			opLogger.Debug("draw_line tool called", "sprite_path", input.SpritePath, "layer_name", input.LayerName, "frame_number", input.FrameNumber)

			// Validate inputs
			if input.LayerName == "" {
				return nil, nil, fmt.Errorf("layer_name cannot be empty")
			}

			if input.FrameNumber < 1 {
				return nil, nil, fmt.Errorf("frame_number must be at least 1, got %d", input.FrameNumber)
			}

			if input.Thickness < 1 || input.Thickness > 100 {
				return nil, nil, fmt.Errorf("thickness must be between 1 and 100, got %d", input.Thickness)
			}

			// Parse color
			var color aseprite.Color
			if err := color.FromHex(input.Color); err != nil {
				return nil, nil, fmt.Errorf("invalid color format: %w", err)
			}

			// Generate Lua script
			script := gen.DrawLine(input.LayerName, input.FrameNumber, input.X1, input.Y1, input.X2, input.Y2, color, input.Thickness, input.UsePalette)

			// Execute Lua script with the sprite
			output, err := client.ExecuteLua(ctx, script, input.SpritePath)
			if err != nil {
				opLogger.Error("Failed to draw line", "error", err)
				return nil, nil, fmt.Errorf("failed to draw line: %w", err)
			}

			// Check for success message
			if !strings.Contains(output, "Line drawn successfully") {
				opLogger.Warning("Unexpected output from draw_line", "output", output)
			}

			opLogger.Information("Line drawn successfully", "sprite", input.SpritePath, "layer", input.LayerName, "frame", input.FrameNumber)

			return nil, &DrawLineOutput{Success: true}, nil
		}),
	)

	// Register draw_contour tool
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "draw_contour",
			Description: "Draw a polyline or polygon by connecting multiple points. Supports both open paths and closed shapes.",
		},
		maybeWrapWithTiming("draw_contour", logger, cfg.EnableTiming, func(ctx context.Context, req *mcp.CallToolRequest, input DrawContourInput) (*mcp.CallToolResult, *DrawContourOutput, error) {
			opLogger := logger.WithContext(ctx)
			opLogger.Debug("draw_contour tool called", "sprite_path", input.SpritePath, "layer_name", input.LayerName, "frame_number", input.FrameNumber, "points", len(input.Points), "closed", input.Closed)

			// Validate inputs
			if input.LayerName == "" {
				return nil, nil, fmt.Errorf("layer_name cannot be empty")
			}

			if input.FrameNumber < 1 {
				return nil, nil, fmt.Errorf("frame_number must be at least 1, got %d", input.FrameNumber)
			}

			if len(input.Points) < 2 {
				return nil, nil, fmt.Errorf("at least 2 points are required, got %d", len(input.Points))
			}

			if input.Thickness < 1 || input.Thickness > 100 {
				return nil, nil, fmt.Errorf("thickness must be between 1 and 100, got %d", input.Thickness)
			}

			// Parse color
			var color aseprite.Color
			if err := color.FromHex(input.Color); err != nil {
				return nil, nil, fmt.Errorf("invalid color format: %w", err)
			}

			// Convert PointInput to aseprite.Point
			points := make([]aseprite.Point, len(input.Points))
			for i, p := range input.Points {
				points[i] = aseprite.Point{X: p.X, Y: p.Y}
			}

			// Generate Lua script
			script := gen.DrawContour(input.LayerName, input.FrameNumber, points, color, input.Thickness, input.Closed, input.UsePalette)

			// Execute Lua script with the sprite
			output, err := client.ExecuteLua(ctx, script, input.SpritePath)
			if err != nil {
				opLogger.Error("Failed to draw contour", "error", err)
				return nil, nil, fmt.Errorf("failed to draw contour: %w", err)
			}

			// Check for success message
			if !strings.Contains(output, "Contour drawn successfully") {
				opLogger.Warning("Unexpected output from draw_contour", "output", output)
			}

			opLogger.Information("Contour drawn successfully", "sprite", input.SpritePath, "layer", input.LayerName, "frame", input.FrameNumber, "points", len(points), "closed", input.Closed)

			return nil, &DrawContourOutput{Success: true}, nil
		}),
	)

	// Register draw_rectangle tool
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "draw_rectangle",
			Description: "Draw a rectangle with specified position, size, color, and fill option.",
		},
		maybeWrapWithTiming("draw_rectangle", logger, cfg.EnableTiming, func(ctx context.Context, req *mcp.CallToolRequest, input DrawRectangleInput) (*mcp.CallToolResult, *DrawRectangleOutput, error) {
			opLogger := logger.WithContext(ctx)
			opLogger.Debug("draw_rectangle tool called", "sprite_path", input.SpritePath, "layer_name", input.LayerName, "frame_number", input.FrameNumber, "filled", input.Filled)

			// Validate inputs
			if input.LayerName == "" {
				return nil, nil, fmt.Errorf("layer_name cannot be empty")
			}

			if input.FrameNumber < 1 {
				return nil, nil, fmt.Errorf("frame_number must be at least 1, got %d", input.FrameNumber)
			}

			if input.Width < 1 || input.Height < 1 {
				return nil, nil, fmt.Errorf("width and height must be at least 1, got width=%d height=%d", input.Width, input.Height)
			}

			// Parse color
			var color aseprite.Color
			if err := color.FromHex(input.Color); err != nil {
				return nil, nil, fmt.Errorf("invalid color format: %w", err)
			}

			// Generate Lua script
			script := gen.DrawRectangle(input.LayerName, input.FrameNumber, input.X, input.Y, input.Width, input.Height, color, input.Filled, input.UsePalette)

			// Execute Lua script with the sprite
			output, err := client.ExecuteLua(ctx, script, input.SpritePath)
			if err != nil {
				opLogger.Error("Failed to draw rectangle", "error", err)
				return nil, nil, fmt.Errorf("failed to draw rectangle: %w", err)
			}

			// Check for success message
			if !strings.Contains(output, "Rectangle drawn successfully") {
				opLogger.Warning("Unexpected output from draw_rectangle", "output", output)
			}

			opLogger.Information("Rectangle drawn successfully", "sprite", input.SpritePath, "layer", input.LayerName, "frame", input.FrameNumber, "filled", input.Filled)

			return nil, &DrawRectangleOutput{Success: true}, nil
		}),
	)

	// Register draw_circle tool
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "draw_circle",
			Description: "Draw a circle with specified center, radius, color, and fill option.",
		},
		maybeWrapWithTiming("draw_circle", logger, cfg.EnableTiming, func(ctx context.Context, req *mcp.CallToolRequest, input DrawCircleInput) (*mcp.CallToolResult, *DrawCircleOutput, error) {
			opLogger := logger.WithContext(ctx)
			opLogger.Debug("draw_circle tool called", "sprite_path", input.SpritePath, "layer_name", input.LayerName, "frame_number", input.FrameNumber, "radius", input.Radius, "filled", input.Filled)

			// Validate inputs
			if input.LayerName == "" {
				return nil, nil, fmt.Errorf("layer_name cannot be empty")
			}

			if input.FrameNumber < 1 {
				return nil, nil, fmt.Errorf("frame_number must be at least 1, got %d", input.FrameNumber)
			}

			if input.Radius < 1 {
				return nil, nil, fmt.Errorf("radius must be at least 1, got %d", input.Radius)
			}

			// Parse color
			var color aseprite.Color
			if err := color.FromHex(input.Color); err != nil {
				return nil, nil, fmt.Errorf("invalid color format: %w", err)
			}

			// Generate Lua script
			script := gen.DrawCircle(input.LayerName, input.FrameNumber, input.CenterX, input.CenterY, input.Radius, color, input.Filled, input.UsePalette)

			// Execute Lua script with the sprite
			output, err := client.ExecuteLua(ctx, script, input.SpritePath)
			if err != nil {
				opLogger.Error("Failed to draw circle", "error", err)
				return nil, nil, fmt.Errorf("failed to draw circle: %w", err)
			}

			// Check for success message
			if !strings.Contains(output, "Circle drawn successfully") {
				opLogger.Warning("Unexpected output from draw_circle", "output", output)
			}

			opLogger.Information("Circle drawn successfully", "sprite", input.SpritePath, "layer", input.LayerName, "frame", input.FrameNumber, "radius", input.Radius, "filled", input.Filled)

			return nil, &DrawCircleOutput{Success: true}, nil
		}),
	)

	// Register fill_area tool
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "fill_area",
			Description: "Flood fill from a starting point with specified color (paint bucket tool).",
		},
		maybeWrapWithTiming("fill_area", logger, cfg.EnableTiming, func(ctx context.Context, req *mcp.CallToolRequest, input FillAreaInput) (*mcp.CallToolResult, *FillAreaOutput, error) {
			opLogger := logger.WithContext(ctx)
			opLogger.Debug("fill_area tool called", "sprite_path", input.SpritePath, "layer_name", input.LayerName, "frame_number", input.FrameNumber, "x", input.X, "y", input.Y, "tolerance", input.Tolerance)

			// Validate inputs
			if input.LayerName == "" {
				return nil, nil, fmt.Errorf("layer_name cannot be empty")
			}

			if input.FrameNumber < 1 {
				return nil, nil, fmt.Errorf("frame_number must be at least 1, got %d", input.FrameNumber)
			}

			if input.Tolerance < 0 || input.Tolerance > 255 {
				return nil, nil, fmt.Errorf("tolerance must be between 0 and 255, got %d", input.Tolerance)
			}

			// Parse color
			var color aseprite.Color
			if err := color.FromHex(input.Color); err != nil {
				return nil, nil, fmt.Errorf("invalid color format: %w", err)
			}

			// Generate Lua script
			script := gen.FillArea(input.LayerName, input.FrameNumber, input.X, input.Y, color, input.Tolerance, input.UsePalette)

			// Execute Lua script with the sprite
			output, err := client.ExecuteLua(ctx, script, input.SpritePath)
			if err != nil {
				opLogger.Error("Failed to fill area", "error", err)
				return nil, nil, fmt.Errorf("failed to fill area: %w", err)
			}

			// Check for success message
			if !strings.Contains(output, "Area filled successfully") {
				opLogger.Warning("Unexpected output from fill_area", "output", output)
			}

			opLogger.Information("Area filled successfully", "sprite", input.SpritePath, "layer", input.LayerName, "frame", input.FrameNumber, "x", input.X, "y", input.Y, "tolerance", input.Tolerance)

			return nil, &FillAreaOutput{Success: true}, nil
		}),
	)
}
