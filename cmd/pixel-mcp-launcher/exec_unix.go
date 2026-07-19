//go:build !windows

package main

import (
	"os"
	"syscall"
)

// execBinary replaces the launcher process with the target binary via execve,
// so stdio, signals, and the exit code all belong to the server process
// directly. Only returns on error.
func execBinary(target string, args []string) error {
	argv := append([]string{target}, args...)
	return syscall.Exec(target, argv, os.Environ())
}
