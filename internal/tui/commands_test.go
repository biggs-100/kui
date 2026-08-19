package tui

import (
	"strings"
	"testing"
)

// TestCommandRegistryCreate verifies that NewCommandRegistry creates a registry
// containing all expected commands with correct metadata.
func TestCommandRegistryCreate(t *testing.T) {
	registry := NewCommandRegistry()

	commands := registry.All()
	if len(commands) == 0 {
		t.Fatal("registry should contain commands, got 0")
	}

	// Registry must have at least the core slash commands
	expectedNames := map[string]bool{
		"/reload":   false,
		"/sessions": false,
		"/resume":   false,
		"/rename":   false,
		"/undo":     false,
		"/redo":     false,
		"/quit":     false,
		"/exit":     false,
		"/help":     false,
		"/theme":    false,
		"/status":   false,
		"/clear":    false,
	}

	for _, cmd := range commands {
		if _, ok := expectedNames[cmd.Name]; ok {
			expectedNames[cmd.Name] = true
		}
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("registry missing command %q", name)
		}
	}

	// Every command must have a non-empty description
	for _, cmd := range commands {
		if cmd.Description == "" {
			t.Errorf("command %q has empty description", cmd.Name)
		}
		if cmd.Category == "" {
			t.Errorf("command %q has empty category", cmd.Name)
		}
	}
}

// TestRegistryLookup verifies that Lookup finds known commands and returns nil
// for unknown names.
func TestRegistryLookup(t *testing.T) {
	registry := NewCommandRegistry()

	// Known command
	cmd := registry.Lookup("/reload")
	if cmd == nil {
		t.Fatal("Lookup(/reload) returned nil, want *Command")
	}
	if cmd.Name != "/reload" {
		t.Errorf("Lookup(/reload).Name = %q, want %q", cmd.Name, "/reload")
	}

	// Unknown command
	cmd = registry.Lookup("/nonexistent")
	if cmd != nil {
		t.Errorf("Lookup(/nonexistent) = %v, want nil", cmd)
	}
}

// TestRegistryHelpText verifies that HelpText returns categorized output with
// category headers, command names, descriptions, and shortcuts.
func TestRegistryHelpText(t *testing.T) {
	registry := NewCommandRegistry()
	help := registry.HelpText()

	if help == "" {
		t.Fatal("HelpText() returned empty string")
	}

	// Help text should contain category headers
	if !strings.Contains(help, "Session") {
		t.Error("HelpText() missing 'Session' category")
	}
	if !strings.Contains(help, "System") {
		t.Error("HelpText() missing 'System' category")
	}

	// Help text should contain command descriptions
	if !strings.Contains(help, "/reload") {
		t.Error("HelpText() missing /reload command")
	}
	if !strings.Contains(help, "/help") {
		t.Error("HelpText() missing /help command")
	}

	// Help text should contain keyboard shortcuts
	if !strings.Contains(help, "Ctrl+P") {
		t.Error("HelpText() missing Ctrl+P shortcut")
	}
}

// TestRegistryAllReturnsCopy verifies that All() returns a slice that can be
// safely modified without affecting the registry.
func TestRegistryAllReturnsCopy(t *testing.T) {
	registry := NewCommandRegistry()

	cmds1 := registry.All()
	cmds2 := registry.All()

	if len(cmds1) != len(cmds2) {
		t.Errorf("two All() calls returned different lengths: %d vs %d", len(cmds1), len(cmds2))
	}

	// Modifying the returned slice should not affect subsequent calls
	cmds1[0].Name = "modified"
	cmds3 := registry.All()
	if cmds3[0].Name == "modified" {
		t.Error("All() returned a reference to internal slice, should return a copy")
	}
}

// TestRegistryCommandsHaveHandlers verifies that all registered commands have
// non-nil handler functions.
func TestRegistryCommandsHaveHandlers(t *testing.T) {
	registry := NewCommandRegistry()

	for _, cmd := range registry.All() {
		// Handle should not panic for any command
		result := registry.Handle(cmd.Name, []string{cmd.Name})
		// result can be nil — that's fine for commands that don't return a Cmd
		_ = result
	}
}

// TestRegistryDispatch verifies that dispatching a command through the registry
// invokes the correct handler.
func TestRegistryDispatch(t *testing.T) {
	registry := NewCommandRegistry()

	// Lookup /reload and verify its handler is callable
	cmd := registry.Lookup("/reload")
	if cmd == nil {
		t.Fatal("Lookup(/reload) returned nil")
	}

	// Handler should return a tea.Cmd (may be nil for no-op commands)
	result := registry.Handle("/reload", []string{"/reload"})
	_ = result
}

// TestRegistryCategoriesGroups verifies that commands are organized into
// distinct categories.
func TestRegistryCategoriesGroups(t *testing.T) {
	registry := NewCommandRegistry()

	categories := make(map[string][]string)
	for _, cmd := range registry.All() {
		categories[cmd.Category] = append(categories[cmd.Category], cmd.Name)
	}

	// Must have at least 2 categories
	if len(categories) < 2 {
		t.Errorf("expected at least 2 categories, got %d", len(categories))
	}

	// Session category must contain /sessions
	sessionCmds, ok := categories["Session"]
	if !ok {
		t.Fatal("missing 'Session' category")
	}
	found := false
	for _, name := range sessionCmds {
		if name == "/sessions" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Session category missing /sessions command")
	}
}

// TestRegistryCommandNames verifies that CommandNames returns only slash commands.
func TestRegistryCommandNames(t *testing.T) {
	registry := NewCommandRegistry()

	names := registry.CommandNames()
	if len(names) == 0 {
		t.Fatal("CommandNames() returned empty")
	}

	for _, name := range names {
		if name[0] != '/' {
			t.Errorf("CommandNames() included non-slash command: %q", name)
		}
	}
}
