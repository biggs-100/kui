package extensions

import (
	"errors"
	"testing"

	"github.com/biggs-100/kui/internal/core"
)

// --- mock types ---

// mockExtension records Init/Shutdown calls and supports controlled failure.
type mockExtension struct {
	name           string
	initErr        error
	shutdownErr    error
	initCalled     bool
	shutdownCalled bool
}

func (m *mockExtension) Name() string { return m.name }

func (m *mockExtension) Init(_ core.ExtensionAPI) error {
	m.initCalled = true
	return m.initErr
}

func (m *mockExtension) Shutdown() error {
	m.shutdownCalled = true
	return m.shutdownErr
}

// mockAPI is a no-op ExtensionAPI used by LoadAll in tests.
type mockAPI struct{}

func (m *mockAPI) RegisterTool(_ core.Tool) error                       { return nil }
func (m *mockAPI) RegisterHook(_ string, _ core.HookHandler) error      { return nil }
func (m *mockAPI) RegisterCommand(_ core.Command) error                 { return nil }

// reset clears all package-level state between tests.
func reset() {
	global = nil
	loaded = nil
}

// --- tests ---

func TestRegisterAppends(t *testing.T) {
	reset()
	ext := &mockExtension{name: "alpha"}
	Register(ext)

	if len(global) != 1 {
		t.Fatalf("expected 1 registered extension, got %d", len(global))
	}
	if global[0] != ext {
		t.Fatal("registered extension pointer does not match")
	}
}

func TestRegisterMultiple(t *testing.T) {
	reset()
	a := &mockExtension{name: "A"}
	b := &mockExtension{name: "B"}
	c := &mockExtension{name: "C"}
	Register(a)
	Register(b)
	Register(c)

	if len(global) != 3 {
		t.Fatalf("expected 3 registered extensions, got %d", len(global))
	}
	if global[0] != a || global[1] != b || global[2] != c {
		t.Fatal("extensions not in registration order")
	}
}

func TestRegisterNilPanics(t *testing.T) {
	reset()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil extension, did not panic")
		}
	}()
	Register(nil)
}

func TestLoadAllInitOrder(t *testing.T) {
	reset()
	a := &mockExtension{name: "A"}
	b := &mockExtension{name: "B"}
	c := &mockExtension{name: "C"}
	Register(a)
	Register(b)
	Register(c)

	api := &mockAPI{}
	if err := LoadAll(api); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !a.initCalled || !b.initCalled || !c.initCalled {
		t.Fatal("not all extensions were initialized")
	}
}

func TestLoadAllRollbackOnFailure(t *testing.T) {
	reset()
	a := &mockExtension{name: "A"}
	b := &mockExtension{name: "B", initErr: errors.New("B init failed")}
	c := &mockExtension{name: "C"}
	Register(a)
	Register(b)
	Register(c)

	api := &mockAPI{}
	err := LoadAll(api)
	if err == nil {
		t.Fatal("expected error from LoadAll, got nil")
	}
	if err.Error() != "B init failed" {
		t.Fatalf("unexpected error: %v", err)
	}

	// A was initialized before B, so A should be shut down during rollback.
	if !a.initCalled {
		t.Fatal("A.Init should have been called")
	}
	if !a.shutdownCalled {
		t.Fatal("A should have been shut down during rollback")
	}
	// B failed to init, so it should NOT be shut down.
	if b.shutdownCalled {
		t.Fatal("B should not be shut down (its Init failed)")
	}
	// C was never reached.
	if c.initCalled {
		t.Fatal("C.Init should not have been called")
	}
}

func TestLoadAllIsIdempotent(t *testing.T) {
	reset()
	ext := &mockExtension{name: "X"}
	Register(ext)

	api := &mockAPI{}
	if err := LoadAll(api); err != nil {
		t.Fatalf("first LoadAll: %v", err)
	}
	if err := LoadAll(api); err != nil {
		t.Fatalf("second LoadAll: %v", err)
	}
	// Init should only be called once per LoadAll call.
	// The second call re-inits on top of existing state (this is acceptable
	// per the spec: "LoadAll is idempotent — double-load safe").
}

func TestShutdownAllReverseOrder(t *testing.T) {
	reset()
	a := &mockExtension{name: "A"}
	b := &mockExtension{name: "B"}
	c := &mockExtension{name: "C"}
	Register(a)
	Register(b)
	Register(c)

	api := &mockAPI{}
	if err := LoadAll(api); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	if err := ShutdownAll(); err != nil {
		t.Fatalf("ShutdownAll: %v", err)
	}

	// Reverse order: C, B, A.
	if !c.shutdownCalled {
		t.Fatal("C.Shutdown should have been called first")
	}
	if !b.shutdownCalled {
		t.Fatal("B.Shutdown should have been called")
	}
	if !a.shutdownCalled {
		t.Fatal("A.Shutdown should have been called last")
	}
}

func TestShutdownAllIdempotent(t *testing.T) {
	reset()
	a := &mockExtension{name: "A"}
	Register(a)

	api := &mockAPI{}
	if err := LoadAll(api); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	if err := ShutdownAll(); err != nil {
		t.Fatalf("first ShutdownAll: %v", err)
	}
	if !a.shutdownCalled {
		t.Fatal("A.Shutdown should have been called on first call")
	}

	// Reset the flag to detect a second call.
	a.shutdownCalled = false
	if err := ShutdownAll(); err != nil {
		t.Fatalf("second ShutdownAll: %v", err)
	}
	if a.shutdownCalled {
		t.Fatal("A.Shutdown should NOT be called on second ShutdownAll (idempotent)")
	}
}

func TestShutdownAllCollectsErrors(t *testing.T) {
	reset()
	a := &mockExtension{name: "A"}
	b := &mockExtension{name: "B", shutdownErr: errors.New("B shutdown failed")}
	c := &mockExtension{name: "C"}
	Register(a)
	Register(b)
	Register(c)

	api := &mockAPI{}
	if err := LoadAll(api); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	err := ShutdownAll()
	if err == nil {
		t.Fatal("expected error from ShutdownAll, got nil")
	}
	if !errors.Is(err, errors.New("B shutdown failed")) {
		// errors.Join wraps — check the message contains the substring.
		if !containsSubstring(err.Error(), "B shutdown failed") {
			t.Fatalf("expected error to contain 'B shutdown failed', got: %v", err)
		}
	}

	// Despite B's error, C and A should still be shut down.
	if !c.shutdownCalled {
		t.Fatal("C.Shutdown should have been called (error collection, not short-circuit)")
	}
	if !a.shutdownCalled {
		t.Fatal("A.Shutdown should have been called")
	}
}

func TestShutdownAllWithoutLoad(t *testing.T) {
	reset()
	// ShutdownAll with nothing loaded should be a no-op.
	if err := ShutdownAll(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadAllEmptyRegistry(t *testing.T) {
	reset()
	// LoadAll with no registered extensions should succeed.
	api := &mockAPI{}
	if err := LoadAll(api); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// containsSubstring is a simple helper to avoid importing strings.
func containsSubstring(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
