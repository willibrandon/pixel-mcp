//go:build integration
// +build integration

package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/willibrandon/pixel-mcp/internal/testutil"
	"github.com/willibrandon/pixel-mcp/pkg/aseprite"
)

// Benchmarks for complete tool workflows (client + generator).
// These test the full stack that MCP tools use.
// Run with: go test -tags=integration -bench=. -benchmem ./pkg/tools

func BenchmarkCompleteWorkflow_Simple(b *testing.B) {
	cfg := testutil.LoadTestConfigTB(b)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create 64x64 canvas
		spritePath := filepath.Join(b.TempDir(), "bench-simple.aseprite")
		createScript := gen.CreateCanvas(64, 64, aseprite.ColorModeRGB, spritePath)
		_, err := client.ExecuteLua(ctx, createScript, "")
		if err != nil {
			b.Fatalf("CreateCanvas failed: %v", err)
		}

		// Draw circle
		drawScript := gen.DrawCircle("Layer 1", 1, 32, 32, 20, aseprite.Color{R: 255, G: 0, B: 0, A: 255}, true, false)
		_, err = client.ExecuteLua(ctx, drawScript, spritePath)
		if err != nil {
			b.Fatalf("DrawCircle failed: %v", err)
		}

		// Export PNG
		exportPath := filepath.Join(b.TempDir(), "bench-output.png")
		exportScript := gen.ExportSprite(exportPath, 0)
		_, err = client.ExecuteLua(ctx, exportScript, spritePath)
		if err != nil {
			b.Fatalf("ExportSprite failed: %v", err)
		}

		os.Remove(spritePath)
		os.Remove(exportPath)
	}
}

func BenchmarkCompleteWorkflow_Animation(b *testing.B) {
	cfg := testutil.LoadTestConfigTB(b)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create canvas
		spritePath := filepath.Join(b.TempDir(), "bench-anim.aseprite")
		createScript := gen.CreateCanvas(48, 48, aseprite.ColorModeRGB, spritePath)
		_, err := client.ExecuteLua(ctx, createScript, "")
		if err != nil {
			b.Fatalf("CreateCanvas failed: %v", err)
		}

		// Add 3 frames
		for j := 0; j < 3; j++ {
			frameScript := gen.AddFrame(100)
			_, err = client.ExecuteLua(ctx, frameScript, spritePath)
			if err != nil {
				b.Fatalf("AddFrame failed: %v", err)
			}
		}

		// Draw on each frame
		for frame := 1; frame <= 4; frame++ {
			drawScript := gen.DrawCircle("Layer 1", frame, 24, 24, 5+frame*2, aseprite.Color{R: 0, G: 255, B: 0, A: 255}, true, false)
			_, err = client.ExecuteLua(ctx, drawScript, spritePath)
			if err != nil {
				b.Fatalf("DrawCircle failed: %v", err)
			}
		}

		// Export GIF
		exportPath := filepath.Join(b.TempDir(), "bench-anim.gif")
		exportScript := gen.ExportSprite(exportPath, 0)
		_, err = client.ExecuteLua(ctx, exportScript, spritePath)
		if err != nil {
			b.Fatalf("ExportSprite failed: %v", err)
		}

		os.Remove(spritePath)
		os.Remove(exportPath)
	}
}

