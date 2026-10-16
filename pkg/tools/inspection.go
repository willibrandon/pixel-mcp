package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/willibrandon/mtlog/core"
	"github.com/willibrandon/pixel-mcp/pkg/aseprite"
	"github.com/willibrandon/pixel-mcp/pkg/config"
)

// GetPixelsInput defines the input parameters for the get_pixels tool.
//
// Reads pixel data from a rectangular region. Supports pagination for large regions
// to avoid memory/bandwidth issues. Default page size is 1000 pixels, max 10000.
type GetPixelsInput struct {
	SpritePath  string `json:"sprite_path" jsonschema:"Path to the Aseprite sprite file"`                                        // Path to the sprite file to read
	LayerName   string `json:"layer_name" jsonschema:"Name of the layer to read from"`                                           // Layer to read from
	FrameNumber int    `json:"frame_number" jsonschema:"Frame number to read from (1-based)"`                                    // 1-based frame index
	X           int    `json:"x" jsonschema:"X coordinate of top-left corner of region"`                                         // Top-left X coordinate
	Y           int    `json:"y" jsonschema:"Y coordinate of top-left corner of region"`                                         // Top-left Y coordinate
	Width       int    `json:"width" jsonschema:"Width of region to read"`                                                       // Region width in pixels
	Height      int    `json:"height" jsonschema:"Height of region to read"`                                                     // Region height in pixels
	Cursor      string `json:"cursor,omitempty" jsonschema:"Pagination cursor for fetching next page (optional)"`                // Cursor for next page (empty for first page)
	PageSize    int    `json:"page_size,omitempty" jsonschema:"Number of pixels to return per page (default: 1000, max: 10000)"` // Page size (default 1000, max 10000)
	Format      string `json:"format,omitempty" jsonschema:"Response format: \"pixels\" (default) returns a paginated array of {x,y,color}; \"grid\" returns the whole region as rows of single-character color keys plus a legend - much more compact, best for inspecting a sprite visually. Grid mode ignores pagination and is limited to 16384 pixels (e.g. 128x128)."`
}

// PixelData represents a single pixel with coordinates and color.
//
// Used by get_pixels to return pixel information from sprite inspection.
type PixelData struct {
	X     int    `json:"x"`     // X coordinate in sprite space
	Y     int    `json:"y"`     // Y coordinate in sprite space
	Color string `json:"color"` // Color in hex format (#RRGGBBAA)
}

// GetPixelsOutput defines the output for the get_pixels tool.
//
// Returns pixel data with pagination support or as a compact character grid.
type GetPixelsOutput struct {
	Format      string            `json:"format" jsonschema:"Response format used (\"pixels\" or \"grid\")"`
	Pixels      []PixelData       `json:"pixels,omitempty" jsonschema:"Array of pixels with coordinates and colors (omitted in grid format or when the page has no pixels)"`
	NextCursor  string            `json:"next_cursor,omitempty" jsonschema:"Cursor for fetching next page (empty if no more pages or in grid format)"`
	TotalPixels int               `json:"total_pixels" jsonschema:"Total number of pixels in the region"`
	Grid        []string          `json:"grid,omitempty" jsonschema:"Character grid representing the region as rows of single-character color keys"`
	Legend      map[string]string `json:"legend,omitempty" jsonschema:"Mapping from character keys in the grid to hex colors (#RRGGBBAA)"`
}

// gridAlphabet holds the single-character keys assigned to colors in grid format,
// in assignment order. It excludes '.', which always means transparent, and
// characters that need escaping in JSON or are easy to misread.
const gridAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz!#$%&()*+,-/:;<=>?@[]^_{|}~"

// isColorTransparent reports whether a #RRGGBBAA color has zero alpha.
func isColorTransparent(color string) bool {
	c := strings.ToUpper(strings.TrimSpace(color))
	if len(c) >= 9 {
		return c[7:9] == "00"
	}
	return false
}

// buildPixelGridFormat renders pixels from the region at (x, y) of the given size
// as one string per row, with one gridAlphabet key per distinct color in order
// of first appearance. Transparent pixels and positions missing from pixels
// (outside the cel) become '.'.
func buildPixelGridFormat(x, y, width, height int, pixels []PixelData) ([]string, map[string]string, error) {
	if width <= 0 || height <= 0 {
		return nil, nil, fmt.Errorf("width and height must be positive, got width=%d height=%d", width, height)
	}

	pixelMap := make(map[[2]int]string, len(pixels))
	for _, p := range pixels {
		if _, exists := pixelMap[[2]int{p.X, p.Y}]; !exists {
			pixelMap[[2]int{p.X, p.Y}] = p.Color
		}
	}

	legend := make(map[string]string)
	colorToKey := make(map[string]byte)
	keyIndex := 0

	grid := make([]string, height)

	for r := 0; r < height; r++ {
		row := make([]byte, width)
		py := y + r
		for c := 0; c < width; c++ {
			px := x + c
			color, exists := pixelMap[[2]int{px, py}]
			if !exists || isColorTransparent(color) {
				row[c] = '.'
				continue
			}

			normColor := strings.ToUpper(strings.TrimSpace(color))
			if !strings.HasPrefix(normColor, "#") {
				normColor = "#" + normColor
			}

			key, seen := colorToKey[normColor]
			if !seen {
				if keyIndex >= len(gridAlphabet) {
					return nil, nil, fmt.Errorf("region contains more distinct non-transparent colors than available keys (%d): use format \"pixels\" or select a smaller region", len(gridAlphabet))
				}
				key = gridAlphabet[keyIndex]
				keyIndex++
				colorToKey[normColor] = key
				legend[string(key)] = normColor
			}
			row[c] = key
		}
		grid[r] = string(row)
	}

	return grid, legend, nil
}

