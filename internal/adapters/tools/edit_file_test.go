package tools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFixture(t *testing.T, root, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
}

// TestEditFileReplace covers REQ-TOOLS-8 "Exact replace with unified diff".
func TestEditFileReplace(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "notes.md", "hello world\n")
	tool := NewEditFile(root)

	res, err := tool.Execute(context.Background(), argsFor(t, map[string]any{"path": "notes.md", "old_text": "world", "new_text": "kui"}))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.HasPrefix(res, "Successfully replaced 1 block(s) in notes.md.") {
		t.Errorf("result missing success line: %q", res)
	}
	if !strings.Contains(res, "diff --git") || !strings.Contains(res, "-hello world") || !strings.Contains(res, "+hello kui") {
		t.Errorf("result missing unified diff: %q", res)
	}
	data, err := os.ReadFile(filepath.Join(root, "notes.md"))
	if err != nil {
		t.Fatalf("read back failed: %v", err)
	}
	if string(data) != "hello kui\n" {
		t.Errorf("file content = %q, want %q", data, "hello kui\n")
	}
}

// TestEditFileNoMatch covers REQ-TOOLS-8 "No match returns guidance".
func TestEditFileNoMatch(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "notes.md", "hello\n")
	tool := NewEditFile(root)

	_, err := tool.Execute(context.Background(), argsFor(t, map[string]any{"path": "notes.md", "old_text": "bye", "new_text": "x"}))
	if err == nil || !strings.Contains(err.Error(), "0 matches") || !strings.Contains(err.Error(), "read_file") {
		t.Fatalf("error = %v, want 0-match guidance mentioning read_file", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "notes.md"))
	if string(data) != "hello\n" {
		t.Errorf("file was modified on no-match: %q", data)
	}
}

// TestEditFileAmbiguous covers REQ-TOOLS-8 "Ambiguous multi-match".
func TestEditFileAmbiguous(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "a.txt", "foo one\nfoo two\n")
	tool := NewEditFile(root)

	_, err := tool.Execute(context.Background(), argsFor(t, map[string]any{"path": "a.txt", "old_text": "foo", "new_text": "bar"}))
	if err == nil || !strings.Contains(err.Error(), "2 regions") || !strings.Contains(err.Error(), "occurrence") {
		t.Fatalf("error = %v, want ambiguity error with count + occurrence hint", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "a.txt"))
	if string(data) != "foo one\nfoo two\n" {
		t.Errorf("file was modified on ambiguous match: %q", data)
	}
}

// TestEditFileOccurrence covers REQ-TOOLS-8 "Occurrence selects Nth match".
func TestEditFileOccurrence(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "a.txt", "foo one\nfoo two\n")
	tool := NewEditFile(root)

	if _, err := tool.Execute(context.Background(), argsFor(t, map[string]any{"path": "a.txt", "old_text": "foo", "new_text": "bar", "occurrence": 2})); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "a.txt"))
	if string(data) != "foo one\nbar two\n" {
		t.Errorf("file content = %q, want second occurrence replaced", data)
	}
}

// TestEditFileOccurrenceOutOfRange covers REQ-TOOLS-8 "Occurrence out of range".
func TestEditFileOccurrenceOutOfRange(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "a.txt", "foo\n")
	tool := NewEditFile(root)

	_, err := tool.Execute(context.Background(), argsFor(t, map[string]any{"path": "a.txt", "old_text": "foo", "new_text": "bar", "occurrence": 3}))
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("error = %v, want range error", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "a.txt"))
	if string(data) != "foo\n" {
		t.Errorf("file was modified on out-of-range occurrence: %q", data)
	}
}

// TestEditFileNoop covers REQ-TOOLS-8 "No-op rejected".
func TestEditFileNoop(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "a.txt", "foo\n")
	tool := NewEditFile(root)

	_, err := tool.Execute(context.Background(), argsFor(t, map[string]any{"path": "a.txt", "old_text": "foo", "new_text": "foo"}))
	if err == nil || !strings.Contains(err.Error(), "no-op") {
		t.Fatalf("error = %v, want no-op error", err)
	}
}

// TestEditFileMissing covers REQ-TOOLS-8 "Missing file".
func TestEditFileMissing(t *testing.T) {
	root := t.TempDir()
	tool := NewEditFile(root)

	_, err := tool.Execute(context.Background(), argsFor(t, map[string]any{"path": "ghost.txt", "old_text": "a", "new_text": "b"}))
	if err == nil || !strings.Contains(err.Error(), "ghost.txt") {
		t.Fatalf("error = %v, want error identifying the missing path", err)
	}
}