func BenchmarkCompleteWorkflow_MultiLayer(b *testing.B) {
	cfg := testutil.LoadTestConfigTB(b)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create canvas
		spritePath := filepath.Join(b.TempDir(), "bench-layers.aseprite")
		createScript := gen.CreateCanvas(100, 100, aseprite.ColorModeRGB, spritePath)
		_, err := client.ExecuteLua(ctx, createScript, "")
		if err != nil {
			b.Fatalf("CreateCanvas failed: %v", err)
		}

		// Add background layer
		layerScript := gen.AddLayer("Background")
		_, err = client.ExecuteLua(ctx, layerScript, spritePath)
		if err != nil {
			b.Fatalf("AddLayer failed: %v", err)
		}

		// Fill background
		fillScript := gen.FillArea("Background", 1, 50, 50, aseprite.Color{R: 50, G: 50, B: 150, A: 255}, 0, false)
		_, err = client.ExecuteLua(ctx, fillScript, spritePath)
		if err != nil {
			b.Fatalf("FillArea failed: %v", err)
		}

		// Draw on Layer 1
		rectScript := gen.DrawRectangle("Layer 1", 1, 30, 30, 40, 40, aseprite.Color{R: 255, G: 255, B: 0, A: 255}, true, false)
		_, err = client.ExecuteLua(ctx, rectScript, spritePath)
		if err != nil {
			b.Fatalf("DrawRectangle failed: %v", err)
		}

		// Export
		exportPath := filepath.Join(b.TempDir(), "bench-layers.png")
		exportScript := gen.ExportSprite(exportPath, 0)
		_, err = client.ExecuteLua(ctx, exportScript, spritePath)
		if err != nil {
			b.Fatalf("ExportSprite failed: %v", err)
		}

		os.Remove(spritePath)
		os.Remove(exportPath)
	}
}

func BenchmarkCompleteWorkflow_PixelBatch(b *testing.B) {
	cfg := testutil.LoadTestConfigTB(b)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	// Generate 1000 pixels once
	pixels := make([]aseprite.Pixel, 1000)
	for i := 0; i < 1000; i++ {
		pixels[i] = aseprite.Pixel{
			Point: aseprite.Point{X: i % 100, Y: i / 100},
			Color: aseprite.Color{R: uint8(i % 256), G: 128, B: uint8((255 - i) % 256), A: 255},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create canvas
		spritePath := filepath.Join(b.TempDir(), "bench-pixels.aseprite")
		createScript := gen.CreateCanvas(100, 100, aseprite.ColorModeRGB, spritePath)
		_, err := client.ExecuteLua(ctx, createScript, "")
		if err != nil {
			b.Fatalf("CreateCanvas failed: %v", err)
		}

		// Draw 1000 pixels
		drawScript := gen.DrawPixels("Layer 1", 1, pixels, false)
		_, err = client.ExecuteLua(ctx, drawScript, spritePath)
		if err != nil {
			b.Fatalf("DrawPixels failed: %v", err)
		}

		// Export
		exportPath := filepath.Join(b.TempDir(), "bench-pixels.png")
		exportScript := gen.ExportSprite(exportPath, 0)
		_, err = client.ExecuteLua(ctx, exportScript, spritePath)
		if err != nil {
			b.Fatalf("ExportSprite failed: %v", err)
		}

		os.Remove(spritePath)
		os.Remove(exportPath)
	}
}

// Professional Pixel Art Feature Benchmarks

func BenchmarkProfessional_DrawWithDither(b *testing.B) {
	cfg := testutil.LoadTestConfigTB(b)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	// Setup: create canvas once
	spritePath := filepath.Join(b.TempDir(), "bench-dither.aseprite")
	createScript := gen.CreateCanvas(128, 128, aseprite.ColorModeRGB, spritePath)
	_, err := client.ExecuteLua(ctx, createScript, "")
	if err != nil {
		b.Fatalf("CreateCanvas failed: %v", err)
	}
	defer os.Remove(spritePath)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ditherScript := gen.DrawWithDither("Layer 1", 1, 0, 0, 128, 128, "#0A1628", "#5B7FB0", "bayer_4x4", 0.5)
		_, err := client.ExecuteLua(ctx, ditherScript, spritePath)
		if err != nil {
			b.Fatalf("DrawWithDither failed: %v", err)
		}
	}
}

