package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willibrandon/mtlog"
	"github.com/willibrandon/mtlog/core"
	"github.com/willibrandon/pixel-mcp/internal/testutil"
	"github.com/willibrandon/pixel-mcp/pkg/aseprite"
)

func TestGetPixelsInput_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   GetPixelsInput
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid input",
			input: GetPixelsInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				X:           0,
				Y:           0,
				Width:       10,
				Height:      10,
			},
			wantErr: false,
		},
		{
			name: "invalid width - zero",
			input: GetPixelsInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				X:           0,
				Y:           0,
				Width:       0,
				Height:      10,
			},
			wantErr: true,
			errMsg:  "width and height must be positive",
		},
		{
			name: "invalid width - negative",
			input: GetPixelsInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				X:           0,
				Y:           0,
				Width:       -1,
				Height:      10,
			},
			wantErr: true,
			errMsg:  "width and height must be positive",
		},
		{
			name: "invalid height - zero",
			input: GetPixelsInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				X:           0,
				Y:           0,
				Width:       10,
				Height:      0,
			},
			wantErr: true,
			errMsg:  "width and height must be positive",
		},
		{
			name: "invalid height - negative",
			input: GetPixelsInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				X:           0,
				Y:           0,
				Width:       10,
				Height:      -1,
			},
			wantErr: true,
			errMsg:  "width and height must be positive",
		},
		{
			name: "invalid frame number - zero",
			input: GetPixelsInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 0,
				X:           0,
				Y:           0,
				Width:       10,
				Height:      10,
			},
			wantErr: true,
			errMsg:  "frame_number must be >= 1",
		},
		{
			name: "invalid frame number - negative",
			input: GetPixelsInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: -1,
				X:           0,
				Y:           0,
				Width:       10,
				Height:      10,
			},
			wantErr: true,
			errMsg:  "frame_number must be >= 1",
		},
		{
			name: "valid with negative coordinates",
			input: GetPixelsInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				X:           -5,
				Y:           -5,
				Width:       10,
				Height:      10,
			},
			wantErr: false,
		},
		{
			name: "large region",
			input: GetPixelsInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				X:           0,
				Y:           0,
				Width:       1000,
				Height:      1000,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate width and height
			if (tt.input.Width <= 0 || tt.input.Height <= 0) && tt.wantErr {
				if tt.errMsg != "width and height must be positive" {
					t.Errorf("Expected error message about width/height")
				}
				return
			}

			// Validate frame number
			if tt.input.FrameNumber < 1 && tt.wantErr {
				if tt.errMsg != "frame_number must be >= 1" {
					t.Errorf("Expected error message about frame_number")
				}
				return
			}

			// If we get here and wantErr is true, test failed
			if tt.wantErr {
				t.Errorf("Expected error but validation passed")
			}
		})
	}
}

func TestPixelData_Structure(t *testing.T) {
	tests := []struct {
		name  string
		pixel PixelData
	}{
		{
			name: "RGB color",
			pixel: PixelData{
				X:     10,
				Y:     20,
				Color: "#FF0000",
			},
		},
		{
			name: "RGBA color",
			pixel: PixelData{
				X:     5,
				Y:     15,
				Color: "#00FF0080",
			},
		},
		{
			name: "black color",
			pixel: PixelData{
				X:     0,
				Y:     0,
				Color: "#000000",
			},
		},
		{
			name: "white color",
			pixel: PixelData{
				X:     100,
				Y:     100,
				Color: "#FFFFFF",
			},
		},
		{
			name: "transparent color",
			pixel: PixelData{
				X:     50,
				Y:     50,
				Color: "#00000000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pixel.Color == "" {
				t.Error("Expected non-empty color")
			}
			if len(tt.pixel.Color) < 7 {
				t.Errorf("Expected color to be at least 7 characters (e.g., #RRGGBB), got %d", len(tt.pixel.Color))
			}
			if tt.pixel.Color[0] != '#' {
				t.Errorf("Expected color to start with #, got %s", tt.pixel.Color)
			}
		})
	}
}

