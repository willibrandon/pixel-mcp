//go:build integration
// +build integration

package tools

import (
	"context"
	"testing"
	"time"

	"github.com/willibrandon/pixel-mcp/internal/testutil"
	"github.com/willibrandon/pixel-mcp/pkg/aseprite"
)

// TestIntegration_SetPalette_TransparentColorCollisionBug tests the bug where
// SetPalette's Palette:resize call causes Aseprite to silently clamp
// Sprite.transparentColor down to the new last valid index (paletteSize-1),
// colliding it with the last real color the caller just set.
//
// CreateCanvas sets Sprite.transparentColor to 255 for indexed sprites so
// that palette index 0 can hold a real color instead of being treated as
// transparent (see the palette index 0 fix). Palette:resize(N) does not
// preserve that: Aseprite clamps transparentColor into [0, N-1], so after
// SetPalette with a locked N-color palette, transparentColor becomes N-1.
//
// EXACT REPRODUCTION:
//  1. Create an indexed sprite and set a 4-color palette: RED, GREEN, BLUE,
//     YELLOW. YELLOW is palette index 3, the palette's last color.
//  2. Draw a YELLOW rectangle (using the last palette color).
//  3. Fill an unrelated background region with a fully transparent color via
//     fill_area, exactly like drawing a locked-palette sprite's background.
//  4. Read back both regions with get_pixels.
//
// BUG: because transparentColor collided with YELLOW's index, the "transparent"
// fill paints pixels that read back as opaque YELLOW instead of transparent,
// and/or the intentionally-drawn YELLOW rectangle gets treated as transparent.
// Either way, YELLOW (the palette's last real color) becomes indistinguishable
// from "no color here."
func TestIntegration_SetPalette_TransparentColorCollisionBug(t *testing.T) {
	cfg := testutil.LoadTestConfig(t)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	spritePath := testutil.TempSpritePath(t, "test-transparent-color-collision.aseprite")
	createScript := gen.CreateCanvas(32, 32, aseprite.ColorModeIndexed, spritePath)
	if _, err := client.ExecuteLua(ctx, createScript, ""); err != nil {
		t.Fatalf("Failed to create canvas: %v", err)
	}

	// 4-color palette; YELLOW (index 3) is the last real color.
	setPaletteScript := gen.SetPalette([]string{
		"#FF0000FF", // Index 0: RED
		"#00FF00FF", // Index 1: GREEN
		"#0000FFFF", // Index 2: BLUE
		"#FFFF00FF", // Index 3: YELLOW (last color -> historically collided with transparentColor)
	})
	if _, err := client.ExecuteLua(ctx, setPaletteScript, spritePath); err != nil {
		t.Fatalf("Failed to set palette: %v", err)
	}

	// Draw a YELLOW rectangle at (0,0)-(3,3), using the palette's last color.
	drawYellowScript := gen.DrawRectangle("Layer 1", 1, 0, 0, 4, 4,
		aseprite.Color{R: 255, G: 255, B: 0, A: 255}, true, true)
	if _, err := client.ExecuteLua(ctx, drawYellowScript, spritePath); err != nil {
		t.Fatalf("Failed to draw YELLOW rectangle: %v", err)
	}

	// Fill an unrelated background pixel with a fully transparent color,
	// exactly like a locked-palette sprite establishing a transparent background.
	fillScript := gen.FillArea("Layer 1", 1, 20, 20, aseprite.Color{R: 0, G: 0, B: 0, A: 0}, 0, false)
	if _, err := client.ExecuteLua(ctx, fillScript, spritePath); err != nil {
		t.Fatalf("Failed to fill background: %v", err)
	}

	getPixelsScript := gen.GetPixels("Layer 1", 1, 0, 0, 32, 32)
	output, err := client.ExecuteLua(ctx, getPixelsScript, spritePath)
	if err != nil {
		// One observed manifestation of this bug: once transparentColor
		// collides with the only real color drawn, Aseprite's cel trimming
		// treats the entire cel as fully transparent and deletes it outright,
		// so get_pixels has no cel left to read and errors out.
		t.Fatalf("BUG CONFIRMED: get_pixels failed after the transparent fill, "+
			"the drawn content was likely trimmed away as transparent: %v", err)
	}

	pixels, err := testutil.ParsePixelData(output)
	if err != nil {
		t.Fatalf("Failed to parse pixel data: %v", err)
	}

	pixelAt := make(map[[2]int]string, len(pixels))
	for _, p := range pixels {
		pixelAt[[2]int{p.X, p.Y}] = p.Color
	}

	// The YELLOW rectangle must still read back as opaque YELLOW.
	if c, ok := pixelAt[[2]int{1, 1}]; !ok || c != "#FFFF00FF" {
		t.Errorf("BUG CONFIRMED: expected opaque YELLOW (#FFFF00FF) at (1,1), got %q (present=%v) - "+
			"the palette's last color was treated as transparent", c, ok)
	}

	// The background fill must read back as transparent, not opaque YELLOW.
	if c, ok := pixelAt[[2]int{20, 20}]; ok {
		t.Errorf("BUG CONFIRMED: expected background pixel (20,20) to be transparent, got opaque color %q - "+
			"transparentColor collided with a real palette color", c)
	}
}
