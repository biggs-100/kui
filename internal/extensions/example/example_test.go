package example

import (
	"testing"

	"github.com/biggs-100/kui/internal/core"
)

// mockAPI is a no-op ExtensionAPI for testing Init behavior.
type mockAPI struct {
	toolsRegistered  int
	hooksRegistered  int
	commandsRegistered int
}

func (m *mockAPI) RegisterTool(_ core.Tool) error {
	m.toolsRegistered++
	return nil
}

func (m *mockAPI) RegisterHook(_ string, _ core.HookHandler) error {
	m.hooksRegistered++
	return nil
}

func (m *mockAPI) RegisterCommand(_ core.Command) error {
	m.commandsRegistered++
	return nil
}

func TestExampleName(t *testing.T) {
	ext := &exampleExtension{}
	if got := ext.Name(); got != "example" {
		t.Fatalf("Name() = %q, want %q", got, "example")
	}
}

func TestExampleInitRegistersTool(t *testing.T) {
	ext := &exampleExtension{}
	api := &mockAPI{}
	if err := ext.Init(api); err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}
	if api.toolsRegistered != 1 {
		t.Fatalf("expected 1 tool registered, got %d", api.toolsRegistered)
	}
}

func TestExampleInitRegistersHook(t *testing.T) {
	ext := &exampleExtension{}
	api := &mockAPI{}
	if err := ext.Init(api); err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}
	if api.hooksRegistered != 1 {
		t.Fatalf("expected 1 hook registered, got %d", api.hooksRegistered)
	}
}

func TestExampleShutdownNoError(t *testing.T) {
	ext := &exampleExtension{}
	if err := ext.Shutdown(); err != nil {
		t.Fatalf("Shutdown() returned error: %v", err)
	}
}

func TestExampleImplementsExtension(t *testing.T) {
	// Compile-time check that exampleExtension satisfies core.Extension.
	var _ core.Extension = &exampleExtension{}
}
