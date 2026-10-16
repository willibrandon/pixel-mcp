package aseprite

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// ExportPreview renders a sprite to a PNG with pixel-perfect nearest-neighbor scaling.
// Scale specifies the integer magnification factor (e.g., 4x or 8x). If scale <= 0, defaults to 4.
// Frame specifies 1-based frame number; if 0, defaults to all/active frame.
// LayerName optionally isolates a single layer.
func (c *Client) ExportPreview(ctx context.Context, spritePath string, scale int, frame int, layerName string) ([]byte, error) {
	if scale <= 0 {
		scale = 4
	}
	if scale > 32 {
		scale = 32
	}

	tempDir, err := os.MkdirTemp(c.tempDir, "pixel-mcp-preview-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory for preview: %w", err)
	}
	defer os.RemoveAll(tempDir)

	previewPath := filepath.Join(tempDir, "preview.png")

	var args []string
	args = append(args, "--batch", spritePath)
	if layerName != "" {
		args = append(args, "--sheet-layer", layerName)
	}
	if frame > 0 {
		// Aseprite CLI frame range is 0-indexed
		args = append(args, "--frame-range", fmt.Sprintf("%d,%d", frame-1, frame-1))
	}
	args = append(args, "--scale", fmt.Sprintf("%d", scale), "--save-as", previewPath)

	if _, err := c.ExecuteCommand(ctx, args); err != nil {
		return nil, fmt.Errorf("failed to render preview: %w", err)
	}

	data, err := os.ReadFile(previewPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rendered preview: %w", err)
	}

	return data, nil
}
