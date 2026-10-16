package aseprite

import (
	"fmt"
)

// GetPixels generates a Lua script to read pixel data from a rectangular region.
//
// Reads all pixel colors from the specified rectangular region on a layer.
// This is a convenience wrapper around GetPixelsWithPagination with no pagination.
//
// Parameters:
//   - layerName: name of the layer to read from (automatically escaped for Lua safety)
//   - frameNumber: 1-based frame index to read from
//   - x, y: top-left corner of the read region
//   - width: region width in pixels
//   - height: region height in pixels
//
// Returns JSON array of pixels with format: [{"x":N,"y":N,"color":"#RRGGBBAA"},...]
// Pixels outside the cel bounds are skipped (not included in output).
//
// For large regions, consider using GetPixelsWithPagination to limit memory usage.
//
// Returns an error if:
//   - No sprite is active
//   - The layer is not found
//   - The frame number is invalid
//   - No cel exists at the specified layer/frame
func (g *LuaGenerator) GetPixels(layerName string, frameNumber int, x, y, width, height int) string {
	return g.GetPixelsWithPagination(layerName, frameNumber, x, y, width, height, 0, 0)
}

// GetPixelsWithPagination generates a Lua script to read pixel data with pagination support.
//
// Reads pixel colors from a rectangular region with optional offset/limit for
// pagination. This is useful for reading large regions in chunks or sampling
// specific portions of a layer.
//
// Parameters:
//   - layerName: name of the layer to read from (automatically escaped for Lua safety)
//   - frameNumber: 1-based frame index to read from
//   - x, y: top-left corner of the read region
//   - width: region width in pixels
//   - height: region height in pixels
//   - offset: number of pixels to skip from start (0 = start from first pixel)
//   - limit: maximum number of pixels to return (0 = return all pixels)
//
// Pixels are indexed in row-major order (left to right, top to bottom).
// For example, in a 10x10 region:
//   - Pixel 0 is at (x, y)
//   - Pixel 10 is at (x, y+1)
//   - offset=5, limit=10 returns pixels 5-14
//
// Returns JSON array of pixels with format: [{"x":N,"y":N,"color":"#RRGGBBAA"},...]
// Pixels outside the cel bounds are skipped (not included in output).
//
// Returns an error if:
//   - No sprite is active
//   - The layer is not found
//   - The frame number is invalid
//   - No cel exists at the specified layer/frame
func (g *LuaGenerator) GetPixelsWithPagination(layerName string, frameNumber int, x, y, width, height int, offset int, limit int) string {
	escapedName := EscapeString(layerName)
	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

-- Find layer by name
local layer = nil
for i, lyr in ipairs(spr.layers) do
	if lyr.name == "%s" then
		layer = lyr
		break
	end
end

if not layer then
	error("Layer not found: %s")
end

local frame = spr.frames[%d]
if not frame then
	error("Frame not found: %d")
end

local cel = layer:cel(frame)
if not cel then
	error("No cel found in frame %d")
end

local img = cel.image
local offset = %d
local limit = %d

-- Calculate pagination bounds
local pixelIndex = 0
local startIdx = offset
local endIdx = (limit > 0) and (offset + limit - 1) or -1

local pixels = {}

-- Get palette for indexed color mode
local palette = spr.palettes[1]

-- Read pixels from the specified region
for py = %d, %d do
	for px = %d, %d do
		-- Check if we're within the pagination range
		if endIdx < 0 or (pixelIndex >= startIdx and pixelIndex <= endIdx) then
			-- Adjust coordinates relative to cel position
			local imgX = px - cel.position.x
			local imgY = py - cel.position.y

			-- Check if coordinates are within image bounds
			if imgX >= 0 and imgX < img.width and imgY >= 0 and imgY < img.height then
				local pixelValue = img:getPixel(imgX, imgY)

				-- Convert pixel value to RGBA
				local r, g, b, a
				if spr.colorMode == ColorMode.INDEXED then
					-- In indexed mode, pixelValue is a palette index
					-- Check if index is within palette bounds
					if pixelValue >= 0 and pixelValue < #palette then
						-- Look up the actual color in the palette
						local palColor = palette:getColor(pixelValue)
						r = palColor.red
						g = palColor.green
						b = palColor.blue
						a = palColor.alpha
					else
						-- Out of bounds index, treat as transparent
						r, g, b, a = 0, 0, 0, 0
					end
				else
					-- In RGB/Grayscale mode, extract color components directly
					r = app.pixelColor.rgbaR(pixelValue)
					g = app.pixelColor.rgbaG(pixelValue)
					b = app.pixelColor.rgbaB(pixelValue)
					a = app.pixelColor.rgbaA(pixelValue)
				end

				-- Store as hex color
				local hexColor = string.format("#%%02X%%02X%%02X%%02X", r, g, b, a)
				table.insert(pixels, string.format('{"x":%%d,"y":%%d,"color":"%%s"}', px, py, hexColor))
			end
		end

		pixelIndex = pixelIndex + 1

		-- Early exit once we've passed the pagination range
		if endIdx >= 0 and pixelIndex > endIdx then
			break
		end
	end

	-- Early exit from outer loop
	if endIdx >= 0 and pixelIndex > endIdx then
		break
	end
end

-- Output as JSON array
print("[" .. table.concat(pixels, ",") .. "]")`,
		escapedName, escapedName,
		frameNumber, frameNumber, frameNumber,
		offset, limit,
		y, y+height-1, x, x+width-1)
}
