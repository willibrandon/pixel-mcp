//go:build integration

package tools

import (
	"context"
	"testing"
	"time"

	"github.com/willibrandon/pixel-mcp/internal/testutil"
	"github.com/willibrandon/pixel-mcp/pkg/aseprite"
)

// TestIntegration_DrawPixels_TransparentBackground_Bug reproduces the bug where a
// later draw_pixels call silently corrupts pixels that an earlier call had
// explicitly made transparent, flipping them to opaque palette-index-0 (e.g.
// opaque black on a DawnBringer 32 sprite).
//
// BUG SCENARIO (reported while drawing DB32 sprites across multiple calls):
//  1. On an indexed sprite CreateCanvas sets transparentColor = 255, so the
//     transparent index is 255 and index 0 is a real, opaque palette color.
//  2. A draw_pixels call explicitly writes transparent pixels (alpha 0, which
//     resolves to index 255) at some coordinates. Those read back as fully
//     transparent.
//  3. A SUBSEQUENT draw_pixels call normalizes the cel to a full-canvas image via
//     `Image(spr.width, spr.height, spr.colorMode)`, which is filled with index 0
//     (opaque), NOT the transparent index. With the default NORMAL blend,
//     `full:drawImage(cel.image, cel.position)` skips the source's transparent
//     (index 255) pixels, leaving the opaque index-0 background showing through.
//  4. RESULT: the previously-transparent pixels the second call did not rewrite
//     flip from transparent (#00000000) to opaque black (#000000FF).
//
// The fix clears the full-canvas image to the sprite's transparent color before
// compositing, so index-255 pixels survive the round-trip.
//
// Palette index 0 is opaque black (#000000FF) so corruption is unambiguous: a
// truly transparent pixel reads back as #00000000 (index 255 is outside the
// 3-color palette's addressable range), a corrupted pixel reads back as
// #000000FF.
func TestIntegration_DrawPixels_TransparentBackground_Bug(t *testing.T) {
	cfg := testutil.LoadTestConfig(t)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	spritePath := testutil.TempSpritePath(t, "test-pixels-transparent-bg.aseprite")
	createScript := gen.CreateCanvas(8, 8, aseprite.ColorModeIndexed, spritePath)
	if _, err := client.ExecuteLua(ctx, createScript, ""); err != nil {
		t.Fatalf("Failed to create canvas: %v", err)
	}

	setPaletteScript := gen.SetPalette([]string{
		"#000000FF", // Index 0: opaque BLACK (what corruption collapses to)
		"#FF0000FF", // Index 1: RED
		"#00FF00FF", // Index 2: GREEN
	})
	if _, err := client.ExecuteLua(ctx, setPaletteScript, spritePath); err != nil {
		t.Fatalf("Failed to set palette: %v", err)
	}

	const transparent = "#00000000"

	colorMap := func(t *testing.T) map[string]string {
		t.Helper()
		output, err := client.ExecuteLua(ctx, gen.GetPixels("Layer 1", 1, 0, 0, 8, 8), spritePath)
		if err != nil {
			t.Fatalf("Failed to get pixels: %v", err)
		}
		pixels, err := testutil.ParsePixelData(output)
		if err != nil {
			t.Fatalf("Failed to parse pixel data: %v", err)
		}
		got := make(map[string]string, len(pixels))
		for _, p := range pixels {
			got[testutil.FormatPixelPos(p.X, p.Y)] = p.Color
		}
		return got
	}

	// Pixels this test explicitly makes transparent in the first call. They must
	// remain transparent after the second call.
	transparentPixels := []aseprite.Point{{X: 2, Y: 2}, {X: 3, Y: 3}, {X: 4, Y: 4}, {X: 6, Y: 6}}

	// First call: a red marker plus an explicitly-transparent set.
	firstBatch := []aseprite.Pixel{
		{Point: aseprite.Point{X: 0, Y: 0}, Color: aseprite.Color{R: 255, G: 0, B: 0, A: 255}},
	}
	for _, p := range transparentPixels {
		firstBatch = append(firstBatch, aseprite.Pixel{Point: p, Color: aseprite.Color{R: 0, G: 0, B: 0, A: 0}})
	}
	if _, err := client.ExecuteLua(ctx, gen.DrawPixels("Layer 1", 1, firstBatch, false), spritePath); err != nil {
		t.Fatalf("Failed to draw first batch: %v", err)
	}

	// Sanity: the explicitly-transparent pixels are transparent after call 1.
	got := colorMap(t)
	for _, p := range transparentPixels {
		pos := testutil.FormatPixelPos(p.X, p.Y)
		if got[pos] != transparent {
			t.Fatalf("setup failed: pixel (%d,%d) = %s after first draw, want %s", p.X, p.Y, got[pos], transparent)
		}
	}

	// Second call: an unrelated edit elsewhere (opaque green) plus a transparent
	// write (the exact scenario from the report). This must NOT disturb the
	// already-transparent pixels from the first call.
	secondBatch := []aseprite.Pixel{
		{Point: aseprite.Point{X: 7, Y: 7}, Color: aseprite.Color{R: 0, G: 255, B: 0, A: 255}}, // opaque green
		{Point: aseprite.Point{X: 1, Y: 1}, Color: aseprite.Color{R: 0, G: 0, B: 0, A: 0}},     // transparent
	}
	if _, err := client.ExecuteLua(ctx, gen.DrawPixels("Layer 1", 1, secondBatch, false), spritePath); err != nil {
		t.Fatalf("Failed to draw second batch: %v", err)
	}

	got = colorMap(t)

	// The reported bug: previously-transparent pixels corrupted to opaque black.
	corrupted := 0
	for _, p := range transparentPixels {
		pos := testutil.FormatPixelPos(p.X, p.Y)
		if got[pos] != transparent {
			if got[pos] == "#000000FF" {
				corrupted++
			}
			t.Errorf("pixel (%d,%d) = %s after second draw, want %s (transparent)", p.X, p.Y, got[pos], transparent)
		}
	}
	if corrupted > 0 {
		t.Logf("BUG CONFIRMED: %d previously-transparent pixels were corrupted to opaque black (#000000FF)", corrupted)
	}

	// The foreground pixels must be intact.
	if got["0,0"] != "#FF0000FF" {
		t.Errorf("red marker (0,0) = %s, want #FF0000FF", got["0,0"])
	}
	if got["7,7"] != "#00FF00FF" {
		t.Errorf("green marker (7,7) = %s, want #00FF00FF", got["7,7"])
	}
}

