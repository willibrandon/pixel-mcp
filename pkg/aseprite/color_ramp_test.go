package aseprite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateColorRamp_Success(t *testing.T) {
	// Test standard green tunic color with warm highlight and cool shadow
	ramp, err := GenerateColorRamp(ColorRampOptions{
		BaseColor: "#38761D",
		Steps:     5,
		Mode:      "warm_highlight_cool_shadow",
	})
	require.NoError(t, err)
	assert.Len(t, ramp, 5)

	// Middle shade should match base color
	assert.Equal(t, "#38761d", ramp[2])

	// Ensure all colors are valid hex
	for _, col := range ramp {
		assert.Regexp(t, "^#[0-9a-fA-F]{6}$", col)
	}
}

func TestGenerateColorRamp_InvalidHex(t *testing.T) {
	_, err := GenerateColorRamp(ColorRampOptions{
		BaseColor: "invalid-color",
		Steps:     5,
	})
	assert.Error(t, err)
}

func TestGenerateColorRamp_ClampSteps(t *testing.T) {
	ramp1, err := GenerateColorRamp(ColorRampOptions{
		BaseColor: "#FF0000",
		Steps:     1, // Should clamp to 3
	})
	require.NoError(t, err)
	assert.Len(t, ramp1, 3)

	ramp2, err := GenerateColorRamp(ColorRampOptions{
		BaseColor: "#FF0000",
		Steps:     15, // Should clamp to 9
	})
	require.NoError(t, err)
	assert.Len(t, ramp2, 9)
}