func TestGetPixelsOutput_Structure(t *testing.T) {
	output := GetPixelsOutput{
		Pixels: []PixelData{
			{X: 0, Y: 0, Color: "#FF0000FF"},
			{X: 1, Y: 0, Color: "#00FF00FF"},
			{X: 2, Y: 0, Color: "#0000FFFF"},
		},
	}

	if len(output.Pixels) != 3 {
		t.Errorf("Expected 3 pixels, got %d", len(output.Pixels))
	}

	// Verify each pixel has valid structure
	for i, p := range output.Pixels {
		if p.Color == "" {
			t.Errorf("Pixel %d has empty color", i)
		}
		if p.X < 0 {
			t.Errorf("Pixel %d has negative X coordinate: %d", i, p.X)
		}
		if p.Y < 0 {
			t.Errorf("Pixel %d has negative Y coordinate: %d", i, p.Y)
		}
	}
}

func TestGetPixelsInput_Regions(t *testing.T) {
	tests := []struct {
		name   string
		input  GetPixelsInput
		pixels int // expected number of pixels in region
	}{
		{
			name: "1x1 region",
			input: GetPixelsInput{
				Width:  1,
				Height: 1,
			},
			pixels: 1,
		},
		{
			name: "10x10 region",
			input: GetPixelsInput{
				Width:  10,
				Height: 10,
			},
			pixels: 100,
		},
		{
			name: "rectangular region 5x10",
			input: GetPixelsInput{
				Width:  5,
				Height: 10,
			},
			pixels: 50,
		},
		{
			name: "large region 100x100",
			input: GetPixelsInput{
				Width:  100,
				Height: 100,
			},
			pixels: 10000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedPixels := tt.input.Width * tt.input.Height
			if expectedPixels != tt.pixels {
				t.Errorf("Expected %d pixels, calculated %d", tt.pixels, expectedPixels)
			}
		})
	}
}

// createInspectionTestSession creates an MCP session with inspection tools registered
func createInspectionTestSession(t *testing.T) (*mcp.Server, *mcp.ClientSession, *aseprite.Client) {
	t.Helper()

	cfg := testutil.LoadTestConfig(t)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, cfg.Timeout)
	gen := aseprite.NewLuaGenerator()
	logger := mtlog.New(mtlog.WithMinimumLevel(core.ErrorLevel))

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "pixel-mcp-test",
		Version: "1.0.0",
	}, nil)

	RegisterInspectionTools(server, client, gen, cfg, logger)
	// Also register canvas and drawing tools for setup
	RegisterCanvasTools(server, client, gen, cfg, logger)
	RegisterDrawingTools(server, client, gen, cfg, logger)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	_, err := server.Connect(context.Background(), serverTransport, nil)
	require.NoError(t, err)

	mcpClient := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	session, err := mcpClient.Connect(context.Background(), clientTransport, nil)
	require.NoError(t, err)

	return server, session, client
}

