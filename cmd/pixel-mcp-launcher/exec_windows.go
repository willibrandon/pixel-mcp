//go:build windows

package main

import (
	"errors"
	"os"
	"os/exec"
)

// execBinary runs the target binary with stdio passed through and exits the
// launcher with the child's exit code. Windows has no execve equivalent, so
// the launcher stays alive as the parent for the server's lifetime.
func execBinary(target string, args []string) error {
	cmd := exec.Command(target, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}
	if err != nil {
		return err
	}
	os.Exit(0)
	return nil
}
