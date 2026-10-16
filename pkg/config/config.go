// Package config provides configuration management for the Aseprite MCP server.
//
// Configuration is layered in the following priority order:
//  1. CLI flags / programmatic options
//  2. Environment variables (PIXEL_MCP_*)
//  3. Configuration file ($PIXEL_MCP_CONFIG, $XDG_CONFIG_HOME/pixel-mcp/config.json, or ~/.config/pixel-mcp/config.json)
//  4. System PATH auto-discovery for Aseprite binary
//  5. Built-in defaults
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config holds the Aseprite MCP server configuration.
type Config struct {
	// AsepritePath is the path to the Aseprite executable.
	AsepritePath string `json:"aseprite_path"`

	// TempDir is the directory for temporary Lua script files.
	// Defaults to /tmp/pixel-mcp if not specified.
	TempDir string `json:"temp_dir"`

	// Timeout is the maximum duration for Aseprite command execution.
	// Defaults to 30 seconds if not specified.
	Timeout time.Duration `json:"timeout"`

	// LogLevel is the logging verbosity level.
	// Valid values: "debug", "info", "warn", "error"
	// Defaults to "info" if not specified.
	LogLevel string `json:"log_level"`

	// LogFile is the optional path to a log file for persistent logging.
	// If empty, logs only go to stderr.
	LogFile string `json:"log_file"`

	// EnableTiming enables request tracking and operation timing for all tools.
	EnableTiming bool `json:"enable_timing"`
}

// Default configuration values.
const (
	DefaultTimeout  = 30 * time.Second
	DefaultLogLevel = "info"
)

// Options allows overriding configuration parameters directly from CLI flags.
type Options struct {
	ConfigFile   string
	AsepritePath string
	TempDir      string
	Timeout      time.Duration
	LogLevel     string
	LogFile      string
	EnableTiming *bool
}

// Load loads configuration from default or environment-configured sources.
func Load() (*Config, error) {
	return LoadWithOptions(Options{})
}

// LoadWithOptions loads configuration with optional CLI flag overrides.
func LoadWithOptions(opts Options) (*Config, error) {
	cfg := &Config{
		Timeout:  DefaultTimeout,
		LogLevel: DefaultLogLevel,
	}

	configPath := opts.ConfigFile
	if configPath == "" {
		if envConfig := os.Getenv("PIXEL_MCP_CONFIG"); envConfig != "" {
			configPath = envConfig
		} else {
			configPath = getConfigFilePath()
		}
	}

	// Try to load from config file
	err := cfg.loadFromPath(configPath)
	if err != nil {
		hasExplicitAseprite := opts.AsepritePath != "" || os.Getenv("PIXEL_MCP_ASEPRITE_PATH") != ""
		if !hasExplicitAseprite || opts.ConfigFile != "" {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("config file not found at %s: %w\nPlease create config file with aseprite_path configured", configPath, err)
			}
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	}

	// Apply Environment Variables (PIXEL_MCP_*)
	if envAse := os.Getenv("PIXEL_MCP_ASEPRITE_PATH"); envAse != "" {
		cfg.AsepritePath = envAse
	}
	if envTemp := os.Getenv("PIXEL_MCP_TEMP_DIR"); envTemp != "" {
		cfg.TempDir = envTemp
	}
	if envTimeout := os.Getenv("PIXEL_MCP_TIMEOUT"); envTimeout != "" {
		if sec, err := strconv.Atoi(envTimeout); err == nil && sec > 0 {
			cfg.Timeout = time.Duration(sec) * time.Second
		}
	}
	if envLog := os.Getenv("PIXEL_MCP_LOG_LEVEL"); envLog != "" {
		cfg.LogLevel = strings.ToLower(envLog)
	}
	if envLogFile := os.Getenv("PIXEL_MCP_LOG_FILE"); envLogFile != "" {
		cfg.LogFile = envLogFile
	}
	if envTiming := os.Getenv("PIXEL_MCP_ENABLE_TIMING"); envTiming != "" {
		if b, err := strconv.ParseBool(envTiming); err == nil {
			cfg.EnableTiming = b
		}
	}

	// Apply CLI flag overrides (highest priority)
	if opts.AsepritePath != "" {
		cfg.AsepritePath = opts.AsepritePath
	}
	if opts.TempDir != "" {
		cfg.TempDir = opts.TempDir
	}
	if opts.Timeout > 0 {
		cfg.Timeout = opts.Timeout
	}
	if opts.LogLevel != "" {
		cfg.LogLevel = opts.LogLevel
	}
	if opts.LogFile != "" {
		cfg.LogFile = opts.LogFile
	}
	if opts.EnableTiming != nil {
		cfg.EnableTiming = *opts.EnableTiming
	}

	// Set defaults and auto-discovery
	if err := cfg.setDefaults(); err != nil {
		return nil, fmt.Errorf("failed to set defaults: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func (c *Config) loadFromFile() error {
	return c.loadFromPath(getConfigFilePath())
}

func (c *Config) loadFromPath(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	var cj configJSON
	if err := json.Unmarshal(data, &cj); err != nil {
		return err
	}
	c.AsepritePath = cj.AsepritePath
	c.TempDir = cj.TempDir
	if cj.Timeout > 0 {
		c.Timeout = time.Duration(cj.Timeout) * time.Second
	}
	if cj.LogLevel != "" {
		c.LogLevel = cj.LogLevel
	}
	c.LogFile = cj.LogFile
	c.EnableTiming = cj.EnableTiming
	return nil
}

type configJSON struct {
	AsepritePath string `json:"aseprite_path"`
	TempDir      string `json:"temp_dir"`
	Timeout      int    `json:"timeout"`
	LogLevel     string `json:"log_level"`
	LogFile      string `json:"log_file"`
	EnableTiming bool   `json:"enable_timing"`
}

func (c *Config) setDefaults() error {
	if c.AsepritePath == "" {
		return fmt.Errorf("aseprite_path must be explicitly configured in config file")
	}

	// Set default temp directory
	if c.TempDir == "" {
		c.TempDir = filepath.Join(os.TempDir(), "pixel-mcp")
	}

	if err := os.MkdirAll(c.TempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}

	if c.LogLevel == "" {
		c.LogLevel = DefaultLogLevel
	}

	return nil
}

// Validate checks if the configuration is valid and usable.
func (c *Config) Validate() error {
	if _, err := os.Stat(c.AsepritePath); os.IsNotExist(err) {
		return fmt.Errorf("aseprite executable not found at %s", c.AsepritePath)
	}

	testFile := filepath.Join(c.TempDir, fmt.Sprintf(".test-%d", time.Now().UnixNano()))
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("temp directory is not writable: %w", err)
	}
	_ = os.Remove(testFile)

	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive, got %v", c.Timeout)
	}

	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s (valid: debug, info, warn, error)", c.LogLevel)
	}

	return nil
}

var getConfigFilePath = func() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "pixel-mcp", "config.json")
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "pixel-mcp", "config.json")
	}
	return filepath.Join(homeDir, ".config", "pixel-mcp", "config.json")
}