func BenchmarkProfessional_ApplyShading(b *testing.B) {
	cfg := testutil.LoadTestConfigTB(b)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	// Setup: create canvas with a circle
	spritePath := filepath.Join(b.TempDir(), "bench-shading.aseprite")
	createScript := gen.CreateCanvas(64, 64, aseprite.ColorModeRGB, spritePath)
	_, err := client.ExecuteLua(ctx, createScript, "")
	if err != nil {
		b.Fatalf("CreateCanvas failed: %v", err)
	}
	defer os.Remove(spritePath)

	// Draw a circle to shade
	drawScript := gen.DrawCircle("Layer 1", 1, 32, 32, 24, aseprite.Color{R: 128, G: 128, B: 200, A: 255}, true, false)
	_, err = client.ExecuteLua(ctx, drawScript, spritePath)
	if err != nil {
		b.Fatalf("DrawCircle failed: %v", err)
	}

	palette := []string{"#1A2639", "#3D5A80", "#5B7FB0", "#8BA3C7"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		shadingScript := gen.ApplyShading("Layer 1", 1, 8, 8, 48, 48, palette, "top_left", 0.7, "smooth")
		_, err := client.ExecuteLua(ctx, shadingScript, spritePath)
		if err != nil {
			b.Fatalf("ApplyShading failed: %v", err)
		}
	}
}

func BenchmarkProfessional_SuggestAntialiasing(b *testing.B) {
	cfg := testutil.LoadTestConfigTB(b)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	// Setup: create canvas with jagged diagonal line once
	spritePath := filepath.Join(b.TempDir(), "bench-aa.aseprite")
	createScript := gen.CreateCanvas(64, 64, aseprite.ColorModeRGB, spritePath)
	_, err := client.ExecuteLua(ctx, createScript, "")
	if err != nil {
		b.Fatalf("CreateCanvas failed: %v", err)
	}
	defer os.Remove(spritePath)

	// Draw jagged diagonal (stair-step pattern)
	jaggedPixels := make([]aseprite.Pixel, 0)
	for i := 0; i < 5; i++ {
		for j := 0; j < 4; j++ {
			jaggedPixels = append(jaggedPixels, aseprite.Pixel{
				Point: aseprite.Point{X: 20 + i + j, Y: 10 + i},
				Color: aseprite.Color{R: 255, G: 0, B: 255, A: 255},
			})
		}
	}
	drawScript := gen.DrawPixels("Layer 1", 1, jaggedPixels, false)
	_, err = client.ExecuteLua(ctx, drawScript, spritePath)
	if err != nil {
		b.Fatalf("DrawPixels failed: %v", err)
	}

	// For benchmarking, we'll use the Lua approach similar to integration tests
	// Since suggestAntialiasing is complex with pixel reading, we'll benchmark
	// the GetPixels operation which is the bulk of the work
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pixelScript := gen.GetPixels("Layer 1", 1, 0, 0, 64, 64)
		_, err := client.ExecuteLua(ctx, pixelScript, spritePath)
		if err != nil {
			b.Fatalf("GetPixels failed: %v", err)
		}
	}
}

func BenchmarkProfessional_DownsampleImage_Small(b *testing.B) {
	cfg := testutil.LoadTestConfigTB(b)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 30*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	// Create a 256x256 test image
	sourcePath := filepath.Join(b.TempDir(), "bench-source-small.aseprite")
	createScript := gen.CreateCanvas(256, 256, aseprite.ColorModeRGB, sourcePath)
	_, err := client.ExecuteLua(ctx, createScript, "")
	if err != nil {
		b.Fatalf("CreateCanvas failed: %v", err)
	}
	defer os.Remove(sourcePath)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		outputPath := filepath.Join(b.TempDir(), "bench-downsampled-small.aseprite")
		downsampleScript := gen.DownsampleImage(sourcePath, outputPath, 64, 64)
		_, err := client.ExecuteLua(ctx, downsampleScript, "")
		if err != nil {
			b.Fatalf("DownsampleImage failed: %v", err)
		}
		os.Remove(outputPath)
	}
}

