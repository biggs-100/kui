package plugin

import (
	"fmt"
	"sync"
)

// CommandHandler is a function that handles a command invocation.
type CommandHandler func(args []string) (string, error)

// CommandInfo holds metadata about a registered command.
type CommandInfo struct {
	Name        string
	Plugin      string
	Description string
	Args        string
}

// CommandDispatcher manages plugin commands: registration, execution, and lookup.
type CommandDispatcher struct {
	registry *PluginRegistry
	handlers map[string]CommandHandler
	commands map[string]CommandInfo
	mu       sync.RWMutex
}

// NewCommandDispatcher creates a new dispatcher backed by the given plugin registry.
func NewCommandDispatcher(registry *PluginRegistry) *CommandDispatcher {
	return &CommandDispatcher{
		registry: registry,
		handlers: make(map[string]CommandHandler),
		commands: make(map[string]CommandInfo),
	}
}

// Register adds a command with its handler. Returns an error if the name is already taken.
func (cd *CommandDispatcher) Register(name, plugin, description, args string, handler CommandHandler) error {
	cd.mu.Lock()
	defer cd.mu.Unlock()

	if _, exists := cd.commands[name]; exists {
		return fmt.Errorf("command %q already registered", name)
	}

	cd.commands[name] = CommandInfo{
		Name:        name,
		Plugin:      plugin,
		Description: description,
		Args:        args,
	}
	cd.handlers[name] = handler
	return nil
}

// Unregister removes a command by name. Returns an error if not found.
func (cd *CommandDispatcher) Unregister(name string) error {
	cd.mu.Lock()
	defer cd.mu.Unlock()

	if _, exists := cd.commands[name]; !exists {
		return fmt.Errorf("command %q not found", name)
	}

	delete(cd.commands, name)
	delete(cd.handlers, name)
	return nil
}

// UnregisterByPlugin removes all commands belonging to a plugin.
// Returns the number of commands removed.
func (cd *CommandDispatcher) UnregisterByPlugin(plugin string) int {
	cd.mu.Lock()
	defer cd.mu.Unlock()

	removed := 0
	for name, info := range cd.commands {
		if info.Plugin == plugin {
			delete(cd.commands, name)
			delete(cd.handlers, name)
			removed++
		}
	}
	return removed
}

// Execute invokes the handler for the given command name with the provided arguments.
func (cd *CommandDispatcher) Execute(name string, args []string) (string, error) {
	cd.mu.RLock()
	handler, exists := cd.handlers[name]
	cd.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("command %q not found", name)
	}

	return handler(args)
}

// List returns metadata for all registered commands.
func (cd *CommandDispatcher) List() []CommandInfo {
	cd.mu.RLock()
	defer cd.mu.RUnlock()

	result := make([]CommandInfo, 0, len(cd.commands))
	for _, info := range cd.commands {
		result = append(result, info)
	}
	return result
}
