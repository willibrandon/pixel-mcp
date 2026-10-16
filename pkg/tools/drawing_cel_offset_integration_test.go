//go:build integration

package tools

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/willibrandon/pixel-mcp/internal/testutil"
	"github.com/willibrandon/pixel-mcp/pkg/aseprite"
)

// TestIntegration_CelOffset_DrawPixels verifies that DrawPixels works correctly
// on a layer whose cel was previously trimmed by a tool (e.g. DrawRectangle),
// ensuring cel normalization prevents misplaced or dropped pixels.
func TestIntegration_CelOffset_DrawPixels(t *testing.T) {
	tests := []struct {
		name       string
		colorMode  aseprite.ColorMode
		usePalette bool
	}{
		{
			name:       "RGB mode",
			colorMode:  aseprite.ColorModeRGB,
			usePalette: false,
		},
		{
			name:       "Indexed mode",
			colorMode:  aseprite.ColorModeIndexed,
			usePalette: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := testutil.LoadTestConfig(t)
			client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
			gen := aseprite.NewLuaGenerator()
			ctx := context.Background()

			spritePath := testutil.TempSpritePath(t, "test-cel-offset-pixels.aseprite")
			defer os.Remove(spritePath)

			createScript := gen.CreateCanvas(32, 32, tc.colorMode, spritePath)
			if _, err := client.ExecuteLua(ctx, createScript, ""); err != nil {
				t.Fatalf("Failed to create canvas: %v", err)
			}

			if tc.colorMode == aseprite.ColorModeIndexed {
				setPaletteScript := gen.SetPalette([]string{
					"#00000000", // 0: transparent
					"#FF0000FF", // 1: red
					"#00FF00FF", // 2: green
					"#0000FFFF", // 3: blue
				})
				if _, err := client.ExecuteLua(ctx, setPaletteScript, spritePath); err != nil {
					t.Fatalf("Failed to set palette: %v", err)
				}
			}

			addLayerScript := gen.AddLayer("Layer 2")
			if _, err := client.ExecuteLua(ctx, addLayerScript, spritePath); err != nil {
				t.Fatalf("Failed to add layer: %v", err)
			}

			// Draw filled rectangle 4x4 at (10, 10) -> trims cel to (10,10)-(13,13)
			redColor := aseprite.Color{R: 255, G: 0, B: 0, A: 255}
			rectScript := gen.DrawRectangle("Layer 2", 1, 10, 10, 4, 4, redColor, true, tc.usePalette)
			if _, err := client.ExecuteLua(ctx, rectScript, spritePath); err != nil {
				t.Fatalf("Failed to draw rectangle: %v", err)
			}

			// Draw a pixel at (2, 2) with green color
			greenColor := aseprite.Color{R: 0, G: 255, B: 0, A: 255}
			pixelScript := gen.DrawPixels("Layer 2", 1, []aseprite.Pixel{
				{Point: aseprite.Point{X: 2, Y: 2}, Color: greenColor},
			}, tc.usePalette)
			if _, err := client.ExecuteLua(ctx, pixelScript, spritePath); err != nil {
				t.Fatalf("Failed to draw pixel: %v", err)
			}

			// Read back all pixels over the whole 32x32 canvas
			getScript := gen.GetPixels("Layer 2", 1, 0, 0, 32, 32)
			output, err := client.ExecuteLua(ctx, getScript, spritePath)
			if err != nil {
				t.Fatalf("Failed to get pixels: %v", err)
			}

			pixels, err := testutil.ParsePixelData(output)
			if err != nil {
				t.Fatalf("Failed to parse pixel data: %v", err)
			}

			pixelMap := make(map[string]string, len(pixels))
			for _, p := range pixels {
				pixelMap[testutil.FormatPixelPos(p.X, p.Y)] = p.Color
			}

			// Assert (2, 2) has the new color (#00FF00FF)
			if p22 := pixelMap["2,2"]; p22 != "#00FF00FF" {
				t.Errorf("Pixel at (2,2) = %q, expected #00FF00FF", p22)
			}

			// Assert (12, 12) still has the rectangle color (#FF0000FF)
			if p1212 := pixelMap["12,12"]; p1212 != "#FF0000FF" {
				t.Errorf("Pixel at (12,12) = %q, expected #FF0000FF", p1212)
			}

			// Assert all 16 rectangle pixels are intact
			intactCount := 0
			for y := 10; y <= 13; y++ {
				for x := 10; x <= 13; x++ {
					pos := testutil.FormatPixelPos(x, y)
					if pixelMap[pos] == "#FF0000FF" {
						intactCount++
					} else {
						t.Errorf("Rectangle pixel at (%d,%d) = %q, expected #FF0000FF", x, y, pixelMap[pos])
					}
				}
			}
			if intactCount != 16 {
				t.Errorf("Expected 16 intact rectangle pixels, got %d", intactCount)
			}
		})
	}
}

