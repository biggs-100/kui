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
	initSeq        int // assigned during Init for ordering assertions
}

var seqCounter int // global sequence counter for ordering

func (m *mockExtension) Name() string { return m.name }

func (m *mockExtension) Init(_ core.ExtensionAPI) error {
	m.initCalled = true
	seqCounter++
	m.initSeq = seqCounter
	return m.initErr
}

func (m *mockExtension) Shutdown() error {
	m.shutdownCalled = true
	return m.shutdownErr
}

// initOrderBefore returns true if this extension was initialized before other.
func (m *mockExtension) initOrderBefore(other *mockExtension) bool {
	return m.initSeq > 0 && other.initSeq > 0 && m.initSeq < other.initSeq
}

// mockAPI is a no-op ExtensionAPI used by LoadAll in tests.
type mockAPI struct{}

func (m *mockAPI) RegisterTool(_ core.Tool) error                       { return nil }
func (m *mockAPI) RegisterHook(_ string, _ core.HookHandler) error      { return nil }
func (m *mockAPI) RegisterCommand(_ core.Command) error                 { return nil }

// reset clears all package-level state between tests.
func reset() {
	global = nil
	dynamic = nil
	loaded = nil
	seqCounter = 0
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

// --- dynamic extension tests ---

func TestRegisterDynamicAppends(t *testing.T) {
	reset()
	ext := &mockExtension{name: "dyn-alpha"}
	RegisterDynamic(ext)

	if len(dynamic) != 1 {
		t.Fatalf("expected 1 registered dynamic extension, got %d", len(dynamic))
	}
	if dynamic[0] != ext {
		t.Fatal("registered dynamic extension pointer does not match")
	}
}

func TestRegisterDynamicMultiple(t *testing.T) {
	reset()
	a := &mockExtension{name: "dynA"}
	b := &mockExtension{name: "dynB"}
	RegisterDynamic(a)
	RegisterDynamic(b)

	if len(dynamic) != 2 {
		t.Fatalf("expected 2 registered dynamic extensions, got %d", len(dynamic))
	}
	if dynamic[0] != a || dynamic[1] != b {
		t.Fatal("dynamic extensions not in registration order")
	}
}

func TestRegisterDynamicNilPanics(t *testing.T) {
	reset()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil dynamic extension, did not panic")
		}
	}()
	RegisterDynamic(nil)
}

func TestLoadAllProcessesGlobalThenDynamic(t *testing.T) {
	reset()
	g1 := &mockExtension{name: "global-1"}
	g2 := &mockExtension{name: "global-2"}
	d1 := &mockExtension{name: "dyn-1"}
	d2 := &mockExtension{name: "dyn-2"}
	Register(g1)
	Register(g2)
	RegisterDynamic(d1)
	RegisterDynamic(d2)

	api := &mockAPI{}
	if err := LoadAll(api); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !g1.initCalled || !g2.initCalled || !d1.initCalled || !d2.initCalled {
		t.Fatal("not all extensions (global + dynamic) were initialized")
	}

	// Verify ordering: global inits before dynamic inits.
	if !g1.initOrderBefore(g2) {
		t.Fatal("global-1 should init before global-2")
	}
	if !g2.initOrderBefore(d1) {
		t.Fatal("global-2 should init before dyn-1")
	}
	if !d1.initOrderBefore(d2) {
		t.Fatal("dyn-1 should init before dyn-2")
	}
}

func TestLoadAllDynamicOnly(t *testing.T) {
	reset()
	d1 := &mockExtension{name: "dyn-only-1"}
	d2 := &mockExtension{name: "dyn-only-2"}
	RegisterDynamic(d1)
	RegisterDynamic(d2)

	api := &mockAPI{}
	if err := LoadAll(api); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !d1.initCalled || !d2.initCalled {
		t.Fatal("not all dynamic extensions were initialized")
	}
}

func TestLoadAllRollbackOnDynamicFailure(t *testing.T) {
	reset()
	g1 := &mockExtension{name: "global-1"}
	d1 := &mockExtension{name: "dyn-1"}
	d2 := &mockExtension{name: "dyn-2", initErr: errors.New("dyn-2 init failed")}
	d3 := &mockExtension{name: "dyn-3"}
	Register(g1)
	RegisterDynamic(d1)
	RegisterDynamic(d2)
	RegisterDynamic(d3)

	api := &mockAPI{}
	err := LoadAll(api)
	if err == nil {
		t.Fatal("expected error from LoadAll, got nil")
	}
	if err.Error() != "dyn-2 init failed" {
		t.Fatalf("unexpected error: %v", err)
	}

	// g1 and d1 were initialized before the failure — both should be shut down.
	if !g1.initCalled {
		t.Fatal("global-1.Init should have been called")
	}
	if !g1.shutdownCalled {
		t.Fatal("global-1 should have been shut down during rollback")
	}
	if !d1.initCalled {
		t.Fatal("dyn-1.Init should have been called")
	}
	if !d1.shutdownCalled {
		t.Fatal("dyn-1 should have been shut down during rollback")
	}

	// d2 failed, should NOT be shut down.
	if d2.shutdownCalled {
		t.Fatal("dyn-2 should not be shut down (its Init failed)")
	}
	// d3 was never reached.
	if d3.initCalled {
		t.Fatal("dyn-3.Init should not have been called")
	}
}

func TestShutdownAllProcessesDynamicInReverseOrder(t *testing.T) {
	reset()
	g1 := &mockExtension{name: "global-1"}
	d1 := &mockExtension{name: "dyn-1"}
	d2 := &mockExtension{name: "dyn-2"}
	Register(g1)
	RegisterDynamic(d1)
	RegisterDynamic(d2)

	api := &mockAPI{}
	if err := LoadAll(api); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	if err := ShutdownAll(); err != nil {
		t.Fatalf("ShutdownAll: %v", err)
	}

	// Shutdown order: dyn-2, dyn-1, global-1 (reverse of init order).
	if !d2.shutdownCalled {
		t.Fatal("dyn-2.Shutdown should have been called")
	}
	if !d1.shutdownCalled {
		t.Fatal("dyn-1.Shutdown should have been called")
	}
	if !g1.shutdownCalled {
		t.Fatal("global-1.Shutdown should have been called")
	}
}

func TestLoadAllEmptyDynamicRegistry(t *testing.T) {
	reset()
	// Only global, no dynamic — should still work.
	g1 := &mockExtension{name: "global-only"}
	Register(g1)

	api := &mockAPI{}
	if err := LoadAll(api); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !g1.initCalled {
		t.Fatal("global extension was not initialized")
	}
}

// --- helpers ---

// containsSubstring is a simple helper to avoid importing strings.
func containsSubstring(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
