package tools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestWriteFileCreate covers REQ-TOOLS-2 "Create new file": the file is
// created with the given content and the tool reports the written path.
func TestWriteFileCreate(t *testing.T) {
	root := t.TempDir()
	tool := NewWriteFile(root)

	reported, err := tool.Execute(context.Background(), argsFor(t, map[string]string{"path": "new.txt", "content": "hello"}))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if reported != filepath.Join(root, "new.txt") {
		t.Errorf("reported path = %q, want %q", reported, filepath.Join(root, "new.txt"))
	}
	data, err := os.ReadFile(reported)
	if err != nil {
		t.Fatalf("reported path is not readable: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("file content = %q, want %q", data, "hello")
	}
}

// TestWriteFileOverwrite covers REQ-TOOLS-2 "Overwrite existing file": the
// file ends up with the new content.
func TestWriteFileOverwrite(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "notes.md"), []byte("old content"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	tool := NewWriteFile(root)

	if _, err := tool.Execute(context.Background(), argsFor(t, map[string]string{"path": "notes.md", "content": "new content"})); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "notes.md"))
	if err != nil {
		t.Fatalf("read back failed: %v", err)
	}
	if string(data) != "new content" {
		t.Errorf("file content = %q, want %q", data, "new content")
	}
}

// TestWriteFileCreatesParentDirectories triangulates the create path with
// missing intermediate directories inside the root.
func TestWriteFileCreatesParentDirectories(t *testing.T) {
	root := t.TempDir()
	tool := NewWriteFile(root)

	_, err := tool.Execute(context.Background(), argsFor(t, map[string]string{"path": filepath.Join("a", "b", "file.txt"), "content": "x"}))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "a", "b", "file.txt"))
	if err != nil {
		t.Fatalf("nested file was not created: %v", err)
	}
	if string(data) != "x" {
		t.Errorf("file content = %q, want %q", data, "x")
	}
}

// TestWriteFileEscapeRejected covers REQ-TOOLS-2 "Path escape rejected": the
// tool returns a path-constraint error and no file is written outside the
// root.
func TestWriteFileEscapeRejected(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(filepath.Dir(root), "kui-evil.txt")
	tool := NewWriteFile(root)

	_, err := tool.Execute(context.Background(), argsFor(t, map[string]string{"path": "../kui-evil.txt", "content": "evil"}))
	var constraint *PathConstraintError
	if !errors.As(err, &constraint) {
		t.Fatalf("error = %v, want *PathConstraintError (no file may be written outside root)", err)
	}
	if _, statErr := os.Stat(outside); !os.IsNotExist(statErr) {
		t.Errorf("file was written outside the root (stat error: %v)", statErr)
	}
	t.Cleanup(func() { _ = os.Remove(outside) })
}

// TestWriteFileSymlinkDirEscape rejects writes that would land outside the
// root through a symlinked directory.
func TestWriteFileSymlinkDirEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	tool := NewWriteFile(root)

	_, err := tool.Execute(context.Background(), argsFor(t, map[string]string{"path": filepath.Join("link", "evil.txt"), "content": "evil"}))
	var constraint *PathConstraintError
	if !errors.As(err, &constraint) {
		t.Fatalf("error = %v, want *PathConstraintError", err)
	}
	if _, statErr := os.Stat(filepath.Join(outside, "evil.txt")); !os.IsNotExist(statErr) {
		t.Errorf("file was written through the symlink (stat error: %v)", statErr)
	}
}
