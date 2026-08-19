package tools

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// writeFile is a test helper that creates a file with the given content.
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// ── Task 1.3 RED: TestGlobValidPattern ─────────────────────────────────────

func TestGlobValidPattern(t *testing.T) {
	root := t.TempDir()

	// Create two nested .go files.
	writeFile(t, root, "main.go", "package main\n")
	writeFile(t, root, "pkg/helper.go", "package pkg\n")

	tool := NewGlobTool(root)
	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"pattern":"**/*.go"}`))
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	var files []string
	if err := json.Unmarshal([]byte(raw), &files); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	// Expect both files returned sorted.
	want := []string{"main.go", "pkg/helper.go"}
	if len(files) != len(want) {
		t.Fatalf("got %d files, want %d: %v", len(files), len(want), files)
	}
	for i, f := range files {
		if f != want[i] {
			t.Errorf("files[%d] = %q, want %q", i, f, want[i])
		}
	}
}

// ── Task 1.5 RED: TestGlobNoMatches ────────────────────────────────────────

func TestGlobNoMatches(t *testing.T) {
	root := t.TempDir()

	// Create Go files — no .toml files exist.
	writeFile(t, root, "main.go", "package main\n")

	tool := NewGlobTool(root)
	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"pattern":"**/*.toml"}`))
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	var files []string
	if err := json.Unmarshal([]byte(raw), &files); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("got %d files, want 0: %v", len(files), files)
	}
}

// ── Task 1.6 RED: TestGlobPathEscapeRejected ───────────────────────────────

func TestGlobPathEscapeRejected(t *testing.T) {
	root := t.TempDir()

	// Create a file inside root.
	writeFile(t, root, "main.go", "package main\n")

	tool := NewGlobTool(root)
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"pattern":"*.go","path":"../../secret"}`))
	if err == nil {
		t.Fatal("expected error for path escaping workspace root, got nil")
	}

	var pathErr *PathConstraintError
	if !errors.As(err, &pathErr) {
		t.Errorf("expected PathConstraintError, got %T: %v", err, err)
	}

	// Verify no files were read outside root (only root/main.go exists).
	// The workspace confinement is enforced by resolvePath before any walk.
}

// ── TestGlobSkipGit ────────────────────────────────────────────────────────

func TestGlobSkipGit(t *testing.T) {
	root := t.TempDir()

	// Create files including inside .git.
	writeFile(t, root, "main.go", "package main\n")
	writeFile(t, root, ".git/objects/pack/data", "binary")
	writeFile(t, root, ".git/HEAD", "ref: refs/heads/main\n")

	tool := NewGlobTool(root)
	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"pattern":"**/*"}`))
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	var files []string
	if err := json.Unmarshal([]byte(raw), &files); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	// .git directory contents must be excluded.
	for _, f := range files {
		if f == ".git/HEAD" || f == ".git/objects/pack/data" {
			t.Errorf("should not include .git content, got %q", f)
		}
	}

	// Only main.go should be returned.
	if len(files) != 1 || files[0] != "main.go" {
		t.Errorf("got %v, want [main.go]", files)
	}
}
