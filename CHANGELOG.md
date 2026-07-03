# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- **draw_pixels: a later call reverts previously-transparent pixels to opaque black**
  - On an indexed sprite `CreateCanvas` sets `Sprite.transparentColor` to 255 so
    palette index 0 can hold a real color, which means index 0 is opaque. When
    `draw_pixels` normalizes the target cel it builds a full-canvas image with
    `Image(spr.width, spr.height, spr.colorMode)`, and a freshly created indexed
    image is filled with index 0 (opaque), not the transparent index
  - Compositing the existing content with the default NORMAL blend
    (`full:drawImage(cel.image, cel.position)`) skips the source's transparent
    (index 255) pixels, so every pixel a previous call had made transparent but
    the current call does not rewrite is left showing the opaque index-0
    background, silently turning transparent areas into opaque black (the exact
    symptom warned about for locked-palette sprites)
  - Fixed by clearing the full-canvas image to the sprite's transparent color
    (`full:clear(spr.transparentColor)`) before compositing/drawing on indexed
    sprites, in both the existing-cel and new-cel paths; RGB and grayscale
    behavior is unchanged
- **create_canvas: a new indexed canvas starts opaque instead of transparent**
  - A new indexed sprite's initial cel is filled with palette index 0, but
    `CreateCanvas` designates index 255 as the transparent color, so index 0 is
    an opaque color (black on DawnBringer 32) and a fresh canvas read back as an
    opaque background rather than transparent
  - Fixed by clearing the initial cel to the sprite's transparent color right
    after creation, so a fresh indexed canvas is genuinely transparent; RGB and
    grayscale are unaffected
- **quantize_palette: median cut blends distinct colors under imbalanced pixel frequency**
  - `MedianCutQuantization` split each bucket at the pixel-array midpoint
    (`mid := len(pixels) / 2`), not at a boundary between distinct colors.
    Pixel art typically has wildly imbalanced color frequency (a large
    solid fill vastly outnumbering a 1-2px accent detail), so splitting at
    the raw pixel-count median lands the cut inside the dominant color's
    own run instead of between colors
  - The dominant color got needlessly re-subdivided into near-duplicate
    buckets while rare accent colors never got isolated into their own
    bucket, and were instead averaged into a blended, off-palette hex
    value - even when `target_colors` comfortably exceeded the number of
    unique colors actually present, which should guarantee lossless
    recovery
  - Fixed by reducing to a frequency-weighted histogram of unique colors
    before bucketing, so a split always falls between distinct colors;
    the split point still favors common colors over rare ones by
    cumulative pixel count, preserving normal median-cut reduction
    behavior when `target_colors` is genuinely smaller than the number of
    unique colors
