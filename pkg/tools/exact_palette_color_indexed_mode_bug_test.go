//go:build integration
// +build integration

package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/willibrandon/pixel-mcp/internal/testutil"
	"github.com/willibrandon/pixel-mcp/pkg/aseprite"
)

// TestIntegration_IndexedMode_ExactColorResolution tests that drawing an exact
// palette color (usePalette=false) on an indexed sprite resolves to the right
// palette index, even though the palette was set in a PREVIOUS aseprite
// process (pixel-mcp runs one process per tool call, so this is the normal
// case, not an edge case).
//
// ROOT CAUSE: on an indexed sprite, img:putPixel and app.useTool both expect
// a palette index. Given a raw Color object instead, Aseprite resolves it to
// an index using an internal color cache that is only correctly populated
// when the palette was set earlier in the SAME process. When the palette was
// set in an earlier process and the sprite is reopened fresh (exactly how
// every pixel-mcp tool call works), that resolution silently returns an
// unrelated index, even though spr.palettes[1] reads back the correct,
// expected palette when inspected directly. The result reads back as the
// wrong color, or as transparent if the resolved index happens to be out of
// the palette's range.
//
// FIX: resolveExactPaletteColor (pkg/aseprite/lua_core.go) looks up the exact
// index itself by reading the sprite's actual active palette at Lua runtime,
// bypassing Aseprite's unreliable internal resolution entirely.
func TestIntegration_IndexedMode_ExactColorResolution(t *testing.T) {
	cfg := testutil.LoadTestConfig(t)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	// 4-color palette, set in its own process (as create_canvas + set_palette
	// always are via the real MCP tools).
	newIndexedSprite := func(t *testing.T, name string) string {
		t.Helper()
		spritePath := testutil.TempSpritePath(t, name)
		createScript := gen.CreateCanvas(32, 32, aseprite.ColorModeIndexed, spritePath)
		if _, err := client.ExecuteLua(ctx, createScript, ""); err != nil {
			t.Fatalf("Failed to create canvas: %v", err)
		}
		setPaletteScript := gen.SetPalette([]string{
			"#FF0000FF", // 0: RED
			"#00FF00FF", // 1: GREEN
			"#0000FFFF", // 2: BLUE
			"#FFFF00FF", // 3: YELLOW
		})
		if _, err := client.ExecuteLua(ctx, setPaletteScript, spritePath); err != nil {
			t.Fatalf("Failed to set palette: %v", err)
		}
		return spritePath
	}

	assertPixel := func(t *testing.T, spritePath string, x, y int, want string) {
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

	t.Run("DrawRectangle_ExactMatch", func(t *testing.T) {
		spritePath := newIndexedSprite(t, "test-exact-color-rect.aseprite")
		green := aseprite.Color{R: 0, G: 255, B: 0, A: 255}
		rectScript := gen.DrawRectangle("Layer 1", 1, 0, 0, 4, 4, green, true, false)
		if _, err := client.ExecuteLua(ctx, rectScript, spritePath); err != nil {
			t.Fatalf("Failed to draw rectangle: %v", err)
		}
		assertPixel(t, spritePath, 1, 1, "#00FF00FF")
	})

	t.Run("DrawPixels_ExactMatch", func(t *testing.T) {
		spritePath := newIndexedSprite(t, "test-exact-color-pixels.aseprite")
		blue := aseprite.Color{R: 0, G: 0, B: 255, A: 255}
		pixels := []aseprite.Pixel{{Point: aseprite.Point{X: 5, Y: 5}, Color: blue}}
		drawScript := gen.DrawPixels("Layer 1", 1, pixels, false)
		if _, err := client.ExecuteLua(ctx, drawScript, spritePath); err != nil {
			t.Fatalf("Failed to draw pixels: %v", err)
		}
		assertPixel(t, spritePath, 5, 5, "#0000FFFF")
	})

	t.Run("NoExactMatch_Errors", func(t *testing.T) {
		spritePath := newIndexedSprite(t, "test-exact-color-no-match.aseprite")
		purple := aseprite.Color{R: 128, G: 0, B: 128, A: 255} // not in the 4-color palette
		rectScript := gen.DrawRectangle("Layer 1", 1, 0, 0, 4, 4, purple, true, false)
		_, err := client.ExecuteLua(ctx, rectScript, spritePath)
		if err == nil {
			t.Fatal("expected an error for a color with no exact palette match, got nil")
		}
		if !strings.Contains(err.Error(), "has no exact match in the sprite's palette") {
			t.Errorf("expected a no-exact-match error, got: %v", err)
		}
	})
}
