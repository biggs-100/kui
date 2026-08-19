package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTestPluginDir creates a plugin directory with a valid kui-plugin.yaml manifest.
func writeTestPluginDir(t *testing.T, parent, name, version, pluginType string) string {
	t.Helper()
	dir := filepath.Join(parent, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", dir, err)
	}
	manifest := "name: " + name + "\nversion: \"" + version + "\"\ntype: " + pluginType + "\nentry_point: ./run\n"
	if err := os.WriteFile(filepath.Join(dir, "kui-plugin.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(kui-plugin.yaml): %v", err)
	}
	return dir
}

// ---------------------------------------------------------------------------
// Plugin subcommand dispatch
// ---------------------------------------------------------------------------

// TestCLIPluginNoSubcommand verifies `kui plugin` without a subcommand prints
// usage and exits 2.
func TestCLIPluginNoSubcommand(t *testing.T) {
	home := t.TempDir()

	_, stderr, code := runCLI(t, map[string]string{"KUI_HOME": home}, "plugin")

	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "plugin") {
		t.Errorf("stderr = %q, want it to mention plugin usage", stderr)
	}
}

// TestCLIPluginUnknownSubcommand verifies `kui plugin foo` prints an error
// and exits 2.
func TestCLIPluginUnknownSubcommand(t *testing.T) {
	home := t.TempDir()

	_, stderr, code := runCLI(t, map[string]string{"KUI_HOME": home}, "plugin", "foo")

	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "unknown") {
		t.Errorf("stderr = %q, want it to say 'unknown'", stderr)
	}
}

// ---------------------------------------------------------------------------
// Plugin list
// ---------------------------------------------------------------------------

// TestCLIPluginListEmpty verifies `kui plugin list` with no installed plugins
// prints a message and exits zero.
func TestCLIPluginListEmpty(t *testing.T) {
	home := t.TempDir()

	stdout, stderr, code := runCLI(t, map[string]string{"KUI_HOME": home}, "plugin", "list")

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "No plugins") && !strings.Contains(stdout, "no plugins") {
		t.Errorf("stdout = %q, want it to mention no plugins", stdout)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

// TestCLIPluginListWithPlugins verifies `kui plugin list` shows installed
// plugins with name, version, type, and status.
func TestCLIPluginListWithPlugins(t *testing.T) {
	home := t.TempDir()
	pluginDir := filepath.Join(home, "plugins")
	writeTestPluginDir(t, pluginDir, "hello-world", "1.0.0", "tool")
	writeTestPluginDir(t, pluginDir, "greet", "0.3.1", "command")

	stdout, stderr, code := runCLI(t, map[string]string{"KUI_HOME": home}, "plugin", "list")

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "hello-world") {
		t.Errorf("stdout = %q, want it to contain 'hello-world'", stdout)
	}
	if !strings.Contains(stdout, "greet") {
		t.Errorf("stdout = %q, want it to contain 'greet'", stdout)
	}
	if !strings.Contains(stdout, "1.0.0") {
		t.Errorf("stdout = %q, want it to contain version '1.0.0'", stdout)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

// TestCLIPluginListJSON verifies `kui plugin list --format json` outputs
// a JSON array of plugin objects.
func TestCLIPluginListJSON(t *testing.T) {
	home := t.TempDir()
	pluginDir := filepath.Join(home, "plugins")
	writeTestPluginDir(t, pluginDir, "alpha", "2.1.0", "hook")

	stdout, stderr, code := runCLI(t, map[string]string{"KUI_HOME": home}, "plugin", "list", "--format", "json")

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}

	var plugins []map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &plugins); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout = %q", err, stdout)
	}
	if len(plugins) != 1 {
		t.Fatalf("got %d plugins, want 1", len(plugins))
	}
	if plugins[0]["name"] != "alpha" {
		t.Errorf("plugin name = %v, want 'alpha'", plugins[0]["name"])
	}
	if plugins[0]["version"] != "2.1.0" {
		t.Errorf("plugin version = %v, want '2.1.0'", plugins[0]["version"])
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

// ---------------------------------------------------------------------------
// Plugin install
// ---------------------------------------------------------------------------

// TestCLIPluginInstallValid verifies `kui plugin install <path>` copies the
// plugin and prints a success message.
func TestCLIPluginInstallValid(t *testing.T) {
	home := t.TempDir()
	srcDir := writeTestPluginDir(t, t.TempDir(), "my-plugin", "1.0.0", "tool")

	stdout, stderr, code := runCLI(t, map[string]string{"KUI_HOME": home}, "plugin", "install", srcDir)

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "my-plugin") {
		t.Errorf("stdout = %q, want it to mention 'my-plugin'", stdout)
	}
	// Verify plugin was installed to the plugins directory
	installedPath := filepath.Join(home, "plugins", "my-plugin", "kui-plugin.yaml")
	if _, err := os.Stat(installedPath); os.IsNotExist(err) {
		t.Errorf("plugin not installed to %s", installedPath)
	}
}

