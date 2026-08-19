package plugin

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNewPluginRegistry(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)
	if r == nil {
		t.Fatal("NewPluginRegistry returned nil")
	}
	if r.discovery != d {
		t.Error("discovery not set correctly")
	}
	if r.plugins == nil {
		t.Error("plugins map not initialized")
	}
}

func TestLoadEmpty(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)
	if err := r.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(r.plugins) != 0 {
		t.Errorf("expected 0 plugins after load, got %d", len(r.plugins))
	}
}

func TestLoadValidPlugins(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()

	// Create a valid plugin in global dir
	pluginDir := filepath.Join(globalDir, "my-tool")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	writePluginManifest(t, pluginDir, `name: my-tool
version: "1.0.0"
type: tool
entry_point: ./bin/run
`)

	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)
	if err := r.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(r.plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(r.plugins))
	}

	p, ok := r.plugins["my-tool"]
	if !ok {
		t.Fatal("plugin 'my-tool' not found in registry")
	}
	if p.Manifest.Name != "my-tool" {
		t.Errorf("Manifest.Name = %q, want %q", p.Manifest.Name, "my-tool")
	}
	if p.Manifest.Version != "1.0.0" {
		t.Errorf("Manifest.Version = %q, want %q", p.Manifest.Version, "1.0.0")
	}
	if p.State != PluginStateLoaded {
		t.Errorf("State = %q, want %q", p.State, PluginStateLoaded)
	}
	if p.Dir != pluginDir {
		t.Errorf("Dir = %q, want %q", p.Dir, pluginDir)
	}
	if p.LoadedAt.IsZero() {
		t.Error("LoadedAt should not be zero")
	}
}

func TestGetFound(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()

	pluginDir := filepath.Join(globalDir, "test-plugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	writePluginManifest(t, pluginDir, `name: test-plugin
version: "1.0.0"
type: hook
entry_point: ./hook.sh
`)

	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)
	if err := r.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	p, err := r.Get("test-plugin")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if p.Manifest.Name != "test-plugin" {
		t.Errorf("Manifest.Name = %q, want %q", p.Manifest.Name, "test-plugin")
	}
}

func TestGetNotFound(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)
	if err := r.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	_, err := r.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent plugin, got nil")
	}
}

func TestListEmpty(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)
	if err := r.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	plugins := r.List()
	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(plugins))
	}
}

func TestListMultiple(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()

	// Create two plugins
	for _, name := range []string{"plugin-a", "plugin-b"} {
		pluginDir := filepath.Join(globalDir, name)
		if err := os.MkdirAll(pluginDir, 0755); err != nil {
			t.Fatal(err)
		}
		writePluginManifest(t, pluginDir, `name: `+name+`
version: "1.0.0"
type: tool
entry_point: ./bin/run
`)
	}

	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)
	if err := r.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	plugins := r.List()
	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}

	names := make(map[string]bool)
	for _, p := range plugins {
		names[p.Manifest.Name] = true
	}
	if !names["plugin-a"] {
		t.Error("plugin-a not found in list")
	}
	if !names["plugin-b"] {
		t.Error("plugin-b not found in list")
	}
}

func TestRegisterValid(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)

	p := &Plugin{
		Manifest: PluginManifest{
			Name:       "manual-plugin",
			Version:    "1.0.0",
			Type:       PluginTool,
			EntryPoint: "./bin/run",
		},
		State:    PluginStateLoaded,
		Dir:      t.TempDir(),
		LoadedAt: time.Now(),
	}

	if err := r.Register(p); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	got, err := r.Get("manual-plugin")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Manifest.Name != "manual-plugin" {
		t.Errorf("Manifest.Name = %q, want %q", got.Manifest.Name, "manual-plugin")
	}
}

func TestRegisterDuplicate(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)

	p := &Plugin{
		Manifest: PluginManifest{
			Name:       "dup-plugin",
			Version:    "1.0.0",
			Type:       PluginTool,
			EntryPoint: "./bin/run",
		},
		State: PluginStateLoaded,
	}

	if err := r.Register(p); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Register same name again — should fail
	p2 := &Plugin{
		Manifest: PluginManifest{
			Name:       "dup-plugin",
			Version:    "2.0.0",
			Type:       PluginTool,
			EntryPoint: "./bin/run2",
		},
		State: PluginStateLoaded,
	}

	if err := r.Register(p2); err == nil {
		t.Fatal("expected error for duplicate registration, got nil")
	}
}

func TestUnregisterFound(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)

	p := &Plugin{
		Manifest: PluginManifest{
			Name:       "removable",
			Version:    "1.0.0",
			Type:       PluginTool,
			EntryPoint: "./bin/run",
		},
		State: PluginStateLoaded,
	}

	if err := r.Register(p); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if err := r.Unregister("removable"); err != nil {
		t.Fatalf("Unregister() error = %v", err)
	}

	_, err := r.Get("removable")
	if err == nil {
		t.Fatal("expected error after unregister, got nil")
	}
	if len(r.plugins) != 0 {
		t.Errorf("expected 0 plugins after unregister, got %d", len(r.plugins))
	}
}

func TestUnregisterNotFound(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)

	if err := r.Unregister("nonexistent"); err == nil {
		t.Fatal("expected error for unregistering nonexistent plugin, got nil")
	}
}

func TestRegisterNilPlugin(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)

	if err := r.Register(nil); err == nil {
		t.Fatal("expected error for nil plugin, got nil")
	}
}

func TestRegistryConcurrency(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)

	var wg sync.WaitGroup
	// Concurrent registrations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := "concurrent-" + string(rune('a'+idx))
			p := &Plugin{
				Manifest: PluginManifest{
					Name:       name,
					Version:    "1.0.0",
					Type:       PluginTool,
					EntryPoint: "./bin/run",
				},
				State: PluginStateLoaded,
			}
			_ = r.Register(p)
		}(i)
	}
	wg.Wait()

	plugins := r.List()
	if len(plugins) != 10 {
		t.Errorf("expected 10 concurrent registrations, got %d", len(plugins))
	}
}

func TestPluginStates(t *testing.T) {
	states := []PluginState{PluginStateDisloaded, PluginStateLoading, PluginStateLoaded, PluginStateError}
	seen := make(map[PluginState]bool)
	for _, s := range states {
		if s == "" {
			t.Error("PluginState should not be empty")
		}
		seen[s] = true
	}
	if len(seen) != 4 {
		t.Errorf("expected 4 plugin states, got %d", len(seen))
	}
}
