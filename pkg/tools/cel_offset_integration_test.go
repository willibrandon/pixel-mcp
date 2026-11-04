//go:build integration

package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/willibrandon/pixel-mcp/internal/testutil"
	"github.com/willibrandon/pixel-mcp/pkg/aseprite"
)

// Aseprite trims a cel to the bounding box of its content. After a shape tool
// (app.useTool) has run, a layer's cel therefore starts somewhere other than
// (0,0) and covers only part of the canvas. Tools that write into cel.image
// directly must offset canvas coordinates by the cel position and grow the cel
// first; without that, pixels land shifted by the cel position, or are dropped
// when they fall outside the cel's image.
//
// The existing DrawPixels cel tests never run a shape tool first, so the cel
// stays canvas-sized and they cannot see this. Each test below trims the cel
// with DrawRectangle before calling the tool under test.

var (
	celRed   = aseprite.Color{R: 255, G: 0, B: 0, A: 255}
	celGreen = aseprite.Color{R: 0, G: 255, B: 0, A: 255}
	celBlue  = aseprite.Color{R: 0, G: 0, B: 255, A: 255}
)

type celTestEnv struct {
	t      *testing.T
	ctx    context.Context
	client *aseprite.Client
	gen    *aseprite.LuaGenerator
	sprite string
}

func newCelTestEnv(t *testing.T, name string) *celTestEnv {
	t.Helper()
	cfg := testutil.LoadTestConfig(t)
	env := &celTestEnv{
		t:      t,
		ctx:    context.Background(),
		client: aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second),
		gen:    aseprite.NewLuaGenerator(),
		sprite: testutil.TempSpritePath(t, name),
	}
	env.run(env.gen.CreateCanvas(32, 32, aseprite.ColorModeRGB, env.sprite), "")
	return env
}

func (e *celTestEnv) run(script, sprite string) string {
	e.t.Helper()
	out, err := e.client.ExecuteLua(e.ctx, script, sprite)
	if err != nil {
		e.t.Fatalf("ExecuteLua failed: %v", err)
	}
	return out
}

// pixels returns the canvas pixels of Layer 1 frame 1, keyed by "x,y".
// Pixels outside the layer's cel are absent, which means transparent.
func (e *celTestEnv) pixels() map[string]string {
	e.t.Helper()
	data, err := testutil.ParsePixelData(e.run(e.gen.GetPixels("Layer 1", 1, 0, 0, 32, 32), e.sprite))
	if err != nil {
		e.t.Fatalf("failed to parse pixel data: %v", err)
	}
	m := make(map[string]string, len(data))
	for _, p := range data {
		m[testutil.FormatPixelPos(p.X, p.Y)] = p.Color
	}
	return m
}

func (e *celTestEnv) expect(m map[string]string, x, y int, want string) {
	e.t.Helper()
	got := m[testutil.FormatPixelPos(x, y)]
	if got == "" {
		got = "#00000000"
	}
	if got != want {
		e.t.Errorf("pixel (%d,%d) = %s, want %s", x, y, got, want)
	}
}

func TestIntegration_DrawPixels_AfterShapeTrimsCel(t *testing.T) {
	e := newCelTestEnv(t, "cel-offset-pixels.aseprite")

	// Trim Layer 1's cel to (8,6)-(23,15).
	e.run(e.gen.DrawRectangle("Layer 1", 1, 8, 6, 16, 10, celBlue, true, false), e.sprite)

	pixels := []aseprite.Pixel{
		{Point: aseprite.Point{X: 2, Y: 2}, Color: celRed},     // above-left of the cel
		{Point: aseprite.Point{X: 12, Y: 10}, Color: celRed},   // inside the cel
		{Point: aseprite.Point{X: 30, Y: 30}, Color: celGreen}, // below-right of the cel
		{Point: aseprite.Point{X: 40, Y: 5}, Color: celGreen},  // outside the canvas
	}
	out := e.run(e.gen.DrawPixels("Layer 1", 1, pixels, false), e.sprite)
	if got := parsePixelsSkipped(out); got != 1 {
		t.Errorf("pixels skipped = %d, want 1 (output: %q)", got, out)
	}

	m := e.pixels()
	e.expect(m, 2, 2, "#FF0000FF")
	e.expect(m, 12, 10, "#FF0000FF")
	e.expect(m, 30, 30, "#00FF00FF")
	e.expect(m, 8, 6, "#0000FFFF")   // the rectangle kept its place when the cel grew
	e.expect(m, 23, 15, "#0000FFFF") // ...at both corners
	e.expect(m, 20, 16, "#00000000") // (12,10) shifted by the old cel position: must be empty
}

