package aseprite

import (
	"fmt"
)

// SelectRectangle generates a Lua script to create a rectangular selection.
//
// Creates or modifies the current selection using a rectangular region.
// Selections are used to limit drawing operations, cut/copy content, or define
// regions for transformations.
//
// Parameters:
//   - x, y: top-left corner of the selection rectangle
//   - width: rectangle width in pixels (must be positive)
//   - height: rectangle height in pixels (must be positive)
//   - mode: selection mode - "replace" (default), "add", "subtract", or "intersect"
//
// Selection modes:
//   - "replace": replaces current selection with new rectangle
//   - "add": adds rectangle to current selection (union)
//   - "subtract": removes rectangle from current selection
//   - "intersect": keeps only the intersection of current and new selection
//
// The selection is persisted to sprite.data as JSON and restored across operations.
// The sprite is saved after the selection is created.
//
// Prints "Rectangle selection created successfully" on success.
// Returns an error if no sprite is active.
func (g *LuaGenerator) SelectRectangle(x, y, width, height int, mode string) string {
	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

local rect = Rectangle(%d, %d, %d, %d)
local sel = Selection(rect)

if "%s" == "replace" then
	spr.selection = sel
else
	spr.selection:add(sel)
	if "%s" == "subtract" then
		spr.selection:subtract(sel)
	elseif "%s" == "intersect" then
		spr.selection:intersect(sel)
	end
end

-- Persist selection state to sprite.data for cross-process persistence
if not spr.selection.isEmpty then
	local bounds = spr.selection.bounds
	spr.data = string.format('{"selection":{"x":%%d,"y":%%d,"w":%%d,"h":%%d}}',
		bounds.x, bounds.y, bounds.width, bounds.height)
	spr:saveAs(spr.filename)
end

print("Rectangle selection created successfully")`, x, y, width, height, mode, mode, mode)
}

// SelectEllipse generates a Lua script to create an elliptical selection.
//
// Creates or modifies the current selection using an elliptical region.
// The ellipse is defined by its bounding rectangle and filled using the
// midpoint ellipse algorithm.
//
// Parameters:
//   - x, y: top-left corner of the ellipse bounding box
//   - width: bounding box width in pixels (ellipse diameter on x-axis)
//   - height: bounding box height in pixels (ellipse diameter on y-axis)
//   - mode: selection mode - "replace" (default), "add", "subtract", or "intersect"
//
// Selection modes:
//   - "replace": replaces current selection with new ellipse
//   - "add": adds ellipse to current selection (union)
//   - "subtract": removes ellipse from current selection
//   - "intersect": keeps only the intersection of current and new selection
//
// The selection is persisted to sprite.data as JSON and restored across operations.
// The sprite is saved after the selection is created.
//
// Prints "Ellipse selection created successfully" on success.
// Returns an error if no sprite is active.
func (g *LuaGenerator) SelectEllipse(x, y, width, height int, mode string) string {
	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

-- Create ellipse selection by using drawPixel for each point on ellipse
local sel = Selection()
local rx = %d / 2
local ry = %d / 2
local cx = %d + rx
local cy = %d + ry

-- Midpoint ellipse algorithm to create selection
for angle = 0, 360 do
	local rad = math.rad(angle)
	local ex = math.floor(cx + rx * math.cos(rad))
	local ey = math.floor(cy + ry * math.sin(rad))
	-- Fill from center to edge
	for fillx = math.floor(cx - rx), math.floor(cx + rx) do
		for filly = math.floor(cy - ry), math.floor(cy + ry) do
			local dx = (fillx - cx) / rx
			local dy = (filly - cy) / ry
			if dx * dx + dy * dy <= 1 then
				sel:add(Rectangle(fillx, filly, 1, 1))
			end
		end
	end
	break  -- Only need one pass to fill
end

if "%s" == "replace" then
	spr.selection = sel
elseif "%s" == "add" then
	spr.selection:add(sel)
elseif "%s" == "subtract" then
	spr.selection:subtract(sel)
elseif "%s" == "intersect" then
	spr.selection:intersect(sel)
end

-- Persist selection state to sprite.data for cross-process persistence
if not spr.selection.isEmpty then
	local bounds = spr.selection.bounds
	spr.data = string.format('{"selection":{"x":%%d,"y":%%d,"w":%%d,"h":%%d}}',
		bounds.x, bounds.y, bounds.width, bounds.height)
	spr:saveAs(spr.filename)
end

print("Ellipse selection created successfully")`, width, height, x, y, mode, mode, mode, mode)
}

