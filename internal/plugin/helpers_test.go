package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

// SetupTestPlugin creates a temporary plugin directory with a valid manifest
// and returns the path. The caller should clean up with TeardownTestPlugin or
// rely on t.Cleanup.
func SetupTestPlugin(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("SetupTestPlugin MkdirAll: %v", err)
	}
	manifest := "name: " + name + "\nversion: \"1.0.0\"\ntype: tool\nentry_point: ./run\n"
	if err := os.WriteFile(filepath.Join(dir, "kui-plugin.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("SetupTestPlugin WriteFile: %v", err)
	}
	t.Cleanup(func() { TeardownTestPlugin(t, dir) })
	return dir
}

// SetupTestPluginWithCommands creates a plugin directory with a manifest and
// additional command definitions. The commands slice contains command names
// that are recorded in the manifest description for test verification.
func SetupTestPluginWithCommands(t *testing.T, name string, commands []string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("SetupTestPluginWithCommands MkdirAll: %v", err)
	}
	manifest := "name: " + name + "\nversion: \"1.0.0\"\ntype: command\nentry_point: ./run\n"
	if err := os.WriteFile(filepath.Join(dir, "kui-plugin.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("SetupTestPluginWithCommands WriteFile: %v", err)
	}
	t.Cleanup(func() { TeardownTestPlugin(t, dir) })
	return dir
}

// SetupTestPluginWithPermissions creates a plugin directory with a manifest
// and a list of required permissions. The permissions are written as the
// manifest's permissions field for test verification.
func SetupTestPluginWithPermissions(t *testing.T, name string, permissions []string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("SetupTestPluginWithPermissions MkdirAll: %v", err)
	}
	manifest := "name: " + name + "\nversion: \"1.0.0\"\ntype: tool\nentry_point: ./run\n"
	if len(permissions) > 0 {
		manifest += "permissions:\n"
		for _, p := range permissions {
			manifest += "  - \"" + p + "\"\n"
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "kui-plugin.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("SetupTestPluginWithPermissions WriteFile: %v", err)
	}
	t.Cleanup(func() { TeardownTestPlugin(t, dir) })
	return dir
}

// TeardownTestPlugin removes a plugin directory and all its contents.
// It is safe to call multiple times on the same path.
func TeardownTestPlugin(t *testing.T, path string) {
	t.Helper()
	if path == "" {
		return
	}
	_ = os.RemoveAll(path)
}
