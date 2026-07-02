package aseprite

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"

	"github.com/lucasb-eyer/go-colorful"
)

// QuantizePalette reduces image colors using specified quantization algorithm.
// Returns palette colors (hex strings), original color count, and error.
func QuantizePalette(img image.Image, targetColors int, algorithm string, preserveTransparency bool) ([]string, int, error) {
	if targetColors < 2 || targetColors > 256 {
		return nil, 0, fmt.Errorf("targetColors must be between 2 and 256, got %d", targetColors)
	}

	// Count unique colors in original image
	originalColors := CountUniqueColors(img, preserveTransparency)

	// Sample pixels (subsample for large images)
	pixels := samplePixels(img, 10000)
	if len(pixels) == 0 {
		return nil, 0, fmt.Errorf("no pixels to sample from image")
	}

	// Filter transparent pixels if preserveTransparency is true
	if preserveTransparency {
		filtered := make([]color.Color, 0, len(pixels))
		for _, p := range pixels {
			_, _, _, a := p.RGBA()
			if a > 0 {
				filtered = append(filtered, p)
			}
		}
		pixels = filtered
	}

	if len(pixels) == 0 {
		return nil, originalColors, fmt.Errorf("no non-transparent pixels to quantize")
	}

	// Run quantization algorithm
	var paletteColors []color.Color
	switch algorithm {
	case "median_cut":
		paletteColors = MedianCutQuantization(pixels, targetColors)
	case "kmeans":
		// Use existing k-means implementation from palette.go
		colorfulPixels := make([]colorful.Color, len(pixels))
		for i, p := range pixels {
			r, g, b, _ := p.RGBA()
			colorfulPixels[i] = colorful.Color{
				R: float64(r) / 65535.0,
				G: float64(g) / 65535.0,
				B: float64(b) / 65535.0,
			}
		}
		centroids := kmeansClustering(colorfulPixels, targetColors, 100)
		paletteColors = make([]color.Color, len(centroids))
		for i, c := range centroids {
			r, g, b := c.RGB255()
			paletteColors[i] = color.RGBA{R: r, G: g, B: b, A: 255}
		}
	case "octree":
		paletteColors = OctreeQuantization(pixels, targetColors)
	default:
		return nil, 0, fmt.Errorf("unknown algorithm: %s (must be median_cut, kmeans, or octree)", algorithm)
	}

	// Add transparency to palette if needed
	if preserveTransparency {
		// Check if image has any transparent pixels
		hasTransparency := false
		bounds := img.Bounds()
		for y := bounds.Min.Y; y < bounds.Max.Y && !hasTransparency; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				_, _, _, a := img.At(x, y).RGBA()
				if a == 0 {
					hasTransparency = true
					break
				}
			}
		}
		if hasTransparency {
			paletteColors = append([]color.Color{color.RGBA{R: 0, G: 0, B: 0, A: 0}}, paletteColors...)
		}
	}

	// Convert to hex strings
	hexPalette := make([]string, len(paletteColors))
	for i, c := range paletteColors {
		r, g, b, a := c.RGBA()
		if a == 0 {
			hexPalette[i] = "#00000000" // Transparent
		} else {
			hexPalette[i] = fmt.Sprintf("#%02X%02X%02X", uint8(r>>8), uint8(g>>8), uint8(b>>8))
		}
	}

	return hexPalette, originalColors, nil
}

// MedianCutQuantization implements the median cut algorithm for color quantization.
// Recursively splits color space by median values to produce a balanced palette.
func MedianCutQuantization(pixels []color.Color, targetColors int) []color.Color {
	if len(pixels) == 0 || targetColors <= 0 {
		return nil
	}

	// Reduce to a frequency histogram of unique colors before bucketing. A
	// raw, duplicate-laden pixel list biases every split toward whatever
	// large solid-fill color dominates the pixel count: cutting at the
	// pixel-COUNT median of a mostly-one-color bucket lands the split point
	// inside that same color's run (not at a boundary between colors), so
	// the algorithm keeps re-subdividing the dominant fill while small
	// accent regions (a 2px highlight next to a 200px fill, typical of
	// pixel art) never get isolated into their own bucket and end up
	// averaged into a blended, off-palette color instead. Splitting the
	// unique-color histogram instead guarantees the split always falls
	// between distinct colors, so targetColors >= the number of unique
	// colors recovers every one of them exactly.
	entries := buildColorHistogram(pixels)

	// Ensure targetColors doesn't exceed number of unique colors
	if targetColors > len(entries) {
		targetColors = len(entries)
	}

	// Create initial bucket with all unique colors
	buckets := []colorBucket{{entries: entries}}

	// Recursively split buckets until we have targetColors
	for len(buckets) < targetColors {
		// Find bucket with largest range
		maxRange := 0.0
		maxIdx := 0
		for i, bucket := range buckets {
			r := bucket.colorRange()
			if r > maxRange {
				maxRange = r
				maxIdx = i
			}
		}

		// Stop if no bucket has any range (all remaining buckets are single colors)
		if maxRange == 0 {
			break
		}

		// Split bucket
		bucket := buckets[maxIdx]
		left, right := bucket.split()

		// Replace bucket with two new buckets
		buckets = append(buckets[:maxIdx], append([]colorBucket{left, right}, buckets[maxIdx+1:]...)...)
	}

	// Calculate average color for each bucket
	palette := make([]color.Color, len(buckets))
	for i, bucket := range buckets {
		palette[i] = bucket.average()
	}

	return palette
}

