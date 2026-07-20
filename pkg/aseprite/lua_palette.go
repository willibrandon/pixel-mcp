package aseprite

import (
	"fmt"
	"strings"
)

// SetPalette generates a Lua script to set the sprite's palette to the specified colors.
//
// Replaces the entire sprite palette with the provided color list. This is useful
// for applying extracted palettes from reference images or enforcing a specific
// color scheme across the sprite.
//
// Parameters:
//   - colors: slice of hex color strings in #RRGGBB or #RRGGBBAA format
//
// The palette is resized to match the color count (1-256 colors).
// Color strings are parsed to extract RGBA components:
//   - #RRGGBB format assumes full opacity (alpha = 255)
//   - #RRGGBBAA format includes alpha channel
//   - Invalid colors are skipped
//
// The sprite is saved after the palette is set.
//
// Prints "Palette set successfully" on success.
// Returns an error if:
//   - No sprite is active
//   - No palette exists in the sprite
//   - No colors are provided
func (g *LuaGenerator) SetPalette(colors []string) string {
	if len(colors) == 0 {
		return `error("No colors provided for palette")`
	}

	// Build palette color list
	colorList := "{\n"
	for i, hexColor := range colors {
		// Parse hex color #RRGGBB
		hexColor = strings.TrimPrefix(hexColor, "#")
		if len(hexColor) != 6 && len(hexColor) != 8 {
			continue
		}

		var r, g, b, a int
		// Parse hex color components (errors ignored as format is validated above)
		_, _ = fmt.Sscanf(hexColor[:2], "%x", &r)
		_, _ = fmt.Sscanf(hexColor[2:4], "%x", &g)
		_, _ = fmt.Sscanf(hexColor[4:6], "%x", &b)
		if len(hexColor) == 8 {
			_, _ = fmt.Sscanf(hexColor[6:8], "%x", &a)
		} else {
			a = 255
		}

		colorList += fmt.Sprintf("\t\tColor{r=%d, g=%d, b=%d, a=%d}", r, g, b, a)
		if i < len(colors)-1 {
			colorList += ","
		}
		colorList += "\n"
	}
	colorList += "\t}"

	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

-- Get or create palette
local palette = spr.palettes[1]
if not palette then
	error("No palette found")
end

-- Resize palette to match color count
palette:resize(%d)

-- Set palette colors
local colors = %s

for i, color in ipairs(colors) do
	palette:setColor(i - 1, color)  -- Palette is 0-indexed
end

%s
spr:saveAs(spr.filename)
print("Palette set successfully")`, len(colors), colorList, ReassertIndexedTransparentColor())
}

// GetPalette generates a Lua script to retrieve the sprite's current palette.
//
// Extracts all colors from the sprite's palette and returns them as a JSON object.
// This is useful for analyzing color usage, extracting palettes for reuse, or
// implementing palette-based drawing operations.
//
// Returns JSON with format: {"colors":["#RRGGBB","#RRGGBB",...],"size":N}
// The colors array contains hex strings without alpha channel.
//
// Returns an error if:
//   - No sprite is active
//   - No palette exists in the sprite
func (g *LuaGenerator) GetPalette() string {
	return `local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

-- Get palette
local palette = spr.palettes[1]
if not palette then
	error("No palette found")
end

-- Extract colors as hex strings
local colors = {}
for i = 0, #palette - 1 do
	local color = palette:getColor(i)
	local hex = string.format("#%02X%02X%02X", color.red, color.green, color.blue)
	table.insert(colors, hex)
end

-- Format as JSON
local colorList = '["' .. table.concat(colors, '","') .. '"]'
local output = string.format('{"colors":%s,"size":%d}', colorList, #palette)

print(output)`
}

// SetPaletteColor generates a Lua script to set a specific palette color at an index.
//
// Modifies a single color in the palette at the specified index. This is useful
// for fine-tuning palettes or implementing color ramps.
//
// Parameters:
//   - index: 0-based palette index (0 to palette_size-1)
//   - hexColor: color in #RRGGBB or #RRGGBBAA format
//
// The index is validated against the current palette size. Setting a color at
// a valid index replaces the existing color at that position.
//
// The sprite is saved after the color is set.
//
// Prints "Palette color set successfully" on success.
// Returns an error if:
//   - No sprite is active
//   - No palette exists in the sprite
//   - The index is out of range (negative or >= palette size)
func (g *LuaGenerator) SetPaletteColor(index int, hexColor string) string {
	// Parse hex color #RRGGBB
	hexColor = strings.TrimPrefix(hexColor, "#")

	var red, green, blue, alpha int
	// Parse hex color components (errors ignored as format is validated by caller)
	_, _ = fmt.Sscanf(hexColor[:2], "%x", &red)
	_, _ = fmt.Sscanf(hexColor[2:4], "%x", &green)
	_, _ = fmt.Sscanf(hexColor[4:6], "%x", &blue)
	if len(hexColor) == 8 {
		_, _ = fmt.Sscanf(hexColor[6:8], "%x", &alpha)
	} else {
		alpha = 255
	}

	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

-- Get palette
local palette = spr.palettes[1]
if not palette then
	error("No palette found")
end

-- Validate index
if %d < 0 or %d >= #palette then
	error(string.format("Palette index %%d out of range (palette has %%d colors)", %d, #palette))
end

-- Set color at index
palette:setColor(%d, Color{r=%d, g=%d, b=%d, a=%d})

spr:saveAs(spr.filename)
print("Palette color set successfully")`, index, index, index, index, red, green, blue, alpha)
}

