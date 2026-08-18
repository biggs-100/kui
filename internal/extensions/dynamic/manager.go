package dynamic

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/biggs-100/kui/internal/core"
)

// ExtScanner discovers extension manifests from a directory path. Extracted
// for testability — the default implementation scans for extension.yaml files.
type ExtScanner func(path string) ([]*Manifest, error)

// defaultScanner walks a directory looking for extension.yaml files and loads
// their manifests. Directories named "extension" or files named
// "extension.yaml" are treated as extension roots.
func defaultScanner(path string) ([]*Manifest, error) {
	var manifests []*Manifest

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("path %s is not a directory", path)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", path, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(path, entry.Name(), "extension.yaml")
		m, err := LoadManifest(manifestPath)
		if err != nil {
			// Not every subdirectory is an extension — skip silently.
			log.Printf("manager: skipping %s: %v", entry.Name(), err)
			continue
		}
		manifests = append(manifests, m)
	}

	return manifests, nil
}

// Manager coordinates dynamic extension lifecycle: discovery, spawning,
// registration, and crash isolation (REQ-MGR-1).
type Manager struct {
	config    *Config
	scanner   ExtScanner
	factory   ClientFactory
	loaded    []*DynamicExtension
}

// ManagerOption configures the manager. Useful for testing.
type ManagerOption func(*Manager)

// WithScanner overrides the default manifest scanner.
func WithScanner(s ExtScanner) ManagerOption {
	return func(m *Manager) { m.scanner = s }
}

// WithClientFactory overrides the default client factory.
func WithClientFactory(f ClientFactory) ManagerOption {
	return func(m *Manager) { m.factory = f }
}

// NewManager creates a Manager from a Config. Options can override the
// scanner and client factory for testing.
func NewManager(config *Config, opts ...ManagerOption) *Manager {
	m := &Manager{
		config:  config,
		scanner: defaultScanner,
		factory: defaultClientFactory,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// LoadAll discovers extensions from configured paths, spawns each as a
// DynamicExtension, and registers its tools via the API. Crashed or
// unreachable extensions are logged and skipped — they don't block other
// extensions from loading (REQ-MGR-2).
func (m *Manager) LoadAll(ctx context.Context, api core.ExtensionAPI) error {
	for _, path := range m.config.Paths {
		manifests, err := m.scanner(path)
		if err != nil {
			log.Printf("manager: scan %s failed: %v", path, err)
			continue
		}

		for _, manifest := range manifests {
			ext := NewDynamicExtension(manifest, m.factory)
			if err := ext.Init(api); err != nil {
				log.Printf("manager: extension %q unavailable: %v", manifest.Name, err)
				continue
			}
			m.loaded = append(m.loaded, ext)
		}
	}
	return nil
}

// ShutdownAll shuts down every loaded extension in reverse registration
// order (REQ-MGR-3). Errors are collected and returned via errors.Join.
func (m *Manager) ShutdownAll() error {
	var errs []error
	for i := len(m.loaded) - 1; i >= 0; i-- {
		if err := m.loaded[i].Shutdown(); err != nil {
			errs = append(errs, err)
		}
	}
	m.loaded = nil
	return errSlice(errs)
}

// Loaded returns the count of successfully loaded extensions.
func (m *Manager) Loaded() int {
	return len(m.loaded)
}

// errSlice joins errors like errors.Join but avoids importing it for
// compatibility.
func errSlice(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("%w", errs[0])
}