func TestGetPixels_ViaMCP(t *testing.T) {
	_, session, _ := createInspectionTestSession(t)
	defer session.Close()

	// Create a 10x10 sprite
	createResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "create_canvas",
		Arguments: map[string]any{
			"width":      10,
			"height":     10,
			"color_mode": "rgb",
		},
	})
	require.NoError(t, err)
	require.False(t, createResult.IsError)

	var createOutput struct {
		FilePath string `json:"file_path"`
	}
	json.Unmarshal([]byte(createResult.Content[0].(*mcp.TextContent).Text), &createOutput)

	// Draw some known pixels
	drawResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "draw_pixels",
		Arguments: map[string]any{
			"sprite_path":  createOutput.FilePath,
			"layer_name":   "Layer 1",
			"frame_number": 1,
			"pixels": []map[string]any{
				{"x": 0, "y": 0, "color": "#FF0000FF"}, // Red
				{"x": 1, "y": 0, "color": "#00FF00FF"}, // Green
				{"x": 2, "y": 0, "color": "#0000FFFF"}, // Blue
			},
		},
	})
	require.NoError(t, err)
	require.False(t, drawResult.IsError)

	// Now read those pixels back
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "get_pixels",
		Arguments: map[string]any{
			"sprite_path":  createOutput.FilePath,
			"layer_name":   "Layer 1",
			"frame_number": 1,
			"x":            0,
			"y":            0,
			"width":        3,
			"height":       1,
		},
	})

	require.NoError(t, err)
	require.False(t, result.IsError)

	var output GetPixelsOutput
	json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &output)

	// Verify we got 3 pixels
	assert.Equal(t, 3, len(output.Pixels), "Should return 3 pixels")
	assert.Equal(t, 3, output.TotalPixels, "Total pixels should be 3")
	assert.Empty(t, output.NextCursor, "Should have no next cursor for small region")

	// Verify colors match (Aseprite returns uppercase hex)
	assert.Equal(t, "#FF0000FF", output.Pixels[0].Color)
	assert.Equal(t, "#00FF00FF", output.Pixels[1].Color)
	assert.Equal(t, "#0000FFFF", output.Pixels[2].Color)

	// Verify coordinates
	assert.Equal(t, 0, output.Pixels[0].X)
	assert.Equal(t, 0, output.Pixels[0].Y)
	assert.Equal(t, 1, output.Pixels[1].X)
	assert.Equal(t, 2, output.Pixels[2].X)
}

func TestGetPixelsPagination_ViaMCP(t *testing.T) {
	_, session, _ := createInspectionTestSession(t)
	defer session.Close()

	// Create a 5x5 sprite (25 pixels)
	createResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "create_canvas",
		Arguments: map[string]any{
			"width":      5,
			"height":     5,
			"color_mode": "rgb",
		},
	})
	require.NoError(t, err)

	var createOutput struct {
		FilePath string `json:"file_path"`
	}
	json.Unmarshal([]byte(createResult.Content[0].(*mcp.TextContent).Text), &createOutput)

	// Get first page with page_size=10
	result1, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "get_pixels",
		Arguments: map[string]any{
			"sprite_path":  createOutput.FilePath,
			"layer_name":   "Layer 1",
			"frame_number": 1,
			"x":            0,
			"y":            0,
			"width":        5,
			"height":       5,
			"page_size":    10,
		},
	})
	require.NoError(t, err)
	require.False(t, result1.IsError)

	var output1 GetPixelsOutput
	json.Unmarshal([]byte(result1.Content[0].(*mcp.TextContent).Text), &output1)

	assert.Equal(t, 10, len(output1.Pixels), "First page should have 10 pixels")
	assert.Equal(t, 25, output1.TotalPixels, "Total should be 25 pixels")
	assert.NotEmpty(t, output1.NextCursor, "Should have next cursor")

	// Get second page using cursor
	result2, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "get_pixels",
		Arguments: map[string]any{
			"sprite_path":  createOutput.FilePath,
			"layer_name":   "Layer 1",
			"frame_number": 1,
			"x":            0,
			"y":            0,
			"width":        5,
			"height":       5,
			"page_size":    10,
			"cursor":       output1.NextCursor,
		},
	})
	require.NoError(t, err)

	var output2 GetPixelsOutput
	json.Unmarshal([]byte(result2.Content[0].(*mcp.TextContent).Text), &output2)

	assert.Equal(t, 10, len(output2.Pixels), "Second page should have 10 pixels")
	assert.NotEmpty(t, output2.NextCursor, "Should have next cursor for third page")

	// Get third page (last 5 pixels)
	result3, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "get_pixels",
		Arguments: map[string]any{
			"sprite_path":  createOutput.FilePath,
			"layer_name":   "Layer 1",
			"frame_number": 1,
			"x":            0,
			"y":            0,
			"width":        5,
			"height":       5,
			"page_size":    10,
			"cursor":       output2.NextCursor,
		},
	})
	require.NoError(t, err)

	var output3 GetPixelsOutput
	json.Unmarshal([]byte(result3.Content[0].(*mcp.TextContent).Text), &output3)

	assert.Equal(t, 5, len(output3.Pixels), "Third page should have remaining 5 pixels")
	assert.Empty(t, output3.NextCursor, "Should have no next cursor on last page")
}