func BenchmarkProfessional_DownsampleImage_Large(b *testing.B) {
	cfg := testutil.LoadTestConfigTB(b)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 60*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	// Create a 1024x1024 test image
	sourcePath := filepath.Join(b.TempDir(), "bench-source-large.aseprite")
	createScript := gen.CreateCanvas(1024, 1024, aseprite.ColorModeRGB, sourcePath)
	_, err := client.ExecuteLua(ctx, createScript, "")
	if err != nil {
		b.Fatalf("CreateCanvas failed: %v", err)
	}
	defer os.Remove(sourcePath)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		outputPath := filepath.Join(b.TempDir(), "bench-downsampled-large.aseprite")
		downsampleScript := gen.DownsampleImage(sourcePath, outputPath, 128, 128)
		_, err := client.ExecuteLua(ctx, downsampleScript, "")
		if err != nil {
			b.Fatalf("DownsampleImage failed: %v", err)
		}
		os.Remove(outputPath)
	}
}

// End-to-end professional workflow benchmark
func BenchmarkProfessional_CompleteWorkflow(b *testing.B) {
	cfg := testutil.LoadTestConfigTB(b)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 60*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Step 1: Create 64x64 canvas
		spritePath := filepath.Join(b.TempDir(), "bench-pro-workflow.aseprite")
		createScript := gen.CreateCanvas(64, 64, aseprite.ColorModeRGB, spritePath)
		_, err := client.ExecuteLua(ctx, createScript, "")
		if err != nil {
			b.Fatalf("CreateCanvas failed: %v", err)
		}

		// Step 2: Apply dithered gradient background
		ditherScript := gen.DrawWithDither("Layer 1", 1, 0, 0, 64, 64, "#0A1628", "#3D5A80", "bayer_4x4", 0.3)
		_, err = client.ExecuteLua(ctx, ditherScript, spritePath)
		if err != nil {
			b.Fatalf("DrawWithDither failed: %v", err)
		}

		// Step 3: Draw and shade a circle
		drawScript := gen.DrawCircle("Layer 1", 1, 32, 32, 20, aseprite.Color{R: 200, G: 100, B: 150, A: 255}, true, false)
		_, err = client.ExecuteLua(ctx, drawScript, spritePath)
		if err != nil {
			b.Fatalf("DrawCircle failed: %v", err)
		}

		palette := []string{"#3B1725", "#8B5580", "#D4A5C0", "#FEF9A7"}
		shadingScript := gen.ApplyShading("Layer 1", 1, 12, 12, 40, 40, palette, "top_left", 0.7, "smooth")
		_, err = client.ExecuteLua(ctx, shadingScript, spritePath)
		if err != nil {
			b.Fatalf("ApplyShading failed: %v", err)
		}

		// Step 4: Export
		exportPath := filepath.Join(b.TempDir(), "bench-pro-output.png")
		exportScript := gen.ExportSprite(exportPath, 0)
		_, err = client.ExecuteLua(ctx, exportScript, spritePath)
		if err != nil {
			b.Fatalf("ExportSprite failed: %v", err)
		}

		os.Remove(spritePath)
		os.Remove(exportPath)
	}
}

