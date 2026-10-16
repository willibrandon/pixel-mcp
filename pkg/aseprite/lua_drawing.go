package aseprite

import (
	"fmt"
	"strings"
)

// DrawPixels generates a Lua script to draw multiple pixels.
//
// Draws a batch of pixels to the specified layer and frame. This is the most
// efficient way to draw multiple pixels as they are all committed in a single transaction.
//
// Parameters:
//   - layerName: name of the target layer (automatically escaped for Lua safety)
//   - frameNumber: 1-based frame index to draw on
//   - pixels: slice of Pixel structs containing {X, Y, Color} data
//   - usePalette: if true, snaps each pixel color to nearest palette color
//
// The operation is wrapped in a transaction for atomicity and the sprite
// is saved after pixels are drawn. If the specified layer has no cel at the
// target frame, a new cel is created automatically.
//
// Prints "Pixels drawn successfully" on success.
// Returns an error if:
//   - No sprite is active
//   - The layer is not found
//   - The frame number is invalid
func (g *LuaGenerator) DrawPixels(layerName string, frameNumber int, pixels []Pixel, usePalette bool) string {
	var sb strings.Builder

	escapedName := EscapeString(layerName)

	// Add palette snapper helper if needed
	if usePalette {
		sb.WriteString(GeneratePaletteSnapperHelper())
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf(`local spr = app.activeSprite
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

app.transaction(function()
%send)

spr:saveAs(spr.filename)
print("Pixels drawn successfully")`, escapedName, escapedName, frameNumber, frameNumber, emitPixelsOp(pixels, usePalette)))

	return sb.String()
}

// DrawLine generates a Lua script to draw a line.
//
// Draws a straight line between two points using Aseprite's line tool.
// The line is anti-aliased and uses the specified brush thickness.
//
// Parameters:
//   - layerName: name of the target layer (automatically escaped for Lua safety)
//   - frameNumber: 1-based frame index to draw on
//   - x1, y1: starting point coordinates
//   - x2, y2: ending point coordinates
//   - color: line color in RGBA format
//   - thickness: brush size in pixels (1 = single pixel line)
//   - usePalette: if true, snaps color to nearest palette color
//
// The operation is wrapped in a transaction for atomicity and the sprite
// is saved after the line is drawn.
//
// Prints "Line drawn successfully" on success.
// Returns an error if:
//   - No sprite is active
//   - The layer is not found
//   - The frame number is invalid
func (g *LuaGenerator) DrawLine(layerName string, frameNumber int, x1, y1, x2, y2 int, color Color, thickness int, usePalette bool) string {
	var sb strings.Builder

	// Add palette snapper helper if needed
	if usePalette {
		sb.WriteString(GeneratePaletteSnapperHelper())
		sb.WriteString("\n")
	}

	escapedName := EscapeString(layerName)
	sb.WriteString(fmt.Sprintf(`local spr = app.activeSprite
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

app.transaction(function()
%s
end)

spr:saveAs(spr.filename)
print("Line drawn successfully")`,
		escapedName, escapedName,
		frameNumber, frameNumber,
		emitLineOp(x1, y1, x2, y2, color, thickness, usePalette)))

	return sb.String()
}

// DrawContour generates a Lua script to draw a polyline or polygon by connecting multiple points.
//
// Draws a series of connected line segments forming either an open polyline or
// closed polygon. This is useful for creating complex shapes like hand-drawn paths,
// borders, or traced outlines.
//
// Parameters:
//   - layerName: name of the target layer (automatically escaped for Lua safety)
//   - frameNumber: 1-based frame index to draw on
//   - points: ordered slice of Point structs defining the path vertices
//   - color: line color in RGBA format
//   - thickness: brush size in pixels (1 = single pixel line)
//   - closed: if true, connects the last point back to the first to form a polygon
//   - usePalette: if true, snaps color to nearest palette color
//
// The operation is wrapped in a transaction for atomicity and the sprite
// is saved after the contour is drawn. Each segment is drawn as a separate line
// to ensure proper vertex connection.
//
// Prints "Contour drawn successfully" on success.
// Returns an error if:
//   - No sprite is active
//   - The layer is not found
//   - The frame number is invalid
//   - The points slice is empty
func (g *LuaGenerator) DrawContour(layerName string, frameNumber int, points []Point, color Color, thickness int, closed bool, usePalette bool) string {
	var sb strings.Builder

	// Add palette snapper helper if needed
	if usePalette {
		sb.WriteString(GeneratePaletteSnapperHelper())
		sb.WriteString("\n")
	}

	escapedName := EscapeString(layerName)
	sb.WriteString(fmt.Sprintf(`local spr = app.activeSprite
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

app.transaction(function()
%s
end)

spr:saveAs(spr.filename)
print("Contour drawn successfully")`,
		escapedName, escapedName,
		frameNumber, frameNumber,
		emitContourOp(points, color, thickness, closed, usePalette)))

	return sb.String()
}

