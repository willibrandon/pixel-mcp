package aseprite

import (
	"strings"
	"testing"
)

func TestEscapeString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple string",
			input: "hello",
			want:  "hello",
		},
		{
			name:  "string with quotes",
			input: `hello "world"`,
			want:  `hello \"world\"`,
		},
		{
			name:  "string with backslash",
			input: `hello\world`,
			want:  `hello\\world`,
		},
		{
			name:  "string with newline",
			input: "hello\nworld",
			want:  `hello\nworld`,
		},
		{
			name:  "string with tab",
			input: "hello\tworld",
			want:  `hello\tworld`,
		},
		{
			name:  "complex string with path and quotes",
			input: `C:\path\to\file "with quotes"`,
			want:  `C:\\path\\to\\file \"with quotes\"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EscapeString(tt.input); got != tt.want {
				t.Errorf("EscapeString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatColor(t *testing.T) {
	tests := []struct {
		name  string
		color Color
		want  string
	}{
		{
			name:  "red",
			color: Color{R: 255, G: 0, B: 0, A: 255},
			want:  "Color(255, 0, 0, 255)",
		},
		{
			name:  "green with alpha",
			color: Color{R: 0, G: 255, B: 0, A: 128},
			want:  "Color(0, 255, 0, 128)",
		},
		{
			name:  "transparent black",
			color: Color{R: 0, G: 0, B: 0, A: 0},
			want:  "Color(0, 0, 0, 0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatColor(tt.color); got != tt.want {
				t.Errorf("FormatColor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatPoint(t *testing.T) {
	p := Point{X: 10, Y: 20}
	want := "Point(10, 20)"

	if got := FormatPoint(p); got != want {
		t.Errorf("FormatPoint() = %v, want %v", got, want)
	}
}

func TestFormatRectangle(t *testing.T) {
	r := Rectangle{X: 10, Y: 20, Width: 30, Height: 40}
	want := "Rectangle(10, 20, 30, 40)"

	if got := FormatRectangle(r); got != want {
		t.Errorf("FormatRectangle() = %v, want %v", got, want)
	}
}

func TestLuaGenerator_CreateCanvas(t *testing.T) {
	gen := NewLuaGenerator()

	script := gen.CreateCanvas(800, 600, ColorModeRGB, "test.aseprite")

	// Verify script contains expected elements
	if !strings.Contains(script, "Sprite(800, 600, ColorMode.RGB)") {
		t.Error("script missing Sprite constructor call")
	}

	if !strings.Contains(script, `spr:saveAs("test.aseprite")`) {
		t.Error("script missing saveAs call")
	}

	if !strings.Contains(script, `print("test.aseprite")`) {
		t.Error("script missing print statement")
	}
}

func TestLuaGenerator_AddLayer(t *testing.T) {
	gen := NewLuaGenerator()

	script := gen.AddLayer("My Layer")

	// Verify script contains expected elements
	if !strings.Contains(script, "spr:newLayer()") {
		t.Error("script missing newLayer call")
	}

	if !strings.Contains(script, `layer.name = "My Layer"`) {
		t.Error("script missing layer name assignment")
	}

	if !strings.Contains(script, "app.transaction(function()") {
		t.Error("script not wrapped in transaction")
	}
}

func TestLuaGenerator_DrawPixels(t *testing.T) {
	gen := NewLuaGenerator()

	pixels := []Pixel{
		{Point: Point{X: 0, Y: 0}, Color: Color{R: 255, G: 0, B: 0, A: 255}},
		{Point: Point{X: 1, Y: 1}, Color: Color{R: 0, G: 255, B: 0, A: 255}},
	}

	script := gen.DrawPixels("Layer 1", 1, pixels, false)

	// Verify script contains expected elements (using loop-based layer lookup)
	if !strings.Contains(script, `lyr.name == "Layer 1"`) {
		t.Error("script missing layer name check")
	}

	if !strings.Contains(script, "for i, lyr in ipairs(spr.layers)") {
		t.Error("script missing layer iteration")
	}

	if !strings.Contains(script, "img:putPixel(0, 0, resolveExactPaletteColor(255, 0, 0, 255))") {
		t.Error("script missing first pixel")
	}

	if !strings.Contains(script, "img:putPixel(1, 1, resolveExactPaletteColor(0, 255, 0, 255))") {
		t.Error("script missing second pixel")
	}
}

func TestLuaGenerator_ExportSprite(t *testing.T) {
	gen := NewLuaGenerator()

	t.Run("export all frames", func(t *testing.T) {
		script := gen.ExportSprite("output.png", 0)

		if !strings.Contains(script, `saveCopyAs("output.png")`) {
			t.Error("script missing saveCopyAs call")
		}
	})

	t.Run("export specific frame", func(t *testing.T) {
		script := gen.ExportSprite("output.png", 2)

		// Check that it creates a temporary sprite
		if !strings.Contains(script, "Sprite(spr.width, spr.height, spr.colorMode)") {
			t.Error("script missing temporary sprite creation")
		}

		// Check that it references the target frame
		if !strings.Contains(script, "spr.frames[2]") {
			t.Error("script missing target frame reference")
		}

		// Check that it flattens layers before export
		if !strings.Contains(script, "FlattenLayers") {
			t.Error("script missing flatten layers command")
		}

		// Check that it uses saveAs (not saveCopyAs which auto-numbers files)
		if !strings.Contains(script, `saveAs("output.png")`) {
			t.Error("script missing saveAs call")
		}

		// Check that it closes the temporary sprite
		if !strings.Contains(script, "close()") {
			t.Error("script missing sprite close")
		}
	})
}

func TestLuaGenerator_DeleteLayer(t *testing.T) {
	gen := NewLuaGenerator()

	script := gen.DeleteLayer("Background")

	// Verify script contains expected elements
	if !strings.Contains(script, `if #spr.layers == 1 then`) {
		t.Error("script missing last layer check")
	}

	if !strings.Contains(script, `if lyr.name == "Background"`) {
		t.Error("script missing layer name check")
	}

	if !strings.Contains(script, `spr:deleteLayer(layer)`) {
		t.Error("script missing deleteLayer call")
	}

	if !strings.Contains(script, "app.transaction(function()") {
		t.Error("script not wrapped in transaction")
	}
}

func TestLuaGenerator_DeleteFrame(t *testing.T) {
	gen := NewLuaGenerator()

	script := gen.DeleteFrame(2)

	// Verify script contains expected elements
	if !strings.Contains(script, `if #spr.frames == 1 then`) {
		t.Error("script missing last frame check")
	}

	if !strings.Contains(script, "spr.frames[2]") {
		t.Error("script missing frame reference")
	}

	if !strings.Contains(script, `spr:deleteFrame(frame)`) {
		t.Error("script missing deleteFrame call")
	}

	if !strings.Contains(script, "app.transaction(function()") {
		t.Error("script not wrapped in transaction")
	}
}

