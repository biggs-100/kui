package providers

import (
	"fmt"
	"io"
	"os"

	"github.com/biggs-100/kui/internal/core"
	"github.com/biggs-100/kui/internal/credentials"
)

// ResolveProvider applies the layered resolution chain for provider selection:
// --provider flag (highest priority) → profile.yaml provider → KUI_PROVIDER
// env → credentials store configured provider → default "openai" (REQ-SEL-2).
func ResolveProvider(flagProvider, profileProvider, root string) string {
	if flagProvider != "" {
		return flagProvider
	}
	if profileProvider != "" {
		return profileProvider
	}
	if env := os.Getenv("KUI_PROVIDER"); env != "" {
		return env
	}
	// Check credentials store for a configured provider.
	store := credentials.NewCredentialStore(root)
	if err := store.Load(); err == nil {
		if configured := store.GetConfiguredProvider(); configured != "" {
			return configured
		}
	}
	return "openai"
}

// CreateProvider uses the registry to construct a provider from the resolved
// name. API key resolution: environment variable → credentials store → error
// (REQ-SEL-3). root is the project root for resolving .kui/credentials.json.
func CreateProvider(reg *Registry, name string, root string) (core.Provider, error) {
	entry, err := reg.Resolve(name)
	if err != nil {
		return nil, err
	}

	// Read API key from the provider's required env var.
	apiKey := os.Getenv(entry.RequiredEnvVar)

	// Fallback to credential store if env var is empty.
	if apiKey == "" {
		store := credentials.NewCredentialStore(root)
		if err := store.Load(); err != nil {
			return nil, fmt.Errorf("load credentials: %w", err)
		}
		key, err := store.GetAPIKey(name)
		if err == nil {
			apiKey = key
		}
	}

	if apiKey == "" {
		return nil, fmt.Errorf("%s is not set: export %s before running kui, or run `kui setup`", entry.RequiredEnvVar, entry.RequiredEnvVar)
	}

	// Read base URL from the provider's base URL env var, falling back to default.
	baseURL := ""
	if entry.BaseURLEnvVar != "" {
		baseURL = os.Getenv(entry.BaseURLEnvVar)
	}
	if baseURL == "" {
		baseURL = entry.DefaultBaseURL
	}

	return entry.Factory(apiKey, baseURL)
}

// WarnThinkingDegradation emits a stderr warning when the resolved provider
// does not support thinking but a non-off thinking level is configured
// (REQ-THINK-13). The client is checked via the SupportsThinking() interface
// method. w is the writer for the warning (typically os.Stderr).
func WarnThinkingDegradation(providerName string, client core.Provider, thinkingLevel string, w io.Writer) {
	if tc, ok := client.(interface{ SupportsThinking() bool }); ok && !tc.SupportsThinking() {
		if thinkingLevel != "off" {
			fmt.Fprintf(w, "kui: WARNING: provider %q does not support thinking; continuing without it\n", providerName)
		}
	}
}
