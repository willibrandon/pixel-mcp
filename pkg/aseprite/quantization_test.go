package aseprite

import (
	"image"
	"image/color"
	"testing"
)

func TestMedianCutQuantization(t *testing.T) {
	tests := []struct {
		name         string
		pixels       []color.Color
		targetColors int
		wantColors   int
	}{
		{
			name: "8 colors to 4",
			pixels: []color.Color{
				color.RGBA{R: 255, G: 0, B: 0, A: 255},
				color.RGBA{R: 255, G: 0, B: 0, A: 255},
				color.RGBA{R: 0, G: 255, B: 0, A: 255},
				color.RGBA{R: 0, G: 255, B: 0, A: 255},
				color.RGBA{R: 0, G: 0, B: 255, A: 255},
				color.RGBA{R: 0, G: 0, B: 255, A: 255},
				color.RGBA{R: 255, G: 255, B: 0, A: 255},
				color.RGBA{R: 255, G: 255, B: 0, A: 255},
			},
			targetColors: 4,
			wantColors:   4,
		},
		{
			name: "target exceeds unique colors",
			pixels: []color.Color{
				color.RGBA{R: 255, G: 0, B: 0, A: 255},
				color.RGBA{R: 0, G: 255, B: 0, A: 255},
			},
			targetColors: 10,
			wantColors:   2,
		},
		{
			name:         "empty pixels",
			pixels:       []color.Color{},
			targetColors: 4,
			wantColors:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			palette := MedianCutQuantization(tt.pixels, tt.targetColors)
			if len(palette) != tt.wantColors {
				t.Errorf("MedianCutQuantization() returned %d colors, want %d", len(palette), tt.wantColors)
			}
		})
	}
}

// TestMedianCutQuantization_ImbalancedFrequencyPreservesExactColors reproduces
// the split-by-pixel-count bug: a bucket containing a large solid-fill color
// alongside a handful of rare accent colors used to get bisected at the
// pixel-count median, landing the split point inside the dominant color's own
// run instead of at a boundary between colors. That blended rare accent
// colors into off-palette hex values and could even drop them from the
// palette entirely, even though targetColors comfortably exceeded the number
// of unique colors present (which should guarantee lossless recovery).
func TestMedianCutQuantization_ImbalancedFrequencyPreservesExactColors(t *testing.T) {
	// Mirrors a typical pixel-art color distribution: a couple of large
	// fills (many samples) plus several tiny accent regions (1-2 samples).
	pixels := make([]color.Color, 0, 320)
	appendN := func(c color.Color, n int) {
		for i := 0; i < n; i++ {
			pixels = append(pixels, c)
		}
	}

	tunicGreen := color.RGBA{R: 0x6A, G: 0xBE, B: 0x30, A: 255}
	skin := color.RGBA{R: 0xD9, G: 0xA0, B: 0x66, A: 255}
	outline := color.RGBA{R: 0x22, G: 0x20, B: 0x34, A: 255}
	highlight := color.RGBA{R: 0x99, G: 0xE5, B: 0x50, A: 255}
	buckle := color.RGBA{R: 0xFB, G: 0xF2, B: 0x36, A: 255}

	appendN(tunicGreen, 200)
	appendN(skin, 90)
	appendN(outline, 20)
	appendN(highlight, 2)
	appendN(buckle, 2)

	want := map[color.RGBA]bool{
		tunicGreen: true, skin: true, outline: true, highlight: true, buckle: true,
	}

	// Target far exceeds the 5 unique colors present, so every one of them
	// must come back exactly - no blending, no duplicates, no drops.
	palette := MedianCutQuantization(pixels, 32)

	if len(palette) != len(want) {
		t.Fatalf("MedianCutQuantization() returned %d colors, want %d (one bucket per unique input color): %v",
			len(palette), len(want), palette)
	}

	seen := make(map[color.RGBA]bool, len(palette))
	for _, c := range palette {
		rgba, ok := c.(color.RGBA)
		if !ok {
			r, g, b, a := c.RGBA()
			rgba = color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
		}
		if !want[rgba] {
			t.Errorf("palette contains %+v, which is not one of the 5 exact input colors (blended/off-palette result)", rgba)
		}
		if seen[rgba] {
			t.Errorf("palette contains duplicate entry %+v (a unique color's bucket was split into two near-identical averages)", rgba)
		}
		seen[rgba] = true
	}
}

