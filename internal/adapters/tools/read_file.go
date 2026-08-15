package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// ReadFile reads the full text content of a file inside the workspace root
// (REQ-TOOLS-1). Paths escaping the root are rejected before any I/O (D11).
type ReadFile struct {
	root string
}

// NewReadFile returns a read_file tool confined to root.
func NewReadFile(root string) *ReadFile {
	return &ReadFile{root: root}
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
	return string(data), nil
}
