package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/biggs-100/kui/internal/tui/views"
)

// CommandRegistry holds all registered commands and provides lookup, listing,
// and help-text generation.
type CommandRegistry struct {
	commands []views.Command
	handlers map[string]func(parts []string) tea.Cmd
}

// NewCommandRegistry creates a CommandRegistry with all known commands.
func NewCommandRegistry() *CommandRegistry {
	r := &CommandRegistry{
		handlers: make(map[string]func(parts []string) tea.Cmd),
	}

	// Session commands
	r.registerWithHandler(views.Command{
		Name:        "/sessions",
		Description: "List and manage sessions",
		Category:    "Session",
	}, nil)
	r.registerWithHandler(views.Command{
		Name:        "/resume",
		Description: "Resume a saved session",
		Category:    "Session",
		Args:        "<session-id>",
	}, nil)
	r.registerWithHandler(views.Command{
		Name:        "/rename",
		Description: "Rename the current session",
		Category:    "Session",
		Args:        "<name>",
	}, nil)

	// Edit commands
	r.registerWithHandler(views.Command{
		Name:        "/undo",
		Description: "Undo last conversation turn",
		Category:    "Edit",
	}, nil)
	r.registerWithHandler(views.Command{
		Name:        "/redo",
		Description: "Redo last undone turn",
		Category:    "Edit",
	}, nil)
	r.registerWithHandler(views.Command{
		Name:        "/clear",
		Description: "Clear chat display",
		Category:    "Edit",
	}, nil)

	// Runtime commands
	r.registerWithHandler(views.Command{
		Name:        "/reload",
		Description: "Hot-reload runtime state",
		Category:    "Runtime",
	}, nil)
	r.registerWithHandler(views.Command{
		Name:        "/theme",
		Description: "Switch UI theme",
		Category:    "Runtime",
		Args:        "<name>",
	}, nil)
	r.registerWithHandler(views.Command{
		Name:        "/status",
		Description: "Show current profile status",
		Category:    "Runtime",
	}, nil)

	// Navigation commands (keyboard shortcuts)
	r.registerWithHandler(views.Command{
		Name:        "Tab",
		Description: "Switch profile",
		Category:    "Navigation",
		Shortcut:    "Tab",
	}, nil)
	r.registerWithHandler(views.Command{
		Name:        "d",
		Description: "Toggle diff view",
		Category:    "Navigation",
		Shortcut:    "d",
	}, nil)
	r.registerWithHandler(views.Command{
		Name:        "gd",
		Description: "Go to definition (LSP)",
		Category:    "Navigation",
		Shortcut:    "gd",
	}, nil)
	r.registerWithHandler(views.Command{
		Name:        "gr",
		Description: "Find references (LSP)",
		Category:    "Navigation",
		Shortcut:    "gr",
	}, nil)
	r.registerWithHandler(views.Command{
		Name:        "K",
		Description: "Show hover info (LSP)",
		Category:    "Navigation",
		Shortcut:    "K",
	}, nil)

	// System commands
	r.registerWithHandler(views.Command{
		Name:        "/quit",
		Description: "Save and exit",
		Category:    "System",
		Shortcut:    "Ctrl+C",
	}, nil)
	r.registerWithHandler(views.Command{
		Name:        "/exit",
		Description: "Save and exit",
		Category:    "System",
	}, nil)
	r.registerWithHandler(views.Command{
		Name:        "/help",
		Description: "Show this help",
		Category:    "System",
	}, nil)

	// Palette shortcut (non-slash, informational only)
	r.registerWithHandler(views.Command{
		Name:        "Ctrl+P",
		Description: "Command palette",
		Category:    "Navigation",
		Shortcut:    "Ctrl+P",
	}, nil)

	return r
}

// registerWithHandler adds a command with an associated handler function.
func (r *CommandRegistry) registerWithHandler(cmd views.Command, handler func(parts []string) tea.Cmd) {
	r.commands = append(r.commands, cmd)
	r.handlers[cmd.Name] = handler
}

// Lookup returns the command with the given name, or nil if not found.
func (r *CommandRegistry) Lookup(name string) *views.Command {
	for i := range r.commands {
		if r.commands[i].Name == name {
			return &r.commands[i]
		}
	}
	return nil
}

// Handle executes the handler for the given command name. Returns the tea.Cmd
// from the handler, or nil if no handler is registered or the handler is nil.
func (r *CommandRegistry) Handle(name string, parts []string) tea.Cmd {
	if h, ok := r.handlers[name]; ok && h != nil {
		return h(parts)
	}
	return nil
}

// All returns a copy of all registered commands.
func (r *CommandRegistry) All() []views.Command {
	out := make([]views.Command, len(r.commands))
	copy(out, r.commands)
	return out
}

// CommandNames returns just the command names (for autocomplete).
func (r *CommandRegistry) CommandNames() []string {
	names := make([]string, 0, len(r.commands))
	for _, cmd := range r.commands {
		if strings.HasPrefix(cmd.Name, "/") {
			names = append(names, cmd.Name)
		}
	}
	return names
}

// HelpText returns a categorized help string with descriptions and shortcuts.
func (r *CommandRegistry) HelpText() string {
	type categoryEntry struct {
		name     string
		commands []views.Command
	}

	seen := make(map[string]int)
	var categories []categoryEntry

	for _, cmd := range r.commands {
		if idx, ok := seen[cmd.Category]; ok {
			categories[idx].commands = append(categories[idx].commands, cmd)
		} else {
			seen[cmd.Category] = len(categories)
			categories = append(categories, categoryEntry{
				name:     cmd.Category,
				commands: []views.Command{cmd},
			})
		}
	}

	var b strings.Builder
	for i, cat := range categories {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "%-12s", cat.name+":")
		for _, cmd := range cat.commands {
			fmt.Fprintf(&b, "\n            %-12s %s", cmd.Name, cmd.Description)
			if cmd.Shortcut != "" {
				fmt.Fprintf(&b, "  (%s)", cmd.Shortcut)
			}
		}
	}

	return b.String()
}