func TestLuaGenerator_DrawContour(t *testing.T) {
	gen := NewLuaGenerator()

	points := []Point{
		{X: 10, Y: 10},
		{X: 50, Y: 10},
		{X: 50, Y: 50},
		{X: 10, Y: 50},
	}
	color := Color{R: 255, G: 0, B: 0, A: 255}

	t.Run("open contour", func(t *testing.T) {
		script := gen.DrawContour("Layer 1", 1, points, color, 2, false, false)

		// Verify script contains expected elements
		if !strings.Contains(script, `lyr.name == "Layer 1"`) {
			t.Error("script missing layer name check")
		}

		if !strings.Contains(script, "Brush(2)") {
			t.Error("script missing brush thickness")
		}

		if !strings.Contains(script, "Color(255, 0, 0, 255)") {
			t.Error("script missing color")
		}

		if !strings.Contains(script, "Point(10, 10)") {
			t.Error("script missing first point")
		}

		if !strings.Contains(script, "Point(50, 50)") {
			t.Error("script missing last point")
		}

		if !strings.Contains(script, "app.transaction(function()") {
			t.Error("script not wrapped in transaction")
		}

		// Should NOT have closing line for open contour
		if strings.Contains(script, "-- Close the contour") {
			t.Error("open contour should not have closing line")
		}
	})

	t.Run("closed contour (polygon)", func(t *testing.T) {
		script := gen.DrawContour("Layer 1", 1, points, color, 2, true, false)

		// Should have closing line connecting last to first
		if !strings.Contains(script, "-- Close the contour") {
			t.Error("closed contour missing closing line comment")
		}

		// Verify it connects last point back to first
		if !strings.Contains(script, "Point(10, 50)") || !strings.Contains(script, "Point(10, 10)") {
			t.Error("closed contour should connect last point to first")
		}
	})

	t.Run("with palette", func(t *testing.T) {
		script := gen.DrawContour("Layer 1", 1, points, color, 2, false, true)

		// Should include palette snapper helper
		if !strings.Contains(script, "function snapToPaletteForTool") {
			t.Error("script with use_palette missing snapToPaletteForTool helper")
		}

		// The color variable should be assigned the result of snapToPaletteForTool (with RGBA values, not Color())
		if !strings.Contains(script, "local color = snapToPaletteForTool(255, 0, 0, 255)") {
			t.Error("script should use snapToPaletteForTool for color assignment")
		}
	})
}

func TestLuaGenerator_SelectRectangle(t *testing.T) {
	gen := NewLuaGenerator()

	t.Run("replace mode", func(t *testing.T) {
		script := gen.SelectRectangle(10, 20, 30, 40, "replace")

		if !strings.Contains(script, "Rectangle(10, 20, 30, 40)") {
			t.Error("script missing rectangle coordinates")
		}

		if !strings.Contains(script, `if "replace" == "replace"`) {
			t.Error("script missing replace mode check")
		}

		if !strings.Contains(script, "Rectangle selection created successfully") {
			t.Error("script missing success message")
		}
	})

	t.Run("add mode", func(t *testing.T) {
		script := gen.SelectRectangle(5, 5, 10, 10, "add")

		if !strings.Contains(script, `if "add" == "replace"`) {
			t.Error("script missing mode check")
		}
	})
}

func TestLuaGenerator_SelectEllipse(t *testing.T) {
	gen := NewLuaGenerator()

	script := gen.SelectEllipse(10, 10, 50, 30, "replace")

	// Check that it uses width/height for radii calculation
	if !strings.Contains(script, "local rx = 50 / 2") {
		t.Error("script missing rx calculation")
	}

	if !strings.Contains(script, "local ry = 30 / 2") {
		t.Error("script missing ry calculation")
	}

	if !strings.Contains(script, "Ellipse selection created successfully") {
		t.Error("script missing success message")
	}
}

func TestLuaGenerator_SelectAll(t *testing.T) {
	gen := NewLuaGenerator()

	script := gen.SelectAll()

	if !strings.Contains(script, "Rectangle(0, 0, spr.width, spr.height)") {
		t.Error("script missing rectangle covering entire sprite")
	}

	if !strings.Contains(script, "spr.selection = sel") {
		t.Error("script missing selection assignment")
	}

	if !strings.Contains(script, "Select all completed successfully") {
		t.Error("script missing success message")
	}
}

func TestLuaGenerator_Deselect(t *testing.T) {
	gen := NewLuaGenerator()

	script := gen.Deselect()

	if !strings.Contains(script, "app.command.DeselectMask()") {
		t.Error("script missing DeselectMask command")
	}

	if !strings.Contains(script, "Deselect completed successfully") {
		t.Error("script missing success message")
	}
}

func TestLuaGenerator_MoveSelection(t *testing.T) {
	gen := NewLuaGenerator()

	script := gen.MoveSelection(15, -10)

	if !strings.Contains(script, "if spr.selection.isEmpty then") {
		t.Error("script missing empty selection check")
	}

	if !strings.Contains(script, "bounds.x + 15") {
		t.Error("script missing dx offset")
	}

	if !strings.Contains(script, "bounds.y + -10") {
		t.Error("script missing dy offset")
	}

	if !strings.Contains(script, "Selection moved successfully") {
		t.Error("script missing success message")
	}
}

func TestLuaGenerator_CutSelection(t *testing.T) {
	gen := NewLuaGenerator()

	script := gen.CutSelection("Layer 1", 2)

	if !strings.Contains(script, "if spr.selection.isEmpty then") {
		t.Error("script missing empty selection check")
	}

	if !strings.Contains(script, `lyr.name == "Layer 1"`) {
		t.Error("script missing layer name check")
	}

	if !strings.Contains(script, "spr.frames[2]") {
		t.Error("script missing frame reference")
	}

	if !strings.Contains(script, "__mcp_clipboard__") {
		t.Error("script missing clipboard layer reference")
	}

	if !strings.Contains(script, "Cut selection completed successfully") {
		t.Error("script missing success message")
	}
}

func TestLuaGenerator_CopySelection(t *testing.T) {
	gen := NewLuaGenerator()

	script := gen.CopySelection()

	if !strings.Contains(script, "if spr.selection.isEmpty then") {
		t.Error("script missing empty selection check")
	}

	if !strings.Contains(script, "__mcp_clipboard__") {
		t.Error("script missing clipboard layer reference")
	}

	if !strings.Contains(script, "Copy selection completed successfully") {
		t.Error("script missing success message")
	}
}

