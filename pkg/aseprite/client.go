package aseprite

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Client executes Aseprite commands and Lua scripts in batch mode.
// It provides a high-level interface for sprite manipulation through
// Aseprite's command-line interface and Lua scripting API.
//
// All operations execute with a configurable timeout and automatic
// temporary file cleanup. The client is safe for concurrent use from
// multiple goroutines.
type Client struct {
	execPath string
	tempDir  string
	timeout  time.Duration
}

// NewClient creates a new Aseprite client with the specified configuration.
//
// Parameters:
//   - execPath: absolute path to the Aseprite executable
//   - tempDir: directory for temporary script files
//   - timeout: maximum duration for command execution
//
// The client will create the temp directory if it doesn't exist.
// All Lua scripts are written to temp files with restricted permissions (0600)
// and automatically cleaned up after execution.
func NewClient(execPath, tempDir string, timeout time.Duration) *Client {
	return &Client{
		execPath: execPath,
		tempDir:  tempDir,
		timeout:  timeout,
	}
}

// ExecuteCommand runs an Aseprite command with the given arguments in batch mode.
//
// The command executes with the configured timeout. If the timeout is exceeded,
// the command is terminated and an error is returned.
//
// Returns the stdout output from Aseprite, or an error if execution fails.
// Stderr output is included in the error message for debugging.
func (c *Client) ExecuteCommand(ctx context.Context, args []string) (string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Build command
	cmd := exec.CommandContext(ctx, c.execPath, args...)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	err := cmd.Run()

	// Check for errors
	if err != nil {
		// Check if it was a timeout
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("aseprite command timed out after %v", c.timeout)
		}

		// Include stderr in error message
		if stderr.Len() > 0 {
			return "", fmt.Errorf("aseprite command failed: %w\nstderr: %s\nstdout: %s", err, stderr.String(), stdout.String())
		}

		return "", fmt.Errorf("aseprite command failed: %w\nstdout: %s", err, stdout.String())
	}

	return stdout.String(), nil
}

// ExecuteLua executes a Lua script in Aseprite batch mode.
//
// The script is written to a temporary file with restricted permissions (0600)
// and automatically cleaned up after execution. Script execution respects the
// configured timeout.
//
// Parameters:
//   - ctx: context for cancellation and timeout control
//   - script: Lua script code to execute
//   - spritePath: path to sprite file to open (empty string to create new sprite)
//
// If spritePath is non-empty, the sprite will be opened before running the script
// and the sprite file must exist. The script will have access to app.activeSprite.
//
// Returns the stdout output from Aseprite, or an error if execution fails.
func (c *Client) ExecuteLua(ctx context.Context, script string, spritePath string) (string, error) {
	// Create temporary script file
	scriptPath, cleanup, err := c.createTempScript(script)
	if err != nil {
		return "", fmt.Errorf("failed to create temp script: %w", err)
	}
	defer cleanup()

	// Build arguments
	args := []string{"--batch"}

	// Add sprite path if specified
	if spritePath != "" {
		// Serialize concurrent operations on the same sprite file
		unlock := GlobalPathLockManager.Lock(spritePath)
		defer unlock()

		// Verify sprite exists
		if _, err := os.Stat(spritePath); os.IsNotExist(err) {
			return "", fmt.Errorf("sprite file not found: %s", spritePath)
		}
		args = append(args, spritePath)
	}

	// Add script argument
	args = append(args, "--script", scriptPath)

	// Execute command
	return c.ExecuteCommand(ctx, args)
}

// GetVersion retrieves the Aseprite version string.
//
// Executes "aseprite --version" and parses the version from the output.
// The version string typically has the format "Aseprite 1.3.x".
//
// Returns the version string, or an error if the command fails or
// the version cannot be parsed from the output.
func (c *Client) GetVersion(ctx context.Context) (string, error) {
	output, err := c.ExecuteCommand(ctx, []string{"--version"})
	if err != nil {
		return "", err
	}

	// Parse version from output (format: "Aseprite 1.3.x")
	lines := strings.Split(output, "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}

	return "", fmt.Errorf("failed to parse version from output: %s", output)
}

// createTempScript creates a temporary Lua script file with restricted permissions.
//
// The script is written to a file in the temp directory with permissions 0600
// to prevent unauthorized access. The file has a .lua extension and a unique
// random name.
//
// Returns:
//   - scriptPath: absolute path to the created script file
//   - cleanup: function to remove the script file (always call this with defer)
//   - error: any error that occurred during file creation
//
// The cleanup function is safe to call multiple times and will not return an error.
func (c *Client) createTempScript(script string) (string, func(), error) {
	// Ensure temp directory exists
	if err := os.MkdirAll(c.tempDir, 0755); err != nil {
		return "", nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Create temp file with .lua extension
	tmpFile, err := os.CreateTemp(c.tempDir, "script-*.lua")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	scriptPath := tmpFile.Name()

	// Write script content
	if _, err := tmpFile.WriteString(script); err != nil {
		tmpFile.Close()
		os.Remove(scriptPath)
		return "", nil, fmt.Errorf("failed to write script: %w", err)
	}

	// Close file
	if err := tmpFile.Close(); err != nil {
		os.Remove(scriptPath)
		return "", nil, fmt.Errorf("failed to close temp file: %w", err)
	}

	// Return cleanup function
	cleanup := func() {
		os.Remove(scriptPath)
	}

	return scriptPath, cleanup, nil
}

// CleanupOldTempFiles removes temporary Lua script files older than the specified duration.
//
// This method scans the temp directory for files matching the pattern "script-*.lua"
// and removes those with modification times older than maxAge. Files are removed
// silently - errors accessing individual files are ignored.
//
// The temp directory itself is not removed, even if empty. If the directory doesn't
// exist, this method returns nil.
//
// This is useful for periodic cleanup of leftover temp files from crashed or
// interrupted operations.
func (c *Client) CleanupOldTempFiles(maxAge time.Duration) error {
	entries, err := os.ReadDir(c.tempDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory doesn't exist, nothing to clean
		}
		return fmt.Errorf("failed to read temp directory: %w", err)
	}

	now := time.Now()
	for _, entry := range entries {
		// Ignore directories
		if entry.IsDir() {
			continue
		}

		// Check if file matches our pattern (script-*.lua)
		if !strings.HasPrefix(entry.Name(), "script-") || !strings.HasSuffix(entry.Name(), ".lua") {
			continue
		}

		// Get file info
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Check age
		if now.Sub(info.ModTime()) > maxAge {
			filePath := filepath.Join(c.tempDir, entry.Name())
			os.Remove(filePath)
		}
	}

	return nil
}
