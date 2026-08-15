package tools

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// argsFor marshals v into the raw JSON arguments a tool Execute call receives.
func argsFor(t *testing.T, v any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	return raw
}

// TestReadFileExisting covers REQ-TOOLS-1 "Read existing file": read_file
// returns the full text content of a file inside the workspace root.
func TestReadFileExisting(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "notes.md"), []byte("hello file"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	tool := NewReadFile(root)

	got, err := tool.Execute(context.Background(), argsFor(t, map[string]string{"path": "notes.md"}))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if got != "hello file" {
		t.Errorf("content = %q, want %q", got, "hello file")
	}
}

// TestReadFileNestedAndAbsolute triangulates the happy path with a nested
// relative path and an absolute path inside the root.
func TestReadFileNestedAndAbsolute(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sub, "deep.txt"), []byte("deep"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	tool := NewReadFile(root)

	if got, err := tool.Execute(context.Background(), argsFor(t, map[string]string{"path": filepath.Join("sub", "deep.txt")})); err != nil || got != "deep" {
		t.Errorf("nested read = %q, %v; want %q, nil error", got, err, "deep")
	}
	abs := filepath.Join(root, "sub", "deep.txt")
	if got, err := tool.Execute(context.Background(), argsFor(t, map[string]string{"path": abs})); err != nil || got != "deep" {
		t.Errorf("absolute read = %q, %v; want %q, nil error", got, err, "deep")
	}
}

// TestReadFileMissing covers REQ-TOOLS-1 "Missing file": the error identifies
// the missing path.
func TestReadFileMissing(t *testing.T) {
	tool := NewReadFile(t.TempDir())

	_, err := tool.Execute(context.Background(), argsFor(t, map[string]string{"path": "nope.md"}))
	if err == nil {
		t.Fatal("Execute returned nil error, want missing-file error")
	}
	if !strings.Contains(err.Error(), "nope.md") {
		t.Errorf("error %q does not identify the missing path", err)
	}
}

// TestReadFileEscapeRejected covers REQ-TOOLS-1 "Path escape rejected": the
// tool returns a path-constraint error and no content is read from outside
// the root.
func TestReadFileEscapeRejected(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(filepath.Dir(root), "kui-secret.txt")
	if err := os.WriteFile(outside, []byte("top secret"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(outside) })
	tool := NewReadFile(root)

	result, err := tool.Execute(context.Background(), argsFor(t, map[string]string{"path": "../kui-secret.txt"}))
	var constraint *PathConstraintError
	if !errors.As(err, &constraint) {
		t.Fatalf("error = %v, want *PathConstraintError (no file outside the root may be read)", err)
	}
	if result != "" {
		t.Errorf("result = %q, want empty on rejection", result)
	}
	if strings.Contains(err.Error(), "top secret") {
		t.Errorf("error leaks file content: %q", err)
	}
}

// TestReadFileInvalidArguments rejects malformed arguments instead of
// touching the file system.
func TestReadFileInvalidArguments(t *testing.T) {
	tool := NewReadFile(t.TempDir())

	if _, err := tool.Execute(context.Background(), json.RawMessage(`not json`)); err == nil {
		t.Error("Execute accepted invalid JSON arguments")
	}
	if _, err := tool.Execute(context.Background(), argsFor(t, map[string]string{"path": ""})); err == nil {
		t.Error("Execute accepted an empty path")
	}
}
