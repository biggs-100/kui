package providers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/biggs-100/kui/internal/adapters/providers/opencode"
	"github.com/biggs-100/kui/internal/core"
)

func TestOpenCodeIntegrationBaseURL(t *testing.T) {
	// REQ-SEL-1 + REQ-PROV-3: httptest server verifies requests hit correct
	// base URL path /v1/chat/completions.
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hi"}}]}`))
	}))
	defer srv.Close()

	os.Setenv("OPENCODE_API_KEY", "test-key")
	defer os.Unsetenv("OPENCODE_API_KEY")

	provider, err := opencode.NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	if provider == nil {
		t.Fatal("provider is nil, want non-nil")
	}

	// Verify the provider can be used (basic smoke test).
	_, err = provider.Chat(t.Context(), []core.Message{{Role: "user", Content: "hello"}}, nil)
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	if !strings.HasSuffix(gotPath, "/chat/completions") {
		t.Errorf("request path = %q, want it to end with /chat/completions", gotPath)
	}
}

func TestRegistryResolveCreatesProvider(t *testing.T) {
	// REQ-SEL-1: registry resolve + factory creates a working provider.
	os.Setenv("OPENCODE_API_KEY", "test-key")
	defer os.Unsetenv("OPENCODE_API_KEY")

	r := NewDefaultRegistry()
	entry, err := r.Resolve("opencode")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	provider, err := entry.Factory("test-key", "http://localhost:1")
	if err != nil {
		t.Fatalf("Factory returned error: %v", err)
	}
	if provider == nil {
		t.Fatal("provider is nil, want non-nil")
	}
}

func TestRegistryOpenAIThinkingSupport(t *testing.T) {
	// REQ-THINK-13: openai provider reports thinking support.
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	r := NewDefaultRegistry()
	entry, err := r.Resolve("openai")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if !entry.SupportsThinking {
		t.Error("openai.SupportsThinking = false, want true")
	}
}

func TestRegistryOpenCodeNoThinkingSupport(t *testing.T) {
	// REQ-THINK-13: opencode provider reports no thinking support.
	r := NewDefaultRegistry()
	entry, err := r.Resolve("opencode")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if entry.SupportsThinking {
		t.Error("opencode.SupportsThinking = true, want false")
	}
}

// mockNoThinkingProvider is a core.Provider that reports SupportsThinking()=false.
type mockNoThinkingProvider struct{}

func (m *mockNoThinkingProvider) Chat(_ context.Context, _ []core.Message, _ []core.Tool) ([]core.Message, error) {
	return nil, nil
}

func (m *mockNoThinkingProvider) SupportsThinking() bool { return false }

func TestThinkingDegradationWarningEmitted(t *testing.T) {
	// REQ-THINK-13: Mock provider with SupportsThinking()=false; verify
	// stderr warning is emitted when thinking level is not "off".
	var buf bytes.Buffer
	mock := &mockNoThinkingProvider{}

	WarnThinkingDegradation("test-provider", mock, "medium", &buf)

	got := buf.String()
	if got == "" {
		t.Fatal("WarnThinkingDegradation wrote no output, want warning on stderr")
	}
	if !strings.Contains(got, "test-provider") {
		t.Errorf("warning = %q, want it to contain provider name %q", got, "test-provider")
	}
	if !strings.Contains(got, "does not support thinking") {
		t.Errorf("warning = %q, want it to contain %q", got, "does not support thinking")
	}
}

func TestThinkingDegradationNoWarningWhenOff(t *testing.T) {
	// REQ-THINK-13: No warning when thinking level is "off".
	var buf bytes.Buffer
	mock := &mockNoThinkingProvider{}

	WarnThinkingDegradation("test-provider", mock, "off", &buf)

	if buf.Len() != 0 {
		t.Errorf("WarnThinkingDegradation wrote %q with thinking off, want empty", buf.String())
	}
}

func TestThinkingDegradationNoWarningWhenSupported(t *testing.T) {
	// REQ-THINK-13: No warning when provider supports thinking.
	var buf bytes.Buffer
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	r := NewDefaultRegistry()
	provider, err := CreateProvider(r, "openai", t.TempDir())
	if err != nil {
		t.Fatalf("CreateProvider returned error: %v", err)
	}

	WarnThinkingDegradation("openai", provider, "medium", &buf)

	if buf.Len() != 0 {
		t.Errorf("WarnThinkingDegradation wrote %q for openai, want empty", buf.String())
	}
}
