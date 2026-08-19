// Package providers defines the provider registry that maps provider names to
// adapter factories. Each registered provider declares its required environment
// variable and capabilities (REQ-SEL-1).
package providers

import (
	"fmt"

	"github.com/biggs-100/kui/internal/adapters/providers/opencode"
	"github.com/biggs-100/kui/internal/adapters/providers/openai"
	"github.com/biggs-100/kui/internal/core"
)

// ProviderFactory creates a Provider from resolved configuration. apiKey and
// baseURL are passed by the caller after layered resolution; the factory reads
// no env vars itself — the caller resolves them from ProviderEntry metadata.
type ProviderFactory func(apiKey, baseURL string) (core.Provider, error)

// ProviderEntry registers a factory with its metadata so the resolution layer
// can read env vars and construct the provider without coupling to each
// adapter's internals (REQ-SEL-1, REQ-SEL-3).
type ProviderEntry struct {
	// Factory constructs a Provider from a resolved API key and base URL.
	Factory ProviderFactory
	// RequiredEnvVar names the env var that must be set (e.g. "OPENAI_API_KEY").
	RequiredEnvVar string
	// BaseURLEnvVar names the optional env var overriding the default base URL.
	BaseURLEnvVar string
	// DefaultBaseURL is used when BaseURLEnvVar is unset.
	DefaultBaseURL string
	// SupportsThinking reports whether the provider supports reasoning effort.
	SupportsThinking bool
}

// Registry maps provider names to entries. It is not safe for concurrent use;
// callers register providers at init time and then only call Resolve.
type Registry struct {
	entries map[string]ProviderEntry
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{entries: make(map[string]ProviderEntry)}
}

// Register adds or replaces a provider entry. It is intended for use at init
// time or in tests — not at runtime.
func (r *Registry) Register(name string, entry ProviderEntry) {
	r.entries[name] = entry
}

// Resolve returns the entry for the named provider. An unknown name returns an
// actionable error (REQ-SEL-1).
func (r *Registry) Resolve(name string) (ProviderEntry, error) {
	entry, ok := r.entries[name]
	if !ok {
		return ProviderEntry{}, fmt.Errorf("unknown provider %q: available providers are openai, opencode, opencode-go", name)
	}
	return entry, nil
}

// NewDefaultRegistry creates a registry pre-populated with the built-in
// providers: "openai", "opencode", and "opencode-go" (task 1.7).
func NewDefaultRegistry() *Registry {
	r := NewRegistry()

	r.Register("openai", ProviderEntry{
		Factory: func(apiKey, baseURL string) (core.Provider, error) {
			return openai.NewClientWithConfig(apiKey, baseURL, "")
		},
		RequiredEnvVar:   "OPENAI_API_KEY",
		BaseURLEnvVar:    "OPENAI_BASE_URL",
		DefaultBaseURL:   "https://api.openai.com/v1",
		SupportsThinking: true,
	})

	r.Register("opencode", ProviderEntry{
		Factory: func(apiKey, baseURL string) (core.Provider, error) {
			return opencode.NewClient(apiKey, baseURL)
		},
		RequiredEnvVar:   "OPENCODE_API_KEY",
		BaseURLEnvVar:    "OPENCODE_BASE_URL",
		DefaultBaseURL:   "https://opencode.ai/zen/v1",
		SupportsThinking: false,
	})

	r.Register("opencode-go", ProviderEntry{
		Factory: func(apiKey, baseURL string) (core.Provider, error) {
			return opencode.NewClient(apiKey, baseURL)
		},
		RequiredEnvVar:   "OPENCODE_GO_API_KEY",
		BaseURLEnvVar:    "OPENCODE_GO_BASE_URL",
		DefaultBaseURL:   "https://opencode.ai/zen/go/v1",
		SupportsThinking: false,
	})

	return r
}