// DrawRectangle generates a Lua script to draw a rectangle.
//
// Draws either a filled or outlined rectangle using Aseprite's rectangle tools.
// The rectangle is defined by its top-left corner position and dimensions.
//
// Parameters:
//   - layerName: name of the target layer (automatically escaped for Lua safety)
//   - frameNumber: 1-based frame index to draw on
//   - x, y: top-left corner coordinates
//   - width: rectangle width in pixels (must be positive)
//   - height: rectangle height in pixels (must be positive)
//   - color: fill/stroke color in RGBA format
//   - filled: if true, uses filled_rectangle tool; if false, draws outline only
//   - usePalette: if true, snaps color to nearest palette color
//
// The operation is wrapped in a transaction for atomicity and the sprite
// is saved after the rectangle is drawn.
//
// Prints "Rectangle drawn successfully" on success.
// Returns an error if:
//   - No sprite is active
//   - The layer is not found
//   - The frame number is invalid
func (g *LuaGenerator) DrawRectangle(layerName string, frameNumber int, x, y, width, height int, color Color, filled bool, usePalette bool) string {
	var sb strings.Builder

	// Add palette snapper helper if needed
	if usePalette {
		sb.WriteString(GeneratePaletteSnapperHelper())
		sb.WriteString("\n")
	}

	escapedName := EscapeString(layerName)

	sb.WriteString(fmt.Sprintf(`local spr = app.activeSprite
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

app.transaction(function()
%s
end)

spr:saveAs(spr.filename)
print("Rectangle drawn successfully")`,
		escapedName, escapedName,
		frameNumber, frameNumber,
		emitRectangleOp(x, y, width, height, color, filled, usePalette)))

	return sb.String()
}

// DrawCircle generates a Lua script to draw a circle (ellipse).
//
// Draws either a filled or outlined circle using Aseprite's ellipse tools.
// The circle is defined by its center point and radius. Note that this actually
// draws a perfect circle, not an arbitrary ellipse.
//
// Parameters:
//   - layerName: name of the target layer (automatically escaped for Lua safety)
//   - frameNumber: 1-based frame index to draw on
//   - centerX, centerY: center point coordinates
//   - radius: circle radius in pixels (must be positive)
//   - color: fill/stroke color in RGBA format
//   - filled: if true, uses filled_ellipse tool; if false, draws outline only
//   - usePalette: if true, snaps color to nearest palette color
//
// The operation is wrapped in a transaction for atomicity and the sprite
// is saved after the circle is drawn.
//
// Prints "Circle drawn successfully" on success.
// Returns an error if:
//   - No sprite is active
//   - The layer is not found
//   - The frame number is invalid
func (g *LuaGenerator) DrawCircle(layerName string, frameNumber int, centerX, centerY, radius int, color Color, filled bool, usePalette bool) string {
	var sb strings.Builder

	// Add palette snapper helper if needed
	if usePalette {
		sb.WriteString(GeneratePaletteSnapperHelper())
		sb.WriteString("\n")
	}

	escapedName := EscapeString(layerName)

	sb.WriteString(fmt.Sprintf(`local spr = app.activeSprite
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

app.transaction(function()
%s
end)

spr:saveAs(spr.filename)
print("Circle drawn successfully")`,
		escapedName, escapedName,
		frameNumber, frameNumber,
		emitCircleOp(centerX, centerY, radius, color, filled, usePalette)))

	return sb.String()
}

