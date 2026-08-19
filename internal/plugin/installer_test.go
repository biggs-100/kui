package plugin

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallFromPathValid(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)
	pluginDir := t.TempDir()

	installer := NewPluginInstaller(r, pluginDir)

	// Create a source plugin directory
	srcDir := filepath.Join(t.TempDir(), "source-plugin")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	writePluginManifest(t, srcDir, `name: source-plugin
version: "1.0.0"
type: tool
entry_point: ./bin/run
`)

	p, err := installer.InstallFromPath(srcDir)
	if err != nil {
		t.Fatalf("InstallFromPath() error = %v", err)
	}
	if p.Manifest.Name != "source-plugin" {
		t.Errorf("Manifest.Name = %q, want %q", p.Manifest.Name, "source-plugin")
	}

	// Verify plugin was copied to install dir
	installedPath := filepath.Join(pluginDir, "source-plugin", "kui-plugin.yaml")
	if _, err := os.Stat(installedPath); os.IsNotExist(err) {
		t.Errorf("plugin not copied to %s", installedPath)
	}

	// Verify plugin is in registry
	got, err := r.Get("source-plugin")
	if err != nil {
		t.Fatalf("plugin not in registry: %v", err)
	}
	if got.Manifest.Name != "source-plugin" {
		t.Errorf("registry plugin name = %q, want %q", got.Manifest.Name, "source-plugin")
	}
}

func TestInstallFromPathInvalidManifest(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)
	pluginDir := t.TempDir()
	installer := NewPluginInstaller(r, pluginDir)

	// Create source with invalid manifest
	srcDir := filepath.Join(t.TempDir(), "bad-plugin")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	writePluginManifest(t, srcDir, `name: bad-plugin
`)

	_, err := installer.InstallFromPath(srcDir)
	if err == nil {
		t.Fatal("expected error for invalid manifest, got nil")
	}
}

func TestInstallFromPathNonexistent(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)
	pluginDir := t.TempDir()
	installer := NewPluginInstaller(r, pluginDir)

	_, err := installer.InstallFromPath("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for nonexistent path, got nil")
	}
}

func TestInstallFromURLValid(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)
	pluginDir := t.TempDir()
	installer := NewPluginInstaller(r, pluginDir)

	// Create a mock HTTP server that serves a valid plugin tarball (as a directory zip)
	// For simplicity, we'll create a temp dir with a plugin, then serve files directly
	srcDir := filepath.Join(t.TempDir(), "url-plugin")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	writePluginManifest(t, srcDir, `name: url-plugin
version: "1.0.0"
type: command
entry_point: ./cmd.sh
`)

	// Serve the manifest file for the URL test
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(srcDir, "kui-plugin.yaml"))
	}))
	defer server.Close()

	p, err := installer.InstallFromURL(server.URL)
	if err != nil {
		t.Fatalf("InstallFromURL() error = %v", err)
	}
	if p.Manifest.Name != "url-plugin" {
		t.Errorf("Manifest.Name = %q, want %q", p.Manifest.Name, "url-plugin")
	}
}

func TestInstallFromURLInvalidURL(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)
	pluginDir := t.TempDir()
	installer := NewPluginInstaller(r, pluginDir)

	_, err := installer.InstallFromURL("http://localhost:99999/invalid")
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestInstallFromURL404(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)
	pluginDir := t.TempDir()
	installer := NewPluginInstaller(r, pluginDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	_, err := installer.InstallFromURL(server.URL + "/nonexistent")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestUninstallFound(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)
	pluginDir := t.TempDir()
	installer := NewPluginInstaller(r, pluginDir)

	// First install a plugin
	srcDir := filepath.Join(t.TempDir(), "uninstall-me")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	writePluginManifest(t, srcDir, `name: uninstall-me
version: "1.0.0"
type: tool
entry_point: ./bin/run
`)

	_, err := installer.InstallFromPath(srcDir)
	if err != nil {
		t.Fatalf("InstallFromPath() error = %v", err)
	}

	// Now uninstall it
	if err := installer.Uninstall("uninstall-me"); err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}

	// Verify removed from registry
	_, err = r.Get("uninstall-me")
	if err == nil {
		t.Fatal("expected error after uninstall, got nil")
	}

	// Verify removed from filesystem
	installedPath := filepath.Join(pluginDir, "uninstall-me")
	if _, err := os.Stat(installedPath); !os.IsNotExist(err) {
		t.Errorf("plugin directory should be removed after uninstall")
	}
}