// SelectAll generates a Lua script to select the entire canvas.
//
// Creates a selection covering the entire sprite canvas from (0,0) to
// (sprite.width, sprite.height). This is useful before copy/cut operations
// or to quickly select all content for transformations.
//
// The selection is persisted to sprite.data as JSON and restored across operations.
// The sprite is saved after the selection is created.
//
// Prints "Select all completed successfully" on success.
// Returns an error if no sprite is active.
func (g *LuaGenerator) SelectAll() string {
	return `local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

-- Create selection covering entire sprite
local rect = Rectangle(0, 0, spr.width, spr.height)
local sel = Selection(rect)
spr.selection = sel

-- Persist selection state to sprite.data for cross-process persistence
local bounds = spr.selection.bounds
spr.data = string.format('{"selection":{"x":%%d,"y":%%d,"w":%%d,"h":%%d}}',
	bounds.x, bounds.y, bounds.width, bounds.height)
spr:saveAs(spr.filename)

print("Select all completed successfully")`
}

// Deselect generates a Lua script to clear the current selection.
//
// Removes the current selection mask, allowing operations to affect the entire
// canvas again. This is the opposite of SelectAll.
//
// The persisted selection state in sprite.data is cleared and the sprite is saved.
//
// Prints "Deselect completed successfully" on success.
// Returns an error if no sprite is active.
func (g *LuaGenerator) Deselect() string {
	return `local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

app.command.DeselectMask()

-- Clear persisted selection state
spr.data = ""
spr:saveAs(spr.filename)

print("Deselect completed successfully")`
}