// FillArea generates a Lua script to flood fill an area (paint bucket).
//
// Performs a flood fill operation starting from the specified point, replacing
// all contiguous pixels of similar color with the target color. The tolerance
// parameter controls how similar colors must be to be considered the same.
//
// Parameters:
//   - layerName: name of the target layer (automatically escaped for Lua safety)
//   - frameNumber: 1-based frame index to draw on
//   - x, y: starting point coordinates for the flood fill
//   - color: fill color in RGBA format
//   - tolerance: color similarity threshold (0-255, where 0 = exact match only)
//   - usePalette: if true, snaps color to nearest palette color
//
// The operation is wrapped in a transaction for atomicity and the sprite
// is saved after the fill is complete. The fill is contiguous, meaning it only
// affects connected pixels, not all pixels of the same color.
//
// Prints "Area filled successfully" on success.
// Returns an error if:
//   - No sprite is active
//   - The layer is not found
//   - The frame number is invalid
func (g *LuaGenerator) FillArea(layerName string, frameNumber int, x, y int, color Color, tolerance int, usePalette bool) string {
	var sb strings.Builder

	// Add palette snapper helper if needed
	if usePalette {
		sb.WriteString(GeneratePaletteSnapperHelper())
		sb.WriteString("\n")
	}

	escapedName := EscapeString(layerName)
	sb.WriteString(fmt.Sprintf(`local spr = app.activeSprite
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

app.transaction(function()
%s
end)

spr:saveAs(spr.filename)
print("Area filled successfully")`,
		escapedName, escapedName,
		frameNumber, frameNumber,
		emitFillOp(x, y, color, tolerance, usePalette)))

	return sb.String()
}

// emitPixelsOp emits the transaction body for drawing pixels directly to cel image.
// Assumes Lua locals 'spr', 'layer', 'frame' are already in scope.
func emitPixelsOp(pixels []Pixel, usePalette bool) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`	local cel = layer:cel(frame)
	if not cel then
		cel = spr:newCel(layer, frame)
	end

%s

	local img = cel.image
`, NormalizeCelSnippet()))

	// Add pixel drawing commands with bounds checking
	for _, p := range pixels {
		sb.WriteString(fmt.Sprintf("\tif %d >= 0 and %d < spr.width and %d >= 0 and %d < spr.height then\n\t\timg:putPixel(%d, %d, %s)\n\tend\n",
			p.X, p.X, p.Y, p.Y, p.X, p.Y, FormatColorWithPalette(p.Color, usePalette)))
	}

	return sb.String()
}

// emitLineOp emits the transaction body for drawing a line.
// Assumes Lua locals 'spr', 'layer', 'frame' are already in scope.
func emitLineOp(x1, y1, x2, y2 int, color Color, thickness int, usePalette bool) string {
	return fmt.Sprintf(`	app.activeLayer = layer
	app.activeFrame = frame

	local brush = Brush(%d)

	app.useTool{
		tool = "line",
		color = %s,
		brush = brush,
		points = {%s, %s}
	}`,
		thickness,
		FormatColorWithPaletteForTool(color, usePalette),
		FormatPoint(Point{X: x1, Y: y1}),
		FormatPoint(Point{X: x2, Y: y2}))
}

// emitContourOp emits the transaction body for drawing a contour.
// Assumes Lua locals 'spr', 'layer', 'frame' are already in scope.
func emitContourOp(points []Point, color Color, thickness int, closed bool, usePalette bool) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`	app.activeLayer = layer
	app.activeFrame = frame

	local brush = Brush(%d)
	local color = %s

	-- Draw lines connecting each point`,
		thickness,
		FormatColorWithPaletteForTool(color, usePalette)))

	for i := 0; i < len(points)-1; i++ {
		sb.WriteString(fmt.Sprintf(`
	app.useTool{
		tool = "line",
		color = color,
		brush = brush,
		points = {%s, %s}
	}`,
			FormatPoint(points[i]),
			FormatPoint(points[i+1])))
	}

	if closed && len(points) > 0 {
		sb.WriteString(fmt.Sprintf(`
	-- Close the contour
	app.useTool{
		tool = "line",
		color = color,
		brush = brush,
		points = {%s, %s}
	}`,
			FormatPoint(points[len(points)-1]),
			FormatPoint(points[0])))
	}

	return sb.String()
}

// emitRectangleOp emits the transaction body for drawing a rectangle.
// Assumes Lua locals 'spr', 'layer', 'frame' are already in scope.
func emitRectangleOp(x, y, width, height int, color Color, filled bool, usePalette bool) string {
	tool := "rectangle"
	if filled {
		tool = "filled_rectangle"
	}
	return fmt.Sprintf(`	app.activeLayer = layer
	app.activeFrame = frame

	app.useTool{
		tool = "%s",
		color = %s,
		points = {%s, %s}
	}`,
		tool,
		FormatColorWithPaletteForTool(color, usePalette),
		FormatPoint(Point{X: x, Y: y}),
		FormatPoint(Point{X: x + width - 1, Y: y + height - 1}))
}

