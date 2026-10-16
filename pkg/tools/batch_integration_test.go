//go:build integration

package tools

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willibrandon/mtlog"
	"github.com/willibrandon/mtlog/core"
	"github.com/willibrandon/pixel-mcp/internal/testutil"
	"github.com/willibrandon/pixel-mcp/pkg/aseprite"
)

func createBatchTestSession(t *testing.T) (*mcp.Server, *mcp.ClientSession, *aseprite.Client, string) {
	t.Helper()

	cfg := testutil.LoadTestConfig(t)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, cfg.Timeout)
	gen := aseprite.NewLuaGenerator()
	logger := mtlog.New(mtlog.WithMinimumLevel(core.ErrorLevel))

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "pixel-mcp-test",
		Version: "1.0.0",
	}, nil)

	RegisterCanvasTools(server, client, gen, cfg, logger)
	RegisterDrawingTools(server, client, gen, cfg, logger)
	RegisterInspectionTools(server, client, gen, cfg, logger)
	RegisterBatchTools(server, client, gen, cfg, logger)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	_, err := server.Connect(context.Background(), serverTransport, nil)
	require.NoError(t, err)

	mcpClient := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	session, err := mcpClient.Connect(context.Background(), clientTransport, nil)
	require.NoError(t, err)

	return server, session, client, cfg.TempDir
}

