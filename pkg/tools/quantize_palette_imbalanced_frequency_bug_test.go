//go:build integration
// +build integration

package tools

import (
	"context"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/willibrandon/pixel-mcp/internal/testutil"
	"github.com/willibrandon/pixel-mcp/pkg/aseprite"
)

// TestIntegration_QuantizePalette_ImbalancedFrequencyPreservesExactColors
// reproduces the median-cut split-by-pixel-count bug end to end through the
// real quantize_palette pipeline: export sprite -> aseprite.QuantizePalette
// (Go clustering) -> ApplyQuantizedPalette (Lua, converts to indexed).
//
// The bug: MedianCutQuantization's bucket split picked the pixel-array
// midpoint (mid := len(pixels) / 2) as the split point, not a boundary
// between distinct colors. A typical pixel-art sprite has wildly imbalanced
// color frequency - large solid fills (skin, clothing) vastly outnumber
// small accent details (a 1-2px highlight or buckle) - so splitting at the
// raw pixel-count median lands the cut inside the dominant fill color's own
// run instead of between colors. The dominant color gets needlessly
// re-subdivided into near-duplicate buckets while rare accent colors never
// get isolated into their own bucket and are averaged into a blended,
// off-palette hex value instead, even when target_colors comfortably
// exceeds the number of unique colors actually present (which should
// guarantee lossless recovery).
//
// This is the exact shape that surfaced the bug on a real 32x32 hero sprite:
// a ~200px tunic fill, a ~90px skin fill, a thin outline, and 2px highlight/
// buckle accents.
func TestIntegration_QuantizePalette_ImbalancedFrequencyPreservesExactColors(t *testing.T) {
	cfg := testutil.LoadTestConfig(t)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	spritePath := testutil.TempSpritePath(t, "test-quantize-imbalanced.aseprite")
	createScript := gen.CreateCanvas(32, 32, aseprite.ColorModeRGB, spritePath)
	if _, err := client.ExecuteLua(ctx, createScript, ""); err != nil {
		t.Fatalf("Failed to create canvas: %v", err)
	}

	tunicGreen := aseprite.Color{R: 0x6A, G: 0xBE, B: 0x30, A: 255}
	skin := aseprite.Color{R: 0xD9, G: 0xA0, B: 0x66, A: 255}
	outline := aseprite.Color{R: 0x22, G: 0x20, B: 0x34, A: 255}
	highlight := aseprite.Color{R: 0x99, G: 0xE5, B: 0x50, A: 255}
	buckle := aseprite.Color{R: 0xFB, G: 0xF2, B: 0x36, A: 255}

	var pixels []aseprite.Pixel
	add := func(x, y int, c aseprite.Color) {
		pixels = append(pixels, aseprite.Pixel{Point: aseprite.Point{X: x, Y: y}, Color: c})
	}
	for y := 11; y < 21; y++ { // tunic: 20 x 10 = 200px
		for x := 6; x < 26; x++ {
			add(x, y, tunicGreen)
		}
	}
	for y := 4; y < 10; y++ { // skin: 16 x 6 = 96px
		for x := 6; x < 22; x++ {
			add(x, y, skin)
		}
	}
	for x := 6; x < 26; x++ { // outline: 20px
		add(x, 2, outline)
	}
	add(15, 14, highlight) // 2px accent
	add(16, 14, highlight)
	add(15, 20, buckle) // 2px accent
	add(16, 20, buckle)

	drawScript := gen.DrawPixels("Layer 1", 1, pixels, false)
	if _, err := client.ExecuteLua(ctx, drawScript, spritePath); err != nil {
		t.Fatalf("Failed to draw pixels: %v", err)
	}

	// Real pipeline: export to PNG, quantize in Go, apply via Lua - matches
	// pkg/tools/quantization.go's actual quantize_palette tool flow.
	tempPNG := filepath.Join(t.TempDir(), "sprite.png")
	exportScript := gen.ExportSprite(tempPNG, 0)
	if _, err := client.ExecuteLua(ctx, exportScript, spritePath); err != nil {
		t.Fatalf("Failed to export sprite: %v", err)
	}
	imgFile, err := os.Open(tempPNG)
	if err != nil {
		t.Fatalf("Failed to open exported PNG: %v", err)
	}
	defer imgFile.Close()
	img, err := png.Decode(imgFile)
	if err != nil {
		t.Fatalf("Failed to decode PNG: %v", err)
	}

	// target_colors (32) comfortably exceeds the 5 unique colors present, so
	// every one of them must survive quantization exactly.
	palette, originalColors, err := aseprite.QuantizePalette(img, 32, "median_cut", true)
	if err != nil {
		t.Fatalf("QuantizePalette failed: %v", err)
	}

	want := map[string]bool{
		"#00000000": true, "#6ABE30": true, "#D9A066": true, "#222034": true,
		"#99E550": true, "#FBF236": true,
	}
	if len(palette) != len(want) {
		t.Fatalf("BUG CONFIRMED: QuantizePalette(target=32) returned %d colors from %d unique input colors, want %d (lossless recovery): %v",
			len(palette), originalColors, len(want), palette)
	}
	for _, c := range palette {
		if !want[c] {
			t.Errorf("BUG CONFIRMED: palette contains %s, not one of the 6 exact input colors (blended/off-palette result)", c)
		}
	}

	applyScript := gen.ApplyQuantizedPalette(palette, originalColors, "median_cut", true, false)
	if _, err := client.ExecuteLua(ctx, applyScript, spritePath); err != nil {
		t.Fatalf("Failed to apply quantized palette: %v", err)
	}

	getPixelsScript := gen.GetPixels("Layer 1", 1, 0, 0, 32, 32)
	output, err := client.ExecuteLua(ctx, getPixelsScript, spritePath)
	if err != nil {
		t.Fatalf("Failed to get pixels after conversion: %v", err)
	}
	got, err := testutil.ParsePixelData(output)
	if err != nil {
		t.Fatalf("Failed to parse pixel data: %v", err)
	}
	pixelAt := make(map[[2]int]string, len(got))
	for _, p := range got {
		pixelAt[[2]int{p.X, p.Y}] = p.Color
	}

	wantPixels := map[[2]int]string{
		{0, 0}:   "#00000000",
		{10, 15}: "#6ABE30FF",
		{10, 6}:  "#D9A066FF",
		{10, 2}:  "#222034FF",
		{15, 14}: "#99E550FF",
		{15, 20}: "#FBF236FF",
	}
	for pos, want := range wantPixels {
		got, ok := pixelAt[pos]
		if want == "#00000000" {
			if ok && got != "#00000000" {
				t.Errorf("BUG CONFIRMED: pixel %v should be transparent, got opaque %q", pos, got)
			}
			continue
		}
		if !ok {
			t.Errorf("BUG CONFIRMED: pixel %v (want %s) missing from converted sprite", pos, want)
			continue
		}
		if got != want {
			t.Errorf("BUG CONFIRMED: pixel %v = %q, want %q", pos, got, want)
		}
	}
}