func TestUninstallNotFound(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)
	pluginDir := t.TempDir()
	installer := NewPluginInstaller(r, pluginDir)

	if err := installer.Uninstall("nonexistent"); err == nil {
		t.Fatal("expected error for uninstalling nonexistent plugin, got nil")
	}
}

func TestValidateSourcePathTraversal(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{"dot-dot traversal", "../escape", true},
		{"nested traversal", "foo/../../etc/passwd", true},
		{"absolute path traversal", "/etc/../passwd", true}, // .. always rejected for security
		{"valid relative path", "plugins/my-plugin", false},
		{"valid absolute path", "/home/user/plugins/my-plugin", false},
		{"empty string", "", true},
		{"dot-dot only", "..", true},
		{"double dot-dot", "../../..", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globalDir := t.TempDir()
			projectDir := t.TempDir()
			d := NewPluginDiscovery(projectDir)
			d.globalDir = globalDir
			d.projectDir = projectDir

			r := NewPluginRegistry(d)
			pluginDir := t.TempDir()
			installer := NewPluginInstaller(r, pluginDir)

			err := installer.ValidateSource(tt.source)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSource(%q) error = %v, wantErr %v", tt.source, err, tt.wantErr)
			}
		})
	}
}

func TestSanitizePathNormal(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)
	pluginDir := t.TempDir()
	installer := NewPluginInstaller(r, pluginDir)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple name", "my-plugin", "my-plugin"},
		{"with spaces", "my plugin", "my_plugin"},
		{"with dots", "my.plugin.v2", "my.plugin.v2"},
		{"with slash", "foo/bar", "foo_bar"},
		{"trailing slash", "plugin/", "plugin"},
		{"leading slash", "/plugin", "plugin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := installer.SanitizePath(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizePathRejectDotDot(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)
	pluginDir := t.TempDir()
	installer := NewPluginInstaller(r, pluginDir)

	// Paths with .. should be rejected (return empty or sanitized away)
	result := installer.SanitizePath("../../../etc")
	if result == "" {
		// If it returns empty, that's one valid rejection approach
		return
	}
	// If it returns something, ensure no .. remains
	if strings.Contains(result, "..") {
		t.Errorf("SanitizePath(%q) should not contain .., got %q", "../../../etc", result)
	}
}

func TestInstallDuplicateOverwrite(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)
	pluginDir := t.TempDir()
	installer := NewPluginInstaller(r, pluginDir)

	// Install first version
	srcDir1 := filepath.Join(t.TempDir(), "dup-plugin-v1")
	if err := os.MkdirAll(srcDir1, 0755); err != nil {
		t.Fatal(err)
	}
	writePluginManifest(t, srcDir1, `name: dup-plugin
version: "1.0.0"
type: tool
entry_point: ./bin/run
`)

	_, err := installer.InstallFromPath(srcDir1)
	if err != nil {
		t.Fatalf("first InstallFromPath() error = %v", err)
	}

	// Install second version (same name)
	srcDir2 := filepath.Join(t.TempDir(), "dup-plugin-v2")
	if err := os.MkdirAll(srcDir2, 0755); err != nil {
		t.Fatal(err)
	}
	writePluginManifest(t, srcDir2, `name: dup-plugin
version: "2.0.0"
type: tool
entry_point: ./bin/run
`)

	p, err := installer.InstallFromPath(srcDir2)
	if err != nil {
		t.Fatalf("second InstallFromPath() error = %v", err)
	}

	// Should have overwritten with v2
	got, err := r.Get("dup-plugin")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Manifest.Version != "2.0.0" {
		t.Errorf("Manifest.Version = %q, want %q (should overwrite)", got.Manifest.Version, "2.0.0")
	}
	if p.Manifest.Version != "2.0.0" {
		t.Errorf("returned Manifest.Version = %q, want %q", p.Manifest.Version, "2.0.0")
	}
}

