package core

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceResolve_InBounds(t *testing.T) {
	dir := t.TempDir()
	ws, err := NewWorkspace(dir)
	if err != nil {
		t.Fatalf("NewWorkspace: %v", err)
	}

	// Create a file inside the workspace
	testFile := filepath.Join(dir, "test.go")
	os.WriteFile(testFile, []byte("package main"), 0644)

	resolved, err := ws.Resolve("test.go")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved != testFile {
		t.Errorf("expected %q, got %q", testFile, resolved)
	}
}

func TestWorkspaceResolve_OutOfBounds(t *testing.T) {
	dir := t.TempDir()
	ws, err := NewWorkspace(dir)
	if err != nil {
		t.Fatalf("NewWorkspace: %v", err)
	}

	_, err = ws.Resolve("../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for out-of-bounds path")
	}
	if !isWorkspaceBoundaryError(err) {
		t.Errorf("expected WorkspaceBoundaryError, got %T: %v", err, err)
	}
}

func TestWorkspaceResolve_Absolute(t *testing.T) {
	dir := t.TempDir()
	ws, err := NewWorkspace(dir)
	if err != nil {
		t.Fatalf("NewWorkspace: %v", err)
	}

	inFile := filepath.Join(dir, "ok.txt")
	os.WriteFile(inFile, []byte("ok"), 0644)

	resolved, err := ws.Resolve(inFile)
	if err != nil {
		t.Fatalf("Resolve absolute in-bounds: %v", err)
	}
	if resolved != inFile {
		t.Errorf("expected %q, got %q", inFile, resolved)
	}
}

func TestWorkspaceResolve_NewFile(t *testing.T) {
	dir := t.TempDir()
	ws, err := NewWorkspace(dir)
	if err != nil {
		t.Fatalf("NewWorkspace: %v", err)
	}

	// File doesn't exist yet — should still resolve if parent is in bounds
	resolved, err := ws.Resolve("newfile.txt")
	if err != nil {
		t.Fatalf("Resolve new file: %v", err)
	}
	expected := filepath.Join(dir, "newfile.txt")
	if resolved != expected {
		t.Errorf("expected %q, got %q", expected, resolved)
	}
}

func TestWorkspaceRoot(t *testing.T) {
	dir := t.TempDir()
	ws, err := NewWorkspace(dir)
	if err != nil {
		t.Fatalf("NewWorkspace: %v", err)
	}
	if ws.Root() == "" {
		t.Error("expected non-empty root")
	}
}

func isWorkspaceBoundaryError(err error) bool {
	var wbe *WorkspaceBoundaryError
	return errors.As(err, &wbe)
}
