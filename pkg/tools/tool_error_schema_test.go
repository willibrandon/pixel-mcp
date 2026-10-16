package tools

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willibrandon/mtlog"
	"github.com/willibrandon/mtlog/core"
	"github.com/willibrandon/pixel-mcp/internal/testutil"
	"github.com/willibrandon/pixel-mcp/pkg/aseprite"
)

// TestToolError_ReachesClientAsIsError verifies that ToolError results arrive as
// isError tool results rather than JSON-RPC errors.
//
// ToolError returns a nil output, and the SDK validates a zero-value output struct
// against the tool's inferred output schema. A slice field without omitempty
// serializes as null and fails that validation, which turns the tool error into
// an opaque "validating tool output" protocol error.
func TestToolError_ReachesClientAsIsError(t *testing.T) {
	cfg := testutil.LoadTestConfig(t)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, cfg.Timeout)
	gen := aseprite.NewLuaGenerator()
	logger := mtlog.New(mtlog.WithMinimumLevel(core.ErrorLevel))

	server := mcp.NewServer(&mcp.Implementation{Name: "pixel-mcp-test", Version: "1.0.0"}, nil)
	RegisterColorRampTools(server, client, gen, cfg, logger)
	RegisterAutoShadingTools(server, client, gen, cfg, logger)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	_, err := server.Connect(context.Background(), serverTransport, nil)
	require.NoError(t, err)
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil).Connect(context.Background(), clientTransport, nil)
	require.NoError(t, err)
	defer session.Close()

	tests := []struct {
		name      string
		tool      string
		args      map[string]any
		wantInMsg string
	}{
		{
			name:      "generate_color_ramp with invalid base color",
			tool:      "generate_color_ramp",
			args:      map[string]any{"base_color": "not-a-color", "steps": 5},
			wantInMsg: "failed to generate color ramp",
		},
		{
			name: "apply_auto_shading with missing sprite",
			tool: "apply_auto_shading",
			args: map[string]any{
				"sprite_path":     "/nonexistent/sprite.aseprite",
				"layer_name":      "Layer 1",
				"frame_number":    1,
				"light_direction": "top_left",
				"intensity":       0.5,
				"style":           "cell",
				"hue_shift":       true,
			},
			wantInMsg: "sprite file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tt.tool, Arguments: tt.args})
			require.NoError(t, err, "tool errors must not surface as JSON-RPC errors")
			require.True(t, result.IsError)
			require.NotEmpty(t, result.Content)
			text, ok := result.Content[0].(*mcp.TextContent)
			require.True(t, ok)
			assert.Contains(t, text.Text, tt.wantInMsg)
		})
	}
}
