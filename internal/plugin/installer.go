package plugin

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// PluginInstaller handles installing and uninstalling plugins on the filesystem.
type PluginInstaller struct {
	registry  *PluginRegistry
	pluginDir string
}

// NewPluginInstaller creates a new installer backed by the given registry and install directory.
func NewPluginInstaller(registry *PluginRegistry, pluginDir string) *PluginInstaller {
	return &PluginInstaller{
		registry:  registry,
		pluginDir: pluginDir,
	}
}

// Install determines whether source is a path or URL and installs accordingly.
func (i *PluginInstaller) Install(source string) (*Plugin, error) {
	if err := i.ValidateSource(source); err != nil {
		return nil, err
	}

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return i.InstallFromURL(source)
	}
	return i.InstallFromPath(source)
}

// InstallFromPath copies a plugin directory from the given local path into the plugin directory.
// It validates the manifest before copying and registers the plugin in the registry.
func (i *PluginInstaller) InstallFromPath(path string) (*Plugin, error) {
	if err := i.ValidateSource(path); err != nil {
		return nil, err
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	// Read and validate manifest
	manifestPath := filepath.Join(absPath, "kui-plugin.yaml")
	m, err := LoadManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("validate manifest: %w", err)
	}

	// Check for hidden directories
	base := filepath.Base(absPath)
	if strings.HasPrefix(base, ".") {
		return nil, fmt.Errorf("cannot install from hidden directory %q", base)
	}

	// Prepare install target
	name := i.SanitizePath(m.Name)
	targetDir := filepath.Join(i.pluginDir, name)

	// Remove existing if present (overwrite)
	if _, err := os.Stat(targetDir); err == nil {
		os.RemoveAll(targetDir)
	}

	// Copy plugin directory
	if err := i.copyDir(absPath, targetDir); err != nil {
		return nil, fmt.Errorf("copy plugin: %w", err)
	}

	// Create plugin entry and register
	p := &Plugin{
		Manifest: *m,
		State:    PluginStateLoaded,
		Dir:      targetDir,
	}

	// If already registered, unregister first (overwrite)
	_ = i.registry.Unregister(name)

	if err := i.registry.Register(p); err != nil {
		return nil, fmt.Errorf("register plugin: %w", err)
	}

	return p, nil
}

// InstallFromURL downloads a plugin manifest from a URL and installs it.
// Currently supports downloading the manifest file directly.
func (i *PluginInstaller) InstallFromURL(url string) (*Plugin, error) {
	resp, err := http.Get(url) //nolint:gosec // user-provided URL from CLI
	if err != nil {
		return nil, fmt.Errorf("download from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download from %s: HTTP %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	m, err := ParseManifest(data)
	if err != nil {
		return nil, fmt.Errorf("parse manifest from URL: %w", err)
	}

	// Create plugin directory from manifest name
	name := i.SanitizePath(m.Name)
	targetDir := filepath.Join(i.pluginDir, name)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("create plugin dir: %w", err)
	}

	// Write the manifest
	manifestPath := filepath.Join(targetDir, "kui-plugin.yaml")
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return nil, fmt.Errorf("write manifest: %w", err)
	}

	p := &Plugin{
		Manifest: *m,
		State:    PluginStateLoaded,
		Dir:      targetDir,
	}

	// If already registered, unregister first (overwrite)
	_ = i.registry.Unregister(name)

	if err := i.registry.Register(p); err != nil {
		return nil, fmt.Errorf("register plugin: %w", err)
	}

	return p, nil
}

// Uninstall removes a plugin from the registry and deletes its directory.
func (i *PluginInstaller) Uninstall(name string) error {
	// Get plugin to find its directory
	p, err := i.registry.Get(name)
	if err != nil {
		return fmt.Errorf("plugin %q: %w", name, err)
	}

	// Remove directory
	if p.Dir != "" {
		if err := os.RemoveAll(p.Dir); err != nil {
			return fmt.Errorf("remove plugin directory %s: %w", p.Dir, err)
		}
	}

	// Unregister
	if err := i.registry.Unregister(name); err != nil {
		return fmt.Errorf("unregister plugin: %w", err)
	}

	return nil
}

// ValidateSource checks that a source path or URL does not contain path traversal.
func (i *PluginInstaller) ValidateSource(source string) error {
	if source == "" {
		return &NotFoundError{Name: "source", Type: "plugin"}
	}

	// Check for path traversal
	if strings.Contains(source, "..") {
		return fmt.Errorf("path traversal detected in source %q", source)
	}

	return nil
}

// SanitizePath cleans a plugin name for safe use as a directory name.
// Slashes, spaces, and leading dots are handled. Path traversal components are stripped.
func (i *PluginInstaller) SanitizePath(path string) string {
	// Trim leading/trailing slashes and dots
	path = strings.Trim(path, "/\\.")
	// Replace slashes with underscores
	path = strings.ReplaceAll(path, "/", "_")
	path = strings.ReplaceAll(path, "\\", "_")
	// Replace spaces with underscores
	path = strings.ReplaceAll(path, " ", "_")
	// Remove any remaining path traversal
	path = strings.ReplaceAll(path, "..", "")
	return path
}

// copyDir recursively copies a directory tree from src to dst.
func (i *PluginInstaller) copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Compute relative path
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		// Copy file
		return copyFile(path, target)
	})
}

// copyFile copies a single file from src to dst, preserving permissions.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	// Preserve permissions
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, info.Mode())
}