// TestIntegration_BatchDraw_AllPrimitives verifies drawing rectangle, circle, line, contour,
// fill, and pixels (including outside trimmed bounds) with preview in a single batch.
func TestIntegration_BatchDraw_AllPrimitives(t *testing.T) {
	_, session, _, _ := createBatchTestSession(t)
	defer session.Close()

	ctx := context.Background()

	// 1. Create a 32x32 RGB sprite
	createRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "create_canvas",
		Arguments: map[string]any{
			"width":      32,
			"height":     32,
			"color_mode": "rgb",
		},
	})
	require.NoError(t, err)
	require.False(t, createRes.IsError)

	var createOut struct {
		FilePath string `json:"file_path"`
	}
	require.NoError(t, json.Unmarshal([]byte(createRes.Content[0].(*mcp.TextContent).Text), &createOut))
	defer os.Remove(createOut.FilePath)

	// 2. Run batch_draw with all primitives
	batchRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "batch_draw",
		Arguments: map[string]any{
			"sprite_path":    createOut.FilePath,
			"layer_name":     "Layer 1",
			"frame_number":   1,
			"return_preview": true,
			"operations": []map[string]any{
				// Filled rectangle 6x6 at (10, 10) -> cel trimmed to (10,10)-(15,15)
				{
					"type":   "rectangle",
					"x":      10,
					"y":      10,
					"width":  6,
					"height": 6,
					"color":  "#FF0000FF",
					"filled": true,
				},
				// Circle outline center (24, 8), radius 4
				{
					"type":     "circle",
					"center_x": 24,
					"center_y": 8,
					"radius":   4,
					"color":    "#00FF00FF",
					"filled":   false,
				},
				// Line from (0, 0) to (5, 0)
				{
					"type":      "line",
					"x1":        0,
					"y1":        0,
					"x2":        5,
					"y2":        0,
					"color":     "#0000FFFF",
					"thickness": 1,
				},
				// Contour path
				{
					"type": "contour",
					"points": []map[string]any{
						{"x": 0, "y": 20},
						{"x": 5, "y": 20},
						{"x": 5, "y": 25},
					},
					"color":     "#FFFF00FF",
					"thickness": 1,
					"closed":    false,
				},
				// Fill empty area
				{
					"type":      "fill",
					"x":         28,
					"y":         28,
					"color":     "#FF00FFFF",
					"tolerance": 0,
				},
				// Pixels op: includes pixel at (2, 2) which is OUTSIDE rectangle bounds,
				// and pixel at (12, 12) which is INSIDE rectangle bounds
				{
					"type": "pixels",
					"pixels": []map[string]any{
						{"x": 2, "y": 2, "color": "#00FFFFFF"},
						{"x": 12, "y": 12, "color": "#00FFFFFF"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.False(t, batchRes.IsError)

	// Verify preview image was returned
	hasImage := false
	for _, c := range batchRes.Content {
		if _, ok := c.(*mcp.ImageContent); ok {
			hasImage = true
			break
		}
	}
	assert.True(t, hasImage, "Expected return_preview to include an ImageContent")

	// 3. Verify representative pixels with get_pixels
	// (11, 11) should be rectangle color (#FF0000FF)
	p11Res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_pixels",
		Arguments: map[string]any{
			"sprite_path":  createOut.FilePath,
			"layer_name":   "Layer 1",
			"frame_number": 1,
			"x":            11,
			"y":            11,
			"width":        1,
			"height":       1,
		},
	})
	require.NoError(t, err)
	var p11Out GetPixelsOutput
	require.NoError(t, json.Unmarshal([]byte(p11Res.Content[0].(*mcp.TextContent).Text), &p11Out))
	require.Len(t, p11Out.Pixels, 1)
	assert.Equal(t, "#FF0000FF", p11Out.Pixels[0].Color, "Pixel at (11,11) should be rectangle red")

	// (12, 12) was drawn by pixels op inside rectangle bounds (#00FFFFFF)
	p12Res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_pixels",
		Arguments: map[string]any{
			"sprite_path":  createOut.FilePath,
			"layer_name":   "Layer 1",
			"frame_number": 1,
			"x":            12,
			"y":            12,
			"width":        1,
			"height":       1,
		},
	})
	require.NoError(t, err)
	var p12Out GetPixelsOutput
	require.NoError(t, json.Unmarshal([]byte(p12Res.Content[0].(*mcp.TextContent).Text), &p12Out))
	require.Len(t, p12Out.Pixels, 1)
	assert.Equal(t, "#00FFFFFF", p12Out.Pixels[0].Color, "Pixel at (12,12) should be cyan from pixels op")

	// (2, 2) was drawn by pixels op OUTSIDE rectangle bounds (#00FFFFFF)
	p2Res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_pixels",
		Arguments: map[string]any{
			"sprite_path":  createOut.FilePath,
			"layer_name":   "Layer 1",
			"frame_number": 1,
			"x":            2,
			"y":            2,
			"width":        1,
			"height":       1,
		},
	})
	require.NoError(t, err)
	var p2Out GetPixelsOutput
	require.NoError(t, json.Unmarshal([]byte(p2Res.Content[0].(*mcp.TextContent).Text), &p2Out))
	require.Len(t, p2Out.Pixels, 1)
	assert.Equal(t, "#00FFFFFF", p2Out.Pixels[0].Color, "Pixel at (2,2) outside rectangle bounds should be cyan")

	// (24, 4) should be circle outline top point (#00FF00FF)
	circleRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_pixels",
		Arguments: map[string]any{
			"sprite_path":  createOut.FilePath,
			"layer_name":   "Layer 1",
			"frame_number": 1,
			"x":            24,
			"y":            4,
			"width":        1,
			"height":       1,
		},
	})
	require.NoError(t, err)
	var circleOut GetPixelsOutput
	require.NoError(t, json.Unmarshal([]byte(circleRes.Content[0].(*mcp.TextContent).Text), &circleOut))
	require.Len(t, circleOut.Pixels, 1)
	assert.Equal(t, "#00FF00FF", circleOut.Pixels[0].Color, "Pixel at (24,4) should be circle outline green")

	// (2, 0) should be line color (#0000FFFF)
	lineRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_pixels",
		Arguments: map[string]any{
			"sprite_path":  createOut.FilePath,
			"layer_name":   "Layer 1",
			"frame_number": 1,
			"x":            2,
			"y":            0,
			"width":        1,
			"height":       1,
		},
	})
	require.NoError(t, err)
	var lineOut GetPixelsOutput
	require.NoError(t, json.Unmarshal([]byte(lineRes.Content[0].(*mcp.TextContent).Text), &lineOut))
	require.Len(t, lineOut.Pixels, 1)
	assert.Equal(t, "#0000FFFF", lineOut.Pixels[0].Color, "Pixel at (2,0) should be line blue")
}

