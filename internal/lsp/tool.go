package lsp

import (
	"context"
	"encoding/json"
	"fmt"
)

// ToolManager is the interface tools depend on to access LSP servers.
// It abstracts the LspManager for testability.
type ToolManager interface {
	GetServer(rootUri string) (*LspClient, error)
	IsRunning(rootUri string) bool
	Cache() *DiagnosticCache
}

// ──────────────────────────────────────────────────────────────────────────────
// lsp_diagnostics
// ──────────────────────────────────────────────────────────────────────────────

// LspDiagnosticsTool returns cached diagnostics for a file.
type LspDiagnosticsTool struct {
	mgr ToolManager
}

// NewLspDiagnosticsTool creates a diagnostics tool backed by the given manager.
func NewLspDiagnosticsTool(mgr ToolManager) *LspDiagnosticsTool {
	return &LspDiagnosticsTool{mgr: mgr}
}

// Name returns the stable tool name.
func (t *LspDiagnosticsTool) Name() string { return "lsp_diagnostics" }

// Description returns the tool description.
func (t *LspDiagnosticsTool) Description() string {
	return "Return LSP diagnostics (errors, warnings) for a file"
}

// Schema returns the JSON parameter schema.
func (t *LspDiagnosticsTool) Schema() string {
	return `{"type":"object","properties":{"uri":{"type":"string"}},"required":["uri"]}`
}

// Execute returns diagnostics from the cache for the given URI.
func (t *LspDiagnosticsTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var in struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if in.URI == "" {
		return "", fmt.Errorf("uri must not be empty")
	}

	diags := t.mgr.Cache().Get(in.URI)
	if diags == nil {
		diags = []Diagnostic{}
	}

	result, err := json.Marshal(map[string]interface{}{
		"diagnostics": diags,
	})
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// ──────────────────────────────────────────────────────────────────────────────
// lsp_hover
// ──────────────────────────────────────────────────────────────────────────────

// LspHoverTool retrieves hover information at a position.
type LspHoverTool struct {
	mgr ToolManager
}

// NewLspHoverTool creates a hover tool backed by the given manager.
func NewLspHoverTool(mgr ToolManager) *LspHoverTool {
	return &LspHoverTool{mgr: mgr}
}

// Name returns the stable tool name.
func (t *LspHoverTool) Name() string { return "lsp_hover" }

// Description returns the tool description.
func (t *LspHoverTool) Description() string {
	return "Show hover information (type signature, docs) at a position in a file"
}

// Schema returns the JSON parameter schema.
func (t *LspHoverTool) Schema() string {
	return `{"type":"object","properties":{"uri":{"type":"string"},"line":{"type":"integer"},"character":{"type":"integer"}},"required":["uri","line","character"]}`
}

// Execute sends a hover request to the LSP server.
func (t *LspHoverTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var in struct {
		URI       string `json:"uri"`
		Line      int    `json:"line"`
		Character int    `json:"character"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if in.URI == "" {
		return "", fmt.Errorf("uri must not be empty")
	}

	client, err := t.mgr.GetServer(in.URI)
	if err != nil {
		return "", err
	}

	hover, err := client.Hover(in.URI, in.Line, in.Character)
	if err != nil {
		return "", err
	}
	if hover == nil {
		return `{"contents":"","range":null}`, nil
	}

	result, err := json.Marshal(hover)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// ──────────────────────────────────────────────────────────────────────────────
// lsp_definition
// ──────────────────────────────────────────────────────────────────────────────

// LspDefinitionTool finds the definition location for a symbol.
type LspDefinitionTool struct {
	mgr ToolManager
}

// NewLspDefinitionTool creates a definition tool backed by the given manager.
func NewLspDefinitionTool(mgr ToolManager) *LspDefinitionTool {
	return &LspDefinitionTool{mgr: mgr}
}

// Name returns the stable tool name.
func (t *LspDefinitionTool) Name() string { return "lsp_definition" }

// Description returns the tool description.
func (t *LspDefinitionTool) Description() string {
	return "Go to the definition of a symbol at a position in a file"
}

// Schema returns the JSON parameter schema.
func (t *LspDefinitionTool) Schema() string {
	return `{"type":"object","properties":{"uri":{"type":"string"},"line":{"type":"integer"},"character":{"type":"integer"}},"required":["uri","line","character"]}`
}

// Execute sends a definition request to the LSP server.
func (t *LspDefinitionTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var in struct {
		URI       string `json:"uri"`
		Line      int    `json:"line"`
		Character int    `json:"character"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if in.URI == "" {
		return "", fmt.Errorf("uri must not be empty")
	}

	client, err := t.mgr.GetServer(in.URI)
	if err != nil {
		return "", err
	}

	locations, err := client.Definition(in.URI, in.Line, in.Character)
	if err != nil {
		return "", err
	}

	result, err := json.Marshal(map[string]interface{}{
		"locations": locations,
	})
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// ──────────────────────────────────────────────────────────────────────────────
// lsp_references
// ──────────────────────────────────────────────────────────────────────────────

// LspReferencesTool finds all references to a symbol.
type LspReferencesTool struct {
	mgr ToolManager
}

// NewLspReferencesTool creates a references tool backed by the given manager.
func NewLspReferencesTool(mgr ToolManager) *LspReferencesTool {
	return &LspReferencesTool{mgr: mgr}
}

// Name returns the stable tool name.
func (t *LspReferencesTool) Name() string { return "lsp_references" }

// Description returns the tool description.
func (t *LspReferencesTool) Description() string {
	return "Find all references to a symbol at a position in a file"
}

// Schema returns the JSON parameter schema.
func (t *LspReferencesTool) Schema() string {
	return `{"type":"object","properties":{"uri":{"type":"string"},"line":{"type":"integer"},"character":{"type":"integer"},"include_declaration":{"type":"boolean"}},"required":["uri","line","character"]}`
}

// Execute sends a references request to the LSP server.
func (t *LspReferencesTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var in struct {
		URI                string `json:"uri"`
		Line               int    `json:"line"`
		Character          int    `json:"character"`
		IncludeDeclaration bool   `json:"include_declaration"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if in.URI == "" {
		return "", fmt.Errorf("uri must not be empty")
	}

	client, err := t.mgr.GetServer(in.URI)
	if err != nil {
		return "", err
	}

	locations, err := client.References(in.URI, in.Line, in.Character, in.IncludeDeclaration)
	if err != nil {
		return "", err
	}

	result, err := json.Marshal(map[string]interface{}{
		"locations": locations,
	})
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// ──────────────────────────────────────────────────────────────────────────────
// LspTool is the common interface satisfied by all LSP tools.
// It mirrors core.Tool but avoids importing core to prevent circular deps.
type LspTool interface {
	Name() string
	Description() string
	Schema() string
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}

// LspTools returns the four LSP tool implementations for registration.
func LspTools(mgr ToolManager) []LspTool {
	return []LspTool{
		NewLspDiagnosticsTool(mgr),
		NewLspHoverTool(mgr),
		NewLspDefinitionTool(mgr),
		NewLspReferencesTool(mgr),
	}
}
