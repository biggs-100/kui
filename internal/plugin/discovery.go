package plugin

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

// PluginDiscovery scans filesystem directories for installed plugins.
type PluginDiscovery struct {
	globalDir   string
	projectDir  string
	envOverride string
}

// NewPluginDiscovery creates a discovery instance for the given project root.
// Global plugins are looked up in ~/.config/kui/plugins/.
func NewPluginDiscovery(projectRoot string) *PluginDiscovery {
	home, _ := os.UserHomeDir()
	globalDir := ""
	if home != "" {
		globalDir = filepath.Join(home, ".config", "kui", "plugins")
	}

	return &PluginDiscovery{
		globalDir:  globalDir,
		projectDir: filepath.Join(projectRoot, ".kui", "plugins"),
	}
}

// NewPluginDiscoveryFromDir creates a discovery instance that scans a single
// directory for plugins. This is used by the CLI to scan a specific plugin
// directory without the global/project split.
func NewPluginDiscoveryFromDir(dir string) *PluginDiscovery {
	return &PluginDiscovery{
		globalDir:  dir,
		projectDir: dir,
	}
}

// Discover scans all plugin directories and returns merged manifests.
// Project-local plugins override global ones with the same name.
// Environment override directory takes highest priority if set.
func (d *PluginDiscovery) Discover() ([]PluginManifest, error) {
	seen := make(map[string]PluginManifest)

	// Scan in priority order: env > project > global
	dirs := d.scanDirs()

	for _, dir := range dirs {
		plugins, err := d.discoverDir(dir)
		if err != nil {
			log.Printf("warning: failed to scan %s: %v", dir, err)
			continue
		}
		for _, p := range plugins {
			seen[p.Name] = p
		}
	}

	result := make([]PluginManifest, 0, len(seen))
	for _, p := range seen {
		result = append(result, p)
	}
	return result, nil
}

// scanDirs returns the directories to scan in priority order.
// Lower priority dirs are scanned first so higher priority can override.
func (d *PluginDiscovery) scanDirs() []string {
	var dirs []string

	// Global has lowest priority
	if d.globalDir != "" {
		dirs = append(dirs, d.globalDir)
	}
	// Project overrides global
	dirs = append(dirs, d.projectDir)
	// Env override has highest priority
	if d.envOverride != "" {
		dirs = append(dirs, d.envOverride)
	}

	return dirs
}

// discoverDir scans a single directory for plugin manifests.
// Each subdirectory is treated as a plugin directory.
func (d *PluginDiscovery) discoverDir(dir string) ([]PluginManifest, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var plugins []PluginManifest
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Skip hidden directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		// Skip symlinks to directories (check via Lstat)
		fullPath := filepath.Join(dir, entry.Name())
		fi, err := os.Lstat(fullPath)
		if err != nil {
			continue
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			continue
		}

		pluginDir := filepath.Join(dir, entry.Name())
		m, err := loadPluginFromDir(pluginDir)
		if err != nil {
			log.Printf("warning: skipping plugin %s: %v", entry.Name(), err)
			continue
		}
		plugins = append(plugins, *m)
	}

	return plugins, nil
}

// loadPluginFromDir attempts to load a plugin manifest from a directory.
// It prefers kui-plugin.yaml over extension.yaml.
func loadPluginFromDir(dir string) (*PluginManifest, error) {
	// Try kui-plugin.yaml first
	pluginYAML := filepath.Join(dir, "kui-plugin.yaml")
	if data, err := os.ReadFile(pluginYAML); err == nil {
		return ParseManifest(data)
	}

	// Fallback to extension.yaml
	extYAML := filepath.Join(dir, "extension.yaml")
	if data, err := os.ReadFile(extYAML); err == nil {
		log.Printf("warning: using legacy extension.yaml in %s — migrate to kui-plugin.yaml", dir)
		return ParseExtensionYAML(data)
	}

	return nil, nil
}
