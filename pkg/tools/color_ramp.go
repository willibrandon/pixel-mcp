package tools

import (
	"context"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/willibrandon/mtlog/core"
	"github.com/willibrandon/pixel-mcp/pkg/aseprite"
	"github.com/willibrandon/pixel-mcp/pkg/config"
)

// GenerateColorRampInput defines input parameters for generating a color ramp.
type GenerateColorRampInput struct {
	BaseColor  string `json:"base_color" jsonschema:"Base hex color (#RRGGBB format)"`
	Steps      int    `json:"steps,omitempty" jsonschema:"Number of shade steps (3-9, default: 5)"`
	Mode       string `json:"mode,omitempty" jsonschema:"Hue shift style: warm_highlight_cool_shadow (default), cool_highlight_warm_shadow, or monochrome"`
	SpritePath string `json:"sprite_path,omitempty" jsonschema:"Optional path to .aseprite file to automatically append ramp to sprite palette"`
}

// GenerateColorRampOutput defines the output of generate_color_ramp.
type GenerateColorRampOutput struct {
	BaseColor        string   `json:"base_color" jsonschema:"Base color used"`
	Ramp             []string `json:"ramp,omitempty" jsonschema:"Ordered array of hex colors from darkest shadow to brightest highlight"`
	Steps            int      `json:"steps" jsonschema:"Number of generated colors"`
	Mode             string   `json:"mode" jsonschema:"Hue shift mode used"`
	AppendedToSprite bool     `json:"appended_to_sprite,omitempty" jsonschema:"Whether colors were appended to the sprite palette"`
}

// RegisterColorRampTools registers the generate_color_ramp tool with the MCP server.
func RegisterColorRampTools(server *mcp.Server, client *aseprite.Client, gen *aseprite.LuaGenerator, cfg *config.Config, logger core.Logger) {
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "generate_color_ramp",
			Description: "Generate a professional pixel-art color ramp with perceptual hue shifting (cool shadows, warm highlights). Essential for shading characters, equipment, and tiles without flat or muddy colors.",
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint: false,
			},
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input GenerateColorRampInput) (*mcp.CallToolResult, *GenerateColorRampOutput, error) {
			opLogger := logger.WithContext(ctx)
			opLogger.Debug("generate_color_ramp called", "base_color", input.BaseColor, "steps", input.Steps)

			ramp, err := aseprite.GenerateColorRamp(aseprite.ColorRampOptions{
				BaseColor: input.BaseColor,
				Steps:     input.Steps,
				Mode:      input.Mode,
			})
			if err != nil {
				return ToolError[GenerateColorRampOutput]("failed to generate color ramp", err)
			}

			out := &GenerateColorRampOutput{
				BaseColor: input.BaseColor,
				Ramp:      ramp,
				Steps:     len(ramp),
				Mode:      input.Mode,
			}

			if input.SpritePath != "" {
				if _, err := os.Stat(input.SpritePath); err == nil {
					for _, col := range ramp {
						script := gen.AddPaletteColor(col)
						_, _ = client.ExecuteLua(ctx, script, input.SpritePath)
					}
					out.AppendedToSprite = true
				}
			}

			return ToolSuccess(out)
		},
	)
}
