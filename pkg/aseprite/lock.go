package aseprite

import (
	"path/filepath"
	"sync"
)

// PathLockManager manages per-file mutex locks to serialize concurrent operations on the same sprite file.
type PathLockManager struct {
	mu    sync.Mutex
	locks map[string]*refLock
}

type refLock struct {
	mu       sync.Mutex
	refCount int
}

// GlobalPathLockManager is the default path lock manager used across Aseprite client invocations.
var GlobalPathLockManager = NewPathLockManager()

// NewPathLockManager creates a new PathLockManager instance.
func NewPathLockManager() *PathLockManager {
	return &PathLockManager{
		locks: make(map[string]*refLock),
	}
}

// Lock acquires a mutex lock for the specified file path and returns a release function.
// Clean paths are used to normalize relative/redundant path representations.
func (m *PathLockManager) Lock(path string) func() {
	if path == "" {
		return func() {}
	}
	cleanPath := filepath.Clean(path)

	m.mu.Lock()
	l, exists := m.locks[cleanPath]
	if !exists {
		l = &refLock{}
		m.locks[cleanPath] = l
	}
	l.refCount++
	m.mu.Unlock()

	l.mu.Lock()

	return func() {
		l.mu.Unlock()

		m.mu.Lock()
		l.refCount--
		if l.refCount <= 0 {
			delete(m.locks, cleanPath)
		}
		m.mu.Unlock()
	}
}