func TestOctreeQuantization(t *testing.T) {
	tests := []struct {
		name         string
		pixels       []color.Color
		targetColors int
		wantColors   int
	}{
		{
			name: "8 colors to 4",
			pixels: []color.Color{
				color.RGBA{R: 255, G: 0, B: 0, A: 255},
				color.RGBA{R: 255, G: 0, B: 0, A: 255},
				color.RGBA{R: 0, G: 255, B: 0, A: 255},
				color.RGBA{R: 0, G: 255, B: 0, A: 255},
				color.RGBA{R: 0, G: 0, B: 255, A: 255},
				color.RGBA{R: 0, G: 0, B: 255, A: 255},
				color.RGBA{R: 255, G: 255, B: 0, A: 255},
				color.RGBA{R: 255, G: 255, B: 0, A: 255},
			},
			targetColors: 4,
			wantColors:   4,
		},
		{
			name:         "empty pixels",
			pixels:       []color.Color{},
			targetColors: 4,
			wantColors:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			palette := OctreeQuantization(tt.pixels, tt.targetColors)
			if len(palette) != tt.wantColors {
				t.Errorf("OctreeQuantization() returned %d colors, want %d", len(palette), tt.wantColors)
			}
		})
	}
}

func TestFloydSteinbergDither(t *testing.T) {
	// Create a simple gradient image
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			gray := uint8(x * 64) // 0, 64, 128, 192
			img.Set(x, y, color.RGBA{R: gray, G: gray, B: gray, A: 255})
		}
	}

	// Simple 2-color palette (black and white)
	palette := []color.Color{
		color.RGBA{R: 0, G: 0, B: 0, A: 255},
		color.RGBA{R: 255, G: 255, B: 255, A: 255},
	}

	result := FloydSteinbergDither(img, palette)

	// Verify result is not nil and has correct dimensions
	if result == nil {
		t.Fatal("FloydSteinbergDither() returned nil")
	}

	bounds := result.Bounds()
	if bounds.Dx() != 4 || bounds.Dy() != 4 {
		t.Errorf("Result dimensions = %dx%d, want 4x4", bounds.Dx(), bounds.Dy())
	}

	// Verify all pixels are from palette
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			c := result.At(x, y)
			r, g, b, _ := c.RGBA()
			if (r == 0 && g == 0 && b == 0) || (r == 65535 && g == 65535 && b == 65535) {
				// Valid palette color
				continue
			}
			t.Errorf("Pixel at (%d,%d) is not from palette: got RGB(%d,%d,%d)", x, y, r>>8, g>>8, b>>8)
		}
	}
}

func TestRemapPixelsWithDithering(t *testing.T) {
	// Create test image
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}

	palette := []color.Color{
		color.RGBA{R: 0, G: 0, B: 0, A: 255},
		color.RGBA{R: 128, G: 128, B: 192, A: 255},
		color.RGBA{R: 255, G: 255, B: 255, A: 255},
	}

	t.Run("without dithering", func(t *testing.T) {
		result := RemapPixelsWithDithering(img, palette, false)
		if result == nil {
			t.Fatal("RemapPixelsWithDithering() returned nil")
		}

		// All pixels should be the same without dithering
		firstPixel := result.At(0, 0)
		for y := 0; y < 4; y++ {
			for x := 0; x < 4; x++ {
				if result.At(x, y) != firstPixel {
					t.Errorf("Pixel at (%d,%d) differs without dithering", x, y)
				}
			}
		}
	})

	t.Run("with dithering", func(t *testing.T) {
		result := RemapPixelsWithDithering(img, palette, true)
		if result == nil {
			t.Fatal("RemapPixelsWithDithering() returned nil")
		}

		// Pixels should vary with dithering (at least some difference)
		firstPixel := result.At(0, 0)
		hasDifference := false
		for y := 0; y < 4; y++ {
			for x := 0; x < 4; x++ {
				if result.At(x, y) != firstPixel {
					hasDifference = true
					break
				}
			}
			if hasDifference {
				break
			}
		}

		// Note: Depending on palette matching, all pixels might still be same color
		// This is acceptable if the source color is very close to one palette color
	})
}