func TestLuaGenerator_PasteClipboard(t *testing.T) {
	gen := NewLuaGenerator()

	t.Run("without position", func(t *testing.T) {
		script := gen.PasteClipboard("Layer 1", 1, nil, nil)

		if !strings.Contains(script, `lyr.name == "Layer 1"`) {
			t.Error("script missing layer name check")
		}

		if !strings.Contains(script, "__mcp_clipboard__") {
			t.Error("script missing clipboard layer reference")
		}

		if !strings.Contains(script, "local pasteX, pasteY = 0, 0") {
			t.Error("script missing default paste position")
		}

		if !strings.Contains(script, "Paste completed successfully") {
			t.Error("script missing success message")
		}
	})

	t.Run("with position", func(t *testing.T) {
		x, y := 25, 35
		script := gen.PasteClipboard("Layer 1", 1, &x, &y)

		if !strings.Contains(script, "local pasteX, pasteY = 25, 35") {
			t.Error("script missing paste position assignment")
		}

		if !strings.Contains(script, "__mcp_clipboard__") {
			t.Error("script missing clipboard layer reference")
		}

		if !strings.Contains(script, "Paste completed successfully") {
			t.Error("script missing success message")
		}
	})
}

func TestLuaGenerator_GetPalette(t *testing.T) {
	gen := NewLuaGenerator()

	script := gen.GetPalette()

	if !strings.Contains(script, "local palette = spr.palettes[1]") {
		t.Error("script missing palette retrieval")
	}

	if !strings.Contains(script, "palette:getColor(i)") {
		t.Error("script missing getColor call")
	}

	if !strings.Contains(script, `string.format("#%02X%02X%02X"`) {
		t.Error("script missing hex color formatting")
	}

	if !strings.Contains(script, `"colors":`) {
		t.Error("script missing JSON colors field")
	}

	if !strings.Contains(script, `"size":`) {
		t.Error("script missing JSON size field")
	}
}

func TestLuaGenerator_SetPaletteColor(t *testing.T) {
	gen := NewLuaGenerator()

	script := gen.SetPaletteColor(5, "#FF0000")

	if !strings.Contains(script, "local palette = spr.palettes[1]") {
		t.Error("script missing palette retrieval")
	}

	if !strings.Contains(script, "if 5 < 0 or 5 >= #palette then") {
		t.Error("script missing index validation")
	}

	if !strings.Contains(script, "palette:setColor(5, Color{r=255, g=0, b=0, a=255})") {
		t.Error("script missing setColor call with correct values")
	}

	if !strings.Contains(script, "Palette color set successfully") {
		t.Error("script missing success message")
	}
}

func TestLuaGenerator_AddPaletteColor(t *testing.T) {
	gen := NewLuaGenerator()

	script := gen.AddPaletteColor("#00FF00")

	if !strings.Contains(script, "local palette = spr.palettes[1]") {
		t.Error("script missing palette retrieval")
	}

	if !strings.Contains(script, "if #palette >= 256 then") {
		t.Error("script missing maximum size check")
	}

	if !strings.Contains(script, "palette:resize(#palette + 1)") {
		t.Error("script missing palette resize")
	}

	if !strings.Contains(script, "Color{r=0, g=255, b=0, a=255}") {
		t.Error("script missing color with correct values")
	}

	if !strings.Contains(script, `"color_index":`) {
		t.Error("script missing JSON color_index field")
	}
}

func TestLuaGenerator_SortPalette(t *testing.T) {
	gen := NewLuaGenerator()

	t.Run("sort by hue ascending", func(t *testing.T) {
		script := gen.SortPalette("hue", true)

		if !strings.Contains(script, "function rgbToHSL(r, g, b)") {
			t.Error("script missing RGB to HSL conversion function")
		}

		if !strings.Contains(script, "local h, s, l = rgbToHSL") {
			t.Error("script missing HSL calculation")
		}

		if !strings.Contains(script, "return a.h < b.h") {
			t.Error("script missing hue comparison for ascending sort")
		}

		if !strings.Contains(script, "Palette sorted by hue successfully") {
			t.Error("script missing success message")
		}
	})

	t.Run("sort by saturation descending", func(t *testing.T) {
		script := gen.SortPalette("saturation", false)

		if !strings.Contains(script, "return a.s > b.s") {
			t.Error("script missing saturation comparison for descending sort")
		}

		if !strings.Contains(script, "Palette sorted by saturation successfully") {
			t.Error("script missing success message")
		}
	})

	t.Run("sort by brightness ascending", func(t *testing.T) {
		script := gen.SortPalette("brightness", true)

		if !strings.Contains(script, "return a.l < b.l") {
			t.Error("script missing lightness comparison for ascending sort")
		}

		if !strings.Contains(script, "Palette sorted by brightness successfully") {
			t.Error("script missing success message")
		}
	})

	t.Run("sort by luminance", func(t *testing.T) {
		script := gen.SortPalette("luminance", true)

		if !strings.Contains(script, "return a.l < b.l") {
			t.Error("script missing lightness comparison")
		}

		if !strings.Contains(script, "Palette sorted by luminance successfully") {
			t.Error("script missing success message")
		}
	})
}

func TestLuaGenerator_FlipSprite(t *testing.T) {
	gen := NewLuaGenerator()

	t.Run("flip horizontal sprite", func(t *testing.T) {
		script := gen.FlipSprite("horizontal", "sprite")

		if !strings.Contains(script, `orientation = "horizontal"`) {
			t.Error("script missing horizontal orientation")
		}

		if !strings.Contains(script, `target = 'sprite'`) {
			t.Error("script missing sprite target")
		}

		if !strings.Contains(script, "app.command.Flip") {
			t.Error("script missing Flip command")
		}

		if !strings.Contains(script, "Sprite flipped horizontal successfully") {
			t.Error("script missing success message")
		}
	})

	t.Run("flip vertical layer", func(t *testing.T) {
		script := gen.FlipSprite("vertical", "layer")

		if !strings.Contains(script, `orientation = "vertical"`) {
			t.Error("script missing vertical orientation")
		}

		if !strings.Contains(script, `target = 'layer'`) {
			t.Error("script missing layer target")
		}
	})

	t.Run("invalid direction defaults to horizontal", func(t *testing.T) {
		script := gen.FlipSprite("invalid", "sprite")

		if !strings.Contains(script, `orientation = "horizontal"`) {
			t.Error("invalid direction should default to horizontal")
		}
	})
}

