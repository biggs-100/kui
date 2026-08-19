package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileSyncer is the port for sending LSP file synchronization notifications.
// Implementations must be safe for concurrent use.
type FileSyncer interface {
	// DidOpen notifies the LSP server that a file was opened.
	DidOpen(uri, languageID, content string) error
	// DidChange notifies the LSP server that a file's content changed.
	DidChange(uri, content string) error
}

// ──────────────────────────────────────────────────────────────────────────────
// read_file
// ──────────────────────────────────────────────────────────────────────────────

// ReadFile reads the full text content of a file inside the workspace root
// (REQ-TOOLS-1). Paths escaping the root are rejected before any I/O (D11).
type ReadFile struct {
	root   string
	syncer FileSyncer // optional — nil disables LSP notifications
}

// NewReadFile returns a read_file tool confined to root.
func NewReadFile(root string) *ReadFile {
	return &ReadFile{root: root}
}

// NewReadFileWithSync returns a read_file tool with LSP file sync support.
func NewReadFileWithSync(root string, syncer FileSyncer) *ReadFile {
	return &ReadFile{root: root, syncer: syncer}
}

// Name returns the stable tool name (REQ-TOOLS-4).
func (t *ReadFile) Name() string { return "read_file" }

// Description returns the tool description (REQ-TOOLS-4).
func (t *ReadFile) Description() string {
	return "Read the full text content of a file inside the workspace"
}

// Schema returns the raw JSON parameter schema (D3, REQ-TOOLS-4).
func (t *ReadFile) Schema() string {
	return `{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}`
}

// Execute reads the file at path. The path must resolve inside the workspace
// root; escapes are rejected before the file system is touched.
func (t *ReadFile) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if in.Path == "" {
		return "", errors.New("path must not be empty")
	}
	resolved, err := resolvePath(t.root, in.Path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", err
	}

	// LSP file sync: notify server that file is open.
	if t.syncer != nil {
		uri := pathToFileURI(resolved)
		langID := languageIDFromPath(in.Path)
		_ = t.syncer.DidOpen(uri, langID, string(data))
	}

	return string(data), nil
}

// ──────────────────────────────────────────────────────────────────────────────
// write_file
// ──────────────────────────────────────────────────────────────────────────────

// WriteFile creates or overwrites a file inside the workspace root and
// reports the written path (REQ-TOOLS-2). Escaping paths are rejected before
// any I/O (D11).
type WriteFile struct {
	root   string
	syncer FileSyncer // optional — nil disables LSP notifications
}

// NewWriteFile returns a write_file tool confined to root.
func NewWriteFile(root string) *WriteFile {
	return &WriteFile{root: root}
}

// NewWriteFileWithSync returns a write_file tool with LSP file sync support.
func NewWriteFileWithSync(root string, syncer FileSyncer) *WriteFile {
	return &WriteFile{root: root, syncer: syncer}
}

// Name returns the stable tool name (REQ-TOOLS-4).
func (t *WriteFile) Name() string { return "write_file" }

// Description returns the tool description (REQ-TOOLS-4).
func (t *WriteFile) Description() string {
	return "Create or overwrite a file inside the workspace with the given content"
}

// Schema returns the raw JSON parameter schema (D3, REQ-TOOLS-4).
func (t *WriteFile) Schema() string {
	return `{"type":"object","properties":{"path":{"type":"string"},"content":{"type":"string"}},"required":["path","content"]}`
}

// Execute writes content to the file at path. The path must resolve inside
// the workspace root; escapes are rejected before the file system is touched.
// It reports the resolved path that was written.
func (t *WriteFile) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if in.Path == "" {
		return "", errors.New("path must not be empty")
	}
	resolved, err := resolvePath(t.root, in.Path)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(resolved, []byte(in.Content), 0o644); err != nil {
		return "", err
	}

	// LSP file sync: notify server that file changed.
	if t.syncer != nil {
		uri := pathToFileURI(resolved)
		_ = t.syncer.DidChange(uri, in.Content)
	}

	return resolved, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// helpers
// ──────────────────────────────────────────────────────────────────────────────

// pathToFileURI converts an absolute file path to a file:// URI.
func pathToFileURI(path string) string {
	// Normalize separators for URI.
	path = filepath.ToSlash(path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return "file://" + path
}

// languageIDFromPath returns the LSP language ID for a file extension.
func languageIDFromPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "typescriptreact"
	case ".js":
		return "javascript"
	case ".jsx":
		return "javascriptreact"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".cxx", ".hpp":
		return "cpp"
	case ".rb":
		return "ruby"
	case ".sh":
		return "shellscript"
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".md":
		return "markdown"
	case ".html":
		return "html"
	case ".css":
		return "css"
	default:
		return "text"
	}
}