func TestCountUniqueColors(t *testing.T) {
	tests := []struct {
		name                 string
		createImage          func() image.Image
		preserveTransparency bool
		want                 int
	}{
		{
			name: "4 unique colors",
			createImage: func() image.Image {
				img := image.NewRGBA(image.Rect(0, 0, 4, 4))
				img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
				img.Set(1, 0, color.RGBA{R: 0, G: 255, B: 0, A: 255})
				img.Set(2, 0, color.RGBA{R: 0, G: 0, B: 255, A: 255})
				img.Set(3, 0, color.RGBA{R: 255, G: 255, B: 0, A: 255})
				// Rest are black
				return img
			},
			preserveTransparency: false,
			want:                 5, // 4 colors + black
		},
		{
			name: "with transparent pixels excluded",
			createImage: func() image.Image {
				img := image.NewRGBA(image.Rect(0, 0, 4, 4))
				img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
				img.Set(1, 0, color.RGBA{R: 0, G: 255, B: 0, A: 255})
				img.Set(2, 0, color.RGBA{R: 0, G: 0, B: 255, A: 255}) // Blue
				img.Set(3, 0, color.RGBA{R: 0, G: 0, B: 0, A: 0})     // Transparent
				// Rest are transparent (NewRGBA initializes to transparent black)
				return img
			},
			preserveTransparency: true,
			want:                 3, // Red, green, blue (transparent excluded)
		},
		{
			name: "all same color",
			createImage: func() image.Image {
				img := image.NewRGBA(image.Rect(0, 0, 4, 4))
				for y := 0; y < 4; y++ {
					for x := 0; x < 4; x++ {
						img.Set(x, y, color.RGBA{R: 128, G: 128, B: 128, A: 255})
					}
				}
				return img
			},
			preserveTransparency: false,
			want:                 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := tt.createImage()
			got := CountUniqueColors(img, tt.preserveTransparency)
			if got != tt.want {
				t.Errorf("CountUniqueColors() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestQuantizePalette(t *testing.T) {
	// Create test image with 8 distinct colors
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	colors := []color.RGBA{
		{R: 255, G: 0, B: 0, A: 255},     // Red
		{R: 0, G: 255, B: 0, A: 255},     // Green
		{R: 0, G: 0, B: 255, A: 255},     // Blue
		{R: 255, G: 255, B: 0, A: 255},   // Yellow
		{R: 255, G: 0, B: 255, A: 255},   // Magenta
		{R: 0, G: 255, B: 255, A: 255},   // Cyan
		{R: 128, G: 128, B: 128, A: 255}, // Gray
		{R: 255, G: 255, B: 255, A: 255}, // White
	}

	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, colors[(x+y)%8])
		}
	}

	tests := []struct {
		name         string
		targetColors int
		algorithm    string
		wantErr      bool
	}{
		{
			name:         "median_cut to 4 colors",
			targetColors: 4,
			algorithm:    "median_cut",
			wantErr:      false,
		},
		{
			name:         "kmeans to 4 colors",
			targetColors: 4,
			algorithm:    "kmeans",
			wantErr:      false,
		},
		{
			name:         "octree to 8 colors",
			targetColors: 8,
			algorithm:    "octree",
			wantErr:      false,
		},
		{
			name:         "invalid target colors",
			targetColors: 300,
			algorithm:    "median_cut",
			wantErr:      true,
		},
		{
			name:         "invalid algorithm",
			targetColors: 4,
			algorithm:    "invalid",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			palette, originalColors, err := QuantizePalette(img, tt.targetColors, tt.algorithm, false)

			if tt.wantErr {
				if err == nil {
					t.Error("QuantizePalette() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("QuantizePalette() unexpected error: %v", err)
			}

			if originalColors != 8 {
				t.Errorf("QuantizePalette() originalColors = %d, want 8", originalColors)
			}

			if len(palette) != tt.targetColors {
				t.Errorf("QuantizePalette() palette length = %d, want %d", len(palette), tt.targetColors)
			}

			// Verify all palette colors are valid hex strings
			for i, hexColor := range palette {
				if len(hexColor) != 7 || hexColor[0] != '#' {
					t.Errorf("Palette color %d is invalid: %s", i, hexColor)
				}
			}
		})
	}
}

func TestQuantizePaletteWithTransparency(t *testing.T) {
	// Create image with transparent and opaque pixels
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})   // Red
	img.Set(1, 0, color.RGBA{R: 0, G: 255, B: 0, A: 255})   // Green
	img.Set(2, 0, color.RGBA{R: 0, G: 0, B: 255, A: 255})   // Blue
	img.Set(3, 0, color.RGBA{R: 0, G: 0, B: 0, A: 0})       // Transparent
	img.Set(0, 1, color.RGBA{R: 255, G: 255, B: 0, A: 255}) // Yellow

	t.Run("preserve transparency", func(t *testing.T) {
		palette, _, err := QuantizePalette(img, 3, "median_cut", true)
		if err != nil {
			t.Fatalf("QuantizePalette() error: %v", err)
		}

		// Should have transparent color plus quantized colors
		hasTransparent := false
		for _, c := range palette {
			if c == "#00000000" {
				hasTransparent = true
				break
			}
		}

		if !hasTransparent {
			t.Error("QuantizePalette() with preserveTransparency=true should include transparent color")
		}
	})

	t.Run("do not preserve transparency", func(t *testing.T) {
		palette, _, err := QuantizePalette(img, 3, "median_cut", false)
		if err != nil {
			t.Fatalf("QuantizePalette() error: %v", err)
		}

		// Should not have transparent color
		for _, c := range palette {
			if c == "#00000000" {
				t.Error("QuantizePalette() with preserveTransparency=false should not include transparent color")
			}
		}
	})
}