func TestLuaGenerator_RotateSprite(t *testing.T) {
	gen := NewLuaGenerator()

	t.Run("rotate 90 degrees", func(t *testing.T) {
		script := gen.RotateSprite(90, "sprite")

		if !strings.Contains(script, "angle = 90") {
			t.Error("script missing 90 degree angle")
		}

		if !strings.Contains(script, "app.command.Rotate") {
			t.Error("script missing Rotate command")
		}

		if !strings.Contains(script, "Sprite rotated 90 degrees successfully") {
			t.Error("script missing success message")
		}
	})

	t.Run("rotate 180 degrees", func(t *testing.T) {
		script := gen.RotateSprite(180, "sprite")

		if !strings.Contains(script, "angle = 180") {
			t.Error("script missing 180 degree angle")
		}
	})

	t.Run("rotate 270 degrees with cel target", func(t *testing.T) {
		script := gen.RotateSprite(270, "cel")

		if !strings.Contains(script, "angle = 270") {
			t.Error("script missing 270 degree angle")
		}

		if !strings.Contains(script, `target = 'cel'`) {
			t.Error("script missing cel target")
		}
	})

	t.Run("invalid angle defaults to 90", func(t *testing.T) {
		script := gen.RotateSprite(45, "sprite")

		if !strings.Contains(script, "angle = 90") {
			t.Error("invalid angle should default to 90")
		}
	})
}

func TestLuaGenerator_ScaleSprite(t *testing.T) {
	gen := NewLuaGenerator()

	t.Run("scale with nearest neighbor", func(t *testing.T) {
		script := gen.ScaleSprite(2.0, 2.0, "nearest")

		if !strings.Contains(script, "oldWidth * 2.000") {
			t.Error("script missing scale factor calculation")
		}

		if !strings.Contains(script, `method = "nearest"`) {
			t.Error("script missing nearest algorithm")
		}

		if !strings.Contains(script, "app.command.SpriteSize") {
			t.Error("script missing SpriteSize command")
		}

		if !strings.Contains(script, `"success":true`) {
			t.Error("script missing JSON output")
		}
	})

	t.Run("scale with bilinear", func(t *testing.T) {
		script := gen.ScaleSprite(0.5, 0.5, "bilinear")

		if !strings.Contains(script, `method = "bilinear"`) {
			t.Error("script missing bilinear algorithm")
		}
	})

	t.Run("scale with rotsprite", func(t *testing.T) {
		script := gen.ScaleSprite(1.5, 1.5, "rotsprite")

		if !strings.Contains(script, `method = "rotsprite"`) {
			t.Error("script missing rotsprite algorithm")
		}
	})

	t.Run("invalid algorithm defaults to nearest", func(t *testing.T) {
		script := gen.ScaleSprite(2.0, 2.0, "invalid")

		if !strings.Contains(script, `method = "nearest"`) {
			t.Error("invalid algorithm should default to nearest")
		}
	})
}

func TestLuaGenerator_CropSprite(t *testing.T) {
	gen := NewLuaGenerator()

	script := gen.CropSprite(10, 20, 100, 80)

	// Check validation
	if !strings.Contains(script, "if 10 < 0 or 20 < 0") {
		t.Error("script missing position validation")
	}

	if !strings.Contains(script, "if 100 <= 0 or 80 <= 0") {
		t.Error("script missing dimension validation")
	}

	// Check bounds checking
	if !strings.Contains(script, "if 10 + 100 > spr.width or 20 + 80 > spr.height") {
		t.Error("script missing bounds validation")
	}

	// Check command
	if !strings.Contains(script, "app.command.CropSprite") {
		t.Error("script missing CropSprite command")
	}

	// Check that selection is set to crop region
	if !strings.Contains(script, "spr.selection = Selection(Rectangle(10, 20, 100, 80))") {
		t.Error("script missing selection setup")
	}

	if !strings.Contains(script, "Sprite cropped successfully") {
		t.Error("script missing success message")
	}
}

func TestLuaGenerator_ResizeCanvas(t *testing.T) {
	gen := NewLuaGenerator()

	t.Run("resize with center anchor", func(t *testing.T) {
		script := gen.ResizeCanvas(200, 150, "center")

		if !strings.Contains(script, "local newWidth = 200") {
			t.Error("script missing new width")
		}

		if !strings.Contains(script, "local newHeight = 150") {
			t.Error("script missing new height")
		}

		if !strings.Contains(script, "left = math.floor((newWidth - oldWidth) / 2)") {
			t.Error("script missing center left padding")
		}

		if !strings.Contains(script, "top = math.floor((newHeight - oldHeight) / 2)") {
			t.Error("script missing center top padding")
		}

		if !strings.Contains(script, "right = math.ceil((newWidth - oldWidth) / 2)") {
			t.Error("script missing center right padding")
		}

		if !strings.Contains(script, "bottom = math.ceil((newHeight - oldHeight) / 2)") {
			t.Error("script missing center bottom padding")
		}

		if !strings.Contains(script, "app.command.CanvasSize") {
			t.Error("script missing CanvasSize command")
		}
	})

	t.Run("resize with top_left anchor", func(t *testing.T) {
		script := gen.ResizeCanvas(200, 150, "top_left")

		if !strings.Contains(script, "left = 0") {
			t.Error("script missing top_left left padding")
		}

		if !strings.Contains(script, "top = 0") {
			t.Error("script missing top_left top padding")
		}

		if !strings.Contains(script, "right = newWidth - oldWidth") {
			t.Error("script missing top_left right padding")
		}

		if !strings.Contains(script, "bottom = newHeight - oldHeight") {
			t.Error("script missing top_left bottom padding")
		}
	})

	t.Run("resize with bottom_right anchor", func(t *testing.T) {
		script := gen.ResizeCanvas(200, 150, "bottom_right")

		if !strings.Contains(script, "left = newWidth - oldWidth") {
			t.Error("script missing bottom_right left padding")
		}

		if !strings.Contains(script, "top = newHeight - oldHeight") {
			t.Error("script missing bottom_right top padding")
		}

		if !strings.Contains(script, "right = 0") {
			t.Error("script missing bottom_right right padding")
		}

		if !strings.Contains(script, "bottom = 0") {
			t.Error("script missing bottom_right bottom padding")
		}
	})

	t.Run("invalid anchor defaults to center", func(t *testing.T) {
		script := gen.ResizeCanvas(200, 150, "invalid")

		if !strings.Contains(script, "math.floor((newWidth - oldWidth) / 2)") {
			t.Error("invalid anchor should default to center")
		}
	})
}

