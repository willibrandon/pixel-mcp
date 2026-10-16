package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/willibrandon/mtlog"
	"github.com/willibrandon/mtlog/core"
	"github.com/willibrandon/mtlog/sinks"
	"github.com/willibrandon/pixel-mcp/pkg/aseprite"
	"github.com/willibrandon/pixel-mcp/pkg/config"
	"github.com/willibrandon/pixel-mcp/pkg/server"
)

var (
	// Version information (set via ldflags during build)
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	// Parse command-line flags
	var (
		showVersion   = flag.Bool("version", false, "Show version information")
		showHealth    = flag.Bool("health", false, "Check health and exit")
		debugMode     = flag.Bool("debug", false, "Enable debug logging")
		configFile    = flag.String("config", "", "Path to JSON configuration file")
		asepritePath  = flag.String("aseprite-path", "", "Path to Aseprite binary")
		tempDir       = flag.String("temp-dir", "", "Directory for temporary files")
		timeoutSec    = flag.Int("timeout", 0, "Execution timeout in seconds")
		logLevel      = flag.String("log-level", "", "Log level (debug, info, warn, error)")
		logFile       = flag.String("log-file", "", "Log file path")
		enableTiming  = flag.Bool("enable-timing", false, "Enable request timing")
	)
	flag.Parse()

	// Show version and exit
	if *showVersion {
		fmt.Printf("pixel-mcp version %s (built %s)\n", Version, BuildTime)
		os.Exit(0)
	}

	opts := config.Options{
		ConfigFile:   *configFile,
		AsepritePath: *asepritePath,
		TempDir:      *tempDir,
		LogLevel:     *logLevel,
		LogFile:      *logFile,
	}
	if *timeoutSec > 0 {
		opts.Timeout = time.Duration(*timeoutSec) * time.Second
	}
	if flag.Lookup("enable-timing") != nil && *enableTiming {
		opts.EnableTiming = enableTiming
	}

	// Load configuration
	cfg, err := config.LoadWithOptions(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nPlease configure ~/.config/pixel-mcp/config.json, use --config / --aseprite-path, or ensure aseprite is in PATH.\n")
		os.Exit(1)
	}

	// Override log level if debug mode enabled
	if *debugMode {
		cfg.LogLevel = "debug"
	}

	// Initialize logger
	logger := createLogger(cfg.LogLevel, cfg.LogFile)
	defer closeLogger(logger)

	// Health check mode
	if *showHealth {
		exitCode := performHealthCheck(cfg, logger)
		closeLogger(logger)
		os.Exit(exitCode)
	}

	// Log startup
	logger.Information("Starting Aseprite MCP Server version {Version} (built {BuildTime})", Version, BuildTime)
	logger.Debug("Configuration loaded: {@Config}", cfg)

	// Create server
	srv, err := server.New(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create server: {Error}", err)
		closeLogger(logger)
		os.Exit(1)
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Run(ctx)
	}()

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		logger.Information("Received shutdown signal: {Signal}", sig)
		cancel()
		select {
		case err := <-errChan:
			if err != nil && err != context.Canceled {
				logger.Error("Server shutdown with error: {Error}", err)
			}
		case <-time.After(2 * time.Second):
			logger.Warning("Server shutdown timed out")
		}
	case err := <-errChan:
		if err != nil {
			logger.Error("Server error: {Error}", err)
			closeLogger(logger)
			os.Exit(1)
		}
	}

	logger.Information("Server stopped")
}

// closeLogger closes the logger and flushes any buffered logs.
func closeLogger(logger core.Logger) {
	if closer, ok := logger.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
}

// createLogger creates a configured logger instance.
func createLogger(logLevel string, logFilePath string) core.Logger {
	template := "[${Timestamp:HH:mm:ss} ${Level:u3}] {SourceContext}: ${Message}"
	if logLevel == "debug" {
		template = "[${Timestamp:HH:mm:ss} ${Level:u3}] {SourceContext} [{MachineName}/{ProcessName}:{ProcessId}]: ${Message}"
	}
	consoleSink, err := sinks.NewConsoleSinkWithTemplateAndTheme(
		template,
		sinks.LiterateTheme(),
	)
	if err != nil {
		consoleSink = sinks.NewConsoleSink()
	}
	consoleSink.SetOutput(os.Stderr)

	var opts []mtlog.Option
	opts = append(opts, mtlog.WithSink(consoleSink))

	if logFilePath != "" {
		fileSink, err := sinks.NewRollingFileSink(sinks.RollingFileOptions{
			FilePath:        logFilePath,
			MaxFileSize:     100 * 1024 * 1024,
			RollingInterval: sinks.RollingIntervalDaily,
			RetainFileCount: 7,
			BufferSize:      1,
		})
		if err == nil {
			opts = append(opts, mtlog.WithSink(fileSink))
		}
	}

	switch logLevel {
	case "debug":
		opts = append(opts, mtlog.Debug())
	case "info":
		opts = append(opts, mtlog.Information())
	case "warn":
		opts = append(opts, mtlog.Warning())
	case "error":
		opts = append(opts, mtlog.Error())
	default:
		opts = append(opts, mtlog.Information())
	}

	return mtlog.New(opts...)
}

func performHealthCheck(cfg *config.Config, logger core.Logger) int {
	fmt.Println("Running pixel-mcp health checks...")

	fmt.Printf("[1/3] Checking Aseprite executable at: %s\n", cfg.AsepritePath)
	if _, err := os.Stat(cfg.AsepritePath); os.IsNotExist(err) {
		fmt.Printf("      FAIL: Aseprite not found at %s\n", cfg.AsepritePath)
		return 1
	}

	client := aseprite.NewClient(cfg.AsepritePath, cfg.TempDir, cfg.Timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	version, err := client.GetVersion(ctx)
	if err != nil {
		fmt.Printf("      FAIL: Failed to run Aseprite: %v\n", err)
		return 1
	}
	fmt.Printf("      PASS: Found %s\n", version)

	fmt.Printf("[2/3] Checking temp directory at: %s\n", cfg.TempDir)
	if err := os.MkdirAll(cfg.TempDir, 0755); err != nil {
		fmt.Printf("      FAIL: Failed to create temp directory: %v\n", err)
		return 1
	}
	testFile := fmt.Sprintf("%s/.health-%d", cfg.TempDir, time.Now().UnixNano())
	if err := os.WriteFile(testFile, []byte("ok"), 0644); err != nil {
		fmt.Printf("      FAIL: Temp directory is not writable: %v\n", err)
		return 1
	}
	_ = os.Remove(testFile)
	fmt.Println("      PASS: Temp directory is writable")

	fmt.Printf("[3/3] Checking configuration parameters...\n")
	fmt.Printf("      PASS: Timeout = %v, LogLevel = %s\n", cfg.Timeout, cfg.LogLevel)

	fmt.Println("\nAll health checks passed successfully!")
	return 0
}
