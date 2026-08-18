package providers

import (
	"os"
	"strings"
	"testing"

	"github.com/biggs-100/kui/internal/core"
)

func TestResolveProvider_FlagTakesPrecedence(t *testing.T) {
	// REQ-SEL-2: --provider flag has highest priority.
	t.Setenv("KUI_PROVIDER", "opencode")
	got := ResolveProvider("openai", "")
	if got != "openai" {
		t.Errorf("ResolveProvider(%q, %q) = %q, want %q", "openai", "", got, "openai")
	}
}

func TestResolveProvider_ProfileTakesPrecedenceOverEnv(t *testing.T) {
	// REQ-SEL-2: profile provider beats KUI_PROVIDER env.
	t.Setenv("KUI_PROVIDER", "opencode")
	got := ResolveProvider("", "openai")
	if got != "openai" {
		t.Errorf("ResolveProvider(%q, %q) = %q, want %q", "", "openai", got, "openai")
	}
}

func TestResolveProvider_EnvTakesPrecedenceOverDefault(t *testing.T) {
	// REQ-SEL-2: KUI_PROVIDER env beats default "openai".
	t.Setenv("KUI_PROVIDER", "opencode")
	got := ResolveProvider("", "")
	if got != "opencode" {
		t.Errorf("ResolveProvider(%q, %q) = %q, want %q", "", "", got, "opencode")
	}
}

func TestResolveProvider_DefaultOpenAI(t *testing.T) {
	// REQ-SEL-2: default is "openai" when nothing is set.
	os.Unsetenv("KUI_PROVIDER")
	got := ResolveProvider("", "")
	if got != "openai" {
		t.Errorf("ResolveProvider(%q, %q) = %q, want %q", "", "", got, "openai")
	}
}

func TestCreateProvider_RegistryResolveError(t *testing.T) {
	// REQ-SEL-1: unknown provider returns error.
	r := NewRegistry()
	_, err := CreateProvider(r, "unknown")
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

	_, err := CreateProvider(r, "test")
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

	_, err := CreateProvider(r, "test")
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

	_, err := CreateProvider(r, "test")
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

	_, err := CreateProvider(r, "test")
	if err == nil {
		t.Fatal("CreateProvider returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "factory failed") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "factory failed")
	}
}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