// TestIntegration_CelOffset_DrawWithDither verifies that DrawWithDither works correctly
// on a previously trimmed cel and writes pixels outside the old cel bounds.
func TestIntegration_CelOffset_DrawWithDither(t *testing.T) {
	tests := []struct {
		name      string
		colorMode aseprite.ColorMode
		pattern   string
	}{
		{
			name:      "RGB mode checkerboard",
			colorMode: aseprite.ColorModeRGB,
			pattern:   "checkerboard",
		},
		{
			name:      "RGB mode bayer_2x2",
			colorMode: aseprite.ColorModeRGB,
			pattern:   "bayer_2x2",
		},
		{
			name:      "Indexed mode checkerboard",
			colorMode: aseprite.ColorModeIndexed,
			pattern:   "checkerboard",
		},
		{
			name:      "Indexed mode bayer_2x2",
			colorMode: aseprite.ColorModeIndexed,
			pattern:   "bayer_2x2",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := testutil.LoadTestConfig(t)
			client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
			gen := aseprite.NewLuaGenerator()
			ctx := context.Background()

			spritePath := testutil.TempSpritePath(t, "test-cel-offset-dither.aseprite")
			defer os.Remove(spritePath)

			createScript := gen.CreateCanvas(32, 32, tc.colorMode, spritePath)
			if _, err := client.ExecuteLua(ctx, createScript, ""); err != nil {
				t.Fatalf("Failed to create canvas: %v", err)
			}

			if tc.colorMode == aseprite.ColorModeIndexed {
				setPaletteScript := gen.SetPalette([]string{
					"#00000000", // 0: transparent
					"#FF0000FF", // 1: red
					"#00FF00FF", // 2: green
					"#0000FFFF", // 3: blue
				})
				if _, err := client.ExecuteLua(ctx, setPaletteScript, spritePath); err != nil {
					t.Fatalf("Failed to set palette: %v", err)
				}
			}

			addLayerScript := gen.AddLayer("Layer 2")
			if _, err := client.ExecuteLua(ctx, addLayerScript, spritePath); err != nil {
				t.Fatalf("Failed to add layer: %v", err)
			}

			// Draw filled rectangle 4x4 at (10, 10) -> cel trimmed to (10,10)-(13,13)
			redColor := aseprite.Color{R: 255, G: 0, B: 0, A: 255}
			usePalette := tc.colorMode == aseprite.ColorModeIndexed
			rectScript := gen.DrawRectangle("Layer 2", 1, 10, 10, 4, 4, redColor, true, usePalette)
			if _, err := client.ExecuteLua(ctx, rectScript, spritePath); err != nil {
				t.Fatalf("Failed to draw rectangle: %v", err)
			}

			// Dither over region (0, 0, 8, 8) extending outside the rectangle's bounds
			ditherScript := gen.DrawWithDither("Layer 2", 1, 0, 0, 8, 8, "#0000FF", "#00FF00", tc.pattern, 0.5)
			output, err := client.ExecuteLua(ctx, ditherScript, spritePath)
			if err != nil {
				t.Fatalf("Failed to draw with dither: %v", err)
			}
			if !strings.Contains(output, "Dithering applied successfully") {
				t.Fatalf("Expected dither success, got: %s", output)
			}

			// Read back pixels over (0, 0, 8, 8)
			getScript := gen.GetPixels("Layer 2", 1, 0, 0, 8, 8)
			getOutput, err := client.ExecuteLua(ctx, getScript, spritePath)
			if err != nil {
				t.Fatalf("Failed to get pixels: %v", err)
			}

			pixels, err := testutil.ParsePixelData(getOutput)
			if err != nil {
				t.Fatalf("Failed to parse pixel data: %v", err)
			}

			pixelMap := make(map[string]string, len(pixels))
			for _, p := range pixels {
				pixelMap[testutil.FormatPixelPos(p.X, p.Y)] = p.Color
			}

			// Assert pixels at (0, 0) and (1, 0) (outside old cel) were written (not transparent)
			p00 := pixelMap["0,0"]
			p10 := pixelMap["1,0"]

			if p00 == "" || p00 == "#00000000" {
				t.Errorf("Pixel at (0,0) was not written: got %q", p00)
			}
			if p10 == "" || p10 == "#00000000" {
				t.Errorf("Pixel at (1,0) was not written: got %q", p10)
			}
		})
	}
}

