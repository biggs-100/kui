package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPluginLifecycle tests the full lifecycle: install → discover → load → execute → uninstall.
func TestPluginLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	projectDir := t.TempDir()
	pluginDir := t.TempDir()
	srcDir := SetupTestPlugin(t, "lifecycle-plugin")

	// Create discovery, registry, and installer.
	discovery := NewPluginDiscovery(projectDir)
	discovery.globalDir = pluginDir
	discovery.projectDir = filepath.Join(projectDir, ".kui", "plugins")

	registry := NewPluginRegistry(discovery)
	installer := NewPluginInstaller(registry, pluginDir)

	// Step 1: Install
	p, err := installer.Install(srcDir)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if p.Manifest.Name != "lifecycle-plugin" {
		t.Errorf("installed plugin name = %q, want %q", p.Manifest.Name, "lifecycle-plugin")
	}

	// Step 2: Discover
	manifests, err := discovery.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	found := false
	for _, m := range manifests {
		if m.Name == "lifecycle-plugin" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Discover() did not find lifecycle-plugin after install")
	}

	// Step 3: Load
	if err := registry.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	loaded, err := registry.Get("lifecycle-plugin")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if loaded.State != PluginStateLoaded {
		t.Errorf("plugin state = %q, want %q", loaded.State, PluginStateLoaded)
	}

	// Step 4: Execute (via command dispatcher)
	dispatcher := NewCommandDispatcher(registry)
	testOutput := "hello from plugin"
	err = dispatcher.Register("greet", "lifecycle-plugin", "Greet command", "", func(args []string) (string, error) {
		return testOutput, nil
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	result, err := dispatcher.Execute("greet", nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != testOutput {
		t.Errorf("Execute() = %q, want %q", result, testOutput)
	}

	// Step 5: Uninstall
	if err := installer.Uninstall("lifecycle-plugin"); err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}
	_, err = registry.Get("lifecycle-plugin")
	if err == nil {
		t.Fatal("expected error after uninstall, got nil")
	}
}

// TestPluginInstallAndRemove verifies install from path, verification, remove,
// and verification that it's gone.
func TestPluginInstallAndRemove(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	projectDir := t.TempDir()
	pluginDir := t.TempDir()
	srcDir := SetupTestPlugin(t, "temp-plugin")

	discovery := NewPluginDiscovery(projectDir)
	discovery.globalDir = pluginDir
	discovery.projectDir = filepath.Join(projectDir, ".kui", "plugins")

	registry := NewPluginRegistry(discovery)
	installer := NewPluginInstaller(registry, pluginDir)

	// Install
	p, err := installer.Install(srcDir)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if p.Manifest.Name != "temp-plugin" {
		t.Errorf("installed name = %q, want %q", p.Manifest.Name, "temp-plugin")
	}

	// Verify installed on filesystem
	installedPath := filepath.Join(pluginDir, "temp-plugin", "kui-plugin.yaml")
	if _, err := os.Stat(installedPath); os.IsNotExist(err) {
		t.Errorf("plugin not installed to %s", installedPath)
	}

	// Verify in registry
	got, err := registry.Get("temp-plugin")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Manifest.Version != "1.0.0" {
		t.Errorf("version = %q, want %q", got.Manifest.Version, "1.0.0")
	}

	// Remove
	if err := installer.Uninstall("temp-plugin"); err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}

	// Verify removed from filesystem
	if _, err := os.Stat(installedPath); !os.IsNotExist(err) {
		t.Errorf("plugin should be removed from filesystem")
	}

	// Verify removed from registry
	_, err = registry.Get("temp-plugin")
	if err == nil {
		t.Fatal("expected error after remove, got nil")
	}
}

// TestPluginPermissionEnforcement verifies that grant/deny permissions are
// enforced correctly in both warn-only and enforce modes.
func TestPluginPermissionEnforcement(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pc := NewPermissionChecker("")

	// Initially unknown
	if pc.Check("test-plugin", "filesystem.read") != PermissionUnknown {
		t.Error("expected unknown before grant")
	}

	// Grant a permission
	if err := pc.Grant("test-plugin", "filesystem.read"); err != nil {
		t.Fatalf("Grant() error = %v", err)
	}
	if pc.Check("test-plugin", "filesystem.read") != PermissionAllowed {
		t.Error("expected allowed after grant")
	}

	// Deny a permission
	if err := pc.Deny("test-plugin", "network.request"); err != nil {
		t.Fatalf("Deny() error = %v", err)
	}
	if pc.Check("test-plugin", "network.request") != PermissionDenied {
		t.Error("expected denied after deny")
	}

	// Enforce mode: denied stays denied
	pc.SetMode(PermissionModeEnforce)
	if pc.Check("test-plugin", "network.request") != PermissionDenied {
		t.Error("expected denied in enforce mode")
	}
	if pc.Check("test-plugin", "filesystem.read") != PermissionAllowed {
		t.Error("expected allowed in enforce mode")
	}

	// Warn-only mode: denied is still denied as a result, but mode is warn-only
	pc.SetMode(PermissionModeWarnOnly)
	if pc.Mode() != PermissionModeWarnOnly {
		t.Errorf("mode = %q, want %q", pc.Mode(), PermissionModeWarnOnly)
	}
	// The Check result is still PermissionDenied — the mode controls
	// whether the caller should block or warn, not the Check result.
	if pc.Check("test-plugin", "network.request") != PermissionDenied {
		t.Error("expected denied in warn-only mode (result is same, caller decides)")
	}
}