// TestIntegration_BatchDraw_Atomicity verifies that a batch containing a failing operation
// (e.g. missing layer) rolls back completely and does not save any prior operations.
func TestIntegration_BatchDraw_Atomicity(t *testing.T) {
	_, session, _, _ := createBatchTestSession(t)
	defer session.Close()

	ctx := context.Background()

	createRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "create_canvas",
		Arguments: map[string]any{
			"width":      32,
			"height":     32,
			"color_mode": "rgb",
		},
	})
	require.NoError(t, err)
	var createOut struct {
		FilePath string `json:"file_path"`
	}
	require.NoError(t, json.Unmarshal([]byte(createRes.Content[0].(*mcp.TextContent).Text), &createOut))
	defer os.Remove(createOut.FilePath)

	// Run batch where op 0 draws pixels and op 1 references a missing layer
	batchRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "batch_draw",
		Arguments: map[string]any{
			"sprite_path":  createOut.FilePath,
			"layer_name":   "Layer 1",
			"frame_number": 1,
			"operations": []map[string]any{
				{
					"type": "pixels",
					"pixels": []map[string]any{
						{"x": 15, "y": 15, "color": "#FF0000FF"},
					},
				},
				{
					"type":       "line",
					"layer_name": "NonExistentLayer",
					"x1":         0,
					"y1":         0,
					"x2":         10,
					"y2":         10,
					"color":      "#00FF00FF",
				},
			},
		},
	})

	require.NoError(t, err, "JSON-RPC should not fail")
	require.True(t, batchRes.IsError, "Expected batch_draw to fail due to missing layer")

	errorText := batchRes.Content[0].(*mcp.TextContent).Text
	assert.True(t, strings.Contains(errorText, "operations[1]"), "Error text should name operations[1], got: %s", errorText)

	// Verify with get_pixels that op 0's pixel at (15, 15) was NOT applied
	pixRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_pixels",
		Arguments: map[string]any{
			"sprite_path":  createOut.FilePath,
			"layer_name":   "Layer 1",
			"frame_number": 1,
			"x":            15,
			"y":            15,
			"width":        1,
			"height":       1,
		},
	})
	require.NoError(t, err)
	var pixOut GetPixelsOutput
	require.NoError(t, json.Unmarshal([]byte(pixRes.Content[0].(*mcp.TextContent).Text), &pixOut))
	if len(pixOut.Pixels) > 0 {
		assert.NotEqual(t, "#FF0000FF", pixOut.Pixels[0].Color, "op 0's pixel should NOT have been saved")
	}
}

// TestIntegration_BatchDraw_LayerAndFrameOverride verifies per-op layer_name override across multiple layers.
func TestIntegration_BatchDraw_LayerAndFrameOverride(t *testing.T) {
	_, session, _, _ := createBatchTestSession(t)
	defer session.Close()

	ctx := context.Background()

	createRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "create_canvas",
		Arguments: map[string]any{
			"width":      32,
			"height":     32,
			"color_mode": "rgb",
		},
	})
	require.NoError(t, err)
	var createOut struct {
		FilePath string `json:"file_path"`
	}
	require.NoError(t, json.Unmarshal([]byte(createRes.Content[0].(*mcp.TextContent).Text), &createOut))
	defer os.Remove(createOut.FilePath)

	// Add second layer
	addLayerRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "add_layer",
		Arguments: map[string]any{
			"sprite_path": createOut.FilePath,
			"layer_name":  "Layer 2",
		},
	})
	require.NoError(t, err)
	require.False(t, addLayerRes.IsError)

	// Batch draw to both layers: op 0 uses default Layer 1, op 1 overrides to Layer 2
	batchRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "batch_draw",
		Arguments: map[string]any{
			"sprite_path":  createOut.FilePath,
			"layer_name":   "Layer 1",
			"frame_number": 1,
			"operations": []map[string]any{
				{
					"type": "pixels",
					"pixels": []map[string]any{
						{"x": 5, "y": 5, "color": "#FF0000FF"},
					},
				},
				{
					"type":       "pixels",
					"layer_name": "Layer 2",
					"pixels": []map[string]any{
						{"x": 10, "y": 10, "color": "#00FF00FF"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.False(t, batchRes.IsError)

	// Read from Layer 1 at (5, 5) -> should be red
	p1Res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_pixels",
		Arguments: map[string]any{
			"sprite_path":  createOut.FilePath,
			"layer_name":   "Layer 1",
			"frame_number": 1,
			"x":            5,
			"y":            5,
			"width":        1,
			"height":       1,
		},
	})
	require.NoError(t, err)
	var p1Out GetPixelsOutput
	require.NoError(t, json.Unmarshal([]byte(p1Res.Content[0].(*mcp.TextContent).Text), &p1Out))
	require.Len(t, p1Out.Pixels, 1)
	assert.Equal(t, "#FF0000FF", p1Out.Pixels[0].Color)

	// Read from Layer 2 at (10, 10) -> should be green
	p2Res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_pixels",
		Arguments: map[string]any{
			"sprite_path":  createOut.FilePath,
			"layer_name":   "Layer 2",
			"frame_number": 1,
			"x":            10,
			"y":            10,
			"width":        1,
			"height":       1,
		},
	})
	require.NoError(t, err)
	var p2Out GetPixelsOutput
	require.NoError(t, json.Unmarshal([]byte(p2Res.Content[0].(*mcp.TextContent).Text), &p2Out))
	require.Len(t, p2Out.Pixels, 1)
	assert.Equal(t, "#00FF00FF", p2Out.Pixels[0].Color)
}

