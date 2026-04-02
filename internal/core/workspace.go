package core

import (
	"path/filepath"
	"strings"
	"sync"
)

// Workspace enforces path boundaries for all tool operations.
type Workspace struct {
	mu   sync.RWMutex
	root string
	cwd  string
}

// NewWorkspace creates a workspace rooted at the given directory.
// The root is resolved to an absolute, symlink-evaluated path.
func NewWorkspace(root string) (*Workspace, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	// Try to eval symlinks; if dir doesn't exist, use abs directly
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		real = abs
	}
	return &Workspace{root: real, cwd: real}, nil
}

// Root returns the workspace root directory.
func (w *Workspace) Root() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.root
}

// CWD returns the current working directory within the workspace.
func (w *Workspace) CWD() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.cwd
}

// Resolve converts a relative or absolute path to a safe absolute path
// within the workspace boundary. Returns WorkspaceBoundaryError if the
// resolved path escapes the root.
func (w *Workspace) Resolve(path string) (string, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var abs string
	if filepath.IsAbs(path) {
		abs = filepath.Clean(path)
	} else {
		abs = filepath.Join(w.cwd, path)
	}

	// Try to resolve symlinks for existing files
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// File may not exist yet — resolve parent instead
		dir := filepath.Dir(abs)
		realDir, dirErr := filepath.EvalSymlinks(dir)
		if dirErr != nil {
			// Parent doesn't exist either — use cleaned path
			real = filepath.Clean(abs)
		} else {
			real = filepath.Join(realDir, filepath.Base(abs))
		}
	}

	// Boundary check: real must be within root (or equal to root)
	if !strings.HasPrefix(real, w.root) {
		return "", &WorkspaceBoundaryError{Path: path, Root: w.root}
	}

	return real, nil
}

// SetCWD changes the working directory within the workspace.
func (w *Workspace) SetCWD(dir string) error {
	resolved, err := w.Resolve(dir)
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.cwd = resolved
	return nil
}
