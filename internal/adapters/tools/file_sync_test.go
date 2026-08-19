package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── FileSyncer interface tests ──────────────────────────────────────────────

// mockFileSyncer records calls for verification.
type mockFileSyncer struct {
	opened  []string // URIs
	changed []string // URIs
}

func (m *mockFileSyncer) DidOpen(uri, languageID, content string) error {
	m.opened = append(m.opened, uri)
	return nil
}

func (m *mockFileSyncer) DidChange(uri, content string) error {
	m.changed = append(m.changed, uri)
	return nil
}

func TestMockFileSyncerSatisfiesInterface(t *testing.T) {
	// Compile-time check: *mockFileSyncer must satisfy FileSyncer.
	var _ FileSyncer = (*mockFileSyncer)(nil)
}

func TestReadFileSyncsDidOpen(t *testing.T) {
	syncer := &mockFileSyncer{}
	root := t.TempDir()

	// Create a test file.
	testContent := "package main\n"
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte(testContent), 0o644); err != nil {
		t.Fatal(err)
	}

	rf := NewReadFileWithSync(root, syncer)
	_, err := rf.Execute(context.Background(), json.RawMessage(`{"path":"main.go"}`))
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if len(syncer.opened) != 1 {
		t.Fatalf("didOpen calls = %d, want 1", len(syncer.opened))
	}
	if !strings.HasSuffix(syncer.opened[0], "main.go") {
		t.Errorf("didOpen URI = %q, want it to end with main.go", syncer.opened[0])
	}
}

func TestReadFileSyncsGracefulDegradation(t *testing.T) {
	// No syncer — should work without error.
	root := t.TempDir()
	testContent := "package main\n"
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte(testContent), 0o644); err != nil {
		t.Fatal(err)
	}

	rf := NewReadFile(root) // no syncer
	result, err := rf.Execute(context.Background(), json.RawMessage(`{"path":"main.go"}`))
	if err != nil {
		t.Fatalf("Execute without syncer should not error: %v", err)
	}
	if result != testContent {
		t.Errorf("content = %q, want %q", result, testContent)
	}
}

func TestWriteFileSyncsDidChange(t *testing.T) {
	syncer := &mockFileSyncer{}
	root := t.TempDir()

	wf := NewWriteFileWithSync(root, syncer)
	_, err := wf.Execute(context.Background(), json.RawMessage(`{"path":"main.go","content":"package main\n"}`))
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if len(syncer.changed) != 1 {
		t.Fatalf("didChange calls = %d, want 1", len(syncer.changed))
	}
	if !strings.HasSuffix(syncer.changed[0], "main.go") {
		t.Errorf("didChange URI = %q, want it to end with main.go", syncer.changed[0])
	}
}