// TestPluginCommandExecution verifies that a command can be registered,
// executed, and unregistered cleanly.
func TestPluginCommandExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	projectDir := t.TempDir()
	discovery := NewPluginDiscovery(projectDir)
	discovery.globalDir = t.TempDir()
	discovery.projectDir = t.TempDir()

	registry := NewPluginRegistry(discovery)
	if err := registry.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	dispatcher := NewCommandDispatcher(registry)

	// Register a command
	err := dispatcher.Register("test-cmd", "test-plugin", "A test command", "<args>", func(args []string) (string, error) {
		return "executed: " + joinArgs(args), nil
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Execute
	result, err := dispatcher.Execute("test-cmd", []string{"hello", "world"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != "executed: hello world" {
		t.Errorf("Execute() = %q, want %q", result, "executed: hello world")
	}

	// List should include it
	cmds := dispatcher.List()
	if len(cmds) != 1 {
		t.Fatalf("List() = %d commands, want 1", len(cmds))
	}
	if cmds[0].Name != "test-cmd" {
		t.Errorf("List()[0].Name = %q, want %q", cmds[0].Name, "test-cmd")
	}
	if cmds[0].Plugin != "test-plugin" {
		t.Errorf("List()[0].Plugin = %q, want %q", cmds[0].Plugin, "test-plugin")
	}

	// Unregister
	if err := dispatcher.Unregister("test-cmd"); err != nil {
		t.Fatalf("Unregister() error = %v", err)
	}

	// Execute should fail after unregister
	_, err = dispatcher.Execute("test-cmd", nil)
	if err == nil {
		t.Fatal("expected error after unregister, got nil")
	}
}

// TestPluginManifestValidation verifies that invalid manifests are rejected
// during the install flow.
func TestPluginManifestValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	projectDir := t.TempDir()
	pluginDir := t.TempDir()

	discovery := NewPluginDiscovery(projectDir)
	discovery.globalDir = pluginDir
	discovery.projectDir = filepath.Join(projectDir, ".kui", "plugins")

	registry := NewPluginRegistry(discovery)
	installer := NewPluginInstaller(registry, pluginDir)

	// Create a source with an invalid manifest (missing required fields)
	srcDir := filepath.Join(t.TempDir(), "bad-plugin")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Manifest with missing version
	badManifest := "name: bad-plugin\ntype: tool\nentry_point: ./run\n"
	if err := os.WriteFile(filepath.Join(srcDir, "kui-plugin.yaml"), []byte(badManifest), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := installer.Install(srcDir)
	if err == nil {
		t.Fatal("expected error for invalid manifest, got nil")
	}

	// Verify nothing was registered
	_, err = registry.Get("bad-plugin")
	if err == nil {
		t.Fatal("bad-plugin should not be in registry after failed install")
	}

	// Also test: missing entry_point
	srcDir2 := filepath.Join(t.TempDir(), "bad-plugin-2")
	if err := os.MkdirAll(srcDir2, 0o755); err != nil {
		t.Fatal(err)
	}
	badManifest2 := "name: bad-plugin-2\nversion: \"1.0.0\"\ntype: tool\n"
	if err := os.WriteFile(filepath.Join(srcDir2, "kui-plugin.yaml"), []byte(badManifest2), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = installer.Install(srcDir2)
	if err == nil {
		t.Fatal("expected error for manifest missing entry_point, got nil")
	}

	// Also test: invalid version
	srcDir3 := filepath.Join(t.TempDir(), "bad-plugin-3")
	if err := os.MkdirAll(srcDir3, 0o755); err != nil {
		t.Fatal(err)
	}
	badManifest3 := "name: bad-plugin-3\nversion: \"not-semver\"\ntype: tool\nentry_point: ./run\n"
	if err := os.WriteFile(filepath.Join(srcDir3, "kui-plugin.yaml"), []byte(badManifest3), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = installer.Install(srcDir3)
	if err == nil {
		t.Fatal("expected error for invalid semver version, got nil")
	}
}

// joinArgs joins a slice of strings with spaces.
func joinArgs(args []string) string {
	result := ""
	for i, a := range args {
		if i > 0 {
			result += " "
		}
		result += a
	}
	return result
}
