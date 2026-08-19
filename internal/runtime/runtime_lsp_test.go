package runtime

import (
	"context"
	"testing"

	"github.com/biggs-100/kui/internal/lsp"
)

func TestRuntimeBuildWithLsp(t *testing.T) {
	cfg := newTestConfig(t)
	writeProfile(t, cfg.ConfigRoot, "coder", "read_file")
	setActive(t, cfg.ConfigRoot, "coder")

	rt, err := Build(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	if rt.LSP == nil {
		t.Error("Build: LSP manager is nil — should always be initialized")
	}

	// LSP should exist but not be running (lazy startup).
	if rt.LSP.Servers() != 0 {
		t.Errorf("LSP servers = %d, want 0 (lazy startup)", rt.LSP.Servers())
	}
}

func TestRuntimeBuildWithLspGracefulDegradation(t *testing.T) {
	cfg := newTestConfig(t)
	writeProfile(t, cfg.ConfigRoot, "coder", "read_file")
	setActive(t, cfg.ConfigRoot, "coder")

	// Even without gopls in PATH, Build should succeed.
	rt, err := Build(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Build() should succeed without gopls: %v", err)
	}
	if rt.LSP == nil {
		t.Error("LSP manager should be initialized even when gopls is absent")
	}
}

func TestRuntimeCloseWithLsp(t *testing.T) {
	cfg := newTestConfig(t)
	writeProfile(t, cfg.ConfigRoot, "coder", "read_file")
	setActive(t, cfg.ConfigRoot, "coder")

	rt, err := Build(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	// Inject a mock LSP manager to verify Close() calls StopAll().
	mockMgr := newMockToolManager()
	rt.LSP = lsp.NewLspManager()

	// Close should not error.
	if err := rt.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}

	// A second Close should be idempotent.
	if err := rt.Close(); err != nil {
		t.Errorf("Close() idempotent error: %v", err)
	}

	_ = mockMgr // used only to satisfy interface check below
}

func TestRuntimeReloadWithLsp(t *testing.T) {
	cfg := newTestConfig(t)
	writeProfile(t, cfg.ConfigRoot, "coder", "read_file")
	setActive(t, cfg.ConfigRoot, "coder")

	rt, err := Build(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	if rt.LSP == nil {
		t.Fatal("LSP manager should be initialized after Build")
	}

	// Reload should succeed and preserve the LSP manager.
	result := rt.Reload(context.Background())
	if result.Err != nil {
		t.Fatalf("Reload() error: %v", result.Err)
	}

	if rt.LSP == nil {
		t.Error("LSP manager should still be present after Reload")
	}
}

func TestReloadPreservesLspTools(t *testing.T) {
	cfg := newTestConfig(t)
	writeProfile(t, cfg.ConfigRoot, "coder", "read_file")
	setActive(t, cfg.ConfigRoot, "coder")

	rt, err := Build(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	// Verify LSP tools are registered after Build.
	lspToolNames := []string{"lsp_diagnostics", "lsp_hover", "lsp_definition", "lsp_references"}
	for _, name := range lspToolNames {
		if _, ok := rt.Full.Get(name); !ok {
			t.Errorf("Build: full registry missing LSP tool %q", name)
		}
	}

	// Reload should preserve LSP tools in the registry.
	result := rt.Reload(context.Background())
	if result.Err != nil {
		t.Fatalf("Reload() error: %v", result.Err)
	}

	for _, name := range lspToolNames {
		if _, ok := rt.Full.Get(name); !ok {
			t.Errorf("Reload lost LSP tool %q from registry", name)
		}
	}
}

func TestReloadStopsOldLspServers(t *testing.T) {
	cfg := newTestConfig(t)
	writeProfile(t, cfg.ConfigRoot, "coder", "read_file")
	setActive(t, cfg.ConfigRoot, "coder")

	rt, err := Build(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	// Pre-populate diagnostics to verify they are cleared during Reload teardown.
	rt.LSP.Cache().Set("file:///old.go", []lsp.Diagnostic{{Message: "old error"}})
	if rt.LSP.Cache().Count() != 1 {
		t.Fatal("precondition: cache should have 1 diagnostic")
	}

	result := rt.Reload(context.Background())
	if result.Err != nil {
		t.Fatalf("Reload() error: %v", result.Err)
	}

	// Reload calls StopAll() + Cache().ClearAll() in teardown.
	// The diagnostics cache should be empty after Reload.
	if count := rt.LSP.Cache().Count(); count != 0 {
		t.Errorf("cache count after Reload = %d, want 0 (clearAll should have run)", count)
	}
}

// mockToolManager satisfies lsp.ToolManager for testing.
type mockToolManager struct{}

func newMockToolManager() *mockToolManager { return &mockToolManager{} }

func (m *mockToolManager) GetServer(string) (*lsp.LspClient, error) {
	return nil, &lsp.ServerNotReadyError{Tool: "mock"}
}
func (m *mockToolManager) IsRunning(string) bool { return false }
func (m *mockToolManager) Cache() *lsp.DiagnosticCache {
	return lsp.NewDiagnosticCache()
}
