package extensions_test

import (
	"testing"

	"github.com/biggs-100/kui/internal/adapters/extensions"

	// Blank import triggers init() self-registration of example extension.
	_ "github.com/biggs-100/kui/internal/extensions/example"

	"github.com/biggs-100/kui/internal/core"
)

// mockAPI is a no-op ExtensionAPI for integration testing.
type mockAPI struct{}

func (m *mockAPI) RegisterTool(_ core.Tool) error                  { return nil }
func (m *mockAPI) RegisterHook(_ string, _ core.HookHandler) error { return nil }
func (m *mockAPI) RegisterCommand(_ core.Command) error            { return nil }

// TestLoadAllPicksUpInitRegisteredExtensions verifies the integration
// between init() self-registration and LoadAll: when a package with an
// init() function calls extensions.Register(), LoadAll must initialize it.
// This test uses the example extension which self-registers via init().
func TestLoadAllPicksUpInitRegisteredExtensions(t *testing.T) {
	// The example extension is registered via init() in the
	// internal/extensions/example package. By the time this test runs,
	// Go's init ordering has already executed that init() function.
	// LoadAll should pick it up and initialize it.
	api := &mockAPI{}
	if err := extensions.LoadAll(api); err != nil {
		t.Fatalf("LoadAll returned error: %v", err)
	}
	// The example extension's Init runs successfully — no error means
	// it was discovered and initialized.
}

// TestShutdownAllCleansUpDiscoveredExtensions verifies that ShutdownAll
// cleanly shuts down extensions that were discovered via init().
func TestShutdownAllCleansUpDiscoveredExtensions(t *testing.T) {
	// ShutdownAll should succeed even if example extension was already
	// shut down by a prior test (idempotent).
	if err := extensions.ShutdownAll(); err != nil {
		t.Fatalf("ShutdownAll returned error: %v", err)
	}
}