func TestIntegration_DrawPixels_InsideTrimmedCel(t *testing.T) {
	e := newCelTestEnv(t, "cel-offset-pixels-inside.aseprite")

	// Trim the cel to (8,6)-(23,15), then draw only inside it: no growth needed,
	// but the coordinates still have to be offset by the cel position.
	e.run(e.gen.DrawRectangle("Layer 1", 1, 8, 6, 16, 10, celBlue, true, false), e.sprite)
	e.run(e.gen.DrawPixels("Layer 1", 1, []aseprite.Pixel{
		{Point: aseprite.Point{X: 10, Y: 8}, Color: celRed},
	}, false), e.sprite)

	m := e.pixels()
	e.expect(m, 10, 8, "#FF0000FF")
	e.expect(m, 18, 14, "#0000FFFF") // (10,8) shifted by (8,6) would have landed here
}

func TestIntegration_DrawWithDither_AfterShapeTrimsCel(t *testing.T) {
	for _, pattern := range []string{"checkerboard", "floyd_steinberg"} {
		t.Run(pattern, func(t *testing.T) {
			e := newCelTestEnv(t, "cel-offset-dither-"+pattern+".aseprite")

			// Trim the cel to (16,16)-(23,23), then dither (0,0)-(7,7), outside it.
			e.run(e.gen.DrawRectangle("Layer 1", 1, 16, 16, 8, 8, celBlue, true, false), e.sprite)
			out := e.run(e.gen.DrawWithDither("Layer 1", 1, 0, 0, 8, 8, "#FF0000", "#00FF00", pattern, 0.5), e.sprite)
			if !strings.Contains(out, "Dithering applied successfully") {
				t.Fatalf("unexpected output: %q", out)
			}

			m := e.pixels()
			for _, pos := range [][2]int{{0, 0}, {7, 0}, {0, 7}, {7, 7}} {
				c := m[testutil.FormatPixelPos(pos[0], pos[1])]
				if c != "#FF0000FF" && c != "#00FF00FF" {
					t.Errorf("pixel (%d,%d) = %q, want a dither colour", pos[0], pos[1], c)
				}
			}
			e.expect(m, 16, 16, "#0000FFFF")
			e.expect(m, 23, 23, "#0000FFFF")
		})
	}
}

func TestIntegration_PasteClipboard_OutsideTrimmedCel(t *testing.T) {
	e := newCelTestEnv(t, "cel-offset-paste.aseprite")

	// Layer 1's cel is trimmed to (0,0)-(3,3). Copy it and paste at (20,20).
	e.run(e.gen.DrawRectangle("Layer 1", 1, 0, 0, 4, 4, celRed, true, false), e.sprite)
	e.run(e.gen.SelectRectangle(0, 0, 4, 4, "replace"), e.sprite)
	e.run(e.gen.CopySelection(), e.sprite)
	x, y := 20, 20
	e.run(e.gen.PasteClipboard("Layer 1", 1, &x, &y), e.sprite)

	m := e.pixels()
	e.expect(m, 0, 0, "#FF0000FF")
	e.expect(m, 20, 20, "#FF0000FF")
	e.expect(m, 23, 23, "#FF0000FF")
}