// AddPaletteColor generates a Lua script to add a new color to the palette.
//
// Appends a new color to the end of the palette, expanding it by one entry.
// This is useful for incrementally building palettes or adding discovered colors.
//
// Parameters:
//   - hexColor: color in #RRGGBB or #RRGGBBAA format
//
// The palette is resized to accommodate the new color (up to maximum of 256 colors).
// The new color is assigned the next available index.
//
// The sprite is saved after the color is added.
//
// Prints JSON with the new color index: {"color_index":N}
// Returns an error if:
//   - No sprite is active
//   - No palette exists in the sprite
//   - Palette is already at maximum size (256 colors)
func (g *LuaGenerator) AddPaletteColor(hexColor string) string {
	// Parse hex color #RRGGBB
	hexColor = strings.TrimPrefix(hexColor, "#")

	var red, green, blue, alpha int
	// Parse hex color components (errors ignored as format is validated by caller)
	_, _ = fmt.Sscanf(hexColor[:2], "%x", &red)
	_, _ = fmt.Sscanf(hexColor[2:4], "%x", &green)
	_, _ = fmt.Sscanf(hexColor[4:6], "%x", &blue)
	if len(hexColor) == 8 {
		_, _ = fmt.Sscanf(hexColor[6:8], "%x", &alpha)
	} else {
		alpha = 255
	}

	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

-- Get palette
local palette = spr.palettes[1]
if not palette then
	error("No palette found")
end

-- Check if palette is at maximum size
if #palette >= 256 then
	error("Palette is already at maximum size (256 colors)")
end

-- Add new color to palette
local newIndex = #palette
palette:resize(#palette + 1)
palette:setColor(newIndex, Color{r=%d, g=%d, b=%d, a=%d})

%s
spr:saveAs(spr.filename)

-- Output JSON with color_index
local output = string.format('{"color_index":%%d}', newIndex)
print(output)`, red, green, blue, alpha, ReassertIndexedTransparentColor())
}

// SortPalette generates a Lua script to sort the palette by a specified method.
//
// Reorders palette colors based on HSL color space properties. This is useful
// for organizing palettes logically, creating color ramps, or improving palette
// usability in indexed color mode.
//
// Parameters:
//   - method: sorting criterion - "hue", "saturation", "brightness", or "luminance"
//   - ascending: if true, sorts from low to high; if false, sorts from high to low
//
// Sorting methods:
//   - "hue": sorts by color hue (red → yellow → green → cyan → blue → magenta)
//   - "saturation": sorts by color intensity (gray → vivid)
//   - "brightness" or "luminance": sorts by perceived lightness (dark → light)
//
// The sort is performed by converting each color to HSL color space, sorting by
// the selected component, then applying the reordered colors back to the palette.
//
// The sprite is saved after the palette is sorted.
//
// Prints "Palette sorted by [method] successfully" on success.
// Returns an error if:
//   - No sprite is active
//   - No palette exists in the sprite
func (g *LuaGenerator) SortPalette(method string, ascending bool) string {
	// Helper function to convert RGB to HSL in Lua
	rgbToHSL := `-- Convert RGB to HSL
local function rgbToHSL(r, g, b)
	r, g, b = r / 255, g / 255, b / 255
	local max = math.max(r, g, b)
	local min = math.min(r, g, b)
	local delta = max - min

	local h, s, l = 0, 0, (max + min) / 2

	if delta ~= 0 then
		-- Calculate saturation
		if l < 0.5 then
			s = delta / (max + min)
		else
			s = delta / (2.0 - max - min)
		end

		-- Calculate hue
		if max == r then
			h = ((g - b) / delta)
			if g < b then
				h = h + 6.0
			end
		elseif max == g then
			h = ((b - r) / delta) + 2.0
		elseif max == b then
			h = ((r - g) / delta) + 4.0
		end

		h = h * 60.0
	end

	return h, s, l
end`

	// Sort function based on method
	var sortKey string
	switch method {
	case "hue":
		sortKey = "h"
	case "saturation":
		sortKey = "s"
	case "brightness", "luminance":
		sortKey = "l"
	default:
		sortKey = "h"
	}

	sortDirection := ">"
	if ascending {
		sortDirection = "<"
	}

	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

-- Get palette
local palette = spr.palettes[1]
if not palette then
	error("No palette found")
end

%s

-- Extract colors with HSL values
local colors = {}
for i = 0, #palette - 1 do
	local color = palette:getColor(i)
	local h, s, l = rgbToHSL(color.red, color.green, color.blue)
	table.insert(colors, {
		index = i,
		color = color,
		h = h,
		s = s,
		l = l
	})
end

-- Sort colors by %s
table.sort(colors, function(a, b)
	return a.%s %s b.%s
end)

-- Apply sorted colors back to palette
for i, entry in ipairs(colors) do
	palette:setColor(i - 1, entry.color)
end

spr:saveAs(spr.filename)
print("Palette sorted by %s successfully")`, rgbToHSL, method, sortKey, sortDirection, sortKey, method)
}