// RegisterInspectionTools registers all inspection tools with the MCP server.
//
// Registers the following tools:
//   - get_pixels: Read pixel data from rectangular regions with pagination support
//
// Inspection tools are read-only and do not modify sprite files.
func RegisterInspectionTools(server *mcp.Server, client *aseprite.Client, gen *aseprite.LuaGenerator, cfg *config.Config, logger core.Logger) {
	// Register get_pixels tool
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "get_pixels",
			Description: "Read pixel data from a rectangular region of a sprite. Supports \"pixels\" format (paginated array of coordinates and colors) and \"grid\" format (compact character grid with color legend for visual inspection up to 16384 pixels).",
		},
		maybeWrapWithTiming("get_pixels", logger, cfg.EnableTiming, func(ctx context.Context, req *mcp.CallToolRequest, input GetPixelsInput) (*mcp.CallToolResult, *GetPixelsOutput, error) {
			opLogger := logger.WithContext(ctx)
			opLogger.Debug("get_pixels tool called", "sprite_path", input.SpritePath, "layer", input.LayerName, "frame", input.FrameNumber, "x", input.X, "y", input.Y, "width", input.Width, "height", input.Height, "cursor", input.Cursor, "page_size", input.PageSize, "format", input.Format)

			format := input.Format
			if format == "" {
				format = "pixels"
			}
			if format != "pixels" && format != "grid" {
				return ToolError[GetPixelsOutput](fmt.Sprintf("invalid format: %q (must be \"pixels\" or \"grid\")", input.Format), nil)
			}

			// Validate inputs
			if input.Width <= 0 || input.Height <= 0 {
				return nil, nil, fmt.Errorf("width and height must be positive, got width=%d height=%d", input.Width, input.Height)
			}

			if input.FrameNumber < 1 {
				return nil, nil, fmt.Errorf("frame_number must be >= 1, got %d", input.FrameNumber)
			}

			// Calculate total pixel count
			totalPixelCount := input.Width * input.Height

			if format == "grid" {
				if totalPixelCount > 16384 {
					return ToolError[GetPixelsOutput](fmt.Sprintf("region size %d pixels exceeds grid format limit of 16384 pixels: split the region into tiles of at most 128x128 or use format \"pixels\" with pagination", totalPixelCount), nil)
				}
				if input.Cursor != "" {
					return ToolError[GetPixelsOutput]("cursor is not supported with format \"grid\" (grid format returns the full region in a single response)", nil)
				}

				// Fetch whole region in one Lua call (offset 0, limit 0 = all)
				script := gen.GetPixelsWithPagination(input.LayerName, input.FrameNumber, input.X, input.Y, input.Width, input.Height, 0, 0)

				output, err := client.ExecuteLua(ctx, script, input.SpritePath)
				if err != nil {
					opLogger.Error("Failed to get pixels", "error", err)
					return nil, nil, fmt.Errorf("failed to get pixels: %w", err)
				}

				var pixels []PixelData
				if err := json.Unmarshal([]byte(output), &pixels); err != nil {
					opLogger.Error("Failed to parse pixel data", "error", err, "output", output)
					return nil, nil, fmt.Errorf("failed to parse pixel data: %w", err)
				}

				grid, legend, err := buildPixelGridFormat(input.X, input.Y, input.Width, input.Height, pixels)
				if err != nil {
					return ToolError[GetPixelsOutput](err.Error(), nil)
				}

				opLogger.Information("Read pixels grid successfully", "sprite", input.SpritePath, "layer", input.LayerName, "frame", input.FrameNumber, "total", totalPixelCount, "colors", len(legend))

				return nil, &GetPixelsOutput{
					Format:      "grid",
					TotalPixels: totalPixelCount,
					Grid:        grid,
					Legend:      legend,
				}, nil
			}

			// format == "pixels"
			// Set default page size and validate
			pageSize := input.PageSize
			if pageSize <= 0 {
				pageSize = 1000 // default
			}
			if pageSize > 10000 {
				pageSize = 10000 // max
			}

			// Parse cursor to get offset
			offset := 0
			if input.Cursor != "" {
				var err error
				offset, err = strconv.Atoi(input.Cursor)
				if err != nil {
					return nil, nil, fmt.Errorf("invalid cursor: %w", err)
				}
			}

			// Generate Lua script with pagination
			script := gen.GetPixelsWithPagination(input.LayerName, input.FrameNumber, input.X, input.Y, input.Width, input.Height, offset, pageSize)

			// Execute Lua script with the sprite
			output, err := client.ExecuteLua(ctx, script, input.SpritePath)
			if err != nil {
				opLogger.Error("Failed to get pixels", "error", err)
				return nil, nil, fmt.Errorf("failed to get pixels: %w", err)
			}

			// Parse JSON output
			var pixels []PixelData
			if err := json.Unmarshal([]byte(output), &pixels); err != nil {
				opLogger.Error("Failed to parse pixel data", "error", err, "output", output)
				return nil, nil, fmt.Errorf("failed to parse pixel data: %w", err)
			}

			// Generate next cursor if there are more pixels
			nextCursor := ""
			nextOffset := offset + pageSize
			if nextOffset < totalPixelCount {
				nextCursor = strconv.Itoa(nextOffset)
			}

			opLogger.Information("Read pixels successfully", "sprite", input.SpritePath, "layer", input.LayerName, "frame", input.FrameNumber, "total", totalPixelCount, "returned", len(pixels), "offset", offset)

			return nil, &GetPixelsOutput{
				Format:      "pixels",
				Pixels:      pixels,
				NextCursor:  nextCursor,
				TotalPixels: totalPixelCount,
			}, nil
		}),
	)
}
