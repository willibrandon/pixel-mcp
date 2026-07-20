package aseprite

import (
	"fmt"
	"strings"
)

// ApplyAutoShadingResult generates a Lua script to apply auto-shading results to a sprite.
//
// This script:
//  1. Loads the shaded image from a temporary file
//  2. Applies the shaded pixels to the specified layer
//  3. Adds generated colors to the palette
//  4. Returns JSON with colors added, final palette, and regions shaded count
//
// Parameters:
//   - tempImagePath: path to temporary PNG with shaded image
//   - layerName: name of layer to apply shading to
//   - frameNumber: frame number (1-based)
//   - generatedColors: array of hex colors generated during shading
//   - regionsShadedCount: number of regions that were shaded
//
// The script imports the shaded image and applies it to the specified layer/frame.
func (g *LuaGenerator) ApplyAutoShadingResult(tempImagePath string, layerName string, frameNumber int, generatedColors []string, regionsShadedCount int) string {
	// Build color list for palette addition
	colorList := "{\n"
	for i, hexColor := range generatedColors {
		hexColor = strings.TrimPrefix(hexColor, "#")
		if len(hexColor) != 6 {
			continue
		}

		var r, g, b int
		_, _ = fmt.Sscanf(hexColor[:2], "%x", &r)
		_, _ = fmt.Sscanf(hexColor[2:4], "%x", &g)
		_, _ = fmt.Sscanf(hexColor[4:6], "%x", &b)

		colorList += fmt.Sprintf("\t\tColor{r=%d, g=%d, b=%d, a=255}", r, g, b)
		if i < len(generatedColors)-1 {
			colorList += ","
		}
		colorList += "\n"
	}
	colorList += "\t}"

	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

-- Find target layer
local targetLayer = nil
for _, layer in ipairs(spr.layers) do
	if layer.name == %q then
		targetLayer = layer
		break
	end
end

if not targetLayer then
	error("Layer not found: " .. %q)
end

-- Load shaded image
local shadedImg = Image{fromFile=%q}
if not shadedImg then
	error("Failed to load shaded image")
end

-- Get target cel
local cel = targetLayer:cel(%d)
if not cel then
	error("No cel found at frame " .. %d)
end

-- Create new image with shaded content
app.transaction(function()
	local finalImg = shadedImg

	-- Convert color mode if needed
	if shadedImg.colorMode ~= spr.colorMode then
		finalImg = Image(shadedImg.width, shadedImg.height, spr.colorMode)
		-- Use SRC blend mode to properly convert colors
		finalImg:drawImage(shadedImg, Point(0, 0), 255, BlendMode.SRC)
	end

	-- Delete the old cel and create a new one with the shaded image
	local celX = cel.position.x
	local celY = cel.position.y
	spr:deleteCel(cel)
	spr:newCel(targetLayer, %d, finalImg, celX, celY)
end)

-- Add generated colors to palette
local palette = spr.palettes[1]
if palette then
	local generatedColors = %s
	local currentSize = #palette

	-- Extend palette to fit new colors
	if currentSize + #generatedColors <= 256 then
		palette:resize(currentSize + #generatedColors)
		for i, color in ipairs(generatedColors) do
			palette:setColor(currentSize + i - 1, color)
		end
	end
end
%s
-- Build final palette for JSON output
local paletteHex = {}
for i = 0, #palette - 1 do
	local c = palette:getColor(i)
	table.insert(paletteHex, string.format("#%%02X%%02X%%02X", c.red, c.green, c.blue))
end

-- Build JSON output
local json = string.format([[{
	"success": true,
	"colors_added": %d,
	"palette": [%%s],
	"regions_shaded": %d
}]], '"' .. table.concat(paletteHex, '", "') .. '"')

-- Save sprite
spr:saveAs(spr.filename)

-- Print JSON result
print(json)`,
		EscapeString(layerName),           // layer name for finding
		EscapeString(layerName),           // layer name for error
		tempImagePath,                     // shaded image path
		frameNumber,                       // frame number for cel lookup
		frameNumber,                       // frame number for error message
		frameNumber,                       // frame number for newCel
		colorList,                         // generated colors
		ReassertIndexedTransparentColor(), // keep transparentColor out of the real palette range
		len(generatedColors),              // colors_added
		regionsShadedCount)                // regions_shaded
}