// MoveSelection generates a Lua script to translate the selection bounds.
//
// Shifts the selection mask by the specified offset without moving the pixel
// content. This is useful for repositioning the selection after creating it,
// or for aligning selections with specific features.
//
// Parameters:
//   - dx: horizontal offset in pixels (positive = right, negative = left)
//   - dy: vertical offset in pixels (positive = down, negative = up)
//
// The selection is restored from sprite.data if needed, then moved and persisted back.
// The sprite is saved after the selection is moved.
//
// Prints "Selection moved successfully" on success.
// Returns an error if:
//   - No sprite is active
//   - No selection exists to move
func (g *LuaGenerator) MoveSelection(dx, dy int) string {
	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

-- Restore selection from persisted state if needed
if spr.selection.isEmpty and spr.data ~= "" then
	local x, y, w, h = spr.data:match('x":(%%d+),"y":(%%d+),"w":(%%d+),"h":(%%d+)')
	if x and y and w and h then
		spr.selection = Selection(Rectangle(tonumber(x), tonumber(y), tonumber(w), tonumber(h)))
	end
end

if spr.selection.isEmpty then
	error("No active selection to move")
end

local bounds = spr.selection.bounds
local newSel = Selection(Rectangle(bounds.x + %d, bounds.y + %d, bounds.width, bounds.height))
spr.selection = newSel

-- Persist updated selection state
local newBounds = spr.selection.bounds
spr.data = string.format('{"selection":{"x":%%d,"y":%%d,"w":%%d,"h":%%d}}',
	newBounds.x, newBounds.y, newBounds.width, newBounds.height)
spr:saveAs(spr.filename)

print("Selection moved successfully")`, dx, dy)
}

// CutSelection generates a Lua script to cut the selected pixels to clipboard.
//
// Removes the pixels within the current selection and copies them to the
// clipboard. The cut area becomes transparent (filled with transparent pixels).
//
// Parameters:
//   - layerName: name of the layer to cut from (automatically escaped for Lua safety)
//   - frameNumber: 1-based frame index to cut from
//
// The operation is wrapped in a transaction for atomicity and the sprite
// is saved after the cut is complete.
//
// Prints "Cut selection completed successfully" on success.
// Returns an error if:
//   - No sprite is active
//   - No selection exists
//   - The layer is not found
//   - The frame number is invalid
func (g *LuaGenerator) CutSelection(layerName string, frameNumber int) string {
	escapedName := EscapeString(layerName)
	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

-- Restore selection from persisted state if needed
if spr.selection.isEmpty and spr.data ~= "" then
	local x, y, w, h = spr.data:match('x":(%%d+),"y":(%%d+),"w":(%%d+),"h":(%%d+)')
	if x and y and w and h then
		spr.selection = Selection(Rectangle(tonumber(x), tonumber(y), tonumber(w), tonumber(h)))
	end
end

if spr.selection.isEmpty then
	error("No active selection to cut")
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

-- Find or create hidden clipboard layer before cutting
local clipboardLayer = nil
for i, lyr in ipairs(spr.layers) do
	if lyr.name == "__mcp_clipboard__" then
		clipboardLayer = lyr
		break
	end
end

if not clipboardLayer then
	clipboardLayer = spr:newLayer()
	clipboardLayer.name = "__mcp_clipboard__"
	clipboardLayer.isVisible = false
end

-- Copy selected region to clipboard layer first
local cel = layer:cel(frame)
if cel then
	local bounds = spr.selection.bounds
	local clipImage = Image(bounds.width, bounds.height, spr.colorMode)
	clipImage:drawImage(cel.image, Point(-bounds.x, -bounds.y))

	-- Store in clipboard layer
	spr:newCel(clipboardLayer, 1, clipImage, Point(bounds.x, bounds.y))
end

-- Now cut from the source layer
app.transaction(function()
	local cel = layer:cel(frame)
	if cel then
		local bounds = spr.selection.bounds
		-- Clear pixels in selection
		for y = bounds.y, bounds.y + bounds.height - 1 do
			for x = bounds.x, bounds.x + bounds.width - 1 do
				if spr.selection:contains(x, y) then
					cel.image:drawPixel(x - cel.position.x, y - cel.position.y, Color{r=0,g=0,b=0,a=0})
				end
			end
		end
	end
end)

-- Selection cleared after cut, clear persisted state
spr.data = ""
spr:saveAs(spr.filename)
print("Cut selection completed successfully")`, escapedName, escapedName, frameNumber, frameNumber)
}