// TestEditFileEmptyOldText covers REQ-TOOLS-8 "Empty old_text rejected".
func TestEditFileEmptyOldText(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "a.txt", "foo\n")
	tool := NewEditFile(root)

	_, err := tool.Execute(context.Background(), argsFor(t, map[string]any{"path": "a.txt", "old_text": "", "new_text": "b"}))
	if err == nil || !strings.Contains(err.Error(), "old_text must not be empty") {
		t.Fatalf("error = %v, want old_text validation error", err)
	}
}

// TestEditFileEscapeRejected covers REQ-TOOLS-8 "Path escape rejected".
func TestEditFileEscapeRejected(t *testing.T) {
	root := t.TempDir()
	tool := NewEditFile(root)

	_, err := tool.Execute(context.Background(), argsFor(t, map[string]any{"path": "../kui-evil.txt", "old_text": "a", "new_text": "b"}))
	var constraint *PathConstraintError
	if !errors.As(err, &constraint) {
		t.Fatalf("error = %v, want *PathConstraintError", err)
	}
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(root), "kui-evil.txt")); !os.IsNotExist(statErr) {
		t.Errorf("file outside root was touched (stat error: %v)", statErr)
	}
}

// TestEditFileBinaryRejected covers REQ-TOOLS-8 "Binary file rejected".
func TestEditFileBinaryRejected(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "img.bin", "PNG\x00binary-data")
	tool := NewEditFile(root)

	_, err := tool.Execute(context.Background(), argsFor(t, map[string]any{"path": "img.bin", "old_text": "binary", "new_text": "text"}))
	if err == nil || !strings.Contains(err.Error(), "binary") {
		t.Fatalf("error = %v, want binary-file error", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "img.bin"))
	if string(data) != "PNG\x00binary-data" {
		t.Errorf("binary file was modified: %q", data)
	}
}

// TestEditFileCRLFPreserved covers REQ-TOOLS-8 "CRLF preserved".
func TestEditFileCRLFPreserved(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "win.txt", "hello world\r\nsecond\r\n")
	tool := NewEditFile(root)

	if _, err := tool.Execute(context.Background(), argsFor(t, map[string]any{"path": "win.txt", "old_text": "world", "new_text": "kui"})); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "win.txt"))
	if string(data) != "hello kui\r\nsecond\r\n" {
		t.Errorf("CRLF not preserved: %q", data)
	}
}

// TestEditFileBOMPreserved covers REQ-TOOLS-8 "BOM preserved".
func TestEditFileBOMPreserved(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "bom.txt", utf8BOM+"hello\n")
	tool := NewEditFile(root)

	if _, err := tool.Execute(context.Background(), argsFor(t, map[string]any{"path": "bom.txt", "old_text": "hello", "new_text": "bye"})); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "bom.txt"))
	if string(data) != utf8BOM+"bye\n" {
		t.Errorf("BOM not preserved: %q", data)
	}
}

// TestEditFileAtomicWrite covers REQ-TOOLS-8 "Atomic write via tmp+rename".
func TestEditFileAtomicWrite(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "a.txt", "foo\n")
	tool := NewEditFile(root)

	if _, err := tool.Execute(context.Background(), argsFor(t, map[string]any{"path": "a.txt", "old_text": "foo", "new_text": "bar"})); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".edit-") {
			t.Errorf("temp file left behind: %q", e.Name())
		}
	}
	data, _ := os.ReadFile(filepath.Join(root, "a.txt"))
	if string(data) != "bar\n" {
		t.Errorf("file content = %q, want %q", data, "bar\n")
	}
}

// TestEditFileSyncsDidChange covers REQ-TOOLS-8 "LSP DidChange notified".
func TestEditFileSyncsDidChange(t *testing.T) {
	syncer := &mockFileSyncer{}
	root := t.TempDir()
	writeFixture(t, root, "main.go", "package main\n")
	tool := NewEditFileWithSync(root, syncer)

	if _, err := tool.Execute(context.Background(), argsFor(t, map[string]any{"path": "main.go", "old_text": "main", "new_text": "kui"})); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(syncer.changed) != 1 {
		t.Fatalf("didChange calls = %d, want 1", len(syncer.changed))
	}
	if !strings.HasSuffix(syncer.changed[0], "main.go") {
		t.Errorf("didChange URI = %q, want it to end with main.go", syncer.changed[0])
	}
}