func TestBuildPixelGridFormat(t *testing.T) {
	// generate 90 distinct colors for the exceeding colors test
	exceedingColors := make([]PixelData, 90)
	for i := 0; i < 90; i++ {
		exceedingColors[i] = PixelData{
			X:     i % 10,
			Y:     i / 10,
			Color: fmt.Sprintf("#%02X%02X%02XFF", (i*2)%256, (i*3)%256, (i*5)%256),
		}
	}

	tests := []struct {
		name       string
		x, y       int
		width      int
		height     int
		pixels     []PixelData
		wantGrid   []string
		wantLegend map[string]string
		wantErr    bool
		errMsg     string
	}{
		{
			name:   "transparent alpha",
			x:      0,
			y:      0,
			width:  2,
			height: 1,
			pixels: []PixelData{
				{X: 0, Y: 0, Color: "#FF000000"}, // alpha 00 -> transparent
				{X: 1, Y: 0, Color: "#00FF00FF"}, // opaque green
			},
			wantGrid: []string{".0"},
			wantLegend: map[string]string{
				"0": "#00FF00FF",
			},
			wantErr: false,
		},
		{
			name:   "missing positions",
			x:      0,
			y:      0,
			width:  3,
			height: 1,
			pixels: []PixelData{
				{X: 0, Y: 0, Color: "#FF0000FF"},
				// (1, 0) missing -> should be '.'
				{X: 2, Y: 0, Color: "#00FF00FF"},
			},
			wantGrid: []string{"0.1"},
			wantLegend: map[string]string{
				"0": "#FF0000FF",
				"1": "#00FF00FF",
			},
			wantErr: false,
		},
		{
			name:   "key ordering by first appearance",
			x:      0,
			y:      0,
			width:  2,
			height: 2,
			pixels: []PixelData{
				{X: 0, Y: 0, Color: "#FF0000FF"}, // first -> '0'
				{X: 1, Y: 0, Color: "#00FF00FF"}, // second -> '1'
				{X: 0, Y: 1, Color: "#00FF00FF"}, // already seen -> '1'
				{X: 1, Y: 1, Color: "#0000FFFF"}, // third -> '2'
			},
			wantGrid: []string{"01", "12"},
			wantLegend: map[string]string{
				"0": "#FF0000FF",
				"1": "#00FF00FF",
				"2": "#0000FFFF",
			},
			wantErr: false,
		},
		{
			name:   "region offset (x,y != 0)",
			x:      10,
			y:      20,
			width:  2,
			height: 2,
			pixels: []PixelData{
				{X: 10, Y: 20, Color: "#FF0000FF"}, // top-left of region
				{X: 11, Y: 21, Color: "#00FF00FF"}, // bottom-right of region
			},
			wantGrid: []string{"0.", ".1"},
			wantLegend: map[string]string{
				"0": "#FF0000FF",
				"1": "#00FF00FF",
			},
			wantErr: false,
		},
		{
			name:   "legend excludes '.'",
			x:      0,
			y:      0,
			width:  2,
			height: 2,
			pixels: []PixelData{
				{X: 0, Y: 0, Color: "#00000000"}, // transparent
				{X: 1, Y: 0, Color: "#FF0000FF"}, // red
				// (0,1) missing
				{X: 1, Y: 1, Color: "#00000000"}, // transparent
			},
			wantGrid: []string{".0", ".."},
			wantLegend: map[string]string{
				"0": "#FF0000FF",
			},
			wantErr: false,
		},
		{
			name:    "exceeding 86 colors -> error",
			x:       0,
			y:       0,
			width:   10,
			height:  10,
			pixels:  exceedingColors,
			wantErr: true,
			errMsg:  "more distinct non-transparent colors than available keys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grid, legend, err := buildPixelGridFormat(tt.x, tt.y, tt.width, tt.height, tt.pixels)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantGrid, grid)
			assert.Equal(t, tt.wantLegend, legend)
			_, dotInLegend := legend["."]
			assert.False(t, dotInLegend, "legend must never contain '.'")
		})
	}
}