// CopySelection generates a Lua script to copy the selected pixels to clipboard.
//
// Copies the pixels within the current selection to the clipboard without
// removing them. The clipboard content can then be pasted elsewhere.
//
// The clipboard content is stored in a hidden layer (__mcp_clipboard__) which
// persists across operations. The selection is restored from sprite.data if needed.
//
// The sprite is saved to persist the clipboard content.
//
// Prints "Copy selection completed successfully" on success.
// Returns an error if:
//   - No sprite is active
//   - No selection exists
func (g *LuaGenerator) CopySelection() string {
	return `local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

-- Restore selection from persisted state if needed
if spr.selection.isEmpty and spr.data ~= "" then
	local x, y, w, h = spr.data:match('x":(%d+),"y":(%d+),"w":(%d+),"h":(%d+)')
	if x and y and w and h then
		spr.selection = Selection(Rectangle(tonumber(x), tonumber(y), tonumber(w), tonumber(h)))
	end
end

if spr.selection.isEmpty then
	error("No active selection to copy")
end

-- Find or create hidden clipboard layer
local clipboardLayer = nil
for i, lyr in ipairs(spr.layers) do
	if lyr.name == "__mcp_clipboard__" then
		clipboardLayer = lyr
		break
	end
end

if not clipboardLayer then
	clipboardLayer = spr:newLayer()
	clipboardLayer.name = "__mcp_clipboard__"
	clipboardLayer.isVisible = false
end

-- Get the selected image from layer 1, frame 1
-- In batch mode, we need to explicitly use frame 1 since app.activeFrame/activeLayer may not be set correctly
local sourceLayer = spr.layers[1]
local sourceFrame = spr.frames[1]
local sourceCel = sourceLayer:cel(sourceFrame)

if sourceCel then
	-- Copy selected region to clipboard layer
	local bounds = spr.selection.bounds
	local clipImage = Image(bounds.width, bounds.height, spr.colorMode)
	clipImage:drawImage(sourceCel.image, Point(-bounds.x, -bounds.y))

	-- Store in clipboard layer at frame 1
	spr:newCel(clipboardLayer, 1, clipImage, Point(bounds.x, bounds.y))
end

spr:saveAs(spr.filename)
print("Copy selection completed successfully")`
}

// PasteClipboard generates a Lua script to paste clipboard content.
//
// Pastes the clipboard content (from a previous copy or cut operation) to the
// specified layer and frame. The paste position can be explicitly set or will
// use the current position if not specified.
//
// Parameters:
//   - layerName: name of the layer to paste to (automatically escaped for Lua safety)
//   - frameNumber: 1-based frame index to paste to
//   - x: optional x-coordinate for paste position (nil = current position)
//   - y: optional y-coordinate for paste position (nil = current position)
//
// The clipboard content is retrieved from the hidden layer (__mcp_clipboard__).
//
// The operation is wrapped in a transaction for atomicity and the sprite
// is saved after the paste is complete.
//
// Prints "Paste completed successfully" on success.
// Returns an error if:
//   - No sprite is active
//   - The layer is not found
//   - The frame number is invalid
//   - Clipboard is empty
func (g *LuaGenerator) PasteClipboard(layerName string, frameNumber int, x, y *int) string {
	escapedName := EscapeString(layerName)

	pastePos := ""
	if x != nil && y != nil {
		pastePos = fmt.Sprintf("local pasteX, pasteY = %d, %d", *x, *y)
	} else {
		pastePos = "local pasteX, pasteY = 0, 0"
	}

	return GenerateCelCoverHelper() + fmt.Sprintf(`
local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

-- Find clipboard layer
local clipboardLayer = nil
for i, lyr in ipairs(spr.layers) do
	if lyr.name == "__mcp_clipboard__" then
		clipboardLayer = lyr
		break
	end
end

if not clipboardLayer then
	error("No clipboard content available")
end

local clipCel = clipboardLayer:cel(1)
if not clipCel then
	error("No clipboard content available")
end

-- Find target layer
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

-- Paste clipboard content to target layer
%s

local clipImg = clipCel.image
local rx1, ry1 = math.max(pasteX, 0), math.max(pasteY, 0)
local rx2 = math.min(pasteX + clipImg.width - 1, spr.width - 1)
local ry2 = math.min(pasteY + clipImg.height - 1, spr.height - 1)
if rx1 > rx2 or ry1 > ry2 then
	error("Paste position is outside the canvas")
end

app.transaction(function()
	-- Grow the target cel over the paste area, so nothing outside its old bounds is clipped
	local targetCel = celCovering(spr, layer, frame, rx1, ry1, rx2, ry2)

	-- Draw clipboard image onto target
	targetCel.image:drawImage(clipImg, Point(pasteX - targetCel.position.x, pasteY - targetCel.position.y))
end)

spr:saveAs(spr.filename)
print("Paste completed successfully")`, escapedName, escapedName, frameNumber, frameNumber, pastePos)
}
