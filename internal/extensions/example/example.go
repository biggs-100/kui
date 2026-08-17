// Package example provides a minimal reference extension that demonstrates
// the Extension lifecycle: Init registers a tool and a hook, Shutdown is a
// no-op. The init() function self-registers the extension into the global
// registry (D6, REQ-DISCOVERY-1).
package example

import (
	"context"
	"encoding/json"

	"github.com/biggs-100/kui/internal/adapters/extensions"
	"github.com/biggs-100/kui/internal/core"
)

// exampleTool is a simple greeting tool for demonstration purposes.
type exampleTool struct{}

func (t *exampleTool) Name() string        { return "example_greet" }
func (t *exampleTool) Description() string { return "A simple greeting tool for demonstration purposes" }
func (t *exampleTool) Schema() string      { return `{"type":"object","properties":{}}` }
func (t *exampleTool) Execute(_ context.Context, _ json.RawMessage) (string, error) {
	return "Hello from the example extension!", nil
}

// exampleExtension is a minimal extension that registers a simple tool
// during Init and performs no-op shutdown.
type exampleExtension struct{}

func (e *exampleExtension) Name() string { return "example" }

// Init registers an example tool and a hook with the extension API (REQ-DISCOVERY-1).
func (e *exampleExtension) Init(api core.ExtensionAPI) error {
	// Register a simple example tool.
	_ = api.RegisterTool(&exampleTool{})

	// Register a hook for demonstration.
	_ = api.RegisterHook("before_provider_request", func(ctx core.HookContext) error {
		return nil
	})

	return nil
}

func (e *exampleExtension) Shutdown() error { return nil }

// init self-registers the example extension into the global registry (D6).
// A blank import of this package triggers init() and makes the extension
// available for LoadAll at startup.
func init() {
	extensions.Register(&exampleExtension{})
}
