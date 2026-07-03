package aseprite

import (
	"strings"
	"testing"
)

func TestFormatColorWithPalette_NoPalette(t *testing.T) {
	color := Color{R: 255, G: 128, B: 64, A: 255}
	result := FormatColorWithPalette(color, false)
	expected := "resolveExactPaletteColor(255, 128, 64, 255)"
	if result != expected {
		t.Errorf("FormatColorWithPalette(usePalette=false) = %s, want %s", result, expected)
	}
}

func TestFormatColorWithPalette_WithPalette(t *testing.T) {
	color := Color{R: 255, G: 128, B: 64, A: 255}
	result := FormatColorWithPalette(color, true)
	expected := "snapToPaletteForPixel(255, 128, 64, 255)"
	if result != expected {
		t.Errorf("FormatColorWithPalette(usePalette=true) = %s, want %s", result, expected)
	}
}

func TestGeneratePaletteSnapperHelper(t *testing.T) {
	helper := GeneratePaletteSnapperHelper()

	// Verify helper contains key components
	if !strings.Contains(helper, "local function findNearestPaletteIndex") {
		t.Error("Helper should define findNearestPaletteIndex function")
	}

	if !strings.Contains(helper, "local function snapToPaletteForPixel") {
		t.Error("Helper should define snapToPaletteForPixel function")
	}

	if !strings.Contains(helper, "local function snapToPaletteForTool") {
		t.Error("Helper should define snapToPaletteForTool function")
	}

	if !strings.Contains(helper, "spr.palettes[1]") {
		t.Error("Helper should access sprite palette")
	}

	if !strings.Contains(helper, "palette:getColor") {
		t.Error("Helper should call getColor on palette")
	}

	if !strings.Contains(helper, "minDist") {
		t.Error("Helper should calculate minimum distance")
	}

	// Verify it handles missing palette gracefully
	if !strings.Contains(helper, "if not palette or #palette == 0") {
		t.Error("Helper should handle missing palette")
	}

	// Verify it returns palette index in indexed mode
	if !strings.Contains(helper, "if spr.colorMode == ColorMode.INDEXED") {
		t.Error("Helper should check for indexed color mode")
	}
}

func TestDrawPixels_WithPalette(t *testing.T) {
	gen := NewLuaGenerator()
	pixels := []Pixel{
		{Point: Point{X: 10, Y: 10}, Color: Color{R: 255, G: 0, B: 0, A: 255}},
		{Point: Point{X: 11, Y: 11}, Color: Color{R: 0, G: 255, B: 0, A: 255}},
	}

	script := gen.DrawPixels("Layer 1", 1, pixels, true)

	// Verify palette snapper helper is included
	if !strings.Contains(script, "local function snapToPaletteForPixel") {
		t.Error("Script should include snapToPaletteForPixel helper when usePalette=true")
	}

	// Verify colors are wrapped in snapToPaletteForPixel calls
	if !strings.Contains(script, "snapToPaletteForPixel(255, 0, 0, 255)") {
		t.Error("Script should use snapToPaletteForPixel for first pixel")
	}

	if !strings.Contains(script, "snapToPaletteForPixel(0, 255, 0, 255)") {
		t.Error("Script should use snapToPaletteForPixel for second pixel")
	}
}

func TestDrawPixels_WithoutPalette(t *testing.T) {
	gen := NewLuaGenerator()
	pixels := []Pixel{
		{Point: Point{X: 10, Y: 10}, Color: Color{R: 255, G: 0, B: 0, A: 255}},
	}

	script := gen.DrawPixels("Layer 1", 1, pixels, false)

	// Verify palette snapper helper is NOT included
	if strings.Contains(script, "local function snapToPalette") {
		t.Error("Script should NOT include snapToPalette helper when usePalette=false")
	}

	// Verify colors use direct Color() constructor
	if !strings.Contains(script, "Color(255, 0, 0, 255)") {
		t.Error("Script should use direct Color() constructor")
	}

	if strings.Contains(script, "snapToPalette") {
		t.Error("Script should NOT call snapToPalette when usePalette=false")
	}
}

func TestDrawLine_WithPalette(t *testing.T) {
	gen := NewLuaGenerator()
	color := Color{R: 128, G: 64, B: 32, A: 255}

	script := gen.DrawLine("Layer 1", 1, 10, 10, 50, 50, color, 2, true)

	// Verify palette snapper helper is included
	if !strings.Contains(script, "local function snapToPaletteForTool") {
		t.Error("Script should include snapToPaletteForTool helper")
	}

	// Verify color is wrapped in snapToPaletteForTool
	if !strings.Contains(script, "snapToPaletteForTool(128, 64, 32, 255)") {
		t.Error("Script should use snapToPaletteForTool for line color")
	}
}

func TestDrawRectangle_WithPalette(t *testing.T) {
	gen := NewLuaGenerator()
	color := Color{R: 200, G: 100, B: 50, A: 255}

	script := gen.DrawRectangle("Layer 1", 1, 10, 10, 30, 30, color, true, true)

	// Verify palette snapper helper is included
	if !strings.Contains(script, "local function snapToPaletteForTool") {
		t.Error("Script should include snapToPaletteForTool helper")
	}

	// Verify color is wrapped in snapToPaletteForTool
	if !strings.Contains(script, "snapToPaletteForTool(200, 100, 50, 255)") {
		t.Error("Script should use snapToPaletteForTool for rectangle color")
	}
}

func TestDrawCircle_WithPalette(t *testing.T) {
	gen := NewLuaGenerator()
	color := Color{R: 150, G: 200, B: 250, A: 255}

	script := gen.DrawCircle("Layer 1", 1, 50, 50, 20, color, true, true)

	// Verify palette snapper helper is included
	if !strings.Contains(script, "local function snapToPaletteForTool") {
		t.Error("Script should include snapToPaletteForTool helper")
	}

	// Verify color is wrapped in snapToPaletteForTool
	if !strings.Contains(script, "snapToPaletteForTool(150, 200, 250, 255)") {
		t.Error("Script should use snapToPaletteForTool for circle color")
	}
}

func TestFillArea_WithPalette(t *testing.T) {
	gen := NewLuaGenerator()
	color := Color{R: 75, G: 125, B: 175, A: 255}

	script := gen.FillArea("Layer 1", 1, 25, 25, color, 10, true)

	// Verify palette snapper helper is included
	if !strings.Contains(script, "local function snapToPaletteForTool") {
		t.Error("Script should include snapToPaletteForTool helper")
	}

	// Verify color is wrapped in snapToPaletteForTool
	if !strings.Contains(script, "snapToPaletteForTool(75, 125, 175, 255)") {
		t.Error("Script should use snapToPaletteForTool for fill color")
	}
}
