// Package aseprite provides types and utilities for interacting with Aseprite's
// command-line interface and Lua scripting API.
//
// This package includes:
//   - Client for executing Aseprite commands and Lua scripts
//   - Domain types (Color, Point, Rectangle, Pixel, SpriteInfo)
//   - Lua script generation utilities
//   - Palette extraction and image analysis functions
//
// All operations require an explicit Aseprite executable path configured via Config.
// No automatic discovery or environment variables are used.
package aseprite

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Color represents an RGBA color value with 8-bit channels.
//
// Each channel (R, G, B, A) ranges from 0-255, where:
//   - R (red): 0 is no red, 255 is full red
//   - G (green): 0 is no green, 255 is full green
//   - B (blue): 0 is no blue, 255 is full blue
//   - A (alpha): 0 is fully transparent, 255 is fully opaque
//
// Color values can be created from RGBA components using NewColor/NewColorRGB,
// or parsed from hex strings using FromHex.
type Color struct {
	R uint8 `json:"r"` // Red channel (0-255)
	G uint8 `json:"g"` // Green channel (0-255)
	B uint8 `json:"b"` // Blue channel (0-255)
	A uint8 `json:"a"` // Alpha channel (0-255)
}

var hexColorPattern = regexp.MustCompile(`^#?([A-Fa-f0-9]{6}|[A-Fa-f0-9]{8})$`)

// NewColor creates a new Color with the specified RGBA values.
//
// All parameters must be in the range 0-255. Values are clamped by the uint8 type.
func NewColor(r, g, b, a uint8) Color {
	return Color{R: r, G: g, B: b, A: a}
}

// NewColorRGB creates a new fully opaque Color with the specified RGB values.
//
// The alpha channel is automatically set to 255 (fully opaque).
// All parameters must be in the range 0-255. Values are clamped by the uint8 type.
func NewColorRGB(r, g, b uint8) Color {
	return Color{R: r, G: g, B: b, A: 255}
}

// FromHex parses a hex color string and updates the Color with the parsed values.
//
// Supported formats:
//   - "#RRGGBB" - RGB with implicit full opacity (alpha = 255)
//   - "#RRGGBBAA" - RGBA with explicit alpha
//   - "RRGGBB" - Same as above, without "#" prefix
//   - "RRGGBBAA" - Same as above, without "#" prefix
//
// Returns an error if the hex string format is invalid.
// The "#" prefix is optional.
func (c *Color) FromHex(hex string) error {
	hex = strings.TrimPrefix(hex, "#")

	if !hexColorPattern.MatchString("#" + hex) {
		return fmt.Errorf("invalid hex color format: %q (expected #RRGGBB or #RRGGBBAA)", hex)
	}

	// Parse RGB
	r, _ := strconv.ParseUint(hex[0:2], 16, 8)
	g, _ := strconv.ParseUint(hex[2:4], 16, 8)
	b, _ := strconv.ParseUint(hex[4:6], 16, 8)

	c.R = uint8(r)
	c.G = uint8(g)
	c.B = uint8(b)

	// Parse alpha if present
	if len(hex) == 8 {
		a, _ := strconv.ParseUint(hex[6:8], 16, 8)
		c.A = uint8(a)
	} else {
		c.A = 255 // Opaque by default
	}

	return nil
}

// ToHex converts the color to a hex string in the format "#RRGGBBAA".
//
// All four channels (R, G, B, A) are included in the output.
// Returns a string like "#FF0000FF" for opaque red.
func (c Color) ToHex() string {
	return fmt.Sprintf("#%02X%02X%02X%02X", c.R, c.G, c.B, c.A)
}

// ToHexRGB converts the color to a hex string in the format "#RRGGBB".
//
// Only RGB channels are included; alpha channel is ignored.
// Returns a string like "#FF0000" for red (regardless of alpha value).
func (c Color) ToHexRGB() string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}

// Point represents a 2D coordinate in pixel space.
//
// Used for specifying positions, offsets, and dimensions in sprite operations.
// The coordinate system has origin (0,0) at the top-left corner.
type Point struct {
	X int `json:"x"` // Horizontal position (pixels from left edge)
	Y int `json:"y"` // Vertical position (pixels from top edge)
}

// Rectangle represents a rectangular region in pixel space.
//
// Defined by top-left corner (X, Y) and size (Width, Height).
// The coordinate system has origin (0,0) at the top-left corner.
type Rectangle struct {
	X      int `json:"x"`      // Left edge position
	Y      int `json:"y"`      // Top edge position
	Width  int `json:"width"`  // Width in pixels
	Height int `json:"height"` // Height in pixels
}

