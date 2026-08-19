package providers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/biggs-100/kui/internal/core"
)

func TestResolveProvider_FlagTakesPrecedence(t *testing.T) {
	// REQ-SEL-2: --provider flag has highest priority.
	t.Setenv("KUI_PROVIDER", "opencode")
	got := ResolveProvider("openai", "", t.TempDir())
	if got != "openai" {
		t.Errorf("ResolveProvider(%q, %q, root) = %q, want %q", "openai", "", got, "openai")
	}
}

func TestResolveProvider_ProfileTakesPrecedenceOverEnv(t *testing.T) {
	// REQ-SEL-2: profile provider beats KUI_PROVIDER env.
	t.Setenv("KUI_PROVIDER", "opencode")
	got := ResolveProvider("", "openai", t.TempDir())
	if got != "openai" {
		t.Errorf("ResolveProvider(%q, %q, root) = %q, want %q", "", "openai", got, "openai")
	}
}

func TestResolveProvider_EnvTakesPrecedenceOverDefault(t *testing.T) {
	// REQ-SEL-2: KUI_PROVIDER env beats default "openai".
	t.Setenv("KUI_PROVIDER", "opencode")
	got := ResolveProvider("", "", t.TempDir())
	if got != "opencode" {
		t.Errorf("ResolveProvider(%q, %q, root) = %q, want %q", "", "", got, "opencode")
	}
}

func TestResolveProvider_DefaultOpenAI(t *testing.T) {
	// REQ-SEL-2: default is "openai" when nothing is set.
	os.Unsetenv("KUI_PROVIDER")
	got := ResolveProvider("", "", t.TempDir())
	if got != "openai" {
		t.Errorf("ResolveProvider(%q, %q, root) = %q, want %q", "", "", got, "openai")
	}
}

func TestCreateProvider_RegistryResolveError(t *testing.T) {
	// REQ-SEL-1: unknown provider returns error.
	r := NewRegistry()
	_, err := CreateProvider(r, "unknown", t.TempDir())
	if err == nil {
		t.Fatal("CreateProvider returned nil error for unknown provider, want error")
	}
	if !strings.Contains(err.Error(), "unknown") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "unknown")
	}
}

func TestCreateProvider_MissingAPIKey(t *testing.T) {
	// REQ-SEL-3: missing API key returns error naming the env var.
	r := NewRegistry()
	r.Register("test", ProviderEntry{
		Factory: func(apiKey, baseURL string) (core.Provider, error) {
			return nil, nil
		},
		RequiredEnvVar: "TEST_API_KEY",
	})
	os.Unsetenv("TEST_API_KEY")

	_, err := CreateProvider(r, "test", t.TempDir())
	if err == nil {
		t.Fatal("CreateProvider returned nil error with missing API key, want error")
	}
	if !strings.Contains(err.Error(), "TEST_API_KEY") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "TEST_API_KEY")
	}
}

func TestCreateProvider_UsesDefaultBaseURL(t *testing.T) {
	// REQ-SEL-3: when BaseURLEnvVar is unset, DefaultBaseURL is used.
	var gotBaseURL string
	r := NewRegistry()
	r.Register("test", ProviderEntry{
		Factory: func(apiKey, baseURL string) (core.Provider, error) {
			gotBaseURL = baseURL
			return nil, nil
		},
		RequiredEnvVar:   "TEST_API_KEY",
		DefaultBaseURL:   "https://default.example.com/v1",
	})
	t.Setenv("TEST_API_KEY", "test-key")

	_, err := CreateProvider(r, "test", t.TempDir())
	if err != nil {
		t.Fatalf("CreateProvider returned error: %v", err)
	}
	if gotBaseURL != "https://default.example.com/v1" {
		t.Errorf("baseURL = %q, want %q", gotBaseURL, "https://default.example.com/v1")
	}
}

