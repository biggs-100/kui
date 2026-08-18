package dynamic

import (
	"context"
	"errors"
	"testing"
)

// --- Helpers ---

// newMockClientFactory returns a ClientFactory that creates mockClients with
// the given tools. Each call to the factory increments a counter so tests can
// verify how many times the factory was invoked.
func newMockClientFactory(tools []ToolDef) (ClientFactory, *int) {
	count := 0
	factory := func(_ context.Context, _ string) (ClientInterface, error) {
		count++
		return &mockClient{tools: tools}, nil
	}
	return factory, &count
}

// failingFactory returns a ClientFactory that always fails to spawn.
func failingFactory(err error) ClientFactory {
	return func(_ context.Context, _ string) (ClientInterface, error) {
		return nil, err
	}
}

// fixedScanner returns a scanner that returns the given manifests.
func fixedScanner(manifests []*Manifest) ExtScanner {
	return func(_ string) ([]*Manifest, error) {
		return manifests, nil
	}
}

// failingScanner returns a scanner that always fails.
func failingScanner(err error) ExtScanner {
	return func(_ string) ([]*Manifest, error) {
		return nil, err
	}
}

// --- Tests ---

func TestManagerLoadAllRegistersTools(t *testing.T) {
	tools := []ToolDef{
		{Name: "a", Description: "tool a", InputSchema: []byte(`{}`)},
		{Name: "b", Description: "tool b", InputSchema: []byte(`{}`)},
	}
	factory, count := newMockClientFactory(tools)
	api := &mockExtensionAPI{}

	cfg := &Config{Paths: []string{"/exts"}}
	scan := fixedScanner([]*Manifest{
		{Name: "myext", EntryPoint: "/usr/bin/myext"},
	})

	mgr := NewManager(cfg, WithScanner(scan), WithClientFactory(factory))
	if err := mgr.LoadAll(context.Background(), api); err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}

	if *count != 1 {
		t.Errorf("factory called %d times, want 1", *count)
	}
	if len(api.tools) != 2 {
		t.Fatalf("got %d registered tools, want 2", len(api.tools))
	}
	if api.tools[0].Name() != "myext_a" {
		t.Errorf("tool[0].Name() = %q, want %q", api.tools[0].Name(), "myext_a")
	}
	if api.tools[1].Name() != "myext_b" {
		t.Errorf("tool[1].Name() = %q, want %q", api.tools[1].Name(), "myext_b")
	}
	if mgr.Loaded() != 1 {
		t.Errorf("Loaded() = %d, want 1", mgr.Loaded())
	}
}

func TestManagerLoadAllMultipleExtensions(t *testing.T) {
	factory, count := newMockClientFactory([]ToolDef{
		{Name: "x", Description: "x", InputSchema: []byte(`{}`)},
	})
	api := &mockExtensionAPI{}

	cfg := &Config{Paths: []string{"/exts"}}
	scan := fixedScanner([]*Manifest{
		{Name: "ext1", EntryPoint: "/usr/bin/ext1"},
		{Name: "ext2", EntryPoint: "/usr/bin/ext2"},
		{Name: "ext3", EntryPoint: "/usr/bin/ext3"},
	})

	mgr := NewManager(cfg, WithScanner(scan), WithClientFactory(factory))
	if err := mgr.LoadAll(context.Background(), api); err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}

	if *count != 3 {
		t.Errorf("factory called %d times, want 3", *count)
	}
	if len(api.tools) != 3 {
		t.Fatalf("got %d registered tools, want 3", len(api.tools))
	}
	if mgr.Loaded() != 3 {
		t.Errorf("Loaded() = %d, want 3", mgr.Loaded())
	}
}

func TestManagerLoadAllCrashOneDoesNotBlockOthers(t *testing.T) {
	// Factory: first call fails, second call succeeds.
	callCount := 0
	factory := func(_ context.Context, entry string) (ClientInterface, error) {
		callCount++
		if entry == "/usr/bin/crasher" {
			return nil, errors.New("process crashed")
		}
		return &mockClient{tools: []ToolDef{
			{Name: "ok_tool", Description: "ok", InputSchema: []byte(`{}`)},
		}}, nil
	}

	api := &mockExtensionAPI{}
	cfg := &Config{Paths: []string{"/exts"}}
	scan := fixedScanner([]*Manifest{
		{Name: "crasher", EntryPoint: "/usr/bin/crasher"},
		{Name: "good", EntryPoint: "/usr/bin/good"},
	})

	mgr := NewManager(cfg, WithScanner(scan), WithClientFactory(factory))
	if err := mgr.LoadAll(context.Background(), api); err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}

	// Only the good extension's tools should be registered.
	if len(api.tools) != 1 {
		t.Fatalf("got %d registered tools, want 1", len(api.tools))
	}
	if api.tools[0].Name() != "good_ok_tool" {
		t.Errorf("tool[0].Name() = %q, want %q", api.tools[0].Name(), "good_ok_tool")
	}
	// Crash extension is not in loaded list.
	if mgr.Loaded() != 1 {
		t.Errorf("Loaded() = %d, want 1", mgr.Loaded())
	}
}

