package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/willibrandon/pixel-mcp/pkg/config"
)

func TestMain_ConfigOptions(t *testing.T) {
	tempDir := t.TempDir()

	opts := config.Options{
		ConfigFile:   filepath.Join(tempDir, "nonexistent.json"),
		AsepritePath: "/nonexistent/path/to/aseprite",
		TempDir:      tempDir,
		Timeout:      15 * time.Second,
		LogLevel:     "debug",
	}

	// Should fail validation because aseprite path does not exist
	cfg, err := config.LoadWithOptions(opts)
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestMain_CreateLogger(t *testing.T) {
	logger := createLogger("debug", "")
	assert.NotNil(t, logger)
	closeLogger(logger)
}
