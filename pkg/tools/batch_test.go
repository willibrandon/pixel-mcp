package tools

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willibrandon/mtlog"
	"github.com/willibrandon/mtlog/core"
	"github.com/willibrandon/pixel-mcp/internal/testutil"
)

func intPtr(i int) *int {
	return &i
}

func TestValidateBatchInput(t *testing.T) {
	tests := []struct {
		name        string
		input       BatchDrawInput
		wantErr     bool
		errContains string
	}{
		{
			name: "valid full batch with all operation types",
			input: BatchDrawInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				Operations: []BatchOperationInput{
					{
						Type:   "pixels",
						Pixels: []PixelInput{{X: 0, Y: 0, Color: "#FF0000FF"}},
					},
					{
						Type:      "line",
						Color:     "#00FF00FF",
						X1:        0,
						Y1:        0,
						X2:        10,
						Y2:        10,
						Thickness: intPtr(2),
					},
					{
						Type:   "rectangle",
						Color:  "#0000FFFF",
						X:      5,
						Y:      5,
						Width:  10,
						Height: 10,
						Filled: true,
					},
					{
						Type:    "circle",
						Color:   "#FFFF00FF",
						CenterX: 15,
						CenterY: 15,
						Radius:  5,
						Filled:  false,
					},
					{
						Type:  "contour",
						Color: "#FF00FFFF",
						Points: []PointInput{
							{X: 1, Y: 1},
							{X: 5, Y: 1},
							{X: 5, Y: 5},
						},
						Closed:    true,
						Thickness: intPtr(1),
					},
					{
						Type:      "fill",
						Color:     "#00FFFFFF",
						X:         2,
						Y:         2,
						Tolerance: 10,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty sprite path",
			input: BatchDrawInput{
				SpritePath:  "",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				Operations: []BatchOperationInput{
					{Type: "fill", Color: "#FF0000FF", X: 0, Y: 0},
				},
			},
			wantErr:     true,
			errContains: "sprite_path cannot be empty",
		},
		{
			name: "empty default layer name",
			input: BatchDrawInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "",
				FrameNumber: 1,
				Operations: []BatchOperationInput{
					{Type: "fill", Color: "#FF0000FF", X: 0, Y: 0},
				},
			},
			wantErr:     true,
			errContains: "layer_name cannot be empty",
		},
		{
			name: "invalid default frame number (zero)",
			input: BatchDrawInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 0,
				Operations: []BatchOperationInput{
					{Type: "fill", Color: "#FF0000FF", X: 0, Y: 0},
				},
			},
			wantErr:     true,
			errContains: "frame_number must be at least 1, got 0",
		},
		{
			name: "empty operations array",
			input: BatchDrawInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				Operations:  []BatchOperationInput{},
			},
			wantErr:     true,
			errContains: "operations must contain between 1 and 1000 items, got 0",
		},
		{
			name: "unknown operation type",
			input: BatchDrawInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				Operations: []BatchOperationInput{
					{Type: "polygon", Color: "#FF0000FF"},
				},
			},
			wantErr:     true,
			errContains: `operations[0]: unknown operation type "polygon", valid types are: pixels, line, rectangle, circle, contour, fill`,
		},
		{
			name: "negative frame_number override on operation",
			input: BatchDrawInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				Operations: []BatchOperationInput{
					{Type: "line", Color: "#FF0000FF", FrameNumber: -1},
				},
			},
			wantErr:     true,
			errContains: "operations[0] (line): frame_number must be at least 1, got -1",
		},
		{
			name: "missing color on non-pixels operation",
			input: BatchDrawInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				Operations: []BatchOperationInput{
					{Type: "line", X1: 0, Y1: 0, X2: 5, Y2: 5},
				},
			},
			wantErr:     true,
			errContains: "operations[0] (line): color is required",
		},
		{
			name: "invalid color format on operation",
			input: BatchDrawInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				Operations: []BatchOperationInput{
					{Type: "circle", Color: "invalid-color", Radius: 5},
				},
			},
			wantErr:     true,
			errContains: "operations[0] (circle): invalid color format",
		},
		{
			name: "line thickness less than 1 (got 0)",
			input: BatchDrawInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				Operations: []BatchOperationInput{
					{Type: "line", Color: "#FF0000FF", Thickness: intPtr(0)},
				},
			},
			wantErr:     true,
			errContains: "operations[0] (line): thickness must be between 1 and 100, got 0",
		},
		{
			name: "line thickness greater than 100",
			input: BatchDrawInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				Operations: []BatchOperationInput{
					{Type: "line", Color: "#FF0000FF", Thickness: intPtr(101)},
				},
			},
			wantErr:     true,
			errContains: "operations[0] (line): thickness must be between 1 and 100, got 101",
		},
		{
			name: "rectangle width less than 1",
			input: BatchDrawInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				Operations: []BatchOperationInput{
					{Type: "rectangle", Color: "#FF0000FF", Width: 0, Height: 10},
				},
			},
			wantErr:     true,
			errContains: "operations[0] (rectangle): width and height must be at least 1, got width=0 height=10",
		},
		{
			name: "rectangle height less than 1",
			input: BatchDrawInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				Operations: []BatchOperationInput{
					{Type: "rectangle", Color: "#FF0000FF", Width: 10, Height: 0},
				},
			},
			wantErr:     true,
			errContains: "operations[0] (rectangle): width and height must be at least 1, got width=10 height=0",
		},
		{
			name: "circle radius less than 1",
			input: BatchDrawInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				Operations: []BatchOperationInput{
					{Type: "circle", Color: "#FF0000FF", Radius: 0},
				},
			},
			wantErr:     true,
			errContains: "operations[0] (circle): radius must be at least 1, got 0",
		},
		{
			name: "contour points fewer than 2",
			input: BatchDrawInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				Operations: []BatchOperationInput{
					{Type: "contour", Color: "#FF0000FF", Points: []PointInput{{X: 0, Y: 0}}},
				},
			},
			wantErr:     true,
			errContains: "operations[0] (contour): at least 2 points are required, got 1",
		},
		{
			name: "contour thickness less than 1",
			input: BatchDrawInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				Operations: []BatchOperationInput{
					{
						Type:      "contour",
						Color:     "#FF0000FF",
						Points:    []PointInput{{X: 0, Y: 0}, {X: 5, Y: 5}},
						Thickness: intPtr(0),
					},
				},
			},
			wantErr:     true,
			errContains: "operations[0] (contour): thickness must be between 1 and 100, got 0",
		},
		{
			name: "fill tolerance less than 0",
			input: BatchDrawInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				Operations: []BatchOperationInput{
					{Type: "fill", Color: "#FF0000FF", Tolerance: -1},
				},
			},
			wantErr:     true,
			errContains: "operations[0] (fill): tolerance must be between 0 and 255, got -1",
		},
		{
			name: "fill tolerance greater than 255",
			input: BatchDrawInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				Operations: []BatchOperationInput{
					{Type: "fill", Color: "#FF0000FF", Tolerance: 256},
				},
			},
			wantErr:     true,
			errContains: "operations[0] (fill): tolerance must be between 0 and 255, got 256",
		},
		{
			name: "pixels array empty",
			input: BatchDrawInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				Operations: []BatchOperationInput{
					{Type: "pixels", Pixels: []PixelInput{}},
				},
			},
			wantErr:     true,
			errContains: "operations[0] (pixels): pixels array cannot be empty",
		},
		{
			name: "invalid pixel color in pixels array",
			input: BatchDrawInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				Operations: []BatchOperationInput{
					{
						Type: "pixels",
						Pixels: []PixelInput{
							{X: 0, Y: 0, Color: "#FF0000FF"},
							{X: 1, Y: 1, Color: "bad-hex"},
						},
					},
				},
			},
			wantErr:     true,
			errContains: "operations[0] (pixels): invalid color format for pixel 1",
		},
		{
			name: "naming operation index correctly in multi-op batch",
			input: BatchDrawInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				Operations: []BatchOperationInput{
					{Type: "pixels", Pixels: []PixelInput{{X: 0, Y: 0, Color: "#FF0000FF"}}},
					{Type: "fill", Color: "#00FF00FF"},
					{Type: "line", Color: "#0000FFFF", Thickness: intPtr(0)},
				},
			},
			wantErr:     true,
			errContains: "operations[2] (line): thickness must be between 1 and 100, got 0",
		},
		{
			name: "exceeds maximum operations limit (1001 operations)",
			input: BatchDrawInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				Operations: func() []BatchOperationInput {
					ops := make([]BatchOperationInput, 1001)
					for i := range ops {
						ops[i] = BatchOperationInput{Type: "fill", Color: "#FF0000FF"}
					}
					return ops
				}(),
			},
			wantErr:     true,
			errContains: "operations must contain between 1 and 1000 items, got 1001",
		},
		{
			name: "exceeds total pixels limit (65537 pixels)",
			input: BatchDrawInput{
				SpritePath:  "/path/to/sprite.aseprite",
				LayerName:   "Layer 1",
				FrameNumber: 1,
				Operations: func() []BatchOperationInput {
					pixels := make([]PixelInput, 65537)
					for i := range pixels {
						pixels[i] = PixelInput{X: i % 100, Y: i / 100, Color: "#FF0000FF"}
					}
					return []BatchOperationInput{
						{Type: "pixels", Pixels: pixels},
					}
				}(),
			},
			wantErr:     true,
			errContains: "total pixels across all operations cannot exceed 65536, got 65537",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBatchInput(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.True(t, strings.Contains(err.Error(), tt.errContains),
						"expected error to contain %q, got %q", tt.errContains, err.Error())
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRegisterBatchTools(t *testing.T) {
	server, client, gen := createTestServer(t)
	cfg := testutil.LoadTestConfig(t)
	logger := mtlog.New(mtlog.WithMinimumLevel(core.ErrorLevel))

	RegisterBatchTools(server, client, gen, cfg, logger)

	assert.NotNil(t, server)
}