// TestCLIPluginInstallInvalidPath verifies `kui plugin install <nonexistent>`
// prints an error and exits non-zero.
func TestCLIPluginInstallInvalidPath(t *testing.T) {
	home := t.TempDir()

	_, stderr, code := runCLI(t, map[string]string{"KUI_HOME": home}, "plugin", "install", "/nonexistent/path")

	if code == 0 {
		t.Error("exit code = 0, want non-zero for nonexistent path")
	}
	if !strings.Contains(stderr, "nonexistent") && !strings.Contains(stderr, "error") && !strings.Contains(stderr, "not") {
		t.Errorf("stderr = %q, want an error about the missing path", stderr)
	}
}

// TestCLIPluginInstallMissingArg verifies `kui plugin install` without a path
// prints usage and exits 2.
func TestCLIPluginInstallMissingArg(t *testing.T) {
	home := t.TempDir()

	_, stderr, code := runCLI(t, map[string]string{"KUI_HOME": home}, "plugin", "install")

	if code != 2 {
		t.Errorf("exit code = %d, want 2 (usage)", code)
	}
	if !strings.Contains(stderr, "plugin") {
		t.Errorf("stderr = %q, want plugin usage text", stderr)
	}
}

// ---------------------------------------------------------------------------
// Plugin remove
// ---------------------------------------------------------------------------

// TestCLIPluginRemoveFound verifies `kui plugin remove <name> --yes` removes
// an installed plugin.
func TestCLIPluginRemoveFound(t *testing.T) {
	home := t.TempDir()
	pluginDir := filepath.Join(home, "plugins")
	writeTestPluginDir(t, pluginDir, "to-remove", "1.0.0", "tool")

	stdout, stderr, code := runCLI(t, map[string]string{"KUI_HOME": home}, "plugin", "remove", "to-remove", "--yes")

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "to-remove") {
		t.Errorf("stdout = %q, want it to mention 'to-remove'", stdout)
	}
	// Verify plugin directory was removed
	removedPath := filepath.Join(pluginDir, "to-remove")
	if _, err := os.Stat(removedPath); !os.IsNotExist(err) {
		t.Errorf("plugin directory should be removed after remove, still exists at %s", removedPath)
	}
}

// TestCLIPluginRemoveNotFound verifies `kui plugin remove <nonexistent>` prints
// an error and exits non-zero.
func TestCLIPluginRemoveNotFound(t *testing.T) {
	home := t.TempDir()

	_, stderr, code := runCLI(t, map[string]string{"KUI_HOME": home}, "plugin", "remove", "nonexistent", "--yes")

	if code == 0 {
		t.Error("exit code = 0, want non-zero for nonexistent plugin")
	}
	if !strings.Contains(stderr, "nonexistent") {
		t.Errorf("stderr = %q, want it to name the missing plugin", stderr)
	}
}

// TestCLIPluginRemoveMissingArg verifies `kui plugin remove` without a name
// prints usage and exits 2.
func TestCLIPluginRemoveMissingArg(t *testing.T) {
	home := t.TempDir()

	_, stderr, code := runCLI(t, map[string]string{"KUI_HOME": home}, "plugin", "remove")

	if code != 2 {
		t.Errorf("exit code = %d, want 2 (usage)", code)
	}
	if !strings.Contains(stderr, "plugin") {
		t.Errorf("stderr = %q, want plugin usage text", stderr)
	}
}

// ---------------------------------------------------------------------------
// Plugin info
// ---------------------------------------------------------------------------

// TestCLIPluginInfoFound verifies `kui plugin info <name>` shows detailed
// information about an installed plugin.
func TestCLIPluginInfoFound(t *testing.T) {
	home := t.TempDir()
	pluginDir := filepath.Join(home, "plugins")
	writeTestPluginDir(t, pluginDir, "my-tool", "2.0.0", "tool")

	stdout, stderr, code := runCLI(t, map[string]string{"KUI_HOME": home}, "plugin", "info", "my-tool")

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "my-tool") {
		t.Errorf("stdout = %q, want it to contain plugin name", stdout)
	}
	if !strings.Contains(stdout, "2.0.0") {
		t.Errorf("stdout = %q, want it to contain version", stdout)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

// TestCLIPluginInfoNotFound verifies `kui plugin info <nonexistent>` prints
// an error and exits non-zero.
func TestCLIPluginInfoNotFound(t *testing.T) {
	home := t.TempDir()

	_, stderr, code := runCLI(t, map[string]string{"KUI_HOME": home}, "plugin", "info", "nonexistent")

	if code == 0 {
		t.Error("exit code = 0, want non-zero for nonexistent plugin")
	}
	if !strings.Contains(stderr, "nonexistent") {
		t.Errorf("stderr = %q, want it to name the missing plugin", stderr)
	}
}

// TestCLIPluginInfoMissingArg verifies `kui plugin info` without a name
// prints usage and exits 2.
func TestCLIPluginInfoMissingArg(t *testing.T) {
	home := t.TempDir()

	_, stderr, code := runCLI(t, map[string]string{"KUI_HOME": home}, "plugin", "info")

	if code != 2 {
		t.Errorf("exit code = %d, want 2 (usage)", code)
	}
	if !strings.Contains(stderr, "plugin") {
		t.Errorf("stderr = %q, want plugin usage text", stderr)
	}
}
