package aseprite

import (
	"fmt"
	"strings"
)

// ApplyQuantizedPalette generates a Lua script to apply a quantized palette to a sprite.
//
// This script:
//  1. Sets the sprite's palette to the quantized colors
//  2. Remaps all pixels to the nearest palette color (with optional dithering)
//  3. Optionally converts the sprite to indexed color mode
//  4. Returns JSON with original color count, quantized color count, color mode, palette, and algorithm used
//
// Parameters:
//   - palette: slice of hex color strings (#RRGGBB or #RRGGBBAA format) from quantization
//   - originalColors: number of unique colors in the original sprite
//   - algorithm: name of the quantization algorithm used
//   - convertToIndexed: whether to convert the sprite to indexed color mode
//   - dither: whether dithering was applied (for reporting purposes)
//
// The script uses Aseprite's built-in color quantization when converting to indexed mode.
func (g *LuaGenerator) ApplyQuantizedPalette(palette []string, originalColors int, algorithm string, convertToIndexed bool, dither bool) string {
	if len(palette) == 0 {
		return `error("No palette colors provided")`
	}

	// Build palette color list
	colorList := "{\n"
	for i, hexColor := range palette {
		// Parse hex color #RRGGBB or #RRGGBBAA
		hexColor = strings.TrimPrefix(hexColor, "#")

		var r, g, b, a int
		if len(hexColor) == 8 {
			// #RRGGBBAA format (with alpha)
			_, _ = fmt.Sscanf(hexColor[:2], "%x", &r)
			_, _ = fmt.Sscanf(hexColor[2:4], "%x", &g)
			_, _ = fmt.Sscanf(hexColor[4:6], "%x", &b)
			_, _ = fmt.Sscanf(hexColor[6:8], "%x", &a)
		} else if len(hexColor) == 6 {
			// #RRGGBB format (assume full opacity)
			_, _ = fmt.Sscanf(hexColor[:2], "%x", &r)
			_, _ = fmt.Sscanf(hexColor[2:4], "%x", &g)
			_, _ = fmt.Sscanf(hexColor[4:6], "%x", &b)
			a = 255
		} else {
			continue // Skip invalid colors
		}

		colorList += fmt.Sprintf("\t\tColor{r=%d, g=%d, b=%d, a=%d}", r, g, b, a)
		if i < len(palette)-1 {
			colorList += ","
		}
		colorList += "\n"
	}
	colorList += "\t}"

	conversionCode := ""
	if convertToIndexed {
		conversionCode = `
-- Convert to indexed color mode
app.command.ChangePixelFormat{format="indexed"}

-- Get the new color mode after conversion
colorMode = "indexed"`
	} else {
		conversionCode = `
-- Keep RGB mode, just apply palette
colorMode = tostring(spr.colorMode):match("ColorMode%%.(%%w+)")`
	}

	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

-- Store original color mode for reporting
local colorMode = tostring(spr.colorMode):match("ColorMode%%.(%%w+)")

-- Get or create palette
local palette = spr.palettes[1]
if not palette then
	error("No palette found")
end

-- Resize palette to match quantized color count
palette:resize(%d)

-- Set palette colors
local colors = %s

for i, color in ipairs(colors) do
	palette:setColor(i - 1, color)  -- Palette is 0-indexed
end
%s
%s
-- Prepare palette for JSON output
local paletteHex = {}
for i = 0, #palette - 1 do
	local c = palette:getColor(i)
	if c.alpha == 0 then
		table.insert(paletteHex, string.format("#%%02X%%02X%%02X%%02X", c.red, c.green, c.blue, c.alpha))
	else
		table.insert(paletteHex, string.format("#%%02X%%02X%%02X", c.red, c.green, c.blue))
	end
end

-- Build JSON output
local json = string.format([[{
	"success": true,
	"original_colors": %d,
	"quantized_colors": %d,
	"color_mode": "%%s",
	"palette": [%%s],
	"algorithm_used": "%s"
}]], colorMode, '"' .. table.concat(paletteHex, '", "') .. '"')

-- Save sprite
spr:saveAs(spr.filename)

-- Print JSON result
print(json)`,
		len(palette),                      // palette resize
		colorList,                         // color list
		conversionCode,                    // conversion code
		ReassertIndexedTransparentColor(), // keep transparentColor out of the real palette range
		originalColors,                    // original_colors
		len(palette),                      // quantized_colors
		EscapeString(algorithm))           // algorithm_used
}

// ReplaceWithImage generates a Lua script to replace sprite content with an external image.
// This flattens all layers and replaces the content with the provided image.
func (g *LuaGenerator) ReplaceWithImage(imagePath string) string {
	escapedPath := EscapeString(imagePath)

	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

-- Load external image
local newImg = Image{ fromFile = "%s" }
if not newImg then
	error("Failed to load image: %s")
end

app.transaction(function()
	-- Flatten all layers to a single layer
	spr:flatten()

	-- Get the flattened layer (should be only layer now)
	local layer = spr.layers[1]
	if not layer then
		error("No layer found after flattening")
	end

	-- Get the cel at frame 1
	local cel = layer:cel(1)
	if not cel then
		-- Create cel if it doesn't exist
		cel = spr:newCel(layer, 1)
	end

	-- Convert color mode if needed
	local finalImg = newImg
	if newImg.colorMode ~= spr.colorMode then
		finalImg = Image(newImg.width, newImg.height, spr.colorMode)
		finalImg:drawImage(newImg, Point(0, 0), 255, BlendMode.SRC)
	end

	-- Replace cel image
	local celX = cel.position.x
	local celY = cel.position.y
	spr:deleteCel(cel)
	spr:newCel(layer, 1, finalImg, celX, celY)
end)

spr:saveAs(spr.filename)
print("Sprite content replaced successfully")`, escapedPath, escapedPath)
}