// TestIntegration_BatchDraw_IndexedMode verifies batch drawing in indexed color mode with use_palette.
func TestIntegration_BatchDraw_IndexedMode(t *testing.T) {
	_, session, _, _ := createBatchTestSession(t)
	defer session.Close()

	ctx := context.Background()

	createRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "create_canvas",
		Arguments: map[string]any{
			"width":      32,
			"height":     32,
			"color_mode": "indexed",
		},
	})
	require.NoError(t, err)
	var createOut struct {
		FilePath string `json:"file_path"`
	}
	require.NoError(t, json.Unmarshal([]byte(createRes.Content[0].(*mcp.TextContent).Text), &createOut))
	defer os.Remove(createOut.FilePath)

	// Run batch draw with use_palette true
	batchRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "batch_draw",
		Arguments: map[string]any{
			"sprite_path":  createOut.FilePath,
			"layer_name":   "Layer 1",
			"frame_number": 1,
			"use_palette":  true,
			"operations": []map[string]any{
				{
					"type":   "rectangle",
					"x":      4,
					"y":      4,
					"width":  8,
					"height": 8,
					"color":  "#FF0000FF",
					"filled": true,
				},
				{
					"type": "pixels",
					"pixels": []map[string]any{
						{"x": 1, "y": 1, "color": "#0000FFFF"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.False(t, batchRes.IsError)

	// Verify pixels are non-transparent
	getRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_pixels",
		Arguments: map[string]any{
			"sprite_path":  createOut.FilePath,
			"layer_name":   "Layer 1",
			"frame_number": 1,
			"x":            4,
			"y":            4,
			"width":        2,
			"height":       2,
		},
	})
	require.NoError(t, err)
	var getOut GetPixelsOutput
	require.NoError(t, json.Unmarshal([]byte(getRes.Content[0].(*mcp.TextContent).Text), &getOut))
	require.NotEmpty(t, getOut.Pixels)
	for _, p := range getOut.Pixels {
		assert.NotEqual(t, "#00000000", p.Color, "Drawn pixel in indexed mode should not be transparent")
	}
}

// TestIntegration_BatchDraw_ValidationError verifies that validation errors return ToolError
// with IsError: true and err == nil at the MCP layer (avoiding JSON-RPC schema errors).
func TestIntegration_BatchDraw_ValidationError(t *testing.T) {
	_, session, _, _ := createBatchTestSession(t)
	defer session.Close()

	ctx := context.Background()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "batch_draw",
		Arguments: map[string]any{
			"sprite_path":  "/tmp/dummy.aseprite",
			"layer_name":   "Layer 1",
			"frame_number": 1,
			"operations": []map[string]any{
				{
					"type":      "line",
					"color":     "#FF0000FF",
					"thickness": 0, // Invalid thickness
				},
			},
		},
	})

	// Critical check: no JSON-RPC error, clean ToolError response
	require.NoError(t, err, "Validation error must not cause JSON-RPC fault")
	require.NotNil(t, result)
	assert.True(t, result.IsError, "Expected result.IsError to be true")
	require.NotEmpty(t, result.Content)

	errorText := result.Content[0].(*mcp.TextContent).Text
	assert.True(t, strings.Contains(errorText, "thickness must be between 1 and 100, got 0"),
		"Expected validation error message, got: %s", errorText)
}
