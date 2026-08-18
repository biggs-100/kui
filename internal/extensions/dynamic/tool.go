package dynamic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/biggs-100/kui/internal/core"
)

// ClientInterface is the subset of Client methods that DynamicExtension and
// DynamicTool need. Extracted so tests can inject a mock without touching the
// real subprocess.
type ClientInterface interface {
	Initialize(ctx context.Context) error
	ListTools(ctx context.Context) ([]ToolDef, error)
	CallTool(ctx context.Context, name string, args json.RawMessage) (string, error)
	Shutdown(ctx context.Context) error
}

// DynamicTool wraps a dynamic extension tool, implementing core.Tool (D3).
// The tool name is prefixed with the extension name to avoid collisions in
// the global registry (D4).
type DynamicTool struct {
	extensionName string
	def           ToolDef
	client        ClientInterface
}

// NewDynamicTool creates a DynamicTool for the given tool definition.
func NewDynamicTool(extensionName string, def ToolDef, client ClientInterface) *DynamicTool {
	return &DynamicTool{
		extensionName: extensionName,
		def:           def,
		client:        client,
	}
}

// Name returns "{extensionName}_{toolName}" to keep the global registry
// unambiguous across extensions (D4).
func (t *DynamicTool) Name() string {
	return fmt.Sprintf("%s_%s", t.extensionName, t.def.Name)
}

// Description returns the tool description provided by the extension.
func (t *DynamicTool) Description() string {
	return t.def.Description
}

// Schema returns the raw JSON input schema for this tool.
func (t *DynamicTool) Schema() string {
	return string(t.def.InputSchema)
}

// Execute delegates to the extension subprocess via Client.CallTool.
func (t *DynamicTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	return t.client.CallTool(ctx, t.def.Name, args)
}

// Verify ClientInterface is satisfied at compile time.
var _ ClientInterface = (*Client)(nil)

// Verify DynamicTool implements core.Tool at compile time.
var _ core.Tool = (*DynamicTool)(nil)
