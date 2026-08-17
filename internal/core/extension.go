package core

import "context"

// Extension is the port every extension implements (D1). It provides a stable
// name, receives the extension API during initialization, and releases
// resources on shutdown.
type Extension interface {
	// Name returns a stable identifier for the extension.
	Name() string

	// Init receives the extension API and MUST complete successfully for
	// the extension to become active.
	Init(api ExtensionAPI) error

	// Shutdown releases all resources held by the extension.
	Shutdown() error
}

// ExtensionAPI is the port through which extensions register tools, hooks,
// and commands during initialization (D2).
type ExtensionAPI interface {
	// RegisterTool adds a tool to the agent's tool registry.
	RegisterTool(tool Tool) error

	// RegisterHook associates a handler with an event name.
	RegisterHook(event string, handler HookHandler) error

	// RegisterCommand adds a command to the agent's command registry.
	// Stubbed for future TUI commands.
	RegisterCommand(cmd Command) error
}

// HookHandler is the function signature for hook handlers (REQ-EXT-3).
// Handlers are invoked by the HookRegistry in registration order.
type HookHandler func(ctx HookContext) error

// Command represents a TUI command that can be registered by an extension (D2).
// The handler receives a context and raw argument string, returning output or
// an error.
type Command struct {
	Name        string
	Description string
	Handler     func(ctx context.Context, args string) (string, error)
}
