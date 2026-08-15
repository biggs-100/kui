package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// WriteFile creates or overwrites a file inside the workspace root and
// reports the written path (REQ-TOOLS-2). Escaping paths are rejected before
// any I/O (D11).
type WriteFile struct {
	root string
}

// NewWriteFile returns a write_file tool confined to root.
func NewWriteFile(root string) *WriteFile {
	return &WriteFile{root: root}
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
	return resolved, nil
}
