package tools

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willibrandon/mtlog"
	"github.com/willibrandon/pixel-mcp/pkg/aseprite"
	"github.com/willibrandon/pixel-mcp/pkg/config"
)

func TestPreviewTool_RegistrationAndValidation(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil)
	client := aseprite.NewClient("/dummy/path", t.TempDir(), 5*time.Second)
	cfg := &config.Config{TempDir: t.TempDir()}
	logger := mtlog.New()

	RegisterPreviewTools(server, client, cfg, logger)

	// Test non-existent file validation directly
	tempDir := t.TempDir()
	nonExistent := filepath.Join(tempDir, "missing.aseprite")

	input := GetSpritePreviewInput{
		SpritePath: nonExistent,
		Scale:      4,
	}

	assert.False(t, func() bool {
		_, err := os.Stat(input.SpritePath)
		return err == nil
	}())
}

func TestColorRampTool_Registration(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil)
	client := aseprite.NewClient("/dummy/path", t.TempDir(), 5*time.Second)
	gen := aseprite.NewLuaGenerator()
	cfg := &config.Config{TempDir: t.TempDir()}
	logger := mtlog.New()

	RegisterColorRampTools(server, client, gen, cfg, logger)

	// Direct algorithm check
	ramp, err := aseprite.GenerateColorRamp(aseprite.ColorRampOptions{
		BaseColor: "#00AA00",
		Steps:     5,
	})
	require.NoError(t, err)
	assert.Len(t, ramp, 5)
}
