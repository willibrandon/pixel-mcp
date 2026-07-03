package aseprite

import "fmt"

// CreateCanvas generates a Lua script to create a new sprite.
//
// Creates a sprite with the specified dimensions and color mode, then saves it to disk.
// The sprite is created using Aseprite's Sprite constructor and saved to the provided path.
//
// For indexed color mode sprites, the transparent color is set to palette index 255 instead
// of the default index 0. This allows palette index 0 to be used for actual colors without
// being treated as transparent by Aseprite's cel trimming operations.
//
// Parameters:
//   - width: sprite width in pixels (1-65535)
//   - height: sprite height in pixels (1-65535)
//   - colorMode: color mode (RGB, Grayscale, or Indexed)
//   - filename: absolute path where the sprite file should be saved
//
// The generated script prints the filename on success.
func (g *LuaGenerator) CreateCanvas(width, height int, colorMode ColorMode, filename string) string {
	escapedFilename := EscapeString(filename)

	// For indexed sprites, set transparent color to index 255 to allow index 0 to be used for actual colors
	if colorMode == ColorModeIndexed {
		return fmt.Sprintf(`local spr = Sprite(%d, %d, %s)
spr.transparentColor = 255
-- A new indexed sprite's initial cel is filled with palette index 0, but index
-- 255 is the transparent index (set above), so index 0 is now an opaque color
-- and the canvas would start as an opaque background instead of transparent.
-- Clear the initial cel to the transparent color so a fresh indexed canvas is
-- genuinely transparent.
local initialCel = spr.layers[1]:cel(spr.frames[1])
if initialCel and initialCel.image then
	initialCel.image:clear(spr.transparentColor)
end
spr:saveAs("%s")
print("%s")`, width, height, colorMode.ToLua(), escapedFilename, escapedFilename)
	}

	return fmt.Sprintf(`local spr = Sprite(%d, %d, %s)
spr:saveAs("%s")
print("%s")`, width, height, colorMode.ToLua(), escapedFilename, escapedFilename)
}

// FlattenLayers generates a Lua script to flatten all visible layers into a single layer.
//
// This merges all visible layers into one layer, compositing them according to their
// blend modes and opacity settings. The result is a single layer containing the
// flattened image.
//
// This is useful before exporting when you want a single composite image.
func (g *LuaGenerator) FlattenLayers() string {
	return `local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

app.transaction(function()
	spr:flatten()
end)

spr:saveAs(spr.filename)
print("Layers flattened successfully")`
}

// GetSpriteInfo generates a Lua script to retrieve sprite metadata.
//
// Extracts complete metadata from the active sprite including:
//   - Dimensions (width, height)
//   - Color mode (RGB, Grayscale, or Indexed)
//   - Frame count
//   - Layer count and names
//
// The script outputs JSON-formatted sprite information to stdout.
// Returns an error if no sprite is active.
func (g *LuaGenerator) GetSpriteInfo() string {
	return `local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

-- Map color mode enum to string
local colorModeStr = "rgb"
if spr.colorMode == ColorMode.GRAYSCALE then
	colorModeStr = "grayscale"
elseif spr.colorMode == ColorMode.INDEXED then
	colorModeStr = "indexed"
end

-- Collect layer names
local layers = {}
for i, layer in ipairs(spr.layers) do
	table.insert(layers, layer.name)
end

-- Format as JSON-like output
local output = string.format([[{
	"width": %d,
	"height": %d,
	"color_mode": "%s",
	"frame_count": %d,
	"layer_count": %d,
	"layers": ["%s"]
}]],
	spr.width,
	spr.height,
	colorModeStr,
	#spr.frames,
	#spr.layers,
	table.concat(layers, '","')
)

print(output)`
}

// AddLayer generates a Lua script to add a new layer.
//
// Creates a new layer in the active sprite with the specified name.
// The layer is added above all existing layers in the layer stack.
//
// The operation is wrapped in a transaction for atomicity and the sprite
// is saved after the layer is created.
//
// Parameters:
//   - layerName: name for the new layer (automatically escaped for Lua safety)
//
// Prints "Layer added successfully" on success.
// Returns an error if no sprite is active.
func (g *LuaGenerator) AddLayer(layerName string) string {
	escapedName := EscapeString(layerName)
	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

app.transaction(function()
	local layer = spr:newLayer()
	layer.name = "%s"
end)

spr:saveAs(spr.filename)
print("Layer added successfully")`, escapedName)
}

// AddFrame generates a Lua script to add a new frame.
//
// Creates a new animation frame at the end of the frame sequence with the
// specified duration. Frame durations control animation playback speed.
//
// The operation is wrapped in a transaction for atomicity and the sprite
// is saved after the frame is created.
//
// Parameters:
//   - durationMs: frame duration in milliseconds (converted to seconds for Aseprite)
//
// Prints the total frame count on success.
// Returns an error if no sprite is active.
func (g *LuaGenerator) AddFrame(durationMs int) string {
	durationSec := float64(durationMs) / 1000.0
	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

app.transaction(function()
	local frame = spr:newFrame()
	frame.duration = %.3f
end)

spr:saveAs(spr.filename)
print(#spr.frames)`, durationSec)
}

// DeleteLayer generates a Lua script to delete a layer.
//
// Removes the specified layer from the active sprite. The layer is found by name.
// As a safety measure, the script prevents deletion of the last remaining layer.
//
// The operation is wrapped in a transaction for atomicity and the sprite
// is saved after the layer is deleted.
//
// Parameters:
//   - layerName: name of the layer to delete (automatically escaped for Lua safety)
//
// Prints "Layer deleted successfully" on success.
// Returns an error if:
//   - No sprite is active
//   - The layer is not found
//   - Attempting to delete the last layer in the sprite
func (g *LuaGenerator) DeleteLayer(layerName string) string {
	escapedName := EscapeString(layerName)
	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

-- Check if this is the last layer
if #spr.layers == 1 then
	error("Cannot delete the last layer")
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

app.transaction(function()
	spr:deleteLayer(layer)
end)

spr:saveAs(spr.filename)
print("Layer deleted successfully")`, escapedName, escapedName)
}

// DeleteFrame generates a Lua script to delete a frame.
//
// Removes the specified frame from the active sprite's animation sequence.
// As a safety measure, the script prevents deletion of the last remaining frame.
//
// The operation is wrapped in a transaction for atomicity and the sprite
// is saved after the frame is deleted.
//
// Parameters:
//   - frameNumber: 1-based frame index to delete
//
// Prints "Frame deleted successfully" on success.
// Returns an error if:
//   - No sprite is active
//   - The frame number is invalid or not found
//   - Attempting to delete the last frame in the sprite
func (g *LuaGenerator) DeleteFrame(frameNumber int) string {
	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

-- Check if this is the last frame
if #spr.frames == 1 then
	error("Cannot delete the last frame")
end

local frame = spr.frames[%d]
if not frame then
	error("Frame not found: %d")
end

app.transaction(function()
	spr:deleteFrame(frame)
end)

spr:saveAs(spr.filename)
print("Frame deleted successfully")`, frameNumber, frameNumber)
}
