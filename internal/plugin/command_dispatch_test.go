package plugin

import (
	"fmt"
	"testing"
)

func TestPluginCommandDispatcherNew(t *testing.T) {
	d := NewPluginDiscovery(t.TempDir())
	r := NewPluginRegistry(d)
	cd := NewCommandDispatcher(r)
	if cd == nil {
		t.Fatal("NewCommandDispatcher returned nil")
	}
	if cd.registry != r {
		t.Error("registry not set correctly")
	}
	if cd.handlers == nil {
		t.Error("handlers map not initialized")
	}
	if cd.commands == nil {
		t.Error("commands map not initialized")
	}
}

func TestPluginCommandDispatcherRegisterValid(t *testing.T) {
	d := NewPluginDiscovery(t.TempDir())
	r := NewPluginRegistry(d)
	cd := NewCommandDispatcher(r)

	handler := func(args []string) (string, error) {
		return "ok", nil
	}

	err := cd.Register("greet", "my-plugin", "Say hello", "<name>", handler)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Verify command appears in List
	commands := cd.List()
	if len(commands) != 1 {
		t.Fatalf("List() = %d commands, want 1", len(commands))
	}
	if commands[0].Name != "greet" {
		t.Errorf("Name = %q, want %q", commands[0].Name, "greet")
	}
	if commands[0].Plugin != "my-plugin" {
		t.Errorf("Plugin = %q, want %q", commands[0].Plugin, "my-plugin")
	}
	if commands[0].Description != "Say hello" {
		t.Errorf("Description = %q, want %q", commands[0].Description, "Say hello")
	}
	if commands[0].Args != "<name>" {
		t.Errorf("Args = %q, want %q", commands[0].Args, "<name>")
	}
}

func TestPluginCommandDispatcherRegisterDuplicate(t *testing.T) {
	d := NewPluginDiscovery(t.TempDir())
	r := NewPluginRegistry(d)
	cd := NewCommandDispatcher(r)

	handler := func(args []string) (string, error) { return "", nil }

	if err := cd.Register("greet", "plugin-a", "desc", "", handler); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}

	err := cd.Register("greet", "plugin-b", "desc2", "", handler)
	if err == nil {
		t.Fatal("expected error for duplicate registration, got nil")
	}
}

func TestPluginCommandDispatcherUnregisterFound(t *testing.T) {
	d := NewPluginDiscovery(t.TempDir())
	r := NewPluginRegistry(d)
	cd := NewCommandDispatcher(r)

	handler := func(args []string) (string, error) { return "", nil }
	_ = cd.Register("greet", "my-plugin", "desc", "", handler)

	if err := cd.Unregister("greet"); err != nil {
		t.Fatalf("Unregister() error = %v", err)
	}

	commands := cd.List()
	if len(commands) != 0 {
		t.Errorf("List() after unregister = %d commands, want 0", len(commands))
	}
}

func TestPluginCommandDispatcherUnregisterNotFound(t *testing.T) {
	d := NewPluginDiscovery(t.TempDir())
	r := NewPluginRegistry(d)
	cd := NewCommandDispatcher(r)

	if err := cd.Unregister("nonexistent"); err == nil {
		t.Fatal("expected error for unregistering nonexistent command, got nil")
	}
}