func TestInstallCopyFiles(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)
	pluginDir := t.TempDir()
	installer := NewPluginInstaller(r, pluginDir)

	// Create source with multiple files
	srcDir := filepath.Join(t.TempDir(), "full-plugin")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	writePluginManifest(t, srcDir, `name: full-plugin
version: "1.0.0"
type: tool
entry_point: ./bin/run
`)
	binDir := filepath.Join(srcDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "run"), []byte("#!/bin/sh\necho hello"), 0755); err != nil {
		t.Fatal(err)
	}

	_, err := installer.InstallFromPath(srcDir)
	if err != nil {
		t.Fatalf("InstallFromPath() error = %v", err)
	}

	// Verify all files copied
	installedBin := filepath.Join(pluginDir, "full-plugin", "bin", "run")
	if _, err := os.Stat(installedBin); os.IsNotExist(err) {
		t.Errorf("bin/run not copied to %s", installedBin)
	}
}

func TestInstallFromPathSkipsDotDir(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)
	pluginDir := t.TempDir()
	installer := NewPluginInstaller(r, pluginDir)

	// Source path with .hidden component
	srcDir := filepath.Join(t.TempDir(), ".hidden-plugin")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	writePluginManifest(t, srcDir, `name: hidden-plugin
version: "1.0.0"
type: tool
entry_point: ./bin/run
`)

	_, err := installer.InstallFromPath(srcDir)
	if err == nil {
		t.Fatal("expected error for hidden directory source, got nil")
	}
}

func TestNewPluginInstaller(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)
	pluginDir := t.TempDir()
	installer := NewPluginInstaller(r, pluginDir)

	if installer == nil {
		t.Fatal("NewPluginInstaller returned nil")
	}
	if installer.registry != r {
		t.Error("registry not set correctly")
	}
	if installer.pluginDir != pluginDir {
		t.Errorf("pluginDir = %q, want %q", installer.pluginDir, pluginDir)
	}
}

func TestUninstallPreservesOtherPlugins(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()
	d := NewPluginDiscovery(projectDir)
	d.globalDir = globalDir
	d.projectDir = projectDir

	r := NewPluginRegistry(d)
	pluginDir := t.TempDir()
	installer := NewPluginInstaller(r, pluginDir)

	// Install two plugins
	for _, name := range []string{"keep-me", "remove-me"} {
		srcDir := filepath.Join(t.TempDir(), name)
		if err := os.MkdirAll(srcDir, 0755); err != nil {
			t.Fatal(err)
		}
		writePluginManifest(t, srcDir, fmt.Sprintf(`name: %s
version: "1.0.0"
type: tool
entry_point: ./bin/run
`, name))
		_, err := installer.InstallFromPath(srcDir)
		if err != nil {
			t.Fatalf("InstallFromPath(%s) error = %v", name, err)
		}
	}

	// Uninstall one
	if err := installer.Uninstall("remove-me"); err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}

	// Verify the other is still there
	_, err := r.Get("keep-me")
	if err != nil {
		t.Errorf("keep-me should still be in registry: %v", err)
	}
	keepPath := filepath.Join(pluginDir, "keep-me")
	if _, err := os.Stat(keepPath); os.IsNotExist(err) {
		t.Errorf("keep-me directory should still exist")
	}
}