// colorCount pairs a unique color with the number of pixels that had it.
type colorCount struct {
	c     color.Color
	count int
}

// buildColorHistogram reduces a (possibly duplicate-laden) pixel slice to a
// list of unique colors with their pixel counts, preserving first-seen order.
func buildColorHistogram(pixels []color.Color) []colorCount {
	index := make(map[uint32]int, len(pixels))
	entries := make([]colorCount, 0, len(pixels))

	for _, p := range pixels {
		r, g, b, _ := p.RGBA()
		key := (r&0xFF00)<<8 | (g & 0xFF00) | (b&0xFF00)>>8
		if i, ok := index[key]; ok {
			entries[i].count++
			continue
		}
		index[key] = len(entries)
		entries = append(entries, colorCount{c: p, count: 1})
	}

	return entries
}

// colorBucket represents a bucket of unique colors (with pixel counts) for the median cut algorithm.
type colorBucket struct {
	entries []colorCount
}

// colorRange calculates the range of the bucket in RGB space.
func (b *colorBucket) colorRange() float64 {
	if len(b.entries) == 0 {
		return 0
	}

	var minR, minG, minB uint32 = 65535, 65535, 65535
	var maxR, maxG, maxB uint32 = 0, 0, 0

	for _, e := range b.entries {
		r, g, bl, _ := e.c.RGBA()
		if r < minR {
			minR = r
		}
		if r > maxR {
			maxR = r
		}
		if g < minG {
			minG = g
		}
		if g > maxG {
			maxG = g
		}
		if bl < minB {
			minB = bl
		}
		if bl > maxB {
			maxB = bl
		}
	}

	rRange := float64(maxR - minR)
	gRange := float64(maxG - minG)
	bRange := float64(maxB - minB)

	return rRange + gRange + bRange
}

// split splits the bucket along the dimension with the largest range. The
// split point is chosen where cumulative pixel count crosses half of the
// bucket's total pixel mass (so common colors still get split priority over
// rare ones, matching standard median-cut behavior), but since entries are
// already unique colors, the split always falls BETWEEN distinct colors -
// never through the middle of one - so a bucket of already-few unique colors
// is never blended together by a split.
func (b *colorBucket) split() (colorBucket, colorBucket) {
	if len(b.entries) < 2 {
		return *b, colorBucket{}
	}

	// Find dimension with largest range
	var minR, minG, minB uint32 = 65535, 65535, 65535
	var maxR, maxG, maxB uint32 = 0, 0, 0

	for _, e := range b.entries {
		r, g, bl, _ := e.c.RGBA()
		if r < minR {
			minR = r
		}
		if r > maxR {
			maxR = r
		}
		if g < minG {
			minG = g
		}
		if g > maxG {
			maxG = g
		}
		if bl < minB {
			minB = bl
		}
		if bl > maxB {
			maxB = bl
		}
	}

	rRange := maxR - minR
	gRange := maxG - minG
	bRange := maxB - minB

	// Sort unique colors by the dimension with largest range
	entries := make([]colorCount, len(b.entries))
	copy(entries, b.entries)

	if rRange >= gRange && rRange >= bRange {
		// Sort by red
		sort.Slice(entries, func(i, j int) bool {
			r1, _, _, _ := entries[i].c.RGBA()
			r2, _, _, _ := entries[j].c.RGBA()
			return r1 < r2
		})
	} else if gRange >= bRange {
		// Sort by green
		sort.Slice(entries, func(i, j int) bool {
			_, g1, _, _ := entries[i].c.RGBA()
			_, g2, _, _ := entries[j].c.RGBA()
			return g1 < g2
		})
	} else {
		// Sort by blue
		sort.Slice(entries, func(i, j int) bool {
			_, _, b1, _ := entries[i].c.RGBA()
			_, _, b2, _ := entries[j].c.RGBA()
			return b1 < b2
		})
	}

	// Split where cumulative pixel count crosses half the bucket's total
	// mass, clamped so both sides always get at least one unique color.
	total := 0
	for _, e := range entries {
		total += e.count
	}
	half := total / 2

	cum := 0
	splitIdx := 1
	for i, e := range entries {
		cum += e.count
		if cum >= half {
			splitIdx = i + 1
			break
		}
	}
	if splitIdx < 1 {
		splitIdx = 1
	}
	if splitIdx > len(entries)-1 {
		splitIdx = len(entries) - 1
	}

	return colorBucket{entries: entries[:splitIdx]}, colorBucket{entries: entries[splitIdx:]}
}

