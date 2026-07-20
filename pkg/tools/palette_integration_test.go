//go:build integration
// +build integration

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

// Integration tests for palette tools with real Aseprite.
// Run with: go test -tags=integration -v ./pkg/tools

func TestIntegration_SetPalette_BasicPalette(t *testing.T) {
	cfg := testutil.LoadTestConfig(t)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	// Create a canvas
	spritePath := testutil.TempSpritePath(t, "test-set-palette.aseprite")
	createScript := gen.CreateCanvas(64, 64, aseprite.ColorModeRGB, spritePath)
	_, err := client.ExecuteLua(ctx, createScript, "")
	if err != nil {
		t.Fatalf("Failed to create canvas: %v", err)
	}
	defer os.Remove(spritePath)

	// Set palette with 8 colors
	colors := []string{
		"#000000", // Black
		"#FF0000", // Red
		"#00FF00", // Green
		"#0000FF", // Blue
		"#FFFF00", // Yellow
		"#FF00FF", // Magenta
		"#00FFFF", // Cyan
		"#FFFFFF", // White
	}

	paletteScript := gen.SetPalette(colors)
	output, err := client.ExecuteLua(ctx, paletteScript, spritePath)
	if err != nil {
		t.Fatalf("ExecuteLua(SetPalette) error = %v", err)
	}

	if !strings.Contains(output, "Palette set successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}

	t.Logf("✓ Set palette with %d colors successfully", len(colors))
}

func TestIntegration_SetPalette_LargePalette(t *testing.T) {
	cfg := testutil.LoadTestConfig(t)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	// Create a canvas
	spritePath := testutil.TempSpritePath(t, "test-set-large-palette.aseprite")
	createScript := gen.CreateCanvas(64, 64, aseprite.ColorModeIndexed, spritePath)
	_, err := client.ExecuteLua(ctx, createScript, "")
	if err != nil {
		t.Fatalf("Failed to create canvas: %v", err)
	}
	defer os.Remove(spritePath)

	// Generate 32 colors (gradient from black to white)
	colors := make([]string, 32)
	for i := 0; i < 32; i++ {
		gray := uint8((i * 255) / 31)
		colors[i] = aseprite.Color{R: gray, G: gray, B: gray, A: 255}.ToHex()
	}

	paletteScript := gen.SetPalette(colors)
	output, err := client.ExecuteLua(ctx, paletteScript, spritePath)
	if err != nil {
		t.Fatalf("ExecuteLua(SetPalette) error = %v", err)
	}

	if !strings.Contains(output, "Palette set successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}

	t.Logf("✓ Set large palette with %d colors successfully", len(colors))
}

func TestIntegration_ApplyShading_SmoothStyle(t *testing.T) {
	cfg := testutil.LoadTestConfig(t)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	// Create a canvas and draw a base shape
	spritePath := testutil.TempSpritePath(t, "test-shading-smooth.aseprite")
	createScript := gen.CreateCanvas(64, 64, aseprite.ColorModeRGB, spritePath)
	_, err := client.ExecuteLua(ctx, createScript, "")
	if err != nil {
		t.Fatalf("Failed to create canvas: %v", err)
	}
	defer os.Remove(spritePath)

	// Draw a filled circle
	circleScript := gen.DrawCircle("Layer 1", 1, 32, 32, 20, aseprite.Color{R: 128, G: 128, B: 128, A: 255}, true, false)
	_, err = client.ExecuteLua(ctx, circleScript, spritePath)
	if err != nil {
		t.Fatalf("Failed to draw circle: %v", err)
	}

	// Apply shading with 3-color palette (dark, mid, light)
	palette := []string{"#404040", "#808080", "#C0C0C0"}
	shadingScript := gen.ApplyShading("Layer 1", 1, 12, 12, 40, 40, palette, "top_left", 0.7, "smooth")
	output, err := client.ExecuteLua(ctx, shadingScript, spritePath)
	if err != nil {
		t.Fatalf("ExecuteLua(ApplyShading) error = %v", err)
	}

	if !strings.Contains(output, "Shading applied") {
		t.Errorf("Expected success message, got: %s", output)
	}

	t.Logf("✓ Applied smooth shading successfully")
}

func TestIntegration_ApplyShading_HardStyle(t *testing.T) {
	cfg := testutil.LoadTestConfig(t)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	// Create a canvas and draw a base shape
	spritePath := testutil.TempSpritePath(t, "test-shading-hard.aseprite")
	createScript := gen.CreateCanvas(64, 64, aseprite.ColorModeRGB, spritePath)
	_, err := client.ExecuteLua(ctx, createScript, "")
	if err != nil {
		t.Fatalf("Failed to create canvas: %v", err)
	}
	defer os.Remove(spritePath)

	// Draw a filled rectangle
	rectScript := gen.DrawRectangle("Layer 1", 1, 10, 10, 44, 44, aseprite.Color{R: 255, G: 100, B: 100, A: 255}, true, false)
	_, err = client.ExecuteLua(ctx, rectScript, spritePath)
	if err != nil {
		t.Fatalf("Failed to draw rectangle: %v", err)
	}

	// Apply hard shading with 4-color palette
	palette := []string{"#800000", "#C00000", "#FF6060", "#FFC0C0"}
	shadingScript := gen.ApplyShading("Layer 1", 1, 10, 10, 44, 44, palette, "top", 0.9, "hard")
	output, err := client.ExecuteLua(ctx, shadingScript, spritePath)
	if err != nil {
		t.Fatalf("ExecuteLua(ApplyShading) error = %v", err)
	}

	if !strings.Contains(output, "Shading applied") {
		t.Errorf("Expected success message, got: %s", output)
	}

	t.Logf("✓ Applied hard shading successfully")
}

func TestIntegration_ApplyShading_AllDirections(t *testing.T) {
	cfg := testutil.LoadTestConfig(t)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	directions := []string{"top_left", "top", "top_right", "left", "right", "bottom_left", "bottom", "bottom_right"}
	palette := []string{"#000000", "#808080", "#FFFFFF"}

	for _, direction := range directions {
		t.Run(direction, func(t *testing.T) {
			// Create a canvas
			spritePath := testutil.TempSpritePath(t, "test-shading-"+direction+".aseprite")
			createScript := gen.CreateCanvas(32, 32, aseprite.ColorModeRGB, spritePath)
			_, err := client.ExecuteLua(ctx, createScript, "")
			if err != nil {
				t.Fatalf("Failed to create canvas: %v", err)
			}
			defer os.Remove(spritePath)

			// Draw filled rectangle
			rectScript := gen.DrawRectangle("Layer 1", 1, 5, 5, 22, 22, aseprite.Color{R: 128, G: 128, B: 128, A: 255}, true, false)
			_, err = client.ExecuteLua(ctx, rectScript, spritePath)
			if err != nil {
				t.Fatalf("Failed to draw rectangle: %v", err)
			}

			// Apply shading from this direction
			shadingScript := gen.ApplyShading("Layer 1", 1, 5, 5, 22, 22, palette, direction, 0.5, "smooth")
			output, err := client.ExecuteLua(ctx, shadingScript, spritePath)
			if err != nil {
				t.Fatalf("ExecuteLua(ApplyShading) error = %v", err)
			}

			if !strings.Contains(output, "Shading applied") {
				t.Errorf("Expected success message, got: %s", output)
			}

			t.Logf("✓ Applied shading from %s successfully", direction)
		})
	}
}

func TestIntegration_SetPalette_ThenApplyShading(t *testing.T) {
	cfg := testutil.LoadTestConfig(t)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	// Create a canvas
	spritePath := testutil.TempSpritePath(t, "test-palette-shading-workflow.aseprite")
	createScript := gen.CreateCanvas(64, 64, aseprite.ColorModeIndexed, spritePath)
	_, err := client.ExecuteLua(ctx, createScript, "")
	if err != nil {
		t.Fatalf("Failed to create canvas: %v", err)
	}
	defer os.Remove(spritePath)

	// Step 1: Set a limited palette (Lospec Pico-8 inspired)
	palette := []string{
		"#000000", "#1D2B53", "#7E2553", "#008751",
		"#AB5236", "#5F574F", "#C2C3C7", "#FFF1E8",
		"#FF004D", "#FFA300", "#FFEC27", "#00E436",
		"#29ADFF", "#83769C", "#FF77A8", "#FFCCAA",
	}

	paletteScript := gen.SetPalette(palette)
	_, err = client.ExecuteLua(ctx, paletteScript, spritePath)
	if err != nil {
		t.Fatalf("Failed to set palette: %v", err)
	}

	// Step 2: Draw a circle with an approximate mid-tone, snapped to the locked
	// palette (this color is not one of the 16 exact palette entries).
	circleScript := gen.DrawCircle("Layer 1", 1, 32, 32, 25, aseprite.Color{R: 195, G: 87, B: 54, A: 255}, true, true)
	_, err = client.ExecuteLua(ctx, circleScript, spritePath)
	if err != nil {
		t.Fatalf("Failed to draw circle: %v", err)
	}

	// Step 3: Apply shading using palette subset
	shadingPalette := []string{"#7E2553", "#AB5236", "#FFA300"}
	shadingScript := gen.ApplyShading("Layer 1", 1, 7, 7, 50, 50, shadingPalette, "top_left", 0.6, "smooth")
	output, err := client.ExecuteLua(ctx, shadingScript, spritePath)
	if err != nil {
		t.Fatalf("ExecuteLua(ApplyShading) error = %v", err)
	}

	if !strings.Contains(output, "Shading applied") {
		t.Errorf("Expected success message, got: %s", output)
	}

	t.Logf("✓ Complete palette workflow: set palette → draw → apply shading")
}

func TestIntegration_GetPalette(t *testing.T) {
	cfg := testutil.LoadTestConfig(t)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	// Create a canvas with indexed color mode
	spritePath := testutil.TempSpritePath(t, "test-get-palette.aseprite")
	createScript := gen.CreateCanvas(32, 32, aseprite.ColorModeIndexed, spritePath)
	_, err := client.ExecuteLua(ctx, createScript, "")
	if err != nil {
		t.Fatalf("Failed to create canvas: %v", err)
	}
	defer os.Remove(spritePath)

	// Set a known palette
	expectedColors := []string{
		"#FF0000", "#00FF00", "#0000FF", "#FFFF00",
	}
	paletteScript := gen.SetPalette(expectedColors)
	_, err = client.ExecuteLua(ctx, paletteScript, spritePath)
	if err != nil {
		t.Fatalf("Failed to set palette: %v", err)
	}

	// Get palette
	getPaletteScript := gen.GetPalette()
	output, err := client.ExecuteLua(ctx, getPaletteScript, spritePath)
	if err != nil {
		t.Fatalf("Failed to get palette: %v", err)
	}

	// Verify output contains expected colors
	for _, color := range expectedColors {
		if !strings.Contains(output, color) {
			t.Errorf("Expected palette to contain %s, got: %s", color, output)
		}
	}

	if !strings.Contains(output, `"colors":`) {
		t.Error("Output missing colors field")
	}

	if !strings.Contains(output, `"size":`) {
		t.Error("Output missing size field")
	}

	t.Logf("✓ Get palette returned: %s", output)
}

func TestIntegration_SetPaletteColor(t *testing.T) {
	cfg := testutil.LoadTestConfig(t)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	// Create a canvas with indexed color mode
	spritePath := testutil.TempSpritePath(t, "test-set-palette-color.aseprite")
	createScript := gen.CreateCanvas(32, 32, aseprite.ColorModeIndexed, spritePath)
	_, err := client.ExecuteLua(ctx, createScript, "")
	if err != nil {
		t.Fatalf("Failed to create canvas: %v", err)
	}
	defer os.Remove(spritePath)

	// Set an initial palette
	initialColors := []string{"#000000", "#111111", "#222222", "#333333"}
	paletteScript := gen.SetPalette(initialColors)
	_, err = client.ExecuteLua(ctx, paletteScript, spritePath)
	if err != nil {
		t.Fatalf("Failed to set initial palette: %v", err)
	}

	// Change color at index 2
	setColorScript := gen.SetPaletteColor(2, "#FF0000")
	output, err := client.ExecuteLua(ctx, setColorScript, spritePath)
	if err != nil {
		t.Fatalf("Failed to set palette color: %v", err)
	}

	if !strings.Contains(output, "Palette color set successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Verify the color was changed
	getPaletteScript := gen.GetPalette()
	paletteOutput, err := client.ExecuteLua(ctx, getPaletteScript, spritePath)
	if err != nil {
		t.Fatalf("Failed to get palette: %v", err)
	}

	if !strings.Contains(paletteOutput, "#FF0000") {
		t.Errorf("Expected palette to contain #FF0000, got: %s", paletteOutput)
	}

	t.Logf("✓ Set palette color at index 2 successfully")
}

func TestIntegration_AddPaletteColor(t *testing.T) {
	cfg := testutil.LoadTestConfig(t)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	// Create a canvas with indexed color mode
	spritePath := testutil.TempSpritePath(t, "test-add-palette-color.aseprite")
	createScript := gen.CreateCanvas(32, 32, aseprite.ColorModeIndexed, spritePath)
	_, err := client.ExecuteLua(ctx, createScript, "")
	if err != nil {
		t.Fatalf("Failed to create canvas: %v", err)
	}
	defer os.Remove(spritePath)

	// Set an initial palette with 3 colors
	initialColors := []string{"#000000", "#808080", "#FFFFFF"}
	paletteScript := gen.SetPalette(initialColors)
	_, err = client.ExecuteLua(ctx, paletteScript, spritePath)
	if err != nil {
		t.Fatalf("Failed to set initial palette: %v", err)
	}

	// Add a new color
	addColorScript := gen.AddPaletteColor("#FF00FF")
	output, err := client.ExecuteLua(ctx, addColorScript, spritePath)
	if err != nil {
		t.Fatalf("Failed to add palette color: %v", err)
	}

	if !strings.Contains(output, `"color_index":`) {
		t.Errorf("Expected JSON with color_index, got: %s", output)
	}

	// Verify the color was added
	getPaletteScript := gen.GetPalette()
	paletteOutput, err := client.ExecuteLua(ctx, getPaletteScript, spritePath)
	if err != nil {
		t.Fatalf("Failed to get palette: %v", err)
	}

	if !strings.Contains(paletteOutput, "#FF00FF") {
		t.Errorf("Expected palette to contain #FF00FF, got: %s", paletteOutput)
	}

	if !strings.Contains(paletteOutput, `"size":4`) {
		t.Errorf("Expected palette size to be 4, got: %s", paletteOutput)
	}

	t.Logf("✓ Added palette color successfully: %s", output)
}

func TestIntegration_SortPalette(t *testing.T) {
	cfg := testutil.LoadTestConfig(t)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	methods := []string{"hue", "saturation", "brightness", "luminance"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			// Create a canvas with indexed color mode
			spritePath := testutil.TempSpritePath(t, "test-sort-palette-"+method+".aseprite")
			createScript := gen.CreateCanvas(32, 32, aseprite.ColorModeIndexed, spritePath)
			_, err := client.ExecuteLua(ctx, createScript, "")
			if err != nil {
				t.Fatalf("Failed to create canvas: %v", err)
			}
			defer os.Remove(spritePath)

			// Set an unsorted palette
			unsortedColors := []string{
				"#FF0000", // Red (high hue, high saturation, mid brightness)
				"#00FF00", // Green (mid hue, high saturation, mid brightness)
				"#0000FF", // Blue (low hue, high saturation, mid brightness)
				"#FFFFFF", // White (no hue, no saturation, max brightness)
				"#000000", // Black (no hue, no saturation, min brightness)
				"#808080", // Gray (no hue, low saturation, mid brightness)
			}
			paletteScript := gen.SetPalette(unsortedColors)
			_, err = client.ExecuteLua(ctx, paletteScript, spritePath)
			if err != nil {
				t.Fatalf("Failed to set palette: %v", err)
			}

			// Sort palette
			sortScript := gen.SortPalette(method, true)
			output, err := client.ExecuteLua(ctx, sortScript, spritePath)
			if err != nil {
				t.Fatalf("Failed to sort palette: %v", err)
			}

			if !strings.Contains(output, "Palette sorted by "+method+" successfully") {
				t.Errorf("Expected success message, got: %s", output)
			}

			// Get sorted palette to verify it changed
			getPaletteScript := gen.GetPalette()
			paletteOutput, err := client.ExecuteLua(ctx, getPaletteScript, spritePath)
			if err != nil {
				t.Fatalf("Failed to get sorted palette: %v", err)
			}

			// Just verify we got a palette back (actual sorting validation would require parsing JSON)
			if !strings.Contains(paletteOutput, `"colors":`) {
				t.Error("Expected sorted palette output")
			}

			t.Logf("✓ Sorted palette by %s successfully", method)
		})
	}
}

func TestIntegration_SortPalette_DescendingOrder(t *testing.T) {
	cfg := testutil.LoadTestConfig(t)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	// Create a canvas
	spritePath := testutil.TempSpritePath(t, "test-sort-palette-descending.aseprite")
	createScript := gen.CreateCanvas(32, 32, aseprite.ColorModeIndexed, spritePath)
	_, err := client.ExecuteLua(ctx, createScript, "")
	if err != nil {
		t.Fatalf("Failed to create canvas: %v", err)
	}
	defer os.Remove(spritePath)

	// Set palette with gradient
	colors := []string{"#111111", "#444444", "#888888", "#BBBBBB", "#EEEEEE"}
	paletteScript := gen.SetPalette(colors)
	_, err = client.ExecuteLua(ctx, paletteScript, spritePath)
	if err != nil {
		t.Fatalf("Failed to set palette: %v", err)
	}

	// Sort descending by brightness
	sortScript := gen.SortPalette("brightness", false)
	output, err := client.ExecuteLua(ctx, sortScript, spritePath)
	if err != nil {
		t.Fatalf("Failed to sort palette: %v", err)
	}

	if !strings.Contains(output, "Palette sorted by brightness successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}

	t.Logf("✓ Sorted palette in descending order successfully")
}
