package aseprite

import (
	"fmt"
)

// ExportSprite generates a Lua script to export a sprite.
//
// Exports the sprite to a standard image format (PNG, GIF, JPG, BMP). Can export
// either all frames or a specific frame. For multi-frame exports, the output
// format determines the result (PNG = image sequence, GIF = animation).
//
// Parameters:
//   - outputPath: absolute path for output file (automatically escaped for Lua safety)
//   - frameNumber: frame to export (0 = export all frames, >0 = export specific frame)
//
// Export behavior:
//   - frameNumber = 0: exports all frames using saveCopyAs (behavior depends on format)
//   - frameNumber > 0: creates temporary sprite with single flattened frame
//
// Supported formats (detected from file extension):
//   - PNG: supports transparency, multi-frame creates image sequence
//   - GIF: supports animation, palette-based
//   - JPG/JPEG: no transparency, lossy compression
//   - BMP: no transparency, uncompressed
//
// Prints "Exported successfully" on success.
// Returns an error if:
//   - No sprite is active
//   - The frame number is invalid (when frameNumber > 0)
func (g *LuaGenerator) ExportSprite(outputPath string, frameNumber int) string {
	escapedPath := EscapeString(outputPath)

	if frameNumber > 0 {
		// Export specific frame by creating a temporary sprite with just that frame
		return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

if #spr.frames < %d then
	error("Frame not found: %d")
end

-- Create a new sprite with the same dimensions and color mode
local tempSpr = Sprite(spr.width, spr.height, spr.colorMode)

-- Copy the palette from the original sprite for indexed mode
if spr.colorMode == ColorMode.INDEXED then
	-- Match the source palette size (a new sprite starts with 256 entries)
	tempSpr.palettes[1]:resize(#spr.palettes[1])
	for i = 0, #spr.palettes[1]-1 do
		tempSpr.palettes[1]:setColor(i, spr.palettes[1]:getColor(i))
	end
	-- Set transparent color to 255 to allow palette index 0 for actual colors
	-- (after the resize, which clamps transparentColor into the palette range)
	tempSpr.transparentColor = 255
end

-- Delete the default layer that comes with new sprites
if #tempSpr.layers > 0 then
	tempSpr:deleteLayer(tempSpr.layers[1])
end

-- Copy the specific frame, preserving layer structure
local targetFrame = spr.frames[%d]
for _, layer in ipairs(spr.layers) do
	local cel = layer:cel(targetFrame)
	if cel then
		-- Create a new layer for each source layer
		local tempLayer = tempSpr:newLayer()
		tempSpr:newCel(tempLayer, 1, cel.image, cel.position)
	end
end

-- Flatten and export
app.command.FlattenLayers()
tempSpr:saveAs("%s")
tempSpr:close()

print("Exported successfully")`, frameNumber, frameNumber, frameNumber, escapedPath)
	}

	// Export all frames
	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

spr:saveCopyAs("%s")
print("Exported successfully")`, escapedPath)
}

// ExportSpritesheet generates a Lua script to export animation frames as a spritesheet.
//
// Combines all animation frames into a single image arranged according to the
// specified layout. Optionally generates JSON metadata with frame positions and
// timing information.
//
// Parameters:
//   - outputPath: absolute path for output image (automatically escaped for Lua safety)
//   - layout: frame arrangement - "horizontal", "vertical", "rows", "columns", or "packed"
//   - padding: spacing in pixels between frames and around sheet edges
//   - includeJSON: if true, exports metadata JSON alongside the image
//
// Layout options:
//   - "horizontal": single row, frames left to right
//   - "vertical": single column, frames top to bottom
//   - "rows": multiple rows, fills left to right then down
//   - "columns": multiple columns, fills top to bottom then right
//   - "packed": bin-packing algorithm for space efficiency
//
// When includeJSON is true, generates a .json file with:
//   - Frame positions and dimensions
//   - Frame durations for animation timing
//   - Layer information
//
// Prints JSON with spritesheet_path, frame_count, and optionally metadata_path.
// Returns an error if no sprite is active.
func (g *LuaGenerator) ExportSpritesheet(outputPath string, layout string, padding int, includeJSON bool) string {
	escapedOutput := EscapeString(outputPath)

	// Map layout to Aseprite's sheet type
	var sheetType string
	switch layout {
	case "horizontal":
		sheetType = "horizontal"
	case "vertical":
		sheetType = "vertical"
	case "rows":
		sheetType = "rows"
	case "columns":
		sheetType = "columns"
	case "packed":
		sheetType = "packed"
	default:
		sheetType = "horizontal"
	}

	dataFormat := ""
	if includeJSON {
		dataFormat = `dataFormat = "json",`
	}

	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

-- Export spritesheet
local outputPath = "%s"
local dataPath = nil

app.command.ExportSpriteSheet{
	type = "%s",
	textureFilename = outputPath,
	%s
	dataFilename = outputPath:gsub("%%.%%w+$", ".json"),
	borderPadding = %d,
	shapePadding = %d,
	innerPadding = %d,
	trim = false,
	extrude = false
}

-- Build output JSON
local result = string.format('{"spritesheet_path":"%%s","frame_count":%%d', outputPath, #spr.frames)
if %t then
	local jsonPath = outputPath:gsub("%%.%%w+$", ".json")
	result = result .. string.format(',"metadata_path":"%%s"', jsonPath)
end
result = result .. "}"
print(result)`, escapedOutput, sheetType, dataFormat, padding, padding, padding, includeJSON)
}

// ImportImage generates a Lua script to import an external image as a layer.
//
// Loads an external image file and places it as a cel in the specified layer
// and frame. If the layer doesn't exist, it is created automatically.
//
// Parameters:
//   - imagePath: absolute path to source image file (automatically escaped for Lua safety)
//   - layerName: name of target layer (created if not found, automatically escaped)
//   - frameNumber: 1-based frame index to import to
//   - x: optional x-position for the imported image (nil = 0)
//   - y: optional y-position for the imported image (nil = 0)
//
// The imported image is placed as a new cel with its top-left corner at (x, y).
// The image retains its original dimensions and color mode.
//
// The operation is wrapped in a transaction for atomicity and the sprite
// is saved after the image is imported.
//
// Prints "Image imported successfully" on success.
// Returns an error if:
//   - No sprite is active
//   - The image file cannot be loaded
//   - The frame number is invalid
func (g *LuaGenerator) ImportImage(imagePath, layerName string, frameNumber int, x, y *int) string {
	escapedImage := EscapeString(imagePath)
	escapedLayer := EscapeString(layerName)

	posX := 0
	posY := 0
	if x != nil {
		posX = *x
	}
	if y != nil {
		posY = *y
	}

	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

-- Load external image
local img = Image{ fromFile = "%s" }
if not img then
	error("Failed to load image: %s")
end

-- Find or create layer
local layer = nil
for i, lyr in ipairs(spr.layers) do
	if lyr.name == "%s" then
		layer = lyr
		break
	end
end

if not layer then
	-- Create new layer
	layer = spr:newLayer()
	layer.name = "%s"
end

local frame = spr.frames[%d]
if not frame then
	error("Frame not found: %d")
end

app.transaction(function()
	local finalImg = img

	-- Convert color mode if needed
	if img.colorMode ~= spr.colorMode then
		finalImg = Image(img.width, img.height, spr.colorMode)
		-- Use SRC blend mode to properly convert indexed colors to RGB
		finalImg:drawImage(img, Point(0, 0), 255, BlendMode.SRC)
	end

	-- Create cel with the image at the specified position
	spr:newCel(layer, frame, finalImg, %d, %d)
end)

spr:saveAs(spr.filename)
print("Image imported successfully")`,
		escapedImage, escapedImage,
		escapedLayer,
		escapedLayer,
		frameNumber, frameNumber,
		posX, posY)
}

// SaveAs generates a Lua script to save the sprite to a new file path.
//
// Saves the active sprite to a new location, effectively creating a copy or
// moving the sprite. The sprite remains open with the new path as its filename.
//
// Parameters:
//   - newPath: absolute path for the saved sprite (automatically escaped for Lua safety)
//
// The file format is determined by the file extension. Supported formats:
//   - .aseprite/.ase: native format (preserves all layers, frames, metadata)
//   - .png, .gif, .jpg, .bmp: exports to image format (see ExportSprite for details)
//
// Prints JSON with success status and file path: {"success":true,"file_path":"path"}
// Returns an error if no sprite is active.
func (g *LuaGenerator) SaveAs(newPath string) string {
	escapedPath := EscapeString(newPath)

	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

local newPath = "%s"

-- Save to new path
spr:saveAs(newPath)

print(string.format('{"success":true,"file_path":"%%s"}', newPath))`, escapedPath)
}
