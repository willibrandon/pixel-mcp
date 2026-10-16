package aseprite

import (
	"fmt"
)

// SetFrameDuration generates a Lua script to set the duration of a frame.
//
// Modifies the display duration of a specific animation frame. Frame duration
// controls how long each frame is displayed during animation playback.
//
// Parameters:
//   - frameNumber: 1-based frame index to modify
//   - durationMs: frame duration in milliseconds (automatically converted to seconds for Aseprite)
//
// The operation is wrapped in a transaction for atomicity and the sprite
// is saved after the duration is set.
//
// Prints "Frame duration set successfully" on success.
// Returns an error if:
//   - No sprite is active
//   - The frame number is invalid
func (g *LuaGenerator) SetFrameDuration(frameNumber int, durationMs int) string {
	durationSec := float64(durationMs) / 1000.0
	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

local frame = spr.frames[%d]
if not frame then
	error("Frame not found: %d")
end

app.transaction(function()
	frame.duration = %.3f
end)

spr:saveAs(spr.filename)
print("Frame duration set successfully")`, frameNumber, frameNumber, durationSec)
}

// CreateTag generates a Lua script to create an animation tag.
//
// Creates a named tag spanning a range of frames. Tags are used to define
// animation sequences within a sprite, such as "walk", "jump", or "idle".
// Each tag can have its own playback direction.
//
// Parameters:
//   - tagName: name for the animation tag (automatically escaped for Lua safety)
//   - fromFrame: 1-based starting frame index (inclusive)
//   - toFrame: 1-based ending frame index (inclusive)
//   - direction: playback direction - "forward" (default), "reverse", or "pingpong"
//
// Direction modes:
//   - "forward": plays frames from start to end
//   - "reverse": plays frames from end to start
//   - "pingpong": plays forward then backward in a loop
//
// The operation is wrapped in a transaction for atomicity and the sprite
// is saved after the tag is created.
//
// Prints "Tag created successfully" on success.
// Returns an error if:
//   - No sprite is active
//   - The frame range exceeds the sprite's frame count
func (g *LuaGenerator) CreateTag(tagName string, fromFrame, toFrame int, direction string) string {
	escapedName := EscapeString(tagName)

	// Map direction to Aseprite AniDir enum
	var aniDir string
	switch direction {
	case "forward":
		aniDir = "AniDir.FORWARD"
	case "reverse":
		aniDir = "AniDir.REVERSE"
	case "pingpong":
		aniDir = "AniDir.PING_PONG"
	default:
		aniDir = "AniDir.FORWARD"
	}

	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

if #spr.frames < %d then
	error("Frame range exceeds sprite frames")
end

app.transaction(function()
	local tag = spr:newTag(%d, %d)
	tag.name = "%s"
	tag.aniDir = %s
end)

spr:saveAs(spr.filename)
print("Tag created successfully")`, toFrame, fromFrame, toFrame, escapedName, aniDir)
}

