package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PluginState represents the lifecycle state of a loaded plugin.
type PluginState string

const (
	PluginStateDisloaded PluginState = "disloaded"
	PluginStateLoading   PluginState = "loading"
	PluginStateLoaded    PluginState = "loaded"
	PluginStateError     PluginState = "error"
)

// Plugin represents a loaded plugin with its manifest, state, and metadata.
type Plugin struct {
	Manifest PluginManifest
	State    PluginState
	Dir      string
	LoadedAt time.Time
}

// PluginRegistry manages the collection of loaded plugins.
// It provides thread-safe access and coordinates with PluginDiscovery.
type PluginRegistry struct {
	discovery *PluginDiscovery
	plugins   map[string]*Plugin
	mu        sync.RWMutex
}

// NewPluginRegistry creates a new registry backed by the given discovery instance.
func NewPluginRegistry(discovery *PluginDiscovery) *PluginRegistry {
	return &PluginRegistry{
		discovery: discovery,
		plugins:   make(map[string]*Plugin),
	}
}

// Load discovers all plugins via the discovery instance and loads them into the registry.
// Plugins that fail to load are logged and skipped.
func (r *PluginRegistry) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	manifests, err := r.discovery.Discover()
	if err != nil {
		return fmt.Errorf("discover plugins: %w", err)
	}

	for _, m := range manifests {
		// Determine the directory for this plugin from the discovery scan.
		// The discovery system already resolved dirs, so we find it by name.
		dir := r.findPluginDir(m.Name)
		r.plugins[m.Name] = &Plugin{
			Manifest: m,
			State:    PluginStateLoaded,
			Dir:      dir,
			LoadedAt: time.Now(),
		}
	}

	return nil
}

// findPluginDir locates the filesystem directory for a plugin by scanning
// the discovery directories. Returns empty string if not found.
func (r *PluginRegistry) findPluginDir(name string) string {
	dirs := r.discovery.scanDirs()
	for _, dir := range dirs {
		candidate := filepath.Join(dir, name)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return ""
}

// Get returns a plugin by name, or an error if not found.
func (r *PluginRegistry) Get(name string) (*Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.plugins[name]
	if !ok {
		return nil, &NotFoundError{Name: name, Type: "plugin"}
	}
	return p, nil
}

// List returns all loaded plugins.
func (r *PluginRegistry) List() []*Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Plugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		result = append(result, p)
	}
	return result
}

// Register adds a plugin to the registry. Returns an error if nil or duplicate.
func (r *PluginRegistry) Register(p *Plugin) error {
	if p == nil {
		return fmt.Errorf("cannot register nil plugin")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[p.Manifest.Name]; exists {
		return fmt.Errorf("plugin %q already registered", p.Manifest.Name)
	}

	r.plugins[p.Manifest.Name] = p
	return nil
}

// Unregister removes a plugin by name. Returns an error if not found.
func (r *PluginRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[name]; !exists {
		return fmt.Errorf("plugin %q not found", name)
	}

	delete(r.plugins, name)
	return nil
}