// TestEditFileSymlinkEscape rejects edits that would land outside the root
// through a symlinked directory (REQ-TOOLS-8 "Path escape rejected").
func TestEditFileSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret\n"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	tool := NewEditFile(root)

	_, err := tool.Execute(context.Background(), argsFor(t, map[string]any{"path": filepath.Join("link", "secret.txt"), "old_text": "secret", "new_text": "evil"}))
	var constraint *PathConstraintError
	if !errors.As(err, &constraint) {
		t.Fatalf("error = %v, want *PathConstraintError", err)
	}
	data, _ := os.ReadFile(filepath.Join(outside, "secret.txt"))
	if string(data) != "secret\n" {
		t.Errorf("file outside root was modified through the symlink: %q", data)
	}
}

// TestEditFileDeletion covers REQ-TOOLS-8 "Deletion via empty new_text":
// new_text:"" removes the matched block and the diff shows the removal.
func TestEditFileDeletion(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "a.txt", "line1\nremove me\nline3\n")
	tool := NewEditFile(root)

	res, err := tool.Execute(context.Background(), argsFor(t, map[string]any{"path": "a.txt", "old_text": "remove me\n", "new_text": ""}))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.Contains(res, "-remove me") {
		t.Errorf("result diff missing removal line: %q", res)
	}
	data, _ := os.ReadFile(filepath.Join(root, "a.txt"))
	if string(data) != "line1\nline3\n" {
		t.Errorf("file content = %q, want %q", data, "line1\nline3\n")
	}
}

// TestEditFileMixedLineEndingsDominantWins covers mixed CRLF/LF files: the
// dominant ending wins and the file is unified to it on write.
func TestEditFileMixedLineEndingsDominantWins(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mix.txt", "a\r\nb\r\nc\r\nd\n")
	tool := NewEditFile(root)

	if _, err := tool.Execute(context.Background(), argsFor(t, map[string]any{"path": "mix.txt", "old_text": "d", "new_text": "D"})); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "mix.txt"))
	if string(data) != "a\r\nb\r\nc\r\nD\r\n" {
		t.Errorf("mixed endings not unified to dominant CRLF: %q", data)
	}
}

// TestEditFileOccurrenceNegative rejects occurrence:-1 with an explicit error.
func TestEditFileOccurrenceNegative(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "a.txt", "foo\n")
	tool := NewEditFile(root)

	_, err := tool.Execute(context.Background(), argsFor(t, map[string]any{"path": "a.txt", "old_text": "foo", "new_text": "bar", "occurrence": -1}))
	if err == nil || !strings.Contains(err.Error(), "occurrence must be >= 1") {
		t.Fatalf("error = %v, want explicit occurrence >= 1 error", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "a.txt"))
	if string(data) != "foo\n" {
		t.Errorf("file was modified on invalid occurrence: %q", data)
	}
}

// TestEditFileOccurrenceExplicitZero rejects an explicit occurrence:0: it is
// distinct from an absent occurrence (unique-match mode) and violates the
// schema minimum of 1.
func TestEditFileOccurrenceExplicitZero(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "a.txt", "foo\n")
	tool := NewEditFile(root)

	_, err := tool.Execute(context.Background(), argsFor(t, map[string]any{"path": "a.txt", "old_text": "foo", "new_text": "bar", "occurrence": 0}))
	if err == nil || !strings.Contains(err.Error(), "occurrence must be >= 1") {
		t.Fatalf("error = %v, want explicit occurrence >= 1 error", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "a.txt"))
	if string(data) != "foo\n" {
		t.Errorf("file was modified on explicit occurrence 0: %q", data)
	}
}

// TestEditFilePathIsDirectory returns an error identifying the path when the
// target is a directory, and leaves the directory intact.
func TestEditFilePathIsDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	tool := NewEditFile(root)

	_, err := tool.Execute(context.Background(), argsFor(t, map[string]any{"path": "sub", "old_text": "a", "new_text": "b"}))
	if err == nil || !strings.Contains(err.Error(), "sub") {
		t.Fatalf("error = %v, want error identifying the directory path", err)
	}
	if fi, statErr := os.Stat(filepath.Join(root, "sub")); statErr != nil || !fi.IsDir() {
		t.Errorf("directory was disturbed (fi=%v, statErr=%v)", fi, statErr)
	}
}

// TestAtomicWriteFailureCleansTemp covers the atomic-write residue guarantee:
// when the rename fails, no .edit-* temp file is left behind.
func TestAtomicWriteFailureCleansTemp(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "dir"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Renaming a file onto an existing directory fails; the temp file must
	// still be cleaned up by atomicWrite's deferred Remove.
	if err := atomicWrite(filepath.Join(root, "dir"), []byte("x")); err == nil {
		t.Fatalf("atomicWrite onto a directory succeeded, want error")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".edit-") {
			t.Errorf("temp file left behind after failed write: %q", e.Name())
		}
	}
}