func TestWriteFileSyncsGracefulDegradation(t *testing.T) {
	root := t.TempDir()
	wf := NewWriteFile(root) // no syncer
	_, err := wf.Execute(context.Background(), json.RawMessage(`{"path":"main.go","content":"package main\n"}`))
	if err != nil {
		t.Fatalf("Execute without syncer should not error: %v", err)
	}
	// Verify file was written.
	data, err := os.ReadFile(filepath.Join(root, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "package main\n" {
		t.Errorf("file content = %q, want %q", string(data), "package main\n")
	}
}

func TestDefaultWithSyncerWiresFileSync(t *testing.T) {
	syncer := &mockFileSyncer{}
	root := t.TempDir()

	// Create a test file.
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("pkg main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	toolSlice := DefaultWithSyncer(root, 0, syncer)
	if len(toolSlice) != 3 {
		t.Fatalf("DefaultWithSyncer returned %d tools, want 3", len(toolSlice))
	}

	// Find read_file and write_file tools.
	var readFile, writeFile interface {
		Execute(context.Context, json.RawMessage) (string, error)
	}
	for _, tool := range toolSlice {
		switch tool.Name() {
		case "read_file":
			readFile = tool
		case "write_file":
			writeFile = tool
		}
	}

	if readFile == nil {
		t.Fatal("DefaultWithSyncer missing read_file tool")
	}
	if writeFile == nil {
		t.Fatal("DefaultWithSyncer missing write_file tool")
	}

	// Execute read_file — should trigger DidOpen via syncer.
	_, err := readFile.Execute(context.Background(), json.RawMessage(`{"path":"main.go"}`))
	if err != nil {
		t.Fatalf("read_file Execute error: %v", err)
	}
	if len(syncer.opened) != 1 {
		t.Errorf("didOpen calls = %d, want 1 after read_file with syncer", len(syncer.opened))
	}

	// Execute write_file — should trigger DidChange via syncer.
	_, err = writeFile.Execute(context.Background(), json.RawMessage(`{"path":"main.go","content":"updated\n"}`))
	if err != nil {
		t.Fatalf("write_file Execute error: %v", err)
	}
	if len(syncer.changed) != 1 {
		t.Errorf("didChange calls = %d, want 1 after write_file with syncer", len(syncer.changed))
	}
}

func TestDefaultWithoutSyncerNoNotifications(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("pkg main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	toolSlice := Default(root, 0)
	for _, tool := range toolSlice {
		if tool.Name() == "read_file" {
			_, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"main.go"}`))
			if err != nil {
				t.Fatalf("read_file Execute error: %v", err)
			}
		}
	}
	// Default() does not wire a syncer — no notifications should be sent.
	// (This just verifies Default still works without syncer.)
}

func TestReadFileSyncerError(t *testing.T) {
	syncer := &errorSyncer{openErr: errors.New("sync failed")}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	rf := NewReadFileWithSync(root, syncer)
	_, err := rf.Execute(context.Background(), json.RawMessage(`{"path":"main.go"}`))
	// File sync error should not fail the read — graceful degradation.
	if err != nil {
		t.Errorf("ReadFile should not fail on sync error: %v", err)
	}
}

func TestWriteFileSyncerError(t *testing.T) {
	syncer := &errorSyncer{changeErr: errors.New("sync failed")}
	root := t.TempDir()

	wf := NewWriteFileWithSync(root, syncer)
	_, err := wf.Execute(context.Background(), json.RawMessage(`{"path":"main.go","content":"x"}`))
	// File sync error should not fail the write — graceful degradation.
	if err != nil {
		t.Errorf("WriteFile should not fail on sync error: %v", err)
	}
}

// errorSyncer returns errors to test graceful degradation.
type errorSyncer struct {
	openErr   error
	changeErr error
}

func (e *errorSyncer) DidOpen(uri, languageID, content string) error { return e.openErr }
func (e *errorSyncer) DidChange(uri, content string) error           { return e.changeErr }

func TestLanguageIDFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"main.go", "go"},
		{"app.ts", "typescript"},
		{"index.js", "javascript"},
		{"lib.py", "python"},
		{"unknown.xyz", "text"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := languageIDFromPath(tt.path)
			if got != tt.want {
				t.Errorf("languageIDFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestPathToFileURI(t *testing.T) {
	uri := pathToFileURI("/home/user/main.go")
	if !strings.HasPrefix(uri, "file:///") {
		t.Errorf("pathToFileURI = %q, want file:/// prefix", uri)
	}
	if !strings.Contains(uri, "main.go") {
		t.Errorf("pathToFileURI = %q, want it to contain main.go", uri)
	}
}

// Ensure ReadFile and WriteFile still satisfy core.Tool.
func TestReadFileSatisfiesToolInterface(t *testing.T) {
	var _ interface {
		Name() string
		Description() string
		Schema() string
		Execute(context.Context, json.RawMessage) (string, error)
	} = NewReadFile("/tmp")
}

func TestWriteFileSatisfiesToolInterface(t *testing.T) {
	var _ interface {
		Name() string
		Description() string
		Schema() string
		Execute(context.Context, json.RawMessage) (string, error)
	} = NewWriteFile("/tmp")
}

// Silence unused import.
var _ = fmt.Sprintf
