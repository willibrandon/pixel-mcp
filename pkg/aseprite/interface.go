package aseprite

import "context"

// ClientInterface defines the methods exposed by the Aseprite execution client.
// This interface allows mock implementations for unit testing without invoking the real Aseprite binary.
type ClientInterface interface {
	ExecuteCommand(ctx context.Context, args []string) (string, error)
	ExecuteLua(ctx context.Context, script string, spritePath string) (string, error)
	GetVersion(ctx context.Context) (string, error)
	ExportPreview(ctx context.Context, spritePath string, scale int, frame int, layerName string) ([]byte, error)
}

// Compile-time check that Client implements ClientInterface
var _ ClientInterface = (*Client)(nil)
