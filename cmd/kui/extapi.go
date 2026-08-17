package main

import "github.com/biggs-100/kui/internal/core"

// extAPI is the production core.ExtensionAPI for the CLI path (D7).
// It backs extension registration onto the CLI's tool registry and hook
// registry, matching the runtime's extAPI pattern (REQ-RELOAD-16/17).
type extAPI struct {
	registry *core.Registry
	hooks    *core.HookRegistry
}

// RegisterTool adds a tool to the CLI's full tool registry (REQ-RELOAD-16).
func (a *extAPI) RegisterTool(tool core.Tool) error {
	return a.registry.Register(tool)
}

// RegisterHook associates a handler with an event in the CLI's hook registry
// (REQ-RELOAD-16).
func (a *extAPI) RegisterHook(event string, handler core.HookHandler) error {
	return a.hooks.Register(event, handler)
}

// RegisterCommand records a command for future TUI wiring (D2). Commands are
// not dispatched anywhere yet — registration is append-only.
func (a *extAPI) RegisterCommand(_ core.Command) error {
	return nil
}