func TestPluginCommandDispatcherExecuteSuccess(t *testing.T) {
	d := NewPluginDiscovery(t.TempDir())
	r := NewPluginRegistry(d)
	cd := NewCommandDispatcher(r)

	handler := func(args []string) (string, error) {
		return fmt.Sprintf("hello %v", args), nil
	}
	_ = cd.Register("greet", "my-plugin", "desc", "", handler)

	result, err := cd.Execute("greet", []string{"world"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != "hello [world]" {
		t.Errorf("Execute() = %q, want %q", result, "hello [world]")
	}
}

func TestPluginCommandDispatcherExecuteError(t *testing.T) {
	d := NewPluginDiscovery(t.TempDir())
	r := NewPluginRegistry(d)
	cd := NewCommandDispatcher(r)

	handler := func(args []string) (string, error) {
		return "", fmt.Errorf("command failed: %s", args[0])
	}
	_ = cd.Register("fail", "my-plugin", "desc", "", handler)

	_, err := cd.Execute("fail", []string{"reason"})
	if err == nil {
		t.Fatal("expected error from Execute, got nil")
	}
	if err.Error() != "command failed: reason" {
		t.Errorf("error = %q, want %q", err.Error(), "command failed: reason")
	}
}

func TestPluginCommandDispatcherExecuteNotFound(t *testing.T) {
	d := NewPluginDiscovery(t.TempDir())
	r := NewPluginRegistry(d)
	cd := NewCommandDispatcher(r)

	_, err := cd.Execute("nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for nonexistent command, got nil")
	}
}

func TestPluginCommandDispatcherListEmpty(t *testing.T) {
	d := NewPluginDiscovery(t.TempDir())
	r := NewPluginRegistry(d)
	cd := NewCommandDispatcher(r)

	commands := cd.List()
	if len(commands) != 0 {
		t.Errorf("List() = %d commands, want 0", len(commands))
	}
}

func TestPluginCommandDispatcherListMultiple(t *testing.T) {
	d := NewPluginDiscovery(t.TempDir())
	r := NewPluginRegistry(d)
	cd := NewCommandDispatcher(r)

	h1 := func(args []string) (string, error) { return "a", nil }
	h2 := func(args []string) (string, error) { return "b", nil }
	h3 := func(args []string) (string, error) { return "c", nil }

	_ = cd.Register("cmd-a", "plugin-a", "Command A", "", h1)
	_ = cd.Register("cmd-b", "plugin-b", "Command B", "", h2)
	_ = cd.Register("cmd-c", "plugin-a", "Command C", "", h3)

	commands := cd.List()
	if len(commands) != 3 {
		t.Fatalf("List() = %d commands, want 3", len(commands))
	}

	names := make(map[string]bool)
	for _, cmd := range commands {
		names[cmd.Name] = true
	}
	if !names["cmd-a"] || !names["cmd-b"] || !names["cmd-c"] {
		t.Errorf("not all commands found: %v", names)
	}
}

func TestPluginCommandDispatcherUnregisterByPlugin(t *testing.T) {
	d := NewPluginDiscovery(t.TempDir())
	r := NewPluginRegistry(d)
	cd := NewCommandDispatcher(r)

	h1 := func(args []string) (string, error) { return "", nil }
	h2 := func(args []string) (string, error) { return "", nil }
	h3 := func(args []string) (string, error) { return "", nil }

	_ = cd.Register("cmd-a", "plugin-a", "desc", "", h1)
	_ = cd.Register("cmd-b", "plugin-b", "desc", "", h2)
	_ = cd.Register("cmd-c", "plugin-a", "desc", "", h3)

	removed := cd.UnregisterByPlugin("plugin-a")
	if removed != 2 {
		t.Errorf("UnregisterByPlugin() = %d, want 2", removed)
	}

	commands := cd.List()
	if len(commands) != 1 {
		t.Fatalf("List() after unregister = %d commands, want 1", len(commands))
	}
	if commands[0].Name != "cmd-b" {
		t.Errorf("remaining command = %q, want %q", commands[0].Name, "cmd-b")
	}
}

func TestPluginCommandDispatcherExecuteEmptyArgs(t *testing.T) {
	d := NewPluginDiscovery(t.TempDir())
	r := NewPluginRegistry(d)
	cd := NewCommandDispatcher(r)

	handler := func(args []string) (string, error) {
		return fmt.Sprintf("args count: %d", len(args)), nil
	}
	_ = cd.Register("count", "my-plugin", "desc", "", handler)

	result, err := cd.Execute("count", nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != "args count: 0" {
		t.Errorf("Execute() = %q, want %q", result, "args count: 0")
	}
}