// TestIntegration_CelOffset_DrawPixelsOutOfBounds verifies that DrawPixels skips
// coordinates outside the canvas without throwing an error and successfully writes in-bounds pixels.
func TestIntegration_CelOffset_DrawPixelsOutOfBounds(t *testing.T) {
	tests := []struct {
		name       string
		colorMode  aseprite.ColorMode
		usePalette bool
	}{
		{
			name:       "RGB mode",
			colorMode:  aseprite.ColorModeRGB,
			usePalette: false,
		},
		{
			name:       "Indexed mode",
			colorMode:  aseprite.ColorModeIndexed,
			usePalette: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := testutil.LoadTestConfig(t)
			client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
			gen := aseprite.NewLuaGenerator()
			ctx := context.Background()

			spritePath := testutil.TempSpritePath(t, "test-cel-offset-oob.aseprite")
			defer os.Remove(spritePath)

			createScript := gen.CreateCanvas(32, 32, tc.colorMode, spritePath)
			if _, err := client.ExecuteLua(ctx, createScript, ""); err != nil {
				t.Fatalf("Failed to create canvas: %v", err)
			}

			if tc.colorMode == aseprite.ColorModeIndexed {
				setPaletteScript := gen.SetPalette([]string{
					"#00000000", // 0: transparent
					"#0000FFFF", // 1: blue
				})
				if _, err := client.ExecuteLua(ctx, setPaletteScript, spritePath); err != nil {
					t.Fatalf("Failed to set palette: %v", err)
				}
			}

			addLayerScript := gen.AddLayer("Layer 2")
			if _, err := client.ExecuteLua(ctx, addLayerScript, spritePath); err != nil {
				t.Fatalf("Failed to add layer: %v", err)
			}

			blueColor := aseprite.Color{R: 0, G: 0, B: 255, A: 255}
			pixelScript := gen.DrawPixels("Layer 2", 1, []aseprite.Pixel{
				{Point: aseprite.Point{X: 5, Y: 5}, Color: blueColor},   // In-bounds
				{Point: aseprite.Point{X: 40, Y: 40}, Color: blueColor}, // Out-of-bounds positive
				{Point: aseprite.Point{X: -1, Y: 0}, Color: blueColor},  // Out-of-bounds negative
			}, tc.usePalette)

			output, err := client.ExecuteLua(ctx, pixelScript, spritePath)
			if err != nil {
				t.Fatalf("DrawPixels failed on out-of-bounds pixels: %v", err)
			}
			if !strings.Contains(output, "Pixels drawn successfully") {
				t.Fatalf("Expected success message, got: %s", output)
			}

			// Read back pixel at (5, 5)
			getScript := gen.GetPixels("Layer 2", 1, 5, 5, 1, 1)
			getOutput, err := client.ExecuteLua(ctx, getScript, spritePath)
			if err != nil {
				t.Fatalf("Failed to get pixel at (5,5): %v", err)
			}

			pixels, err := testutil.ParsePixelData(getOutput)
			if err != nil {
				t.Fatalf("Failed to parse pixel data: %v", err)
			}

			if len(pixels) != 1 || pixels[0].Color != "#0000FFFF" {
				t.Errorf("Expected in-bounds pixel at (5,5) to be #0000FFFF, got %+v", pixels)
			}
		})
	}
}