// TestIntegration_CreateCanvas_IndexedBackgroundTransparent verifies that a
// freshly created indexed canvas starts fully transparent rather than opaque
// palette-index-0.
//
// A new indexed sprite's initial cel is filled with index 0. Because CreateCanvas
// designates index 255 as transparent, index 0 is an opaque color, so without
// clearing the initial cel the whole canvas reads back as opaque black
// (#000000FF) instead of transparent (#00000000).
func TestIntegration_CreateCanvas_IndexedBackgroundTransparent(t *testing.T) {
	cfg := testutil.LoadTestConfig(t)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	spritePath := testutil.TempSpritePath(t, "test-fresh-indexed-transparent.aseprite")
	if _, err := client.ExecuteLua(ctx, gen.CreateCanvas(4, 4, aseprite.ColorModeIndexed, spritePath), ""); err != nil {
		t.Fatalf("Failed to create canvas: %v", err)
	}

	// Index 0 = opaque black; a corrupted (uncleared) background collapses to it.
	setPaletteScript := gen.SetPalette([]string{"#000000FF", "#FF0000FF"})
	if _, err := client.ExecuteLua(ctx, setPaletteScript, spritePath); err != nil {
		t.Fatalf("Failed to set palette: %v", err)
	}

	output, err := client.ExecuteLua(ctx, gen.GetPixels("Layer 1", 1, 0, 0, 4, 4), spritePath)
	if err != nil {
		t.Fatalf("Failed to get pixels: %v", err)
	}
	pixels, err := testutil.ParsePixelData(output)
	if err != nil {
		t.Fatalf("Failed to parse pixel data: %v", err)
	}
	if len(pixels) != 16 {
		t.Fatalf("Expected 16 pixels for a 4x4 canvas, got %d", len(pixels))
	}

	opaqueBlack := 0
	for _, p := range pixels {
		if p.Color != "#00000000" {
			if p.Color == "#000000FF" {
				opaqueBlack++
			}
			t.Errorf("fresh-canvas pixel (%d,%d) = %s, want #00000000 (transparent)", p.X, p.Y, p.Color)
		}
	}
	if opaqueBlack > 0 {
		t.Logf("BUG CONFIRMED: %d/16 fresh-canvas pixels are opaque black (#000000FF) instead of transparent", opaqueBlack)
	}
}
