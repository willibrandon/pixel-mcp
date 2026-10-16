//go:build integration

package tools

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willibrandon/mtlog"
	"github.com/willibrandon/mtlog/core"
	"github.com/willibrandon/pixel-mcp/internal/testutil"
	"github.com/willibrandon/pixel-mcp/pkg/aseprite"
)

func createGridIntegrationSession(t *testing.T) (*mcp.ClientSession, func()) {
	t.Helper()

	cfg := testutil.LoadTestConfig(t)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, cfg.Timeout)
	gen := aseprite.NewLuaGenerator()
	logger := mtlog.New(mtlog.WithMinimumLevel(core.ErrorLevel))

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "pixel-mcp-grid-test",
		Version: "1.0.0",
	}, nil)

	RegisterInspectionTools(server, client, gen, cfg, logger)
	RegisterCanvasTools(server, client, gen, cfg, logger)
	RegisterDrawingTools(server, client, gen, cfg, logger)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()

	_, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)

	mcpClient := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	session, err := mcpClient.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)

	cleanup := func() {
		session.Close()
	}
	return session, cleanup
}

func TestGetPixels_GridAndPagination_MCPIntegration(t *testing.T) {
	session, cleanup := createGridIntegrationSession(t)
	defer cleanup()

	ctx := context.Background()

	// 1. Create a 16x16 canvas
	createResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "create_canvas",
		Arguments: map[string]any{
			"width":      16,
			"height":     16,
			"color_mode": "rgb",
		},
	})
	require.NoError(t, err)
	require.False(t, createResult.IsError)

	var createOutput struct {
		FilePath string `json:"file_path"`
	}
	require.NoError(t, json.Unmarshal([]byte(createResult.Content[0].(*mcp.TextContent).Text), &createOutput))
	defer os.Remove(createOutput.FilePath)

	// 2. Add a layer (cels on new layers start untrimmed/uncreated until drawn on)
	addLayerResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "add_layer",
		Arguments: map[string]any{
			"sprite_path": createOutput.FilePath,
			"layer_name":  "ContentLayer",
		},
	})
	require.NoError(t, err)
	require.False(t, addLayerResult.IsError)

	// 3. Draw a filled rectangle (4x4 at x=4, y=4) so the cel is trimmed
	drawRectResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "draw_rectangle",
		Arguments: map[string]any{
			"sprite_path":  createOutput.FilePath,
			"layer_name":   "ContentLayer",
			"frame_number": 1,
			"x":            4,
			"y":            4,
			"width":        4,
			"height":       4,
			"color":        "#FF0000FF", // Red
			"filled":       true,
		},
	})
	require.NoError(t, err)
	require.False(t, drawRectResult.IsError)

	// Draw a couple of pixels with distinct colors inside the rectangle using 1x1 rectangles
	// (so cel remains trimmed by app.useTool to content bounds)
	for _, p := range []struct {
		x, y  int
		color string
	}{
		{4, 4, "#00FF00FF"}, // Green
		{5, 4, "#0000FFFF"}, // Blue
	} {
		drawPixelResult, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "draw_rectangle",
			Arguments: map[string]any{
				"sprite_path":  createOutput.FilePath,
				"layer_name":   "ContentLayer",
				"frame_number": 1,
				"x":            p.x,
				"y":            p.y,
				"width":        1,
				"height":       1,
				"color":        p.color,
				"filled":       true,
			},
		})
		require.NoError(t, err)
		require.False(t, drawPixelResult.IsError)
	}

	// 4. Call get_pixels with format "grid" over the whole canvas (16x16)
	gridResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_pixels",
		Arguments: map[string]any{
			"sprite_path":  createOutput.FilePath,
			"layer_name":   "ContentLayer",
			"frame_number": 1,
			"x":            0,
			"y":            0,
			"width":        16,
			"height":       16,
			"format":       "grid",
		},
	})
	require.NoError(t, err)
	require.False(t, gridResult.IsError)

	gridText := gridResult.Content[0].(*mcp.TextContent).Text
	var gridOutput GetPixelsOutput
	require.NoError(t, json.Unmarshal([]byte(gridText), &gridOutput))

	// Verify grid metadata
	assert.Equal(t, "grid", gridOutput.Format)
	assert.Equal(t, 256, gridOutput.TotalPixels)
	assert.Empty(t, gridOutput.NextCursor)
	assert.Empty(t, gridOutput.Pixels)

	// Check grid dimensions
	assert.Equal(t, 16, len(gridOutput.Grid), "Grid should have 16 rows")
	for r, row := range gridOutput.Grid {
		assert.Equal(t, 16, len(row), "Row %d should have 16 characters", r)
	}

	// Legend checks: '.' must never be in legend
	_, dotInLegend := gridOutput.Legend["."]
	assert.False(t, dotInLegend, "Legend must not contain '.'")
	assert.Equal(t, "#00FF00FF", gridOutput.Legend["0"], "Key '0' should be first color (Green)")
	assert.Equal(t, "#0000FFFF", gridOutput.Legend["1"], "Key '1' should be second color (Blue)")
	assert.Equal(t, "#FF0000FF", gridOutput.Legend["2"], "Key '2' should be third color (Red)")

	// Check '.' for empty areas outside the trimmed cel
	assert.Equal(t, byte('.'), gridOutput.Grid[0][0], "Top-left outside cel should be '.'")
	assert.Equal(t, byte('.'), gridOutput.Grid[3][3], "Row 3 col 3 should be '.'")
	assert.Equal(t, byte('.'), gridOutput.Grid[15][15], "Bottom-right outside cel should be '.'")

	// Check right keys at drawn positions
	// Row 4: cols 4=Green ('0'), 5=Blue ('1'), 6=Red ('2'), 7=Red ('2')
	assert.Equal(t, "....0122........", gridOutput.Grid[4])
	assert.Equal(t, "....2222........", gridOutput.Grid[5])
	assert.Equal(t, "....2222........", gridOutput.Grid[6])
	assert.Equal(t, "....2222........", gridOutput.Grid[7])

	// Measure text size for comparison (Report requirement 4)
	pixelsFullResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_pixels",
		Arguments: map[string]any{
			"sprite_path":  createOutput.FilePath,
			"layer_name":   "ContentLayer",
			"frame_number": 1,
			"x":            0,
			"y":            0,
			"width":        16,
			"height":       16,
			"format":       "pixels",
			"page_size":    1000,
		},
	})
	require.NoError(t, err)
	require.False(t, pixelsFullResult.IsError)
	pixelsText := pixelsFullResult.Content[0].(*mcp.TextContent).Text

	t.Logf("Measured text size for 16x16 region: format=\"pixels\" = %d bytes, format=\"grid\" = %d bytes", len(pixelsText), len(gridText))

	// 5. Call grid with a 200x200 region -> IsError true
	oversizeResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_pixels",
		Arguments: map[string]any{
			"sprite_path":  createOutput.FilePath,
			"layer_name":   "ContentLayer",
			"frame_number": 1,
			"x":            0,
			"y":            0,
			"width":        200,
			"height":       200,
			"format":       "grid",
		},
	})
	require.NoError(t, err)
	assert.True(t, oversizeResult.IsError, "Grid format with 200x200 region should return tool error")
	assert.Contains(t, oversizeResult.Content[0].(*mcp.TextContent).Text, "16384")

	// Also verify non-empty cursor with grid format returns IsError true
	cursorGridResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_pixels",
		Arguments: map[string]any{
			"sprite_path":  createOutput.FilePath,
			"layer_name":   "ContentLayer",
			"frame_number": 1,
			"x":            0,
			"y":            0,
			"width":        16,
			"height":       16,
			"format":       "grid",
			"cursor":       "10",
		},
	})
	require.NoError(t, err)
	assert.True(t, cursorGridResult.IsError, "Grid format with cursor should return tool error")

	// 6. Pagination test: page_size small enough (10) that pages contain skipped positions,
	// walk all cursors to the end, assert no duplicate (x,y) and walk terminates with full set of opaque pixels.
	cursor := ""
	seen := make(map[[2]int]bool)
	var allPixels []PixelData
	pageCount := 0
	maxPages := 40 // circuit breaker against infinite loop

	for {
		pageCount++
		require.Less(t, pageCount, maxPages, "Pagination walk exceeded maxPages limit (infinite loop)")

		args := map[string]any{
			"sprite_path":  createOutput.FilePath,
			"layer_name":   "ContentLayer",
			"frame_number": 1,
			"x":            0,
			"y":            0,
			"width":        16,
			"height":       16,
			"page_size":    10,
		}
		if cursor != "" {
			args["cursor"] = cursor
		}

		pageResult, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "get_pixels",
			Arguments: args,
		})
		require.NoError(t, err)
		require.False(t, pageResult.IsError)

		var pageOutput GetPixelsOutput
		require.NoError(t, json.Unmarshal([]byte(pageResult.Content[0].(*mcp.TextContent).Text), &pageOutput))

		for _, p := range pageOutput.Pixels {
			coord := [2]int{p.X, p.Y}
			assert.False(t, seen[coord], "Duplicate pixel received at (%d, %d)", p.X, p.Y)
			seen[coord] = true
			allPixels = append(allPixels, p)
		}

		if pageOutput.NextCursor == "" {
			break
		}
		cursor = pageOutput.NextCursor
	}

	// Verify all 16 opaque pixels from the 4x4 trimmed cel are returned
	assert.Equal(t, 16, len(allPixels), "Walk should terminate with exactly the 16 drawn pixels")
	for y := 4; y <= 7; y++ {
		for x := 4; x <= 7; x++ {
			assert.True(t, seen[[2]int{x, y}], "Expected pixel at (%d, %d)", x, y)
		}
	}
}
