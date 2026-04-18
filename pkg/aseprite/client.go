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

type Client struct {
	execPath string
	tempDir  string
	timeout  time.Duration
}

func NewClient(execPath, tempDir string, timeout time.Duration) *Client {
	return &Client{
		execPath: execPath,
		tempDir:  tempDir,
		timeout:  timeout,
	}
}

func (c *Client) ExecuteCommand(ctx context.Context, args []string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	needBatch := true
	for _, a := range args {
		if a == "--batch" || a == "-b" {
			needBatch = false
			break
		}
	}
	if needBatch {
		args = append([]string{"--batch"}, args...)
	}

	cmd := exec.CommandContext(ctx, c.execPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("aseprite command timed out after %v", c.timeout)
		}

		if stderr.Len() > 0 {
			return "", fmt.Errorf("aseprite command failed: %w\nstderr: %s\nstdout: %s", err, stderr.String(), stdout.String())
		}

		return "", fmt.Errorf("aseprite command failed: %w\nstdout: %s", err, stdout.String())
	}

	return stdout.String(), nil
}

func (c *Client) ExecuteLua(ctx context.Context, script string, spritePath string) (string, error) {
	scriptPath, cleanup, err := c.createTempScript(script)
	if err != nil {
		return "", fmt.Errorf("failed to create temp script: %w", err)
	}
	defer cleanup()

	args := []string{"--batch"}

	if spritePath != "" {
		if _, err := os.Stat(spritePath); os.IsNotExist(err) {
			return "", fmt.Errorf("sprite file not found: %s", spritePath)
		}
		args = append(args, spritePath)
	}

	args = append(args, "--script", scriptPath)

	return c.ExecuteCommand(ctx, args)
}

func (c *Client) GetVersion(ctx context.Context) (string, error) {
	output, err := c.ExecuteCommand(ctx, []string{"--version"})
	if err != nil {
		return "", err
	}

	lines := strings.Split(output, "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}

	return "", fmt.Errorf("failed to parse version from output: %s", output)
}

func (c *Client) createTempScript(script string) (string, func(), error) {
	if err := os.MkdirAll(c.tempDir, 0755); err != nil {
		return "", nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	tmpFile, err := os.CreateTemp(c.tempDir, "script-*.lua")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	scriptPath := tmpFile.Name()

	if _, err := tmpFile.WriteString(script); err != nil {
		tmpFile.Close()
		os.Remove(scriptPath)
		return "", nil, fmt.Errorf("failed to write script: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(scriptPath)
		return "", nil, fmt.Errorf("failed to close temp file: %w", err)
	}

	cleanup := func() {
		os.Remove(scriptPath)
	}

	return scriptPath, cleanup, nil
}

func (c *Client) CleanupOldTempFiles(maxAge time.Duration) error {
	entries, err := os.ReadDir(c.tempDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read temp directory: %w", err)
	}

	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasPrefix(entry.Name(), "script-") || !strings.HasSuffix(entry.Name(), ".lua") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if now.Sub(info.ModTime()) > maxAge {
			filePath := filepath.Join(c.tempDir, entry.Name())
			os.Remove(filePath)
		}
	}

	return nil
}