func TestLuaGenerator_ApplyOutline(t *testing.T) {
	gen := NewLuaGenerator()

	color := Color{R: 255, G: 0, B: 0, A: 255}
	script := gen.ApplyOutline("Layer 1", 1, color, 2)

	// Check layer lookup
	if !strings.Contains(script, `for i, lyr in ipairs(spr.layers)`) {
		t.Error("script missing layer iteration")
	}

	if !strings.Contains(script, `if lyr.name == "Layer 1"`) {
		t.Error("script missing layer name comparison")
	}

	// Check frame lookup
	if !strings.Contains(script, "spr.frames[1]") {
		t.Error("script missing frame lookup")
	}

	// Check cel existence check
	if !strings.Contains(script, "if not cel then") {
		t.Error("script missing cel existence check")
	}

	// Check Outline command
	if !strings.Contains(script, "app.command.Outline") {
		t.Error("script missing Outline command")
	}

	if !strings.Contains(script, "color = Color(255, 0, 0, 255)") {
		t.Error("script missing color parameter")
	}

	if !strings.Contains(script, "size = 2") {
		t.Error("script missing size parameter")
	}

	if !strings.Contains(script, "Outline applied successfully") {
		t.Error("script missing success message")
	}
}

func TestLuaGenerator_ExportSpritesheet(t *testing.T) {
	gen := NewLuaGenerator()

	t.Run("export with horizontal layout", func(t *testing.T) {
		script := gen.ExportSpritesheet("/tmp/output.png", "horizontal", 2, false)

		if !strings.Contains(script, `type = "horizontal"`) {
			t.Error("script missing horizontal layout")
		}

		if !strings.Contains(script, `textureFilename = outputPath`) {
			t.Error("script missing texture filename")
		}

		if !strings.Contains(script, "borderPadding = 2") {
			t.Error("script missing padding")
		}

		if !strings.Contains(script, "app.command.ExportSpriteSheet") {
			t.Error("script missing ExportSpriteSheet command")
		}
	})

	t.Run("export with JSON metadata", func(t *testing.T) {
		script := gen.ExportSpritesheet("/tmp/output.png", "packed", 0, true)

		if !strings.Contains(script, `dataFormat = "json"`) {
			t.Error("script missing JSON data format")
		}

		if !strings.Contains(script, `metadata_path`) {
			t.Error("script missing metadata path in output")
		}
	})

	t.Run("invalid layout defaults to horizontal", func(t *testing.T) {
		script := gen.ExportSpritesheet("/tmp/output.png", "invalid", 0, false)

		if !strings.Contains(script, `type = "horizontal"`) {
			t.Error("invalid layout should default to horizontal")
		}
	})
}

func TestLuaGenerator_ImportImage(t *testing.T) {
	gen := NewLuaGenerator()

	t.Run("import with default position", func(t *testing.T) {
		script := gen.ImportImage("/tmp/image.png", "Imported Layer", 1, nil, nil)

		if !strings.Contains(script, `Image{ fromFile = "/tmp/image.png" }`) {
			t.Error("script missing image load")
		}

		if !strings.Contains(script, `if lyr.name == "Imported Layer"`) {
			t.Error("script missing layer name check")
		}

		if !strings.Contains(script, "Point(0, 0)") {
			t.Error("script missing Point(0, 0) for color conversion")
		}

		if !strings.Contains(script, "spr:newCel(layer, frame, finalImg, 0, 0)") {
			t.Error("script missing cel creation with default position")
		}
	})

	t.Run("import with custom position", func(t *testing.T) {
		x := 50
		y := 100
		script := gen.ImportImage("/tmp/image.png", "Layer 1", 2, &x, &y)

		if !strings.Contains(script, "spr:newCel(layer, frame, finalImg, 50, 100)") {
			t.Error("script missing cel creation with custom position")
		}

		if !strings.Contains(script, "spr.frames[2]") {
			t.Error("script missing frame number")
		}
	})
}

func TestLuaGenerator_SaveAs(t *testing.T) {
	gen := NewLuaGenerator()

	script := gen.SaveAs("/tmp/new-sprite.aseprite")

	if !strings.Contains(script, `local newPath = "/tmp/new-sprite.aseprite"`) {
		t.Error("script missing escaped path")
	}

	if !strings.Contains(script, "spr:saveAs(newPath)") {
		t.Error("script missing saveAs call")
	}

	if !strings.Contains(script, `"success":true`) {
		t.Error("script missing success flag")
	}

	if !strings.Contains(script, `"file_path"`) {
		t.Error("script missing file_path in output")
	}
}

func TestLuaGenerator_DeleteTag(t *testing.T) {
	gen := NewLuaGenerator()

	script := gen.DeleteTag("walk")

	// Check tag iteration
	if !strings.Contains(script, "for i, t in ipairs(spr.tags)") {
		t.Error("script missing tag iteration")
	}

	if !strings.Contains(script, `if t.name == "walk"`) {
		t.Error("script missing tag name check")
	}

	// Check error handling
	if !strings.Contains(script, `error("Tag not found: walk")`) {
		t.Error("script missing tag not found error")
	}

	// Check deletion
	if !strings.Contains(script, "spr:deleteTag(tag)") {
		t.Error("script missing tag deletion")
	}

	if !strings.Contains(script, "app.transaction") {
		t.Error("script missing transaction wrapper")
	}

	if !strings.Contains(script, "Tag deleted successfully") {
		t.Error("script missing success message")
	}
}

func TestLuaGenerator_SetFrameDuration(t *testing.T) {
	gen := NewLuaGenerator()

	script := gen.SetFrameDuration(2, 150)

	if !strings.Contains(script, "local frame = spr.frames[2]") {
		t.Error("script missing frame retrieval")
	}

	if !strings.Contains(script, "frame.duration = 0.150") {
		t.Error("script missing duration assignment")
	}

	if !strings.Contains(script, "Frame duration set successfully") {
		t.Error("script missing success message")
	}
}

func TestLuaGenerator_CreateTag(t *testing.T) {
	gen := NewLuaGenerator()

	script := gen.CreateTag("run", 1, 4, "forward")

	if !strings.Contains(script, `tag.name = "run"`) {
		t.Error("script missing tag name assignment")
	}

	if !strings.Contains(script, "spr:newTag(1, 4)") {
		t.Error("script missing newTag with frame range")
	}

	if !strings.Contains(script, "tag.aniDir = AniDir.FORWARD") {
		t.Error("script missing animation direction")
	}

	if !strings.Contains(script, "Tag created successfully") {
		t.Error("script missing success message")
	}
}

func TestLuaGenerator_AddFrame(t *testing.T) {
	gen := NewLuaGenerator()

	script := gen.AddFrame(100)

	if !strings.Contains(script, "local frame = spr:newFrame()") {
		t.Error("script missing newFrame call")
	}

	if !strings.Contains(script, "frame.duration = 0.100") {
		t.Error("script missing duration assignment")
	}

	if !strings.Contains(script, "print(#spr.frames)") {
		t.Error("script missing frame count output")
	}
}