// average calculates the pixel-count-weighted average color of the bucket.
// A single-entry bucket returns that entry's exact color unchanged.
func (b *colorBucket) average() color.Color {
	if len(b.entries) == 0 {
		return color.RGBA{R: 0, G: 0, B: 0, A: 255}
	}

	var sumR, sumG, sumB uint64
	var totalCount uint64
	for _, e := range b.entries {
		r, g, bl, _ := e.c.RGBA()
		w := uint64(e.count)
		sumR += uint64(r) * w
		sumG += uint64(g) * w
		sumB += uint64(bl) * w
		totalCount += w
	}

	return color.RGBA{
		R: uint8((sumR / totalCount) >> 8),
		G: uint8((sumG / totalCount) >> 8),
		B: uint8((sumB / totalCount) >> 8),
		A: 255,
	}
}

// OctreeQuantization implements octree color quantization algorithm.
// Fast, tree-based color reduction suitable for large images.
func OctreeQuantization(pixels []color.Color, targetColors int) []color.Color {
	if len(pixels) == 0 || targetColors <= 0 {
		return nil
	}

	// Build octree
	root := &octreeNode{
		level:    0,
		children: make([]*octreeNode, 8),
	}

	// Track nodes at each level for reduction
	levels := make([][]*octreeNode, 9)

	// Insert all pixels into octree
	for _, p := range pixels {
		r, g, b, _ := p.RGBA()
		root.insert(uint8(r>>8), uint8(g>>8), uint8(b>>8), 0, levels)
	}

	// Reduce tree to targetColors
	for root.leafCount() > targetColors {
		// Find deepest level with nodes
		level := 7
		for level >= 0 && len(levels[level]) == 0 {
			level--
		}
		if level < 0 {
			break
		}

		// Reduce one node at this level
		node := levels[level][0]
		levels[level] = levels[level][1:]
		node.reduce()
	}

	// Extract palette from leaf nodes
	palette := make([]color.Color, 0, targetColors)
	root.getPalette(&palette)

	return palette
}

// octreeNode represents a node in the octree.
type octreeNode struct {
	level      int
	r, g, b    uint64
	pixelCount uint64
	children   []*octreeNode
	isLeaf     bool
}

// insert adds a pixel to the octree.
func (n *octreeNode) insert(r, g, b uint8, level int, levels [][]*octreeNode) {
	if level >= 8 {
		// Leaf node
		n.isLeaf = true
		n.r += uint64(r)
		n.g += uint64(g)
		n.b += uint64(b)
		n.pixelCount++
		return
	}

	// Calculate octree index from RGB bits
	idx := 0
	if r&(1<<(7-level)) != 0 {
		idx |= 4
	}
	if g&(1<<(7-level)) != 0 {
		idx |= 2
	}
	if b&(1<<(7-level)) != 0 {
		idx |= 1
	}

	// Create child if needed
	if n.children[idx] == nil {
		n.children[idx] = &octreeNode{
			level:    level + 1,
			children: make([]*octreeNode, 8),
		}
		// Track node at this level for reduction
		if level < 8 {
			levels[level] = append(levels[level], n.children[idx])
		}
	}

	n.children[idx].insert(r, g, b, level+1, levels)

	// Accumulate color values for averaging
	n.r += uint64(r)
	n.g += uint64(g)
	n.b += uint64(b)
	n.pixelCount++
}

// reduce merges all children into this node (converts to leaf).
func (n *octreeNode) reduce() {
	// Already accumulated color values from children
	n.isLeaf = true
	n.children = make([]*octreeNode, 8)
}

// leafCount counts the number of leaf nodes.
func (n *octreeNode) leafCount() int {
	if n.isLeaf {
		return 1
	}

	count := 0
	for _, child := range n.children {
		if child != nil {
			count += child.leafCount()
		}
	}
	return count
}

// getPalette extracts colors from leaf nodes.
func (n *octreeNode) getPalette(palette *[]color.Color) {
	if n.isLeaf && n.pixelCount > 0 {
		*palette = append(*palette, color.RGBA{
			R: uint8(n.r / n.pixelCount),
			G: uint8(n.g / n.pixelCount),
			B: uint8(n.b / n.pixelCount),
			A: 255,
		})
		return
	}

	for _, child := range n.children {
		if child != nil {
			child.getPalette(palette)
		}
	}
}

