package runtime

import (
	"github.com/biggs-100/kui/internal/core"
)

// extAPI is the production core.ExtensionAPI (D7, REQ-RELOAD-16). It backs
// extension registration onto the runtime's tool registry and hook registry,
// and records commands for future TUI wiring. Build passes one to
// extensions.LoadAll so compiled-in extensions can register tools, hooks, and
// commands during Init (REQ-RELOAD-17).
type extAPI struct {
	registry *core.Registry
	hooks    *core.HookRegistry
	commands []core.Command
}

// RegisterTool adds a tool to the runtime's full tool registry. A duplicate
// name is rejected with a DuplicateToolError while the first tool stays
// registered (REQ-RELOAD-16).
func (a *extAPI) RegisterTool(tool core.Tool) error {
	return a.registry.Register(tool)
}

// RegisterHook associates a handler with an event in the runtime's hook
// registry (REQ-RELOAD-16).
func (a *extAPI) RegisterHook(event string, handler core.HookHandler) error {
	return a.hooks.Register(event, handler)
}

// RegisterCommand records a command for future TUI wiring (D2). Commands are
// not dispatched anywhere yet — registration is append-only.
func (a *extAPI) RegisterCommand(cmd core.Command) error {
	a.commands = append(a.commands, cmd)
	return nil
}