func TestLuaGenerator_GetSpriteInfo(t *testing.T) {
	gen := NewLuaGenerator()

	script := gen.GetSpriteInfo()

	if !strings.Contains(script, `"color_mode":`) {
		t.Error("script missing color_mode field")
	}

	if !strings.Contains(script, `"width":`) {
		t.Error("script missing width field")
	}

	if !strings.Contains(script, `"height":`) {
		t.Error("script missing height field")
	}

	if !strings.Contains(script, `"frame_count":`) {
		t.Error("script missing frame_count field")
	}

	if !strings.Contains(script, `"layer_count":`) {
		t.Error("script missing layer_count field")
	}

	if !strings.Contains(script, `"layers":`) {
		t.Error("script missing layers array")
	}
}

func TestLuaGenerator_SetPalette(t *testing.T) {
	gen := NewLuaGenerator()

	colors := []string{"#FF0000", "#00FF00", "#0000FF"}
	script := gen.SetPalette(colors)

	if !strings.Contains(script, "palette:resize(3)") {
		t.Error("script missing palette resize")
	}

	if !strings.Contains(script, "Color{r=255, g=0, b=0, a=255}") {
		t.Error("script missing first color")
	}

	if !strings.Contains(script, "Color{r=0, g=255, b=0, a=255}") {
		t.Error("script missing second color")
	}

	if !strings.Contains(script, "Color{r=0, g=0, b=255, a=255}") {
		t.Error("script missing third color")
	}

	if !strings.Contains(script, "for i, color in ipairs(colors)") {
		t.Error("script missing color iteration")
	}

	if !strings.Contains(script, "palette:setColor(i - 1, color)") {
		t.Error("script missing setColor call")
	}

	if !strings.Contains(script, "Palette set successfully") {
		t.Error("script missing success message")
	}
}

func TestLuaGenerator_DuplicateFrame(t *testing.T) {
	gen := NewLuaGenerator()

	script := gen.DuplicateFrame(2, 3)

	if !strings.Contains(script, "local srcFrame = spr.frames[2]") {
		t.Error("script missing source frame reference")
	}

	if !strings.Contains(script, "spr:newFrame(3 + 1)") && !strings.Contains(script, "spr:newFrame(4)") {
		t.Error("script missing newFrame at position")
	}

	if !strings.Contains(script, "srcCel.image") {
		t.Error("script missing image reference")
	}

	if !strings.Contains(script, "print(3 + 1)") && !strings.Contains(script, "print(4)") {
		t.Error("script missing frame number output")
	}
}

func TestLuaGenerator_LinkCel(t *testing.T) {
	gen := NewLuaGenerator()

	script := gen.LinkCel("Layer 1", 1, 2)

	if !strings.Contains(script, `lyr.name == "Layer 1"`) {
		t.Error("script missing layer name check")
	}

	if !strings.Contains(script, "local srcFrame = spr.frames[1]") {
		t.Error("script missing source frame reference")
	}

	if !strings.Contains(script, "local tgtFrame = spr.frames[2]") {
		t.Error("script missing target frame reference")
	}

	if !strings.Contains(script, "local srcCel = layer:cel(srcFrame)") {
		t.Error("script missing source cel reference")
	}

	if !strings.Contains(script, "spr:newCel(layer, tgtFrame, srcCel.image, srcCel.position)") {
		t.Error("script missing linked cel creation")
	}

	if !strings.Contains(script, "Cel linked successfully") {
		t.Error("script missing success message")
	}
}

func TestWrapInTransaction(t *testing.T) {
	innerCode := `img:putPixel(0, 0, Color(255, 0, 0, 255))`
	script := WrapInTransaction(innerCode)

	if !strings.Contains(script, "app.transaction(function()") {
		t.Error("script missing transaction wrapper start")
	}

	if !strings.Contains(script, innerCode) {
		t.Error("script missing inner code")
	}

	if !strings.Contains(script, "end)") {
		t.Error("script missing transaction wrapper end")
	}
}