// RemapPixelsWithDithering remaps image pixels to palette colors with optional Floyd-Steinberg dithering.
func RemapPixelsWithDithering(img image.Image, palette []color.Color, dither bool) *image.RGBA {
	bounds := img.Bounds()
	result := image.NewRGBA(bounds)

	if dither {
		return FloydSteinbergDither(img, palette)
	}

	// Simple nearest-color mapping without dithering
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)

			// Preserve transparency
			_, _, _, a := c.RGBA()
			if a == 0 {
				result.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 0})
				continue
			}

			// Find nearest palette color
			nearest := findNearestColor(c, palette)
			result.Set(x, y, nearest)
		}
	}

	return result
}

// FloydSteinbergDither applies Floyd-Steinberg error diffusion dithering.
func FloydSteinbergDither(img image.Image, palette []color.Color) *image.RGBA {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Create working buffer for error accumulation
	buffer := make([][]color.RGBA, height)
	for y := 0; y < height; y++ {
		buffer[y] = make([]color.RGBA, width)
		for x := 0; x < width; x++ {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			buffer[y][x] = color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(a >> 8),
			}
		}
	}

	result := image.NewRGBA(bounds)

	// Apply Floyd-Steinberg dithering
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			oldPixel := buffer[y][x]

			// Preserve transparency
			if oldPixel.A == 0 {
				result.Set(bounds.Min.X+x, bounds.Min.Y+y, color.RGBA{R: 0, G: 0, B: 0, A: 0})
				continue
			}

			// Find nearest palette color
			newPixel := findNearestColor(oldPixel, palette)
			result.Set(bounds.Min.X+x, bounds.Min.Y+y, newPixel)

			// Calculate error
			nr, ng, nb, _ := newPixel.RGBA()
			errR := int(oldPixel.R) - int(nr>>8)
			errG := int(oldPixel.G) - int(ng>>8)
			errB := int(oldPixel.B) - int(nb>>8)

			// Distribute error to neighboring pixels (7/16, 3/16, 5/16, 1/16)
			if x+1 < width {
				buffer[y][x+1] = addError(buffer[y][x+1], errR*7/16, errG*7/16, errB*7/16)
			}
			if y+1 < height {
				if x > 0 {
					buffer[y+1][x-1] = addError(buffer[y+1][x-1], errR*3/16, errG*3/16, errB*3/16)
				}
				buffer[y+1][x] = addError(buffer[y+1][x], errR*5/16, errG*5/16, errB*5/16)
				if x+1 < width {
					buffer[y+1][x+1] = addError(buffer[y+1][x+1], errR*1/16, errG*1/16, errB*1/16)
				}
			}
		}
	}

	return result
}

// addError adds error values to a color, clamping to valid range.
func addError(c color.RGBA, errR, errG, errB int) color.RGBA {
	r := clamp(int(c.R) + errR)
	g := clamp(int(c.G) + errG)
	b := clamp(int(c.B) + errB)
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: c.A}
}

// clamp clamps a value to 0-255 range.
func clamp(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

// findNearestColor finds the nearest color in the palette using LAB color space.
func findNearestColor(c color.Color, palette []color.Color) color.Color {
	if len(palette) == 0 {
		return c
	}

	r1, g1, b1, _ := c.RGBA()
	color1 := colorful.Color{
		R: float64(r1) / 65535.0,
		G: float64(g1) / 65535.0,
		B: float64(b1) / 65535.0,
	}

	minDist := math.MaxFloat64
	var nearest color.Color = palette[0]

	for _, p := range palette {
		r2, g2, b2, _ := p.RGBA()
		color2 := colorful.Color{
			R: float64(r2) / 65535.0,
			G: float64(g2) / 65535.0,
			B: float64(b2) / 65535.0,
		}

		dist := colorDistanceLab(color1, color2)
		if dist < minDist {
			minDist = dist
			nearest = p
		}
	}

	return nearest
}

// CountUniqueColors counts the number of unique colors in an image.
func CountUniqueColors(img image.Image, preserveTransparency bool) int {
	colorSet := make(map[uint32]bool)
	bounds := img.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()

			// Skip transparent pixels if preserveTransparency is true
			if preserveTransparency && a == 0 {
				continue
			}

			// Pack RGBA into uint32 for set membership
			packed := (r&0xFF00)<<8 | (g & 0xFF00) | (b&0xFF00)>>8
			colorSet[packed] = true
		}
	}

	return len(colorSet)
}
