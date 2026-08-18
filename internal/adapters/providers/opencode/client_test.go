package opencode

import (
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestOpenCodeMissingAPIKey(t *testing.T) {
	// REQ-SEL-3: missing API key fails with error naming the variable.
	orig := os.Getenv("OPENCODE_API_KEY")
	os.Unsetenv("OPENCODE_API_KEY")
	defer os.Setenv("OPENCODE_API_KEY", orig)

	_, err := NewClient("https://opencode.ai/zen/go/v1")
	if err == nil {
		t.Fatal("NewClient returned nil error with missing OPENCODE_API_KEY, want error")
	}
	if !strings.Contains(err.Error(), "OPENCODE_API_KEY") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "OPENCODE_API_KEY")
	}
}

func TestOpenCodeBaseURLOverride(t *testing.T) {
	// REQ-PROV-3: custom base URL override works with httptest server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hi"}}]}`))
	}))
	defer srv.Close()

	os.Setenv("OPENCODE_API_KEY", "test-key")
	defer os.Unsetenv("OPENCODE_API_KEY")

	client, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	if client == nil {
		t.Fatal("client is nil, want non-nil")
	}
}

func TestOpenCodeDefaultBaseURLFallback(t *testing.T) {
	// When OPENCODE_API_KEY is set but OPENCODE_BASE_URL is not, the client
	// must fall back to the hardcoded default (https://opencode.ai/zen/go/v1).
	os.Setenv("OPENCODE_API_KEY", "test-key")
	defer os.Unsetenv("OPENCODE_API_KEY")
	os.Unsetenv("OPENCODE_BASE_URL")

	client, err := NewClient("")
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	if client == nil {
		t.Fatal("client is nil, want non-nil")
	}

	// Verify the internal baseURL field holds the default value via reflection.
	v := reflect.ValueOf(client).Elem()
	baseURL := v.FieldByName("baseURL").String()
	want := "https://opencode.ai/zen/go/v1"
	if baseURL != want {
		t.Errorf("baseURL = %q, want %q", baseURL, want)
	}
}