func TestCreateProvider_UsesEnvBaseURL(t *testing.T) {
	// REQ-SEL-3: BaseURLEnvVar overrides DefaultBaseURL.
	var gotBaseURL string
	r := NewRegistry()
	r.Register("test", ProviderEntry{
		Factory: func(apiKey, baseURL string) (core.Provider, error) {
			gotBaseURL = baseURL
			return nil, nil
		},
		RequiredEnvVar:   "TEST_API_KEY",
		BaseURLEnvVar:    "TEST_BASE_URL",
		DefaultBaseURL:   "https://default.example.com/v1",
	})
	t.Setenv("TEST_API_KEY", "test-key")
	t.Setenv("TEST_BASE_URL", "https://custom.example.com/v1")

	_, err := CreateProvider(r, "test", t.TempDir())
	if err != nil {
		t.Fatalf("CreateProvider returned error: %v", err)
	}
	if gotBaseURL != "https://custom.example.com/v1" {
		t.Errorf("baseURL = %q, want %q", gotBaseURL, "https://custom.example.com/v1")
	}
}

func TestCreateProvider_FactoryError(t *testing.T) {
	// REQ-SEL-3: factory errors propagate.
	r := NewRegistry()
	r.Register("test", ProviderEntry{
		Factory: func(apiKey, baseURL string) (core.Provider, error) {
			return nil, &testError{msg: "factory failed"}
		},
		RequiredEnvVar: "TEST_API_KEY",
	})
	t.Setenv("TEST_API_KEY", "test-key")

	_, err := CreateProvider(r, "test", t.TempDir())
	if err == nil {
		t.Fatal("CreateProvider returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "factory failed") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "factory failed")
	}
}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }

// --- Credential store integration tests (Phase 3) ---

// writeCredsFile writes a .kui/credentials.json with the given provider key.
func writeCredsFile(t *testing.T, root, provider, apiKey string) {
	t.Helper()
	dir := filepath.Join(root, ".kui")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	type providerCreds struct {
		APIKey string `json:"api_key"`
	}
	type file struct {
		Providers map[string]providerCreds `json:"providers"`
	}
	data := file{
		Providers: map[string]providerCreds{
			provider: {APIKey: apiKey},
		},
	}
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "credentials.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCreateProvider_EnvVarTakesPrecedenceOverFile(t *testing.T) {
	// REQ-SEL-3: env var takes precedence over credential store.
	r := NewRegistry()
	r.Register("test", ProviderEntry{
		Factory: func(apiKey, baseURL string) (core.Provider, error) {
			if apiKey != "env-secret-key" {
				t.Errorf("apiKey = %q, want %q", apiKey, "env-secret-key")
			}
			return nil, nil
		},
		RequiredEnvVar: "TEST_CRED_API_KEY",
	})

	root := t.TempDir()
	writeCredsFile(t, root, "test", "file-secret-key")
	t.Setenv("TEST_CRED_API_KEY", "env-secret-key")

	_, err := CreateProvider(r, "test", root)
	if err != nil {
		t.Fatalf("CreateProvider returned error: %v", err)
	}
}

func TestCreateProvider_FallbackToCredentialStore(t *testing.T) {
	// REQ-SEL-3: when env is unset, credential store is used.
	var gotAPIKey string
	r := NewRegistry()
	r.Register("test", ProviderEntry{
		Factory: func(apiKey, baseURL string) (core.Provider, error) {
			gotAPIKey = apiKey
			return nil, nil
		},
		RequiredEnvVar: "TEST_CRED_MISSING_KEY",
	})
	os.Unsetenv("TEST_CRED_MISSING_KEY")

	root := t.TempDir()
	writeCredsFile(t, root, "test", "stored-secret-key")

	_, err := CreateProvider(r, "test", root)
	if err != nil {
		t.Fatalf("CreateProvider returned error: %v", err)
	}
	if gotAPIKey != "stored-secret-key" {
		t.Errorf("apiKey = %q, want %q", gotAPIKey, "stored-secret-key")
	}
}

func TestCreateProvider_NoKeyFound(t *testing.T) {
	// REQ-SEL-3: missing everywhere returns error.
	r := NewRegistry()
	r.Register("test", ProviderEntry{
		Factory: func(apiKey, baseURL string) (core.Provider, error) {
			return nil, nil
		},
		RequiredEnvVar: "TEST_CRED_NOWHERE_KEY",
	})
	os.Unsetenv("TEST_CRED_NOWHERE_KEY")

	root := t.TempDir() // empty — no credentials file

	_, err := CreateProvider(r, "test", root)
	if err == nil {
		t.Fatal("CreateProvider returned nil error when no key found, want error")
	}
	if !strings.Contains(err.Error(), "TEST_CRED_NOWHERE_KEY") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "TEST_CRED_NOWHERE_KEY")
	}
}