// DeleteTag generates a Lua script to delete an animation tag.
//
// Removes a named animation tag from the sprite. The frames themselves are not
// deleted, only the tag metadata is removed.
//
// Parameters:
//   - tagName: name of the tag to delete (automatically escaped for Lua safety)
//
// The operation is wrapped in a transaction for atomicity and the sprite
// is saved after the tag is deleted.
//
// Prints "Tag deleted successfully" on success.
// Returns an error if:
//   - No sprite is active
//   - The tag is not found
func (g *LuaGenerator) DeleteTag(tagName string) string {
	escapedName := EscapeString(tagName)

	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

-- Find tag by name
local tag = nil
for i, t in ipairs(spr.tags) do
	if t.name == "%s" then
		tag = t
		break
	end
end

if not tag then
	error("Tag not found: %s")
end

app.transaction(function()
	spr:deleteTag(tag)
end)

spr:saveAs(spr.filename)
print("Tag deleted successfully")`, escapedName, escapedName)
}

// DuplicateFrame generates a Lua script to duplicate a frame.
//
// Creates a complete copy of a frame, including all layer cels and the frame's
// duration. The new frame can be inserted at a specific position or at the end
// of the timeline.
//
// Parameters:
//   - sourceFrame: 1-based frame index to duplicate
//   - insertAfter: 1-based position to insert after (0 = insert at end of timeline)
//
// The operation copies:
//   - All layer cels (pixel data and positions)
//   - Frame duration settings
//
// The operation is wrapped in a transaction for atomicity and the sprite
// is saved after the frame is duplicated.
//
// Prints the new frame number on success.
// Returns an error if:
//   - No sprite is active
//   - The source frame is not found
//   - The insert position exceeds the sprite's frame count (when insertAfter > 0)
func (g *LuaGenerator) DuplicateFrame(sourceFrame int, insertAfter int) string {
	if insertAfter == 0 {
		// Insert at end
		return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

local srcFrame = spr.frames[%d]
if not srcFrame then
	error("Source frame not found: %d")
end

local newFrame
app.transaction(function()
	newFrame = spr:newFrame(srcFrame)
	newFrame.duration = srcFrame.duration
end)

spr:saveAs(spr.filename)
print(#spr.frames)`, sourceFrame, sourceFrame)
	}

	// Insert after specific frame
	return fmt.Sprintf(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

local srcFrame = spr.frames[%d]
if not srcFrame then
	error("Source frame not found: %d")
end

if #spr.frames < %d then
	error("Insert position exceeds sprite frames")
end

local newFrame
app.transaction(function()
	newFrame = spr:newFrame(%d + 1)
	newFrame.duration = srcFrame.duration

	-- Copy cels from source frame to new frame
	for _, layer in ipairs(spr.layers) do
		local srcCel = layer:cel(srcFrame)
		if srcCel then
			local newCel = spr:newCel(layer, newFrame, srcCel.image, srcCel.position)
		end
	end
end)

spr:saveAs(spr.filename)
print(%d + 1)`, sourceFrame, sourceFrame, insertAfter, insertAfter, insertAfter)
}

// LinkCel generates a Lua script to create a linked cel.
//
// Creates a cel in the target frame that shares the same image data as the
// source cel. Linked cels are useful for animation efficiency - when the same
// image appears in multiple frames, linking saves memory and ensures consistency.
//
// Parameters:
//   - layerName: name of the layer to link within (automatically escaped for Lua safety)
//   - sourceFrame: 1-based frame index containing the cel to link from
//   - targetFrame: 1-based frame index where the linked cel will be created
//
// IMPORTANT: Changes to the image in either the source or target cel will affect
// both cels since they share the same image reference.
//
// The operation is wrapped in a transaction for atomicity and the sprite
// is saved after the cel is linked.
//
// Prints "Cel linked successfully" on success.
// Returns an error if:
//   - No sprite is active
//   - The layer is not found
//   - The source frame is not found
//   - The target frame is not found
//   - No cel exists in the source frame
func (g *LuaGenerator) LinkCel(layerName string, sourceFrame, targetFrame int) string {
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

local srcFrame = spr.frames[%d]
if not srcFrame then
	error("Source frame not found: %d")
end

local tgtFrame = spr.frames[%d]
if not tgtFrame then
	error("Target frame not found: %d")
end

local srcCel = layer:cel(srcFrame)
if not srcCel then
	error("Source cel not found in frame %d")
end

app.transaction(function()
	-- Delete existing cel if present to avoid collision
	local existingCel = layer:cel(tgtFrame)
	if existingCel then
		spr:deleteCel(existingCel)
	end
	-- Create linked cel by copying with same image reference
	spr:newCel(layer, tgtFrame, srcCel.image, srcCel.position)
end)

spr:saveAs(spr.filename)
print("Cel linked successfully")`,
		escapedName, escapedName,
		sourceFrame, sourceFrame,
		targetFrame, targetFrame,
		sourceFrame)
}
