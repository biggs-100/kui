package runtime

import (
	"context"
	"errors"
	"testing"

	"github.com/biggs-100/kui/internal/core"
)

// ── concrete ExtensionAPI tests (REQ-RELOAD-16) ───────────────────────────

func TestExtAPIRegisterTool(t *testing.T) {
	reg := core.NewRegistry()
	api := &extAPI{registry: reg, hooks: core.NewHookRegistry()}

	if err := api.RegisterTool(&fakeTool{name: "ext_tool_a"}); err != nil {
		t.Fatalf("RegisterTool() error: %v", err)
	}
	if _, ok := reg.Get("ext_tool_a"); !ok {
		t.Error("RegisterTool() did not register the tool into the registry")
	}
}

func TestExtAPIRegisterHook(t *testing.T) {
	hooks := core.NewHookRegistry()
	api := &extAPI{registry: core.NewRegistry(), hooks: hooks}

	called := false
	handler := func(ctx core.HookContext) error {
		called = true
		return nil
	}
	if err := api.RegisterHook("on_turn_start", handler); err != nil {
		t.Fatalf("RegisterHook() error: %v", err)
	}
	if !hooks.HasHooks("on_turn_start") {
		t.Error("RegisterHook() did not register the hook in the hook registry")
	}
	_ = hooks.Emit("on_turn_start", core.NewHookContext("on_turn_start", nil))
	if !called {
		t.Error("registered hook handler was not invoked by Emit")
	}
}

func TestExtAPIRegisterToolDuplicate(t *testing.T) {
	reg := core.NewRegistry()
	api := &extAPI{registry: reg, hooks: core.NewHookRegistry()}

	if err := api.RegisterTool(&fakeTool{name: "dup"}); err != nil {
		t.Fatalf("first RegisterTool() error: %v", err)
	}
	err := api.RegisterTool(&fakeTool{name: "dup"})
	if err == nil {
		t.Fatal("RegisterTool() duplicate should return an error")
	}
	var dup *core.DuplicateToolError
	if !errors.As(err, &dup) {
		t.Errorf("duplicate error = %v, want *core.DuplicateToolError", err)
	}
	if _, ok := reg.Get("dup"); !ok {
		t.Error("first registration should remain after the duplicate is rejected")
	}
}

func TestExtAPIRegisterCommand(t *testing.T) {
	api := &extAPI{registry: core.NewRegistry(), hooks: core.NewHookRegistry()}

	cmd := core.Command{
		Name:        "ext-cmd",
		Description: "a test command",
		Handler: func(_ context.Context, _ string) (string, error) {
			return "out", nil
		},
	}
	if err := api.RegisterCommand(cmd); err != nil {
		t.Fatalf("RegisterCommand() error: %v", err)
	}
	if len(api.commands) != 1 || api.commands[0].Name != "ext-cmd" {
		t.Errorf("commands = %+v, want [ext-cmd]", api.commands)
	}
}