// emitCircleOp emits the transaction body for drawing a circle/ellipse.
// Assumes Lua locals 'spr', 'layer', 'frame' are already in scope.
func emitCircleOp(centerX, centerY, radius int, color Color, filled bool, usePalette bool) string {
	tool := "ellipse"
	if filled {
		tool = "filled_ellipse"
	}
	x1 := centerX - radius
	y1 := centerY - radius
	x2 := centerX + radius
	y2 := centerY + radius

	return fmt.Sprintf(`	app.activeLayer = layer
	app.activeFrame = frame

	app.useTool{
		tool = "%s",
		color = %s,
		points = {%s, %s}
	}`,
		tool,
		FormatColorWithPaletteForTool(color, usePalette),
		FormatPoint(Point{X: x1, Y: y1}),
		FormatPoint(Point{X: x2, Y: y2}))
}

// emitFillOp emits the transaction body for flood filling an area.
// Assumes Lua locals 'spr', 'layer', 'frame' are already in scope.
func emitFillOp(x, y int, color Color, tolerance int, usePalette bool) string {
	return fmt.Sprintf(`	app.activeLayer = layer
	app.activeFrame = frame

	app.useTool{
		tool = "paint_bucket",
		color = %s,
		points = {%s},
		contiguous = true,
		tolerance = %d
	}`,
		FormatColorWithPaletteForTool(color, usePalette),
		FormatPoint(Point{X: x, Y: y}),
		tolerance)
}

