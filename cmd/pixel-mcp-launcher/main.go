// Command pixel-mcp-launcher is a cross-platform launcher for the pixel-mcp
// MCP server. It detects the current OS and architecture at runtime, locates
// the matching pixel-mcp-{os}-{arch}[.exe] binary in its own directory, and
// runs it with all arguments and standard streams passed through.
//
// It exists so that pixel-plugin's .mcp.json can point at a single native
// executable instead of a bash wrapper script, which is not executable on
// native Windows (see willibrandon/pixel-mcp#18).
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// binaryName returns the platform-specific pixel-mcp binary filename for the
// given OS and architecture, e.g. "pixel-mcp-linux-amd64" or
// "pixel-mcp-windows-amd64.exe".
func binaryName(goos, goarch string) string {
	name := fmt.Sprintf("pixel-mcp-%s-%s", goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

// run locates the platform binary next to the launcher and executes it with
// the given arguments. The launcher's own path is resolved through symlinks so
// the sibling binary is found relative to the real install location.
func run(args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine launcher location: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("cannot resolve launcher location: %w", err)
	}

	target := filepath.Join(filepath.Dir(resolved), binaryName(runtime.GOOS, runtime.GOARCH))
	if _, err := os.Stat(target); err != nil {
		return fmt.Errorf("platform binary %s not found next to launcher: %w", filepath.Base(target), err)
	}

	return execBinary(target, args)
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "pixel-mcp-launcher: %v\n", err)
		os.Exit(1)
	}
}
