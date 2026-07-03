//go:build integration

package tools

import (
	"context"
	"testing"
	"time"

	"github.com/willibrandon/pixel-mcp/internal/testutil"
	"github.com/willibrandon/pixel-mcp/pkg/aseprite"
)

// TestIntegration_DrawPixels_AfterShape_CelOffset reproduces the bug where
// draw_pixels places pixels at the wrong coordinates once a shape tool has
// caused the layer's cel to be trimmed to a non-(0,0) origin.
//
// A filled circle drawn in the middle of the canvas leaves the cel positioned
// at the circle's bounding-box top-left (not the sprite origin). DrawPixels then
// wrote with cel-LOCAL coordinates via Image:putPixel, so a pixel requested at
// sprite (16,16) actually landed at (16 + cel.x, 16 + cel.y). This test asserts
// the pixel ends up exactly where it was requested.
func TestIntegration_DrawPixels_AfterShape_CelOffset(t *testing.T) {
	cfg := testutil.LoadTestConfig(t)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	spritePath := testutil.TempSpritePath(t, "test-pixels-after-shape.aseprite")
	createScript := gen.CreateCanvas(32, 32, aseprite.ColorModeRGB, spritePath)
	if _, err := client.ExecuteLua(ctx, createScript, ""); err != nil {
		t.Fatalf("Failed to create canvas: %v", err)
	}

	// Draw a filled circle in the center. After save, Aseprite trims the cel to
	// this content, so the cel origin becomes non-zero (≈(2,2) for r14 on 32px).
	circleColor := aseprite.Color{R: 0, G: 170, B: 255, A: 255} // #00AAFF
	circleScript := gen.DrawCircle("Layer 1", 1, 16, 16, 14, circleColor, true, false)
	if _, err := client.ExecuteLua(ctx, circleScript, spritePath); err != nil {
		t.Fatalf("Failed to draw circle: %v", err)
	}

	// Draw a single distinct pixel at a known sprite coordinate inside the circle.
	dotColor := aseprite.Color{R: 255, G: 0, B: 0, A: 255} // #FF0000
	pixels := []aseprite.Pixel{
		{Point: aseprite.Point{X: 16, Y: 16}, Color: dotColor},
	}
	drawScript := gen.DrawPixels("Layer 1", 1, pixels, false)
	if _, err := client.ExecuteLua(ctx, drawScript, spritePath); err != nil {
		t.Fatalf("Failed to draw pixel: %v", err)
	}

	// The pixel must be exactly at sprite (16,16), not offset by the cel origin.
	getScript := gen.GetPixels("Layer 1", 1, 16, 16, 1, 1)
	output, err := client.ExecuteLua(ctx, getScript, spritePath)
	if err != nil {
		t.Fatalf("Failed to get pixels: %v", err)
	}
	got, err := testutil.ParsePixelData(output)
	if err != nil {
		t.Fatalf("Failed to parse pixel data: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 pixel for region (16,16,1,1), got %d", len(got))
	}
	if got[0].Color != "#FF0000FF" {
		t.Errorf("BUG CONFIRMED: pixel at sprite (16,16) = %s, want #FF0000FF "+
			"(draw_pixels offset by the cel origin)", got[0].Color)
	}
}