func TestColor_ToHexRGB(t *testing.T) {
	tests := []struct {
		name  string
		color Color
		want  string
	}{
		{
			name:  "black",
			color: Color{R: 0, G: 0, B: 0, A: 255},
			want:  "#000000",
		},
		{
			name:  "white",
			color: Color{R: 255, G: 255, B: 255, A: 255},
			want:  "#FFFFFF",
		},
		{
			name:  "red",
			color: Color{R: 255, G: 0, B: 0, A: 255},
			want:  "#FF0000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.color.ToHexRGB()
			if got != tt.want {
				t.Errorf("Color.ToHexRGB() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestColorMode_String(t *testing.T) {
	tests := []struct {
		mode ColorMode
		want string
	}{
		{mode: ColorModeRGB, want: "rgb"},
		{mode: ColorModeGrayscale, want: "grayscale"},
		{mode: ColorModeIndexed, want: "indexed"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.mode.String()
			if got != tt.want {
				t.Errorf("ColorMode.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLuaGenerator_DrawWithDither(t *testing.T) {
	gen := &LuaGenerator{}

	tests := []struct {
		name     string
		pattern  string
		checkFor []string
	}{
		{
			name:     "bayer_2x2 pattern",
			pattern:  "bayer_2x2",
			checkFor: []string{"local matrix = {{0, 2}, {3, 1}}", "local matrixSize = 2"},
		},
		{
			name:     "bayer_4x4 pattern",
			pattern:  "bayer_4x4",
			checkFor: []string{"local matrix = {", "local matrixSize = 4"},
		},
		{
			name:     "checkerboard pattern",
			pattern:  "checkerboard",
			checkFor: []string{"local matrix = {{0, 1}, {1, 0}}", "local matrixSize = 2"},
		},
		{
			name:     "grass pattern",
			pattern:  "grass",
			checkFor: []string{"local matrixSize = 6"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := gen.DrawWithDither("Layer 1", 1, 0, 0, 64, 64, "#FF0000", "#00FF00", tt.pattern, 0.5)

			for _, check := range tt.checkFor {
				if !strings.Contains(script, check) {
					t.Errorf("DrawWithDither() missing %q in generated script", check)
				}
			}

			if !strings.Contains(script, "Layer 1") {
				t.Error("DrawWithDither() missing layer name in script")
			}
		})
	}
}

func TestLuaGenerator_GetPixels(t *testing.T) {
	gen := &LuaGenerator{}
	script := gen.GetPixels("Layer 1", 1, 10, 20, 100, 50)

	if !strings.Contains(script, "Layer 1") {
		t.Error("GetPixels() missing layer name")
	}
	if !strings.Contains(script, "spr.frames[1]") {
		t.Error("GetPixels() missing frame reference")
	}
	if !strings.Contains(script, "for py = 20, 69 do") {
		t.Error("GetPixels() missing correct y loop bounds")
	}
	if !strings.Contains(script, "for px = 10, 109 do") {
		t.Error("GetPixels() missing correct x loop bounds")
	}
}

func TestLuaGenerator_GetPixelsWithPagination(t *testing.T) {
	gen := &LuaGenerator{}

	tests := []struct {
		name     string
		offset   int
		limit    int
		checkFor []string
	}{
		{
			name:     "with pagination",
			offset:   10,
			limit:    20,
			checkFor: []string{"local offset = 10", "local limit = 20"},
		},
		{
			name:     "without limit",
			offset:   0,
			limit:    0,
			checkFor: []string{"local offset = 0", "local limit = 0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := gen.GetPixelsWithPagination("Layer 1", 1, 0, 0, 100, 100, tt.offset, tt.limit)

			for _, check := range tt.checkFor {
				if !strings.Contains(script, check) {
					t.Errorf("GetPixelsWithPagination() missing %q", check)
				}
			}

			if !strings.Contains(script, "Layer 1") {
				t.Error("GetPixelsWithPagination() missing layer name")
			}
		})
	}
}

func TestLuaGenerator_ApplyShading(t *testing.T) {
	gen := &LuaGenerator{}
	palette := []string{"#000000", "#808080", "#FFFFFF"}

	tests := []struct {
		name      string
		direction string
		style     string
		checkFor  []string
	}{
		{
			name:      "top_left smooth shading",
			direction: "top_left",
			style:     "smooth",
			checkFor:  []string{"local lightDx = -1", "local lightDy = -1", `local style = "smooth"`},
		},
		{
			name:      "bottom_right hard shading",
			direction: "bottom_right",
			style:     "hard",
			checkFor:  []string{"local lightDx = 1", "local lightDy = 1", `local style = "hard"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := gen.ApplyShading("Layer 1", 1, 0, 0, 64, 64, palette, tt.direction, 0.5, tt.style)

			for _, check := range tt.checkFor {
				if !strings.Contains(script, check) {
					t.Errorf("ApplyShading() missing %q", check)
				}
			}

			if !strings.Contains(script, "{r=0, g=0, b=0, a=255}") {
				t.Error("ApplyShading() missing palette color conversion")
			}
		})
	}
}

func TestParseHexColor(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Color
	}{
		{
			name:  "RGB with hash",
			input: "#FF0000",
			want:  Color{R: 255, G: 0, B: 0, A: 255},
		},
		{
			name:  "RGB without hash",
			input: "00FF00",
			want:  Color{R: 0, G: 255, B: 0, A: 255},
		},
		{
			name:  "RGBA with hash",
			input: "#0000FF80",
			want:  Color{R: 0, G: 0, B: 255, A: 128},
		},
		{
			name:  "invalid short",
			input: "#FFF",
			want:  Color{R: 0, G: 0, B: 0, A: 255},
		},
		{
			name:  "invalid long",
			input: "#FFFFFFFFFF",
			want:  Color{R: 0, G: 0, B: 0, A: 255},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseHexColor(tt.input)
			if got != tt.want {
				t.Errorf("parseHexColor(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestLuaGenerator_DownsampleImage(t *testing.T) {
	gen := &LuaGenerator{}
	script := gen.DownsampleImage("/path/to/source.png", "/path/to/output.aseprite", 64, 48)

	checks := []string{
		`app.open("/path/to/source.png")`,
		"local targetWidth = 64",
		"local targetHeight = 48",
		"local scaleX = srcWidth / targetWidth",
		"local scaleY = srcHeight / targetHeight",
		"-- Downsample using box filter",
		`targetSprite:saveAs("/path/to/output.aseprite")`,
	}

	for _, check := range checks {
		if !strings.Contains(script, check) {
			t.Errorf("DownsampleImage() missing %q", check)
		}
	}
}

func TestLuaGenerator_CreateTag_EdgeCases(t *testing.T) {
	gen := &LuaGenerator{}

	tests := []struct {
		name      string
		direction string
		checkFor  string
	}{
		{
			name:      "forward direction",
			direction: "forward",
			checkFor:  `aniDir = AniDir.FORWARD`,
		},
		{
			name:      "reverse direction",
			direction: "reverse",
			checkFor:  `aniDir = AniDir.REVERSE`,
		},
		{
			name:      "pingpong direction",
			direction: "pingpong",
			checkFor:  `aniDir = AniDir.PING_PONG`,
		},
		{
			name:      "invalid defaults to forward",
			direction: "invalid",
			checkFor:  `aniDir = AniDir.FORWARD`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := gen.CreateTag("test_tag", 1, 5, tt.direction)

			if !strings.Contains(script, tt.checkFor) {
				t.Errorf("CreateTag() missing %q", tt.checkFor)
			}
		})
	}
}

func TestLuaGenerator_DuplicateFrame_EdgeCases(t *testing.T) {
	gen := &LuaGenerator{}

	tests := []struct {
		name        string
		sourceFrame int
		insertAfter int
	}{
		{
			name:        "insert at end",
			sourceFrame: 1,
			insertAfter: 0,
		},
		{
			name:        "insert after frame 2",
			sourceFrame: 1,
			insertAfter: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := gen.DuplicateFrame(tt.sourceFrame, tt.insertAfter)

			if len(script) == 0 {
				t.Error("DuplicateFrame() generated empty script")
			}

			if !strings.Contains(script, "spr:newFrame") {
				t.Error("DuplicateFrame() missing newFrame operation")
			}
		})
	}
}

func TestLuaGenerator_ExportSpritesheet_EdgeCases(t *testing.T) {
	gen := &LuaGenerator{}

	tests := []struct {
		name        string
		layout      string
		includeJSON bool
		checkLayout string
	}{
		{
			name:        "horizontal with JSON",
			layout:      "horizontal",
			includeJSON: true,
			checkLayout: `type = "horizontal"`,
		},
		{
			name:        "vertical without JSON",
			layout:      "vertical",
			includeJSON: false,
			checkLayout: `type = "vertical"`,
		},
		{
			name:        "invalid defaults to horizontal",
			layout:      "invalid",
			includeJSON: false,
			checkLayout: `type = "horizontal"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := gen.ExportSpritesheet("/path/to/output.png", tt.layout, 2, tt.includeJSON)

			if len(script) == 0 {
				t.Error("ExportSpritesheet() generated empty script")
			}

			if !strings.Contains(script, tt.checkLayout) {
				t.Errorf("ExportSpritesheet() missing %q", tt.checkLayout)
			}

			if !strings.Contains(script, "ExportSpriteSheet") {
				t.Error("ExportSpritesheet() missing ExportSpriteSheet command")
			}
		})
	}
}

func TestLuaGenerator_SetPalette_EdgeCases(t *testing.T) {
	gen := &LuaGenerator{}

	tests := []struct {
		name   string
		colors []string
	}{
		{
			name:   "single color",
			colors: []string{"#FF0000"},
		},
		{
			name:   "many colors",
			colors: []string{"#FF0000", "#00FF00", "#0000FF", "#FFFF00"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := gen.SetPalette(tt.colors)

			if len(script) == 0 {
				t.Error("SetPalette() generated empty script")
			}

			// Check for Color{...} format which indicates color conversion happened
			if !strings.Contains(script, "Color{r=") {
				t.Error("SetPalette() missing color conversion")
			}
		})
	}
}

func TestLuaGenerator_SetPaletteColor_EdgeCases(t *testing.T) {
	gen := &LuaGenerator{}

	script := gen.SetPaletteColor(0, "#FF0000")

	if len(script) == 0 {
		t.Error("SetPaletteColor() generated empty script")
	}

	if !strings.Contains(script, "Color(") {
		t.Error("SetPaletteColor() missing Color conversion")
	}

	script2 := gen.SetPaletteColor(255, "#0000FF")

	if len(script2) == 0 {
		t.Error("SetPaletteColor() generated empty script for index 255")
	}
}

func TestLuaGenerator_AddPaletteColor_EdgeCases(t *testing.T) {
	gen := &LuaGenerator{}

	script := gen.AddPaletteColor("#ABCDEF")

	if len(script) == 0 {
		t.Error("AddPaletteColor() generated empty script")
	}

	// Check that it generates a valid script with palette operations
	if !strings.Contains(script, "Color(") {
		t.Error("AddPaletteColor() missing Color conversion")
	}
}

func TestLuaGenerator_SortPalette_EdgeCases(t *testing.T) {
	gen := &LuaGenerator{}

	tests := []struct {
		name      string
		method    string
		ascending bool
		wantSort  string
	}{
		{
			name:      "hue ascending",
			method:    "hue",
			ascending: true,
			wantSort:  "table.sort",
		},
		{
			name:      "saturation descending",
			method:    "saturation",
			ascending: false,
			wantSort:  "table.sort",
		},
		{
			name:      "brightness ascending",
			method:    "brightness",
			ascending: true,
			wantSort:  "table.sort",
		},
		{
			name:      "luminance descending",
			method:    "luminance",
			ascending: false,
			wantSort:  "table.sort",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := gen.SortPalette(tt.method, tt.ascending)

			if !strings.Contains(script, tt.wantSort) {
				t.Errorf("SortPalette() missing sort operation")
			}

			if !strings.Contains(script, tt.method) {
				t.Errorf("SortPalette() missing sort method %q", tt.method)
			}
		})
	}
}

func TestLuaGenerator_FlipSprite_AllTargets(t *testing.T) {
	gen := &LuaGenerator{}

	tests := []struct {
		name      string
		direction string
		target    string
	}{
		{name: "horizontal sprite", direction: "horizontal", target: "sprite"},
		{name: "vertical layer", direction: "vertical", target: "layer"},
		{name: "horizontal cel", direction: "horizontal", target: "cel"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := gen.FlipSprite(tt.direction, tt.target)

			if !strings.Contains(script, "app.command.Flip") {
				t.Error("FlipSprite() missing Flip command")
			}
		})
	}
}

func TestLuaGenerator_RotateSprite_AllAngles(t *testing.T) {
	gen := &LuaGenerator{}

	tests := []struct {
		name   string
		angle  int
		target string
	}{
		{name: "90 degrees", angle: 90, target: "sprite"},
		{name: "180 degrees", angle: 180, target: "sprite"},
		{name: "270 degrees", angle: 270, target: "sprite"},
		{name: "invalid angle defaults to 90", angle: 45, target: "sprite"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := gen.RotateSprite(tt.angle, tt.target)

			if !strings.Contains(script, "app.command.Rotate") {
				t.Error("RotateSprite() missing Rotate command")
			}
		})
	}
}

func TestLuaGenerator_ResizeCanvas_AllAnchors(t *testing.T) {
	gen := &LuaGenerator{}

	tests := []struct {
		name   string
		anchor string
	}{
		{name: "top_left", anchor: "top_left"},
		{name: "top_right", anchor: "top_right"},
		{name: "bottom_left", anchor: "bottom_left"},
		{name: "bottom_right", anchor: "bottom_right"},
		{name: "center", anchor: "center"},
		{name: "invalid defaults to center", anchor: "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := gen.ResizeCanvas(100, 100, tt.anchor)

			if !strings.Contains(script, "app.command.CanvasSize") {
				t.Error("ResizeCanvas() missing CanvasSize command")
			}

			if !strings.Contains(script, "newWidth = 100") {
				t.Error("ResizeCanvas() missing width parameter")
			}

			if !strings.Contains(script, "newHeight = 100") {
				t.Error("ResizeCanvas() missing height parameter")
			}
		})
	}
}

func TestLuaGenerator_DrawWithDither_AllPatterns(t *testing.T) {
	gen := &LuaGenerator{}

	patterns := []string{
		"bayer_2x2", "bayer_4x4", "bayer_8x8", "checkerboard",
		"grass", "water", "stone", "cloud", "brick", "dots",
		"diagonal", "cross", "noise", "horizontal_lines", "vertical_lines",
	}

	for _, pattern := range patterns {
		t.Run(pattern, func(t *testing.T) {
			script := gen.DrawWithDither("Layer 1", 1, 0, 0, 64, 64, "#FF0000", "#00FF00", pattern, 0.5)

			if !strings.Contains(script, "local matrix") {
				t.Errorf("DrawWithDither(%s) missing matrix definition", pattern)
			}

			if !strings.Contains(script, "local matrixSize") {
				t.Errorf("DrawWithDither(%s) missing matrixSize", pattern)
			}
		})
	}
}

func TestLuaGenerator_ApplyShading_AllDirections(t *testing.T) {
	gen := &LuaGenerator{}
	palette := []string{"#000000", "#808080", "#FFFFFF"}

	directions := []string{
		"top_left", "top", "top_right",
		"left", "right",
		"bottom_left", "bottom", "bottom_right",
	}

	for _, direction := range directions {
		t.Run(direction, func(t *testing.T) {
			script := gen.ApplyShading("Layer 1", 1, 0, 0, 64, 64, palette, direction, 0.5, "smooth")

			if !strings.Contains(script, "local lightDx") {
				t.Errorf("ApplyShading(%s) missing lightDx", direction)
			}

			if !strings.Contains(script, "local lightDy") {
				t.Errorf("ApplyShading(%s) missing lightDy", direction)
			}
		})
	}
}