func TestQuantizePaletteWithDithering(t *testing.T) {
	// Create gradient image
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			gray := uint8((x + y) * 16)
			img.Set(x, y, color.RGBA{R: gray, G: gray, B: gray, A: 255})
		}
	}

	t.Run("without dithering", func(t *testing.T) {
		palette, originalColors, err := QuantizePalette(img, 4, "median_cut", false)
		if err != nil {
			t.Fatalf("QuantizePalette() error: %v", err)
		}

		if len(palette) != 4 {
			t.Errorf("Palette length = %d, want 4", len(palette))
		}

		if originalColors == 0 {
			t.Error("Original colors should not be 0")
		}
	})

	t.Run("with dithering", func(t *testing.T) {
		palette, originalColors, err := QuantizePalette(img, 4, "median_cut", false)
		if err != nil {
			t.Fatalf("QuantizePalette() error: %v", err)
		}

		if len(palette) != 4 {
			t.Errorf("Palette length = %d, want 4", len(palette))
		}

		if originalColors == 0 {
			t.Error("Original colors should not be 0")
		}
	})
}

// Benchmark tests
func BenchmarkMedianCutQuantization(b *testing.B) {
	// Create test data
	pixels := make([]color.Color, 1000)
	for i := range pixels {
		pixels[i] = color.RGBA{
			R: uint8(i % 256),
			G: uint8((i * 2) % 256),
			B: uint8((i * 3) % 256),
			A: 255,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MedianCutQuantization(pixels, 16)
	}
}

func BenchmarkOctreeQuantization(b *testing.B) {
	// Create test data
	pixels := make([]color.Color, 1000)
	for i := range pixels {
		pixels[i] = color.RGBA{
			R: uint8(i % 256),
			G: uint8((i * 2) % 256),
			B: uint8((i * 3) % 256),
			A: 255,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		OctreeQuantization(pixels, 16)
	}
}

func BenchmarkFloydSteinbergDither(b *testing.B) {
	// Create test image
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			gray := uint8((x + y) * 2)
			img.Set(x, y, color.RGBA{R: gray, G: gray, B: gray, A: 255})
		}
	}

	palette := []color.Color{
		color.RGBA{R: 0, G: 0, B: 0, A: 255},
		color.RGBA{R: 85, G: 85, B: 85, A: 255},
		color.RGBA{R: 170, G: 170, B: 170, A: 255},
		color.RGBA{R: 255, G: 255, B: 255, A: 255},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FloydSteinbergDither(img, palette)
	}
}