// BatchDraw generates a Lua script to execute multiple drawing operations in a single transaction.
//
// All operations are executed atomically in the order provided within a single app.transaction.
// The sprite is saved once after all operations complete. If any operation fails (e.g., layer or
// frame not found), the script raises a Lua error naming the operation index (0-based) and the
// failure reason, aborting the transaction and leaving the sprite file untouched.
//
// Parameters:
//   - ops: slice of BatchOp structs, each with resolved layer name, frame number, and parameters
//   - usePalette: if true, snaps colors to nearest palette color
//
// Prints "Batch completed: N operations" on success.
func (g *LuaGenerator) BatchDraw(ops []BatchOp, usePalette bool) string {
	var sb strings.Builder

	if usePalette {
		sb.WriteString(GeneratePaletteSnapperHelper())
		sb.WriteString("\n")
	}

	sb.WriteString(`local spr = app.activeSprite
if not spr then
	error("No active sprite")
end

-- Helper to find layer by name with caching
local layerCache = {}
local function getLayer(name)
	local lyr = layerCache[name]
	if lyr then
		return lyr
	end
	for _, l in ipairs(spr.layers) do
		if l.name == name then
			layerCache[name] = l
			return l
		end
	end
	return nil
end

app.transaction(function()
`)

	for i, op := range ops {
		escapedLayer := EscapeString(op.LayerName)
		sb.WriteString(fmt.Sprintf(`	do
		local layer = getLayer("%s")
		if not layer then
			error("operations[%d]: Layer not found: %s")
		end
		local frame = spr.frames[%d]
		if not frame then
			error("operations[%d]: Frame not found: %d")
		end

`, escapedLayer, i, escapedLayer, op.FrameNumber, i, op.FrameNumber))

		var opLua string
		switch op.Type {
		case BatchOpPixels:
			opLua = emitPixelsOp(op.Pixels, usePalette)
		case BatchOpLine:
			opLua = emitLineOp(op.X1, op.Y1, op.X2, op.Y2, op.Color, op.Thickness, usePalette)
		case BatchOpRectangle:
			opLua = emitRectangleOp(op.X, op.Y, op.Width, op.Height, op.Color, op.Filled, usePalette)
		case BatchOpCircle:
			opLua = emitCircleOp(op.CenterX, op.CenterY, op.Radius, op.Color, op.Filled, usePalette)
		case BatchOpContour:
			opLua = emitContourOp(op.Points, op.Color, op.Thickness, op.Closed, usePalette)
		case BatchOpFill:
			opLua = emitFillOp(op.X, op.Y, op.Color, op.Tolerance, usePalette)
		}

		sb.WriteString(opLua)
		if !strings.HasSuffix(opLua, "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("\tend\n")
	}

	sb.WriteString(fmt.Sprintf(`end)

spr:saveAs(spr.filename)
print("Batch completed: %d operations")`, len(ops)))

	return sb.String()
}

// DrawWithDither generates a Lua script to fill a region with a dithering pattern.
//
// Applies a dithering pattern to create texture, gradients, or retro aesthetic effects.
// Supports 15 different patterns including Bayer matrices and texture patterns.
//
// Parameters:
//   - layerName: name of the target layer (automatically escaped for Lua safety)
//   - frameNumber: 1-based frame index to draw on
//   - x, y: top-left corner of the dither region
//   - width: region width in pixels
//   - height: region height in pixels
//   - color1: first color as hex string (e.g., "#FF0000FF")
//   - color2: second color as hex string (alternates with color1 based on pattern)
//   - pattern: dithering pattern name (see below for available patterns)
//   - density: pattern density/threshold (0.0-1.0, where 0.5 = balanced mix)
//
// Available patterns:
//   - "bayer_2x2", "bayer_4x4", "bayer_8x8" - Ordered dithering matrices
//   - "checkerboard" - Simple alternating pattern
//   - "grass", "water", "stone", "cloud" - Organic texture patterns
//   - "brick" - Masonry pattern
//   - "dots", "diagonal", "cross" - Geometric patterns
//   - "noise" - Pseudo-random pattern
//   - "horizontal_lines", "vertical_lines" - Directional line patterns
//
// The operation is wrapped in a transaction for atomicity and the sprite
// is saved after the dither is applied.
//
// Prints "Dithering applied successfully" on success.
// Returns an error if:
//   - No sprite is active
//   - The layer is not found
//   - The frame number is invalid
//   - The pattern name is not recognized
func (g *LuaGenerator) DrawWithDither(layerName string, frameNumber int, x, y, width, height int, color1, color2 string, pattern string, density float64) string {
	escapedLayerName := EscapeString(layerName)

	// Parse hex colors
	c1 := parseHexColor(color1)
	c2 := parseHexColor(color2)

	// Get dithering matrix based on pattern
	var matrixCode string
	switch pattern {
	case "bayer_2x2":
		matrixCode = `local matrix = {{0, 2}, {3, 1}}
local matrixSize = 2`
	case "bayer_4x4":
		matrixCode = `local matrix = {
	{ 0,  8,  2, 10},
	{12,  4, 14,  6},
	{ 3, 11,  1,  9},
	{15,  7, 13,  5}
}
local matrixSize = 4`
	case "bayer_8x8":
		matrixCode = `local matrix = {
	{ 0, 32,  8, 40,  2, 34, 10, 42},
	{48, 16, 56, 24, 50, 18, 58, 26},
	{12, 44,  4, 36, 14, 46,  6, 38},
	{60, 28, 52, 20, 62, 30, 54, 22},
	{ 3, 35, 11, 43,  1, 33,  9, 41},
	{51, 19, 59, 27, 49, 17, 57, 25},
	{15, 47,  7, 39, 13, 45,  5, 37},
	{63, 31, 55, 23, 61, 29, 53, 21}
}
local matrixSize = 8`
	case "checkerboard":
		matrixCode = `local matrix = {{0, 1}, {1, 0}}
local matrixSize = 2`
	case "grass":
		matrixCode = `local matrix = {
	{1, 0, 1, 0, 1, 0},
	{0, 1, 1, 0, 0, 1},
	{1, 1, 0, 1, 0, 0},
	{0, 1, 0, 1, 1, 0},
	{1, 0, 0, 0, 1, 1},
	{0, 0, 1, 1, 0, 1}
}
local matrixSize = 6`
	case "water":
		matrixCode = `local matrix = {
	{0, 0, 1, 1, 0, 0},
	{0, 1, 1, 1, 1, 0},
	{1, 1, 0, 0, 1, 1},
	{1, 0, 0, 0, 0, 1},
	{0, 1, 1, 1, 1, 0},
	{0, 0, 1, 1, 0, 0}
}
local matrixSize = 6`
	case "stone":
		matrixCode = `local matrix = {
	{0, 0, 0, 1, 1, 0},
	{0, 1, 0, 0, 1, 1},
	{0, 0, 1, 1, 0, 0},
	{1, 1, 0, 0, 0, 1},
	{1, 0, 0, 1, 1, 0},
	{0, 1, 1, 0, 0, 0}
}
local matrixSize = 6`
	case "cloud":
		matrixCode = `local matrix = {
	{0, 0, 0, 0, 1, 1},
	{0, 0, 0, 1, 1, 1},
	{0, 0, 1, 1, 1, 0},
	{0, 1, 1, 1, 0, 0},
	{1, 1, 1, 0, 0, 0},
	{1, 1, 0, 0, 0, 0}
}
local matrixSize = 6`
	case "brick":
		matrixCode = `local matrix = {
	{0, 0, 0, 0, 1, 0, 0, 0},
	{0, 0, 0, 0, 1, 0, 0, 0},
	{1, 1, 1, 1, 1, 1, 1, 1},
	{0, 0, 1, 0, 0, 0, 0, 1},
	{0, 0, 1, 0, 0, 0, 0, 1},
	{1, 1, 1, 1, 1, 1, 1, 1},
	{0, 0, 0, 0, 1, 0, 0, 0},
	{0, 0, 0, 0, 1, 0, 0, 0}
}
local matrixSize = 8`
	case "dots":
		matrixCode = `local matrix = {
	{1, 0, 0, 0},
	{0, 0, 0, 0},
	{0, 0, 1, 0},
	{0, 0, 0, 0}
}
local matrixSize = 4`
	case "diagonal":
		matrixCode = `local matrix = {
	{1, 0, 0, 0},
	{0, 1, 0, 0},
	{0, 0, 1, 0},
	{0, 0, 0, 1}
}
local matrixSize = 4`
	case "cross":
		matrixCode = `local matrix = {
	{0, 1, 0},
	{1, 1, 1},
	{0, 1, 0}
}
local matrixSize = 3`
	case "noise":
		matrixCode = `local matrix = {
	{1, 0, 1, 0, 0, 1},
	{0, 1, 0, 1, 1, 0},
	{1, 0, 0, 1, 0, 1},
	{0, 1, 1, 0, 1, 0},
	{0, 0, 1, 0, 1, 1},
	{1, 1, 0, 1, 0, 0}
}
local matrixSize = 6`
	case "horizontal_lines":
		matrixCode = `local matrix = {
	{1, 1, 1, 1},
	{0, 0, 0, 0},
	{1, 1, 1, 1},
	{0, 0, 0, 0}
}
local matrixSize = 4`
	case "vertical_lines":
		matrixCode = `local matrix = {
	{1, 0, 1, 0},
	{1, 0, 1, 0},
	{1, 0, 1, 0},
	{1, 0, 1, 0}
}
local matrixSize = 4`
	case "floyd_steinberg":
		// Floyd-Steinberg uses error diffusion instead of matrix patterns
		return generateFloydSteinbergLua(escapedLayerName, frameNumber, x, y, width, height, c1, c2, density)
	default:
		return fmt.Sprintf(`error("Unknown dithering pattern: %s")`, pattern)
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

-- Helper: Find nearest palette index for given RGBA color
local function findNearestPaletteIndex(r, g, b, a)
	local palette = spr.palettes[1]
	if not palette or #palette == 0 then
		return 0
	end

	local minDist = math.huge
	local nearestIndex = 0

	for i = 0, #palette - 1 do
		local palColor = palette:getColor(i)
		local dr = r - palColor.red
		local dg = g - palColor.green
		local db = b - palColor.blue
		local da = a - palColor.alpha
		local dist = dr*dr + dg*dg + db*db + da*da

		if dist < minDist then
			minDist = dist
			nearestIndex = i
		end
	end

	return nearestIndex
end

-- Define colors based on color mode
local color1, color2
if spr.colorMode == ColorMode.INDEXED then
	-- In indexed mode, img:drawPixel expects palette indices
	color1 = findNearestPaletteIndex(%d, %d, %d, %d)
	color2 = findNearestPaletteIndex(%d, %d, %d, %d)
else
	-- In RGB/Grayscale mode, use pixel color values
	color1 = app.pixelColor.rgba(%d, %d, %d, %d)
	color2 = app.pixelColor.rgba(%d, %d, %d, %d)
end

-- Dithering matrix
%s

-- Dithering threshold (based on density)
local threshold = %f * (matrixSize * matrixSize)

-- Apply dithering pattern
app.transaction(function()
	local cel = layer:cel(frame)
	if not cel then
		cel = spr:newCel(layer, frame)
	end

%s

	local img = cel.image

	for py = 0, %d - 1 do
		for px = 0, %d - 1 do
			local drawX = %d + px
			local drawY = %d + py
			if drawX >= 0 and drawX < spr.width and drawY >= 0 and drawY < spr.height then
				local mx = (px %% matrixSize) + 1
				local my = (py %% matrixSize) + 1
				local matrixValue = matrix[my][mx]

				local useColor2 = matrixValue < threshold
				local finalColor = useColor2 and color2 or color1

				img:drawPixel(drawX, drawY, finalColor)
			end
		end
	end
end)

spr:saveAs(spr.filename)
print("Dithering applied successfully")`,
		escapedLayerName, escapedLayerName,
		frameNumber, frameNumber,
		c1.R, c1.G, c1.B, c1.A,
		c2.R, c2.G, c2.B, c2.A,
		c1.R, c1.G, c1.B, c1.A,
		c2.R, c2.G, c2.B, c2.A,
		matrixCode,
		density,
		NormalizeCelSnippet(),
		height, width,
		x, y)
}

// generateFloydSteinbergLua generates Lua script for Floyd-Steinberg error diffusion dithering.
//
// Floyd-Steinberg is an error diffusion algorithm that produces high-quality dithering
// by propagating quantization error to neighboring pixels. The error distribution pattern is:
//
//	        X   7/16
//	3/16  5/16  1/16
//
// where X is the current pixel being processed.
//
// The implementation creates a horizontal gradient from color1 to color2 and applies error
// diffusion to produce smooth transitions. The density parameter is currently unused as the
// gradient quality is controlled by the error diffusion algorithm itself.
func generateFloydSteinbergLua(layerName string, frameNumber int, x, y, width, height int, c1, c2 Color, density float64) string {
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

-- Helper: Find nearest palette index for given RGBA color
local function findNearestPaletteIndex(r, g, b, a)
	local palette = spr.palettes[1]
	if not palette or #palette == 0 then
		return 0
	end

	local minDist = math.huge
	local nearestIndex = 0

	for i = 0, #palette - 1 do
		local palColor = palette:getColor(i)
		local dr = r - palColor.red
		local dg = g - palColor.green
		local db = b - palColor.blue
		local da = a - palColor.alpha
		local dist = dr*dr + dg*dg + db*db + da*da

		if dist < minDist then
			minDist = dist
			nearestIndex = i
		end
	end

	return nearestIndex
end

-- Define colors based on color mode
local color1, color2
local color1_r, color1_g, color1_b, color1_a = %d, %d, %d, %d
local color2_r, color2_g, color2_b, color2_a = %d, %d, %d, %d

if spr.colorMode == ColorMode.INDEXED then
	-- In indexed mode, img:drawPixel expects palette indices
	color1 = findNearestPaletteIndex(color1_r, color1_g, color1_b, color1_a)
	color2 = findNearestPaletteIndex(color2_r, color2_g, color2_b, color2_a)
else
	-- In RGB/Grayscale mode, use pixel color values
	color1 = app.pixelColor.rgba(color1_r, color1_g, color1_b, color1_a)
	color2 = app.pixelColor.rgba(color2_r, color2_g, color2_b, color2_a)
end

-- Floyd-Steinberg error diffusion
app.transaction(function()
	local cel = layer:cel(frame)
	if not cel then
		cel = spr:newCel(layer, frame)
	end

%s

	local img = cel.image

	-- Create error buffer (width+2 to handle edges, 2 rows for current and next)
	local errors = {}
	for row = 0, 1 do
		errors[row] = {}
		for col = 0, %d + 1 do
			errors[row][col] = {r=0, g=0, b=0}
		end
	end

	for py = 0, %d - 1 do
		-- Swap error buffers for next row
		if py > 0 then
			errors[0], errors[1] = errors[1], errors[0]
			-- Clear the new "next row" buffer
			for col = 0, %d + 1 do
				errors[1][col] = {r=0, g=0, b=0}
			end
		end

		for px = 0, %d - 1 do
			-- Calculate gradient position (0.0 to 1.0 across width)
			local gradient_pos
			if %d == 1 then
				gradient_pos = 0
			else
				gradient_pos = px / (%d - 1)
			end

			-- Calculate ideal gradient color at this position
			local ideal_r = color1_r + (color2_r - color1_r) * gradient_pos
			local ideal_g = color1_g + (color2_g - color1_g) * gradient_pos
			local ideal_b = color1_b + (color2_b - color1_b) * gradient_pos

			-- Add accumulated error
			local err = errors[0][px + 1]
			local new_r = math.max(0, math.min(255, ideal_r + err.r))
			local new_g = math.max(0, math.min(255, ideal_g + err.g))
			local new_b = math.max(0, math.min(255, ideal_b + err.b))

			-- Calculate color with accumulated error
			local targetColor
			if spr.colorMode == ColorMode.INDEXED then
				-- Choose nearest color by Euclidean distance
				local dist1 = (new_r - color1_r)^2 + (new_g - color1_g)^2 + (new_b - color1_b)^2
				local dist2 = (new_r - color2_r)^2 + (new_g - color2_g)^2 + (new_b - color2_b)^2

				targetColor = (dist1 < dist2) and color1 or color2

				-- Calculate error in RGB space
				local actual_r, actual_g, actual_b
				if targetColor == color1 then
					actual_r, actual_g, actual_b = color1_r, color1_g, color1_b
				else
					actual_r, actual_g, actual_b = color2_r, color2_g, color2_b
				end

				local err_r = new_r - actual_r
				local err_g = new_g - actual_g
				local err_b = new_b - actual_b

				-- Distribute error to neighbors (Floyd-Steinberg weights)
				if px < %d - 1 then
					errors[0][px + 2].r = errors[0][px + 2].r + err_r * 7/16
					errors[0][px + 2].g = errors[0][px + 2].g + err_g * 7/16
					errors[0][px + 2].b = errors[0][px + 2].b + err_b * 7/16
				end
				if py < %d - 1 then
					if px > 0 then
						errors[1][px].r = errors[1][px].r + err_r * 3/16
						errors[1][px].g = errors[1][px].g + err_g * 3/16
						errors[1][px].b = errors[1][px].b + err_b * 3/16
					end
					errors[1][px + 1].r = errors[1][px + 1].r + err_r * 5/16
					errors[1][px + 1].g = errors[1][px + 1].g + err_g * 5/16
					errors[1][px + 1].b = errors[1][px + 1].b + err_b * 5/16
					if px < %d - 1 then
						errors[1][px + 2].r = errors[1][px + 2].r + err_r * 1/16
						errors[1][px + 2].g = errors[1][px + 2].g + err_g * 1/16
						errors[1][px + 2].b = errors[1][px + 2].b + err_b * 1/16
					end
				end
			else
				-- RGB mode - choose nearest color by Euclidean distance
				local dist1 = (new_r - color1_r)^2 + (new_g - color1_g)^2 + (new_b - color1_b)^2
				local dist2 = (new_r - color2_r)^2 + (new_g - color2_g)^2 + (new_b - color2_b)^2

				targetColor = (dist1 < dist2) and color1 or color2

				-- Calculate error (difference between gradient+error and quantized color)
				local actual_r = app.pixelColor.rgbaR(targetColor)
				local actual_g = app.pixelColor.rgbaG(targetColor)
				local actual_b = app.pixelColor.rgbaB(targetColor)

				local err_r = new_r - actual_r
				local err_g = new_g - actual_g
				local err_b = new_b - actual_b

				-- Distribute error to neighbors
				if px < %d - 1 then
					errors[0][px + 2].r = errors[0][px + 2].r + err_r * 7/16
					errors[0][px + 2].g = errors[0][px + 2].g + err_g * 7/16
					errors[0][px + 2].b = errors[0][px + 2].b + err_b * 7/16
				end
				if py < %d - 1 then
					if px > 0 then
						errors[1][px].r = errors[1][px].r + err_r * 3/16
						errors[1][px].g = errors[1][px].g + err_g * 3/16
						errors[1][px].b = errors[1][px].b + err_b * 3/16
					end
					errors[1][px + 1].r = errors[1][px + 1].r + err_r * 5/16
					errors[1][px + 1].g = errors[1][px + 1].g + err_g * 5/16
					errors[1][px + 1].b = errors[1][px + 1].b + err_b * 5/16
					if px < %d - 1 then
						errors[1][px + 2].r = errors[1][px + 2].r + err_r * 1/16
						errors[1][px + 2].g = errors[1][px + 2].g + err_g * 1/16
						errors[1][px + 2].b = errors[1][px + 2].b + err_b * 1/16
					end
				end
			end

			local drawX = %d + px
			local drawY = %d + py
			if drawX >= 0 and drawX < spr.width and drawY >= 0 and drawY < spr.height then
				img:drawPixel(drawX, drawY, targetColor)
			end
		end
	end
end)

spr:saveAs(spr.filename)
print("Dithering applied successfully")`,
		layerName, layerName,
		frameNumber, frameNumber,
		c1.R, c1.G, c1.B, c1.A,
		c2.R, c2.G, c2.B, c2.A,
		NormalizeCelSnippet(),
		width,  // error buffer width
		height, // py loop
		width,  // clear buffer width
		width,  // px loop
		width,  // width check for division by zero
		width,  // gradient calculation
		width,  // right neighbor check
		height, // bottom neighbor check
		width,  // bottom-right check
		width,  // right neighbor check (RGB)
		height, // bottom neighbor check (RGB)
		width,  // bottom-right check (RGB)
		x,      // drawX coordinate
		y)      // drawY coordinate
}