func getBenchmark20Ops() []aseprite.BatchOp {
	return []aseprite.BatchOp{
		{Type: aseprite.BatchOpRectangle, LayerName: "Layer 1", FrameNumber: 1, X: 0, Y: 0, Width: 10, Height: 10, Color: aseprite.Color{R: 255, G: 0, B: 0, A: 255}, Filled: true},
		{Type: aseprite.BatchOpRectangle, LayerName: "Layer 1", FrameNumber: 1, X: 15, Y: 0, Width: 10, Height: 10, Color: aseprite.Color{R: 0, G: 255, B: 0, A: 255}, Filled: false},
		{Type: aseprite.BatchOpRectangle, LayerName: "Layer 1", FrameNumber: 1, X: 30, Y: 0, Width: 10, Height: 10, Color: aseprite.Color{R: 0, G: 0, B: 255, A: 255}, Filled: true},
		{Type: aseprite.BatchOpRectangle, LayerName: "Layer 1", FrameNumber: 1, X: 45, Y: 0, Width: 10, Height: 10, Color: aseprite.Color{R: 255, G: 255, B: 0, A: 255}, Filled: false},

		{Type: aseprite.BatchOpCircle, LayerName: "Layer 1", FrameNumber: 1, CenterX: 8, CenterY: 20, Radius: 6, Color: aseprite.Color{R: 255, G: 0, B: 255, A: 255}, Filled: true},
		{Type: aseprite.BatchOpCircle, LayerName: "Layer 1", FrameNumber: 1, CenterX: 24, CenterY: 20, Radius: 6, Color: aseprite.Color{R: 0, G: 255, B: 255, A: 255}, Filled: false},
		{Type: aseprite.BatchOpCircle, LayerName: "Layer 1", FrameNumber: 1, CenterX: 40, CenterY: 20, Radius: 6, Color: aseprite.Color{R: 128, G: 128, B: 128, A: 255}, Filled: true},
		{Type: aseprite.BatchOpCircle, LayerName: "Layer 1", FrameNumber: 1, CenterX: 56, CenterY: 20, Radius: 6, Color: aseprite.Color{R: 200, G: 100, B: 50, A: 255}, Filled: false},

		{Type: aseprite.BatchOpLine, LayerName: "Layer 1", FrameNumber: 1, X1: 0, Y1: 30, X2: 60, Y2: 30, Color: aseprite.Color{R: 255, G: 255, B: 255, A: 255}, Thickness: 1},
		{Type: aseprite.BatchOpLine, LayerName: "Layer 1", FrameNumber: 1, X1: 0, Y1: 35, X2: 60, Y2: 35, Color: aseprite.Color{R: 100, G: 200, B: 100, A: 255}, Thickness: 2},
		{Type: aseprite.BatchOpLine, LayerName: "Layer 1", FrameNumber: 1, X1: 0, Y1: 40, X2: 60, Y2: 40, Color: aseprite.Color{R: 200, G: 100, B: 200, A: 255}, Thickness: 1},
		{Type: aseprite.BatchOpLine, LayerName: "Layer 1", FrameNumber: 1, X1: 0, Y1: 45, X2: 60, Y2: 45, Color: aseprite.Color{R: 255, G: 150, B: 0, A: 255}, Thickness: 2},

		{Type: aseprite.BatchOpContour, LayerName: "Layer 1", FrameNumber: 1, Points: []aseprite.Point{{X: 0, Y: 50}, {X: 10, Y: 55}, {X: 20, Y: 50}}, Color: aseprite.Color{R: 255, G: 0, B: 0, A: 255}, Thickness: 1, Closed: false},
		{Type: aseprite.BatchOpContour, LayerName: "Layer 1", FrameNumber: 1, Points: []aseprite.Point{{X: 25, Y: 50}, {X: 35, Y: 55}, {X: 35, Y: 60}, {X: 25, Y: 60}}, Color: aseprite.Color{R: 0, G: 255, B: 0, A: 255}, Thickness: 1, Closed: true},
		{Type: aseprite.BatchOpContour, LayerName: "Layer 1", FrameNumber: 1, Points: []aseprite.Point{{X: 40, Y: 50}, {X: 50, Y: 55}, {X: 60, Y: 50}}, Color: aseprite.Color{R: 0, G: 0, B: 255, A: 255}, Thickness: 1, Closed: false},
		{Type: aseprite.BatchOpContour, LayerName: "Layer 1", FrameNumber: 1, Points: []aseprite.Point{{X: 40, Y: 55}, {X: 50, Y: 60}, {X: 60, Y: 55}}, Color: aseprite.Color{R: 255, G: 255, B: 0, A: 255}, Thickness: 1, Closed: false},

		{Type: aseprite.BatchOpPixels, LayerName: "Layer 1", FrameNumber: 1, Pixels: []aseprite.Pixel{{Point: aseprite.Point{X: 1, Y: 62}, Color: aseprite.Color{R: 255, G: 0, B: 0, A: 255}}}},
		{Type: aseprite.BatchOpPixels, LayerName: "Layer 1", FrameNumber: 1, Pixels: []aseprite.Pixel{{Point: aseprite.Point{X: 5, Y: 62}, Color: aseprite.Color{R: 0, G: 255, B: 0, A: 255}}}},
		{Type: aseprite.BatchOpPixels, LayerName: "Layer 1", FrameNumber: 1, Pixels: []aseprite.Pixel{{Point: aseprite.Point{X: 10, Y: 62}, Color: aseprite.Color{R: 0, G: 0, B: 255, A: 255}}}},
		{Type: aseprite.BatchOpPixels, LayerName: "Layer 1", FrameNumber: 1, Pixels: []aseprite.Pixel{{Point: aseprite.Point{X: 15, Y: 62}, Color: aseprite.Color{R: 255, G: 255, B: 255, A: 255}}}},
	}
}

