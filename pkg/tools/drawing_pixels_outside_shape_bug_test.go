//go:build integration

package tools

import (
	"context"
	"testing"
	"time"

	"github.com/willibrandon/pixel-mcp/internal/testutil"
	"github.com/willibrandon/pixel-mcp/pkg/aseprite"
)

// TestIntegration_DrawPixels_AfterShape_OutsideShapeBounds verifies that
// draw_pixels can place a pixel anywhere on the sprite after a shape tool has
// trimmed the cel to a non-(0,0), ASYMMETRIC origin — including a coordinate
// OUTSIDE the shape's bounding box.
//
// This covers two facets the simple offset test does not:
//   - asymmetric origin (x != y), proving both axes are handled independently
//   - a pixel outside the trimmed cel, which the stock code dropped entirely
//     (writing to cel-local coords that map elsewhere, leaving the requested
//     coordinate transparent)
func TestIntegration_DrawPixels_AfterShape_OutsideShapeBounds(t *testing.T) {
	cfg := testutil.LoadTestConfig(t)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	spritePath := testutil.TempSpritePath(t, "test-pixels-outside-shape.aseprite")
	createScript := gen.CreateCanvas(32, 32, aseprite.ColorModeRGB, spritePath)
	if _, err := client.ExecuteLua(ctx, createScript, ""); err != nil {
		t.Fatalf("Failed to create canvas: %v", err)
	}

	// Asymmetric, inset rectangle -> cel origin becomes ≈(10,6) after trimming,
	// so the x and y offsets differ.
	rect := aseprite.Color{R: 0, G: 170, B: 255, A: 255} // #00AAFF
	rectScript := gen.DrawRectangle("Layer 1", 1, 10, 6, 8, 8, rect, true, false)
	if _, err := client.ExecuteLua(ctx, rectScript, spritePath); err != nil {
		t.Fatalf("Failed to draw rectangle: %v", err)
	}

	// One pixel inside the rect, one OUTSIDE it near the top-left corner.
	red := aseprite.Color{R: 255, G: 0, B: 0, A: 255}   // #FF0000 inside
	green := aseprite.Color{R: 0, G: 255, B: 0, A: 255} // #00FF00 outside
	pixels := []aseprite.Pixel{
		{Point: aseprite.Point{X: 12, Y: 8}, Color: red},
		{Point: aseprite.Point{X: 1, Y: 1}, Color: green},
	}
	drawScript := gen.DrawPixels("Layer 1", 1, pixels, false)
	if _, err := client.ExecuteLua(ctx, drawScript, spritePath); err != nil {
		t.Fatalf("Failed to draw pixels: %v", err)
	}

	assertPixel := func(x, y int, want string) {
		t.Helper()
		out, err := client.ExecuteLua(ctx, gen.GetPixels("Layer 1", 1, x, y, 1, 1), spritePath)
		if err != nil {
			t.Fatalf("get_pixels(%d,%d): %v", x, y, err)
		}
		got, err := testutil.ParsePixelData(out)
		if err != nil {
			t.Fatalf("parse (%d,%d): %v", x, y, err)
		}
		if len(got) != 1 || got[0].Color != want {
			t.Errorf("BUG CONFIRMED: pixel (%d,%d) = %v, want %s", x, y, got, want)
		}
	}

	assertPixel(12, 8, "#FF0000FF") // inside the shape: placed at the right spot
	assertPixel(1, 1, "#00FF00FF")  // outside the shape: present, not lost
}