// ApplyShading generates a Lua script to apply palette-constrained shading based on light direction.
//
// Applies realistic shading to a region by simulating light and shadow using only
// colors from the provided palette. This enables professional pixel art shading
// techniques while maintaining palette constraints.
//
// Parameters:
//   - layerName: name of the layer to shade (automatically escaped for Lua safety)
//   - frameNumber: 1-based frame index to shade
//   - x, y: top-left corner of the shading region
//   - width: region width in pixels
//   - height: region height in pixels
//   - palette: ordered slice of colors from darkest to lightest (hex strings #RRGGBB)
//   - lightDirection: light source position - "top_left", "top", "top_right", "left", "right", "bottom_left", "bottom", "bottom_right"
//   - intensity: shading strength (0.0-1.0, where 0 = no shading, 1 = maximum contrast)
//   - style: shading style - "smooth", "hard", or "pillow"
//
// Shading styles:
//   - "smooth": gradual luminance-based transitions between palette colors
//   - "hard": sharp, cell-shaded look with minimal blending
//   - "pillow": pillow shading with highlights in center, shadows at edges
//
// Algorithm:
//  1. For each pixel, determine light/shadow based on light direction and style
//  2. Calculate target luminance adjustment based on intensity
//  3. Find closest palette color matching the target luminance
//  4. Replace pixel with palette-constrained shaded color
//
// The operation is wrapped in a transaction for atomicity and the sprite
// is saved after shading is applied.
//
// Prints "Shading applied successfully" on success.
// Returns an error if:
//   - No sprite is active
//   - The layer is not found
//   - The frame number is invalid
//   - No cel exists at the specified layer/frame
func (g *LuaGenerator) ApplyShading(layerName string, frameNumber int, x, y, width, height int, palette []string, lightDirection string, intensity float64, style string) string {
	escapedLayerName := EscapeString(layerName)

	// Build palette color list
	paletteColors := "{\n"
	for i, hexColor := range palette {
		c := parseHexColor(hexColor)
		paletteColors += fmt.Sprintf("\t\t{r=%d, g=%d, b=%d, a=%d}", c.R, c.G, c.B, c.A)
		if i < len(palette)-1 {
			paletteColors += ","
		}
		paletteColors += "\n"
	}
	paletteColors += "\t}"

	// Map light direction to offset vectors
	var dx, dy int
	switch lightDirection {
	case "top_left":
		dx, dy = -1, -1
	case "top":
		dx, dy = 0, -1
	case "top_right":
		dx, dy = 1, -1
	case "left":
		dx, dy = -1, 0
	case "right":
		dx, dy = 1, 0
	case "bottom_left":
		dx, dy = -1, 1
	case "bottom":
		dx, dy = 0, 1
	case "bottom_right":
		dx, dy = 1, 1
	default:
		dx, dy = -1, -1 // Default to top_left
	}

	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

-- Find layer
local layer = nil
for _, l in ipairs(spr.layers) do
	if l.name == "%s" then
		layer = l
		break
	end
end

if not layer then
	error("Layer not found: %s")
end

-- Get frame
local frame = spr.frames[%d]
if not frame then
	error("Frame not found: %d")
end

-- Get or create cel
local cel = layer:cel(frame)
if not cel then
	error("No cel found in frame %d")
end

local img = cel.image

-- Palette colors (ordered darkest to lightest)
local palette = %s

-- Light direction offset
local lightDx = %d
local lightDy = %d
local intensity = %f
local style = "%s"

-- Helper: Find nearest palette color
local function findNearestPaletteColor(r, g, b, a)
	local minDist = math.huge
	local nearestColor = palette[1]

	for _, palColor in ipairs(palette) do
		local dr = r - palColor.r
		local dg = g - palColor.g
		local db = b - palColor.b
		local da = a - palColor.a
		local dist = dr*dr + dg*dg + db*db + da*da

		if dist < minDist then
			minDist = dist
			nearestColor = palColor
		end
	end

	return app.pixelColor.rgba(nearestColor.r, nearestColor.g, nearestColor.b, nearestColor.a)
end

-- Helper: Calculate shading factor based on position and light direction
local function calculateShadingFactor(px, py, regionX, regionY, regionW, regionH)
	-- Normalize position to 0-1
	local normX = (px - regionX) / regionW
	local normY = (py - regionY) / regionH

	-- Calculate distance from light source direction
	local lightFactor = 0.5
	if lightDx < 0 then
		lightFactor = lightFactor + (1 - normX) * 0.5
	elseif lightDx > 0 then
		lightFactor = lightFactor + normX * 0.5
	end

	if lightDy < 0 then
		lightFactor = lightFactor + (1 - normY) * 0.5
	elseif lightDy > 0 then
		lightFactor = lightFactor + normY * 0.5
	end

	-- Normalize to 0-1 range
	lightFactor = math.max(0, math.min(1, lightFactor))

	return lightFactor
end

-- Helper: Apply shading to color
local function applyShading(pixelValue, shadingFactor)
	local r = app.pixelColor.rgbaR(pixelValue)
	local g = app.pixelColor.rgbaG(pixelValue)
	local b = app.pixelColor.rgbaB(pixelValue)
	local a = app.pixelColor.rgbaA(pixelValue)

	if a == 0 then
		return pixelValue  -- Skip transparent pixels
	end

	-- Apply shading based on style
	local shadedR, shadedG, shadedB

	if style == "hard" then
		-- Hard shading: quantize to palette steps
		local paletteIndex = math.floor(shadingFactor * (#palette - 1)) + 1
		paletteIndex = math.max(1, math.min(#palette, paletteIndex))
		return app.pixelColor.rgba(palette[paletteIndex].r, palette[paletteIndex].g, palette[paletteIndex].b, a)
	elseif style == "smooth" then
		-- Smooth shading: blend current color toward palette extremes
		local targetBrightness = shadingFactor
		local currentBrightness = (0.299 * r + 0.587 * g + 0.114 * b) / 255

		-- Blend toward target brightness
		local blend = intensity * 0.5
		shadedR = math.floor(r * (1 - blend) + r * (targetBrightness / math.max(0.01, currentBrightness)) * blend)
		shadedG = math.floor(g * (1 - blend) + g * (targetBrightness / math.max(0.01, currentBrightness)) * blend)
		shadedB = math.floor(b * (1 - blend) + b * (targetBrightness / math.max(0.01, currentBrightness)) * blend)
	else  -- pillow (avoid - but included for completeness)
		-- Pillow shading: center highlight regardless of light direction
		local centerX = 0.5
		local centerY = 0.5
		local distFromCenter = math.sqrt((shadingFactor - centerX)^2 + (shadingFactor - centerY)^2)
		local pillow = 1 - distFromCenter

		shadedR = math.floor(r * (1 + pillow * intensity))
		shadedG = math.floor(g * (1 + pillow * intensity))
		shadedB = math.floor(b * (1 + pillow * intensity))
	end

	-- Clamp values
	shadedR = math.max(0, math.min(255, shadedR))
	shadedG = math.max(0, math.min(255, shadedG))
	shadedB = math.max(0, math.min(255, shadedB))

	-- Find nearest palette color
	return findNearestPaletteColor(shadedR, shadedG, shadedB, a)
end

-- Apply shading to region
local affectedPixels = 0

app.transaction(function()
	for py = %d, %d do
		for px = %d, %d do
			-- Adjust coordinates relative to cel position
			local imgX = px - cel.position.x
			local imgY = py - cel.position.y

			-- Check if coordinates are within image bounds
			if imgX >= 0 and imgX < img.width and imgY >= 0 and imgY < img.height then
				local pixelValue = img:getPixel(imgX, imgY)
				local alpha = app.pixelColor.rgbaA(pixelValue)

				if alpha > 0 then
					local shadingFactor = calculateShadingFactor(px, py, %d, %d, %d, %d)
					local shadedColor = applyShading(pixelValue, shadingFactor)
					img:drawPixel(imgX, imgY, shadedColor)
					affectedPixels = affectedPixels + 1
				end
			end
		end
	end
end)

spr:saveAs(spr.filename)
print(string.format("Shading applied to %%d pixels", affectedPixels))`,
		escapedLayerName, escapedLayerName,
		frameNumber, frameNumber, frameNumber,
		paletteColors,
		dx, dy, intensity, style,
		y, y+height-1, x, x+width-1,
		x, y, width, height)
}

// parseHexColor parses a hex color string and returns RGBA components.
func parseHexColor(hexColor string) Color {
	hexColor = strings.TrimPrefix(hexColor, "#")
	if len(hexColor) != 6 && len(hexColor) != 8 {
		return Color{R: 0, G: 0, B: 0, A: 255}
	}

	var r, g, b, a int
	// Parse hex color components (errors ignored as format is validated above)
	_, _ = fmt.Sscanf(hexColor[:2], "%x", &r)
	_, _ = fmt.Sscanf(hexColor[2:4], "%x", &g)
	_, _ = fmt.Sscanf(hexColor[4:6], "%x", &b)
	if len(hexColor) == 8 {
		_, _ = fmt.Sscanf(hexColor[6:8], "%x", &a)
	} else {
		a = 255
	}

	return Color{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}
}