// BenchmarkBatchDraw_Separate20Ops benchmarks executing 20 drawing operations as 20 separate Aseprite executions.
func BenchmarkBatchDraw_Separate20Ops(b *testing.B) {
	cfg := testutil.LoadTestConfigTB(b)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 60*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()
	ops := getBenchmark20Ops()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		spritePath := filepath.Join(b.TempDir(), "bench-separate-ops.aseprite")
		createScript := gen.CreateCanvas(64, 64, aseprite.ColorModeRGB, spritePath)
		if _, err := client.ExecuteLua(ctx, createScript, ""); err != nil {
			b.Fatalf("CreateCanvas failed: %v", err)
		}

		for _, op := range ops {
			var script string
			switch op.Type {
			case aseprite.BatchOpRectangle:
				script = gen.DrawRectangle(op.LayerName, op.FrameNumber, op.X, op.Y, op.Width, op.Height, op.Color, op.Filled, false)
			case aseprite.BatchOpCircle:
				script = gen.DrawCircle(op.LayerName, op.FrameNumber, op.CenterX, op.CenterY, op.Radius, op.Color, op.Filled, false)
			case aseprite.BatchOpLine:
				script = gen.DrawLine(op.LayerName, op.FrameNumber, op.X1, op.Y1, op.X2, op.Y2, op.Color, op.Thickness, false)
			case aseprite.BatchOpContour:
				script = gen.DrawContour(op.LayerName, op.FrameNumber, op.Points, op.Color, op.Thickness, op.Closed, false)
			case aseprite.BatchOpPixels:
				script = gen.DrawPixels(op.LayerName, op.FrameNumber, op.Pixels, false)
			}
			if _, err := client.ExecuteLua(ctx, script, spritePath); err != nil {
				b.Fatalf("Separate op execution failed: %v", err)
			}
		}

		os.Remove(spritePath)
	}
}

// BenchmarkBatchDraw_Batch20Ops benchmarks executing the same 20 drawing operations in a single BatchDraw execution.
func BenchmarkBatchDraw_Batch20Ops(b *testing.B) {
	cfg := testutil.LoadTestConfigTB(b)
	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, 60*time.Second)
	gen := aseprite.NewLuaGenerator()
	ctx := context.Background()
	ops := getBenchmark20Ops()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		spritePath := filepath.Join(b.TempDir(), "bench-batch-ops.aseprite")
		createScript := gen.CreateCanvas(64, 64, aseprite.ColorModeRGB, spritePath)
		if _, err := client.ExecuteLua(ctx, createScript, ""); err != nil {
			b.Fatalf("CreateCanvas failed: %v", err)
		}

		batchScript := gen.BatchDraw(ops, false)
		if _, err := client.ExecuteLua(ctx, batchScript, spritePath); err != nil {
			b.Fatalf("BatchDraw execution failed: %v", err)
		}

		os.Remove(spritePath)
	}
}