- **Indexed palette: transparentColor collides with a real color after resize**
  - `Palette:resize` silently clamps `Sprite.transparentColor` into the new
    palette's valid index range, so after `set_palette`, `quantize_palette`,
    `add_palette_color`, or `apply_auto_shading` resize an indexed sprite's
    palette, the sprite's transparent-color index can land on the same index
    as a real, in-use color (typically the palette's last entry)
  - Once that collision happens, painting "transparent" (e.g. `fill_area`
    with a fully-transparent color) writes that colliding index, so
    background fills read back as an opaque real color instead of
    transparent, and pixels intentionally drawn in that color can be
    auto-trimmed away as if they were transparent
  - Fixed by re-asserting `transparentColor = 255` after every palette
    resize on an indexed sprite, keeping it safely out of the palette's
    real, addressable range

## [0.5.0] - 2025-10-18

### Added
- **Layer Flattening Tool** (`flatten_layers`)
  - Flattens all layers in a sprite into a single layer
  - Uses Aseprite's built-in flatten operation
  - Integration test and example demonstration included

## [0.4.0] - 2025-10-18

### Added
- **Palette Quantization Tool** (`quantize_palette`)
  - Reduces colors in images using three algorithms: median_cut, k-means, and octree
  - Supports color counts from 2-256 colors
  - Optional Floyd-Steinberg dithering for smoother gradients
  - Converts sprites to indexed color mode with optimized palettes
  - Example demonstrating all three algorithms with side-by-side comparison

- **Automatic Shading Tool** (`apply_auto_shading`)
  - Automatically adds shading to sprites based on light direction (8 directions)
  - Geometry-based analysis identifies surfaces and calculates per-pixel normals
  - Three shading styles: cell (hard-edged bands), smooth (dithered gradients), soft (subtle blending)
  - Configurable intensity (0.0-1.0) and optional hue shifting (shadows→cool, highlights→warm)
  - Generates shadow and highlight colors for each base color
  - Example demonstrating all styles, intensities, light directions, and hue shifting

### Fixed
- `ImportImage` tool now correctly handles color mode conversion and cel creation
  - Uses proper image buffer and BlendMode.SRC for accurate color conversion
  - Fixes issue where imported images appeared as empty/transparent pixels
- `ApplyAutoShadingResult` Lua script properly applies shaded pixels
  - Creates new cel with shaded image instead of drawing onto existing cel
  - Fixes issue where shading wasn't visible in exported sprites

## [0.3.0] - 2025-10-18

### Added
- Floyd-Steinberg error diffusion dithering pattern
  - Brings total dithering patterns to 16 (from 15)
  - Creates smooth gradients using error propagation algorithm
  - Uses standard 7/16, 5/16, 3/16, 1/16 weight distribution
  - Color selection by Euclidean distance for accurate transitions
  - Handles edge case where width=1 to prevent division by zero
- Test for Floyd-Steinberg gradient verification using pixel inspection
- Side-by-side example comparing Bayer 4x4 vs Floyd-Steinberg dithering

### Changed
- Updated `draw_with_dither` tool description to reflect 16 patterns

## [0.2.0] - 2025-10-17

### Changed
- Renamed project from `aseprite-mcp` to `pixel-mcp`

## [0.1.0] - 2025-10-16

First official release of the Aseprite MCP Server.

### Added

#### Core Infrastructure
- Initial project structure and Go module initialization
- MCP server with stdio transport
- Configuration management (file-based at ~/.config/pixel-mcp/config.json)
- Aseprite client with Lua script execution and timeout handling
- Lua script generation utilities with proper escaping and security
- Comprehensive GoDoc documentation for all packages (pkg/aseprite, pkg/config, pkg/server, pkg/tools)
- Integration test suite with real Aseprite testing (80+ tests)
- End-to-end workflow tests
- Performance benchmarks for all critical operations
- Benchmark documentation showing all PRD targets exceeded
- Build system with cross-platform Makefile
- Health check and version flags
- Structured logging with mtlog (message template logging)

#### Canvas Management Tools (6 tools)
- `create_canvas` - Create new sprites with RGB, grayscale, or indexed color modes
- `add_layer` - Add layers to existing sprites
- `add_frame` - Add animation frames with duration control
- `delete_layer` - Remove layers (prevents deletion of last layer)
- `delete_frame` - Remove frames (prevents deletion of last frame)
- `get_sprite_info` - Retrieve sprite metadata (dimensions, color mode, layers, frames)

#### Drawing Tools (7 tools)
- `draw_pixels` - Draw individual pixels (batch operations, supports 10K+ pixels)
- `draw_line` - Draw lines with thickness control
- `draw_rectangle` - Draw rectangles (filled or outline)
- `draw_circle` - Draw circles/ellipses (filled or outline)
- `draw_contour` - Draw multi-point polylines and polygons
- `fill_area` - Flood fill with color tolerance
- `draw_with_dither` - Apply dithering patterns (16 patterns: Bayer matrices, Floyd-Steinberg, textures, noise)

#### Animation Tools (5 tools)
- `set_frame_duration` - Control frame timing in milliseconds
- `create_tag` - Define animation sequences with playback direction (forward, reverse, pingpong)
- `delete_tag` - Remove animation tags
- `duplicate_frame` - Copy frames with all layer content
- `link_cel` - Create linked cels for animation efficiency

#### Selection Tools (8 tools)
- `select_rectangle` - Select rectangular regions with blend modes (replace, add, subtract, intersect)
- `select_ellipse` - Select elliptical regions
- `select_all` - Select entire canvas
- `deselect` - Clear current selection
- `move_selection` - Translate selection mask
- `cut_selection` - Cut pixels to clipboard
- `copy_selection` - Copy pixels to clipboard
- `paste_clipboard` - Paste clipboard content at position

#### Palette Tools (6 tools)
- `get_palette` - Retrieve current palette colors
- `set_palette` - Set entire palette (1-256 colors)
- `set_palette_color` - Modify individual palette entry
- `add_palette_color` - Append color to palette
- `sort_palette` - Sort by hue, saturation, brightness, or luminance
- `analyze_palette_harmonies` - Analyze color relationships (complementary, triadic, analogous, temperature)

#### Transform & Filter Tools (7 tools)
- `flip_sprite` - Flip horizontally or vertically (sprite/layer/cel scope)
- `rotate_sprite` - Rotate 90/180/270 degrees
- `scale_sprite` - Scale with algorithm choice (nearest, bilinear, rotsprite)
- `crop_sprite` - Crop to rectangular region
- `resize_canvas` - Change canvas size with anchor positioning
- `apply_outline` - Add colored outline effect to layers
- `downsample_image` - High-quality image downsampling using box filter (area averaging)

#### Export & Import Tools (4 tools)
- `export_sprite` - Export to PNG, GIF, JPG, BMP (single frame or animation)
- `export_spritesheet` - Export animation as spritesheet with layouts (horizontal, vertical, rows, columns, packed)
- `import_image` - Import external images as layers
- `save_as` - Save sprite to new file path

#### Professional Pixel Art Tools (4 tools)
- `analyze_reference` - Extract palette from reference images using k-means clustering
- `apply_shading` - Apply palette-constrained shading (smooth, hard, pillow styles, 8 light directions)
- `suggest_antialiasing` - Detect jagged edges and suggest smoothing colors (auto-apply option)
- Palette-aware drawing - All drawing tools support `use_palette` flag for color snapping

#### Advanced Features
- Pixel data inspection with `get_pixels` tool (pagination support for large regions)
- Brightness map generation for shading analysis
- Sobel edge detection for composition and antialiasing
- Composition analysis (rule of thirds, focal points)
- 16 dithering patterns for gradients and textures
- LAB color space for perceptually accurate color matching
- Color temperature analysis (warm/cool classification)