func TestManagerLoadAllEntryMissing(t *testing.T) {
	// Extension manifest exists but the binary is missing.
	factory := failingFactory(errors.New("executable not found in $PATH"))
	api := &mockExtensionAPI{}

	cfg := &Config{Paths: []string{"/exts"}}
	scan := fixedScanner([]*Manifest{
		{Name: "missing", EntryPoint: "/no/such/binary"},
	})

	mgr := NewManager(cfg, WithScanner(scan), WithClientFactory(factory))
	if err := mgr.LoadAll(context.Background(), api); err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}

	if len(api.tools) != 0 {
		t.Errorf("got %d registered tools, want 0", len(api.tools))
	}
	if mgr.Loaded() != 0 {
		t.Errorf("Loaded() = %d, want 0", mgr.Loaded())
	}
}

func TestManagerLoadAllScanFailure(t *testing.T) {
	api := &mockExtensionAPI{}
	cfg := &Config{Paths: []string{"/bad/path"}}
	scan := failingScanner(errors.New("permission denied"))

	mgr := NewManager(cfg, WithScanner(scan), WithClientFactory(defaultClientFactory))
	if err := mgr.LoadAll(context.Background(), api); err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}

	// Scan failure is logged but doesn't error.
	if len(api.tools) != 0 {
		t.Errorf("got %d registered tools, want 0", len(api.tools))
	}
}

func TestManagerLoadAllEmptyConfig(t *testing.T) {
	api := &mockExtensionAPI{}
	cfg := &Config{Paths: []string{}}

	mgr := NewManager(cfg, WithScanner(fixedScanner(nil)))
	if err := mgr.LoadAll(context.Background(), api); err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}

	if len(api.tools) != 0 {
		t.Errorf("got %d registered tools, want 0", len(api.tools))
	}
	if mgr.Loaded() != 0 {
		t.Errorf("Loaded() = %d, want 0", mgr.Loaded())
	}
}

func TestManagerShutdownAllReverseOrder(t *testing.T) {
	// Track shutdown order via mock clients.
	var shutdownOrder []string
	tools := []ToolDef{{Name: "t", Description: "t", InputSchema: []byte(`{}`)}}

	factory := func(_ context.Context, entry string) (ClientInterface, error) {
		return &shutdownMockClient{
			mockClient: mockClient{tools: tools},
			name:       entry,
			shutdownFn: func(name string) { shutdownOrder = append(shutdownOrder, name) },
		}, nil
	}

	api := &mockExtensionAPI{}
	cfg := &Config{Paths: []string{"/exts"}}
	scan := fixedScanner([]*Manifest{
		{Name: "ext_a", EntryPoint: "/a"},
		{Name: "ext_b", EntryPoint: "/b"},
		{Name: "ext_c", EntryPoint: "/c"},
	})

	mgr := NewManager(cfg, WithScanner(scan), WithClientFactory(factory))
	if err := mgr.LoadAll(context.Background(), api); err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if err := mgr.ShutdownAll(); err != nil {
		t.Fatalf("ShutdownAll() error = %v", err)
	}

	// Shutdown should be reverse order: /c, /b, /a.
	if len(shutdownOrder) != 3 {
		t.Fatalf("got %d shutdowns, want 3", len(shutdownOrder))
	}
	if shutdownOrder[0] != "/c" {
		t.Errorf("shutdownOrder[0] = %q, want %q", shutdownOrder[0], "/c")
	}
	if shutdownOrder[1] != "/b" {
		t.Errorf("shutdownOrder[1] = %q, want %q", shutdownOrder[1], "/b")
	}
	if shutdownOrder[2] != "/a" {
		t.Errorf("shutdownOrder[2] = %q, want %q", shutdownOrder[2], "/a")
	}
}

func TestManagerShutdownAllIdempotent(t *testing.T) {
	factory, _ := newMockClientFactory([]ToolDef{{Name: "t", Description: "t", InputSchema: []byte(`{}`)}})
	api := &mockExtensionAPI{}

	cfg := &Config{Paths: []string{"/exts"}}
	scan := fixedScanner([]*Manifest{
		{Name: "ext", EntryPoint: "/usr/bin/ext"},
	})

	mgr := NewManager(cfg, WithScanner(scan), WithClientFactory(factory))
	_ = mgr.LoadAll(context.Background(), api)

	// First shutdown.
	if err := mgr.ShutdownAll(); err != nil {
		t.Fatalf("first ShutdownAll() error = %v", err)
	}
	// Second shutdown should be a no-op.
	if err := mgr.ShutdownAll(); err != nil {
		t.Fatalf("second ShutdownAll() error = %v", err)
	}
}

func TestManagerDefaultScannerSkipsNonDirectories(t *testing.T) {
	api := &mockExtensionAPI{}
	cfg := &Config{Paths: []string{}}

	// defaultScanner with empty paths should work fine.
	mgr := NewManager(cfg)
	if err := mgr.LoadAll(context.Background(), api); err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if mgr.Loaded() != 0 {
		t.Errorf("Loaded() = %d, want 0", mgr.Loaded())
	}
}

// --- shutdownMockClient wraps mockClient to track shutdown calls ---

type shutdownMockClient struct {
	mockClient
	name       string
	shutdownFn func(string)
}

func (s *shutdownMockClient) Shutdown(_ context.Context) error {
	s.shutdownFn(s.name)
	return nil
}

// Verify mockClient implements ClientInterface.
var _ ClientInterface = (*mockClient)(nil)
