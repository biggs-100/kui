package providers

import (
	"strings"
	"testing"

	"github.com/biggs-100/kui/internal/core"
)

func TestRegistryKnownProviderLookup(t *testing.T) {
	// REQ-SEL-1: known provider lookup returns the registered entry.
	r := NewRegistry()
	r.Register("openai", ProviderEntry{
		Factory: func(apiKey, baseURL string) (core.Provider, error) {
			return nil, nil // stub
		},
		RequiredEnvVar: "OPENAI_API_KEY",
	})
	entry, err := r.Resolve("openai")
	if err != nil {
		t.Fatalf("Resolve(openai) returned error: %v", err)
	}
	if entry.Factory == nil {
		t.Error("entry.Factory is nil, want non-nil")
	}
	if entry.RequiredEnvVar != "OPENAI_API_KEY" {
		t.Errorf("RequiredEnvVar = %q, want %q", entry.RequiredEnvVar, "OPENAI_API_KEY")
	}
}

func TestRegistryUnknownProviderLookup(t *testing.T) {
	// REQ-SEL-1: unknown provider lookup returns an actionable error naming it.
	r := NewRegistry()
	_, err := r.Resolve("anthropic")
	if err == nil {
		t.Fatal("Resolve(anthropic) returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "anthropic") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "anthropic")
	}
}