// Pixel represents a single pixel with both position and color information.
//
// Combines a Point (X, Y coordinates) with a Color (RGBA values).
// Used for bulk pixel operations and pixel data inspection.
type Pixel struct {
	Point       // Embedded point for X, Y coordinates
	Color Color `json:"color"` // RGBA color value
}

// SpriteInfo contains metadata about an Aseprite sprite.
//
// Retrieved from sprites using the get_sprite_info tool.
// Provides dimensions, color mode, animation frame count, and layer information.
type SpriteInfo struct {
	Width      int      `json:"width"`       // Sprite width in pixels
	Height     int      `json:"height"`      // Sprite height in pixels
	ColorMode  string   `json:"color_mode"`  // Color mode: "0" (RGB), "1" (Grayscale), "2" (Indexed)
	FrameCount int      `json:"frame_count"` // Number of animation frames
	LayerCount int      `json:"layer_count"` // Number of layers
	Layers     []string `json:"layers"`      // Layer names in order
}

// ColorMode represents the color mode of a sprite.
//
// Aseprite supports three color modes:
//   - RGB: Full color with 8-bit RGB channels (most common)
//   - Grayscale: 8-bit grayscale values
//   - Indexed: Palette-based colors (pixel art friendly)
type ColorMode string

const (
	// ColorModeRGB represents RGB color mode (8-bit per channel)
	ColorModeRGB ColorMode = "rgb"

	// ColorModeGrayscale represents grayscale color mode (8-bit values)
	ColorModeGrayscale ColorMode = "grayscale"

	// ColorModeIndexed represents indexed color mode (palette-based)
	ColorModeIndexed ColorMode = "indexed"
)

// String returns the string representation of the color mode.
func (cm ColorMode) String() string {
	return string(cm)
}

// ToLua returns the Aseprite Lua API constant for the color mode.
//
// Returns one of:
//   - "ColorMode.RGB"
//   - "ColorMode.GRAYSCALE"
//   - "ColorMode.INDEXED"
//
// Defaults to "ColorMode.RGB" for unrecognized modes.
func (cm ColorMode) ToLua() string {
	switch cm {
	case ColorModeRGB:
		return "ColorMode.RGB"
	case ColorModeGrayscale:
		return "ColorMode.GRAYSCALE"
	case ColorModeIndexed:
		return "ColorMode.INDEXED"
	default:
		return "ColorMode.RGB"
	}
}

// BatchOpType represents the type of drawing operation within a batch.
type BatchOpType string

const (
	// BatchOpPixels draws individual pixels directly to the cel image.
	BatchOpPixels BatchOpType = "pixels"
	// BatchOpLine draws a line segment.
	BatchOpLine BatchOpType = "line"
	// BatchOpRectangle draws a rectangle (outline or filled).
	BatchOpRectangle BatchOpType = "rectangle"
	// BatchOpCircle draws an ellipse/circle (outline or filled).
	BatchOpCircle BatchOpType = "circle"
	// BatchOpContour draws a polyline or closed polygon.
	BatchOpContour BatchOpType = "contour"
	// BatchOpFill flood fills an area with color.
	BatchOpFill BatchOpType = "fill"
)

// BatchOp represents a single drawing operation within a batch.
//
// Carries the resolved layer name, frame number, operation type, and all
// primitive-specific drawing parameters.
type BatchOp struct {
	Type        BatchOpType // Operation type (pixels, line, rectangle, circle, contour, fill)
	LayerName   string      // Target layer name (resolved)
	FrameNumber int         // 1-based target frame number (resolved)
	Color       Color       // Drawing color (used by line, rectangle, circle, contour, fill)
	Pixels      []Pixel     // Pixels array (used by pixels)
	X1          int         // Line start X
	Y1          int         // Line start Y
	X2          int         // Line end X
	Y2          int         // Line end Y
	X           int         // Top-left X for rectangle, start X for fill
	Y           int         // Top-left Y for rectangle, start Y for fill
	Width       int         // Rectangle width
	Height      int         // Rectangle height
	CenterX     int         // Circle center X
	CenterY     int         // Circle center Y
	Radius      int         // Circle radius
	Filled      bool        // Fill flag for rectangle and circle
	Points      []Point     // Points array for contour
	Closed      bool        // Closed loop flag for contour
	Thickness   int         // Brush thickness for line and contour (1-100)
	Tolerance   int         // Tolerance for fill (0-255)
}
