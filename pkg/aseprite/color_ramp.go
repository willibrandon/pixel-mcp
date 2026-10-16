package aseprite

import (
	"fmt"
	"math"
	"strings"

	"github.com/lucasb-eyer/go-colorful"
)

// ColorRampOptions specifies options for generating a pixel art color ramp.
type ColorRampOptions struct {
	BaseColor string // Hex format #RRGGBB
	Steps     int    // Number of steps (3-9, default 5)
	Mode      string // "warm_highlight_cool_shadow", "cool_highlight_warm_shadow", "monochrome"
}

// shiftHue shifts hue h towards targetHue by angle deltaDegrees, taking the shortest angular path.
func shiftHue(h, targetHue, deltaDegrees float64) float64 {
	diff := targetHue - h
	for diff > 180 {
		diff -= 360
	}
	for diff < -180 {
		diff += 360
	}
	sign := 1.0
	if diff < 0 {
		sign = -1.0
	}
	maxShift := math.Min(math.Abs(diff), deltaDegrees)
	result := h + sign*maxShift
	for result >= 360 {
		result -= 360
	}
	for result < 0 {
		result += 360
	}
	return result
}

// GenerateColorRamp generates a scientifically balanced pixel art color ramp with hue shifting.
func GenerateColorRamp(opts ColorRampOptions) ([]string, error) {
	hexStr := strings.TrimSpace(opts.BaseColor)
	if !strings.HasPrefix(hexStr, "#") {
		hexStr = "#" + hexStr
	}

	col, err := colorful.Hex(hexStr)
	if err != nil {
		return nil, fmt.Errorf("invalid base color hex %q: %w", opts.BaseColor, err)
	}

	steps := opts.Steps
	if steps <= 0 {
		steps = 5
	}
	if steps < 3 {
		steps = 3
	}
	if steps > 9 {
		steps = 9
	}

	mode := opts.Mode
	if mode == "" {
		mode = "warm_highlight_cool_shadow"
	}

	h, s, l := col.Hsl()
	ramp := make([]string, steps)
	mid := steps / 2

	for i := 0; i < steps; i++ {
		if i == mid {
			ramp[i] = col.Hex()
			continue
		}

		// offset ranges from -1.0 (deep shadow) to +1.0 (peak highlight)
		var offset float64
		if i < mid {
			offset = float64(i-mid) / float64(mid)
		} else {
			offset = float64(i-mid) / float64(steps-1-mid)
		}

		stepH := h
		stepS := s
		var stepL float64

		if offset < 0 {
			// Shadows
			stepL = math.Max(0.05, l+offset*(l*0.80))
			stepS = math.Max(0.05, math.Min(1.0, s+math.Abs(offset)*0.10)) // Rich shadow saturation

			switch mode {
			case "warm_highlight_cool_shadow":
				// Shift towards cool blue/purple (~250 deg)
				stepH = shiftHue(h, 250.0, math.Abs(offset)*25.0)
			case "cool_highlight_warm_shadow":
				// Shift towards warm red/orange (~15 deg)
				stepH = shiftHue(h, 15.0, math.Abs(offset)*25.0)
			}
		} else {
			// Highlights
			stepL = math.Min(0.97, l+offset*((1.0-l)*0.80))
			stepS = math.Max(0.05, math.Min(1.0, s-offset*0.12)) // Soft highlight desaturation

			switch mode {
			case "warm_highlight_cool_shadow":
				// Shift towards warm golden yellow (~55 deg)
				stepH = shiftHue(h, 55.0, offset*25.0)
			case "cool_highlight_warm_shadow":
				// Shift towards cool cyan (~190 deg)
				stepH = shiftHue(h, 190.0, offset*25.0)
			}
		}

		rampCol := colorful.Hsl(stepH, stepS, stepL)
		ramp[i] = rampCol.Hex()
	}

	return ramp, nil
}
