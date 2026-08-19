package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverEmptyDirs(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()

	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	plugins, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(plugins))
	}
}

func TestDiscoverValidPlugins(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()

	// Create a plugin in global dir
	globalPluginDir := filepath.Join(globalDir, "global-plugin")
	if err := os.MkdirAll(globalPluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	writePluginManifest(t, globalPluginDir, `name: global-plugin
version: "1.0.0"
type: tool
entry_point: ./bin/run
`)

	// Create a plugin in project dir
	projectPluginDir := filepath.Join(projectDir, "project-plugin")
	if err := os.MkdirAll(projectPluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	writePluginManifest(t, projectPluginDir, `name: project-plugin
version: "2.0.0"
type: hook
entry_point: ./hook.sh
`)

	// Debug: check that files exist
	t.Logf("globalDir contents: %v", listDir(t, globalDir))
	t.Logf("projectDir contents: %v", listDir(t, projectDir))
	t.Logf("globalPluginDir exists: %v", dirExists(t, globalPluginDir))
	t.Logf("projectPluginDir exists: %v", dirExists(t, projectPluginDir))

	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	// Debug: test discoverDir directly
	globalPlugins, err := d.discoverDir(globalDir)
	t.Logf("globalPlugins: %d, err: %v", len(globalPlugins), err)
	for _, p := range globalPlugins {
		t.Logf("  global plugin: %s", p.Name)
	}

	projectPlugins, err := d.discoverDir(projectDir)
	t.Logf("projectPlugins: %d, err: %v", len(projectPlugins), err)
	for _, p := range projectPlugins {
		t.Logf("  project plugin: %s", p.Name)
	}

	plugins, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	t.Logf("total plugins: %d", len(plugins))
	for _, p := range plugins {
		t.Logf("  plugin: %s v%s", p.Name, p.Version)
	}
	if len(plugins) != 2 {
		t.Errorf("expected 2 plugins, got %d", len(plugins))
	}

	names := make(map[string]bool)
	for _, p := range plugins {
		names[p.Name] = true
	}
	if !names["global-plugin"] {
		t.Error("global-plugin not found")
	}
	if !names["project-plugin"] {
		t.Error("project-plugin not found")
	}
}

func TestDiscoverPriorityProjectOverridesGlobal(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()

	// Create same-named plugin in both dirs
	for _, dir := range []string{globalDir, projectDir} {
		pluginDir := filepath.Join(dir, "shared-plugin")
		if err := os.MkdirAll(pluginDir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Global version 1.0.0
	writePluginManifest(t, filepath.Join(globalDir, "shared-plugin"), `name: shared-plugin
version: "1.0.0"
type: tool
entry_point: ./bin/run
`)

	// Project version 2.0.0
	writePluginManifest(t, filepath.Join(projectDir, "shared-plugin"), `name: shared-plugin
version: "2.0.0"
type: tool
entry_point: ./bin/run
`)

	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	plugins, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin (project overrides global), got %d", len(plugins))
	}
	if plugins[0].Version != "2.0.0" {
		t.Errorf("Version = %q, want %q (project should override global)", plugins[0].Version, "2.0.0")
	}
}

func TestDiscoverEnvOverride(t *testing.T) {
	projectDir := t.TempDir()
	envDir := t.TempDir()

	// Create plugin in env dir
	pluginDir := filepath.Join(envDir, "env-plugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	writePluginManifest(t, pluginDir, `name: env-plugin
version: "3.0.0"
type: command
entry_point: ./cmd.sh
`)

	d := NewPluginDiscovery(projectDir)
	d.globalDir = t.TempDir() // empty global
	d.projectDir = projectDir
	d.envOverride = envDir

	plugins, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin from env dir, got %d", len(plugins))
	}
	if plugins[0].Name != "env-plugin" {
		t.Errorf("Name = %q, want %q", plugins[0].Name, "env-plugin")
	}
}

func TestDiscoverMissingDirs(t *testing.T) {
	d := NewPluginDiscovery("/nonexistent/project")
	d.globalDir = "/nonexistent/global"
	d.envOverride = "" // no env

	plugins, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() should not error on missing dirs, got: %v", err)
	}
	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(plugins))
	}
}

func TestDiscoverInvalidManifestSkipped(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()

	// Create plugin with invalid manifest
	badPluginDir := filepath.Join(globalDir, "bad-plugin")
	if err := os.MkdirAll(badPluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	writePluginManifest(t, badPluginDir, `name: bad-plugin
`)

	// Create valid plugin
	goodPluginDir := filepath.Join(globalDir, "good-plugin")
	if err := os.MkdirAll(goodPluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	writePluginManifest(t, goodPluginDir, `name: good-plugin
version: "1.0.0"
type: tool
entry_point: ./bin/run
`)

	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	plugins, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	// Invalid manifest should be skipped, not fail
	if len(plugins) != 1 {
		t.Fatalf("expected 1 valid plugin, got %d", len(plugins))
	}
	if plugins[0].Name != "good-plugin" {
		t.Errorf("Name = %q, want %q", plugins[0].Name, "good-plugin")
	}
}

func TestDiscoverHiddenDirsIgnored(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()

	// Create hidden plugin dir (starts with .)
	hiddenDir := filepath.Join(globalDir, ".hidden-plugin")
	if err := os.MkdirAll(hiddenDir, 0755); err != nil {
		t.Fatal(err)
	}
	writePluginManifest(t, hiddenDir, `name: hidden-plugin
version: "1.0.0"
type: tool
entry_point: ./bin/run
`)

	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	plugins, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins (hidden dirs ignored), got %d", len(plugins))
	}
}

func TestDiscoverSymlinkDirsIgnored(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()

	// Create a real plugin dir
	realDir := filepath.Join(globalDir, "real-plugin")
	if err := os.MkdirAll(realDir, 0755); err != nil {
		t.Fatal(err)
	}
	writePluginManifest(t, realDir, `name: real-plugin
version: "1.0.0"
type: tool
entry_point: ./bin/run
`)

	// Create symlink to it
	symlinkDir := filepath.Join(globalDir, "symlink-plugin")
	if err := os.Symlink(realDir, symlinkDir); err != nil {
		t.Skip("symlinks not supported on this platform")
	}

	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	plugins, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	// Symlinks should be ignored for security
	if len(plugins) != 1 {
		t.Errorf("expected 1 plugin (symlink ignored), got %d", len(plugins))
	}
}

func TestDiscoverLegacyExtensionYAML(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()

	// Create plugin with extension.yaml (no kui-plugin.yaml)
	pluginDir := filepath.Join(globalDir, "legacy-plugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Write extension.yaml directly
	extPath := filepath.Join(pluginDir, "extension.yaml")
	if err := os.WriteFile(extPath, []byte(`name: legacy-plugin
version: "1.0.0"
protocol_version: kui-ext/1
entry_point: ./bin/legacy
`), 0644); err != nil {
		t.Fatal(err)
	}

	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	plugins, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 legacy plugin, got %d", len(plugins))
	}
	if plugins[0].Name != "legacy-plugin" {
		t.Errorf("Name = %q, want %q", plugins[0].Name, "legacy-plugin")
	}
	if plugins[0].Type != PluginTool {
		t.Errorf("Type = %q, want %q (legacy defaults to tool)", plugins[0].Type, PluginTool)
	}
}

func TestDiscoverKuiPluginYAMLPrecedence(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()

	// Create plugin with both kui-plugin.yaml and extension.yaml
	pluginDir := filepath.Join(globalDir, "dual-plugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	writePluginManifest(t, pluginDir, `name: dual-plugin
version: "2.0.0"
type: hook
entry_point: ./hook.sh
`)
	extPath := filepath.Join(pluginDir, "extension.yaml")
	if err := os.WriteFile(extPath, []byte(`name: dual-plugin
version: "1.0.0"
protocol_version: kui-ext/1
entry_point: ./old-hook.sh
`), 0644); err != nil {
		t.Fatal(err)
	}

	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	plugins, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	// kui-plugin.yaml should take precedence
	if plugins[0].Version != "2.0.0" {
		t.Errorf("Version = %q, want %q (kui-plugin.yaml should take precedence)", plugins[0].Version, "2.0.0")
	}
	if plugins[0].Type != PluginHook {
		t.Errorf("Type = %q, want %q (from kui-plugin.yaml)", plugins[0].Type, PluginHook)
	}
}

func listDir(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func dirExists(t *testing.T, path string) bool {
	t.Helper()
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
