package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBinaryName(t *testing.T) {
	tests := []struct {
		name   string
		goos   string
		goarch string
		want   string
	}{
		{"linux amd64", "linux", "amd64", "pixel-mcp-linux-amd64"},
		{"linux arm64", "linux", "arm64", "pixel-mcp-linux-arm64"},
		{"darwin amd64", "darwin", "amd64", "pixel-mcp-darwin-amd64"},
		{"darwin arm64", "darwin", "arm64", "pixel-mcp-darwin-arm64"},
		{"windows amd64 gets exe suffix", "windows", "amd64", "pixel-mcp-windows-amd64.exe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := binaryName(tt.goos, tt.goarch)
			if got != tt.want {
				t.Errorf("binaryName(%q, %q) = %q, want %q", tt.goos, tt.goarch, got, tt.want)
			}
		})
	}
}

// stubSource is a minimal program standing in for the real pixel-mcp binary.
// It prints its arguments, echoes stdin to stdout, and exits with code 3 when
// its first argument is "fail" so tests can verify exit code propagation.
const stubSource = `package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "fail" {
		os.Exit(3)
	}
	fmt.Printf("args:%v\n", os.Args[1:])
	if _, err := io.Copy(os.Stdout, os.Stdin); err != nil {
		os.Exit(1)
	}
}
`

// buildLauncher compiles the launcher into dir and returns its path.
func buildLauncher(t *testing.T, dir string) string {
	t.Helper()
	name := "launcher"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(dir, name)
	cmd := exec.Command("go", "build", "-o", path, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build launcher: %v\n%s", err, out)
	}
	return path
}

// buildStub compiles the stub target binary as pixel-mcp-{os}-{arch}[.exe]
// into dir.
func buildStub(t *testing.T, dir string) {
	t.Helper()
	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "main.go")
	if err := os.WriteFile(src, []byte(stubSource), 0o644); err != nil {
		t.Fatalf("failed to write stub source: %v", err)
	}
	target := filepath.Join(dir, binaryName(runtime.GOOS, runtime.GOARCH))
	cmd := exec.Command("go", "build", "-o", target, src)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build stub: %v\n%s", err, out)
	}
}

func TestLauncherRunsPlatformBinary(t *testing.T) {
	dir := t.TempDir()
	launcher := buildLauncher(t, dir)
	buildStub(t, dir)

	cmd := exec.Command(launcher, "--flag", "value")
	cmd.Stdin = strings.NewReader("hello over stdin")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("launcher failed: %v\nstderr: %s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "args:[--flag value]") {
		t.Errorf("arguments not passed through, stdout: %q", out)
	}
	if !strings.Contains(out, "hello over stdin") {
		t.Errorf("stdin not passed through, stdout: %q", out)
	}
}

func TestLauncherPropagatesExitCode(t *testing.T) {
	dir := t.TempDir()
	launcher := buildLauncher(t, dir)
	buildStub(t, dir)

	cmd := exec.Command(launcher, "fail")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit, got success")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %v", err)
	}
	if code := exitErr.ExitCode(); code != 3 {
		t.Errorf("exit code = %d, want 3", code)
	}
}

func TestLauncherReportsMissingBinary(t *testing.T) {
	dir := t.TempDir()
	launcher := buildLauncher(t, dir)
	// No stub built — the platform binary is missing.

	cmd := exec.Command(launcher)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit when platform binary is missing")
	}
	want := binaryName(runtime.GOOS, runtime.GOARCH)
	if !strings.Contains(stderr.String(), want) {
		t.Errorf("stderr should name the missing binary %q, got: %q", want, stderr.String())
	}
}

func TestLauncherResolvesSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on Windows")
	}
	dir := t.TempDir()
	launcher := buildLauncher(t, dir)
	buildStub(t, dir)

	// Invoke the launcher through a symlink from another directory; it must
	// resolve its real location to find the sibling binary.
	linkDir := t.TempDir()
	link := filepath.Join(linkDir, "pixel-mcp")
	if err := os.Symlink(launcher, link); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	cmd := exec.Command(link)
	cmd.Stdin = strings.NewReader("")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("launcher via symlink failed: %v\nstderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "args:[]") {
		t.Errorf("expected stub output, got: %q", stdout.String())
	}
}
