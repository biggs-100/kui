package core

import (
	"errors"
	"strings"
	"testing"
)

// TestHookRegistryRegisterOrderPreserved verifies that handlers execute in
// the order they were registered (D3, REQ-HOOK-1).
func TestHookRegistryRegisterOrderPreserved(t *testing.T) {
	registry := NewHookRegistry()
	var order []int

	h1 := func(ctx HookContext) error { order = append(order, 1); return nil }
	h2 := func(ctx HookContext) error { order = append(order, 2); return nil }
	h3 := func(ctx HookContext) error { order = append(order, 3); return nil }

	if err := registry.Register("test", h1); err != nil {
		t.Fatalf("Register(h1) failed: %v", err)
	}
	if err := registry.Register("test", h2); err != nil {
		t.Fatalf("Register(h2) failed: %v", err)
	}
	if err := registry.Register("test", h3); err != nil {
		t.Fatalf("Register(h3) failed: %v", err)
	}

	ctx := NewHookContext("test", nil)
	if err := registry.Emit("test", ctx); err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	if len(order) != 3 {
		t.Fatalf("handlers called %d times, want 3", len(order))
	}
	for i, got := range order {
		if got != i+1 {
			t.Errorf("handler[%d] = %d, want %d", i, got, i+1)
		}
	}
}

// TestHookRegistryEmitOrderMatchesRegistration verifies that emit order
// exactly matches registration order (REQ-HOOK-2).
func TestHookRegistryEmitOrderMatchesRegistration(t *testing.T) {
	registry := NewHookRegistry()
	var names []string

	registry.Register("ev", func(ctx HookContext) error {
		names = append(names, "first")
		return nil
	})
	registry.Register("ev", func(ctx HookContext) error {
		names = append(names, "second")
		return nil
	})
	registry.Register("ev", func(ctx HookContext) error {
		names = append(names, "third")
		return nil
	})

	registry.Emit("ev", NewHookContext("ev", nil))

	want := []string{"first", "second", "third"}
	if len(names) != len(want) {
		t.Fatalf("got %d calls, want %d", len(names), len(want))
	}
	for i, n := range names {
		if n != want[i] {
			t.Errorf("call[%d] = %q, want %q", i, n, want[i])
		}
	}
}

// TestHookRegistryNilHandlerRejected verifies that registering a nil handler
// returns an error (REQ-HOOK-1).
func TestHookRegistryNilHandlerRejected(t *testing.T) {
	registry := NewHookRegistry()

	err := registry.Register("test", nil)
	if err == nil {
		t.Fatal("Register(nil) returned nil, want error")
	}
	if !strings.Contains(err.Error(), "must not be nil") {
		t.Errorf("error = %q, want it to contain 'must not be nil'", err.Error())
	}
}

// TestHookRegistryErrorShortCircuit verifies that when a handler returns an
// error, subsequent handlers are not called and the error is returned
// (REQ-HOOK-3).
func TestHookRegistryErrorShortCircuit(t *testing.T) {
	registry := NewHookRegistry()
	var order []int
	cause := errors.New("handler failed")

	registry.Register("test", func(ctx HookContext) error {
		order = append(order, 1)
		return nil
	})
	registry.Register("test", func(ctx HookContext) error {
		order = append(order, 2)
		return cause
	})
	registry.Register("test", func(ctx HookContext) error {
		order = append(order, 3)
		return nil
	})

	err := registry.Emit("test", NewHookContext("test", nil))
	if err == nil {
		t.Fatal("Emit returned nil, want error")
	}
	if !errors.Is(err, cause) {
		t.Errorf("Emit error = %v, want errors.Is to match %v", err, cause)
	}

	// Third handler must not have been called
	if len(order) != 2 {
		t.Fatalf("handlers called %d times, want 2 (third should be skipped)", len(order))
	}
	if order[0] != 1 || order[1] != 2 {
		t.Errorf("order = %v, want [1, 2]", order)
	}
}

// TestHookRegistryHasHooksTrueWhenRegistered verifies HasHooks returns true
// when at least one handler is registered for the event (REQ-HOOK-4).
func TestHookRegistryHasHooksTrueWhenRegistered(t *testing.T) {
	registry := NewHookRegistry()

	if registry.HasHooks("test") {
		t.Error("HasHooks returned true before any registration")
	}

	registry.Register("test", func(ctx HookContext) error { return nil })

	if !registry.HasHooks("test") {
		t.Error("HasHooks returned false after registration")
	}
}

// TestHookRegistryHasHooksFalseForDifferentEvent verifies HasHooks only
// matches the exact event name.
func TestHookRegistryHasHooksFalseForDifferentEvent(t *testing.T) {
	registry := NewHookRegistry()
	registry.Register("event_a", func(ctx HookContext) error { return nil })

	if registry.HasHooks("event_b") {
		t.Error("HasHooks returned true for a different event")
	}
}

// TestHookRegistryEmitUnknownEventNoOp verifies that emitting an event with
// no handlers returns nil.
func TestHookRegistryEmitUnknownEventNoOp(t *testing.T) {
	registry := NewHookRegistry()

	err := registry.Emit("nonexistent", NewHookContext("nonexistent", nil))
	if err != nil {
		t.Errorf("Emit on unknown event returned %v, want nil", err)
	}
}

// --- Nil-safety tests (D7) ---

// TestHookRegistryNilReceiverRegister verifies that calling Register on a nil
// *HookRegistry returns an error (not a panic).
func TestHookRegistryNilReceiverRegister(t *testing.T) {
	var registry *HookRegistry

	err := registry.Register("test", func(ctx HookContext) error { return nil })
	if err == nil {
		t.Fatal("Register on nil registry returned nil, want error")
	}
}

// TestHookRegistryNilReceiverEmit verifies that calling Emit on a nil
// *HookRegistry is a no-op returning nil (D7).
func TestHookRegistryNilReceiverEmit(t *testing.T) {
	var registry *HookRegistry

	err := registry.Emit("test", NewHookContext("test", nil))
	if err != nil {
		t.Errorf("Emit on nil registry returned %v, want nil", err)
	}
}

// TestHookRegistryNilReceiverHasHooks verifies that calling HasHooks on a nil
// *HookRegistry returns false (D7).
func TestHookRegistryNilReceiverHasHooks(t *testing.T) {
	var registry *HookRegistry

	if registry.HasHooks("test") {
		t.Error("HasHooks on nil registry returned true, want false")
	}
}

// TestHookRegistrySeparateEvents verifies that handlers for different events
// are independent.
func TestHookRegistrySeparateEvents(t *testing.T) {
	registry := NewHookRegistry()
	var calls []string

	registry.Register("event_a", func(ctx HookContext) error {
		calls = append(calls, "a")
		return nil
	})
	registry.Register("event_b", func(ctx HookContext) error {
		calls = append(calls, "b")
		return nil
	})

	registry.Emit("event_a", NewHookContext("event_a", nil))

	if len(calls) != 1 || calls[0] != "a" {
		t.Errorf("calls = %v, want [a] (event_b handler must not fire)", calls)
	}
}
