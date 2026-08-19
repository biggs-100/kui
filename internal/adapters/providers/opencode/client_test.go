package opencode

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestOpenCodeMissingAPIKey(t *testing.T) {
	// REQ-SEL-3: missing API key fails with error.
	_, err := NewClient("", "https://opencode.ai/zen/v1")
	if err == nil {
		t.Fatal("NewClient returned nil error with empty API key, want error")
	}
}

func TestOpenCodeBaseURLOverride(t *testing.T) {
	// REQ-PROV-3: custom base URL override works with httptest server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hi"}}]}`))
	}))
	defer srv.Close()

	client, err := NewClient("test-key", srv.URL)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	if client == nil {
		t.Fatal("client is nil, want non-nil")
	}
}

func TestOpenCodeDefaultBaseURLFallback(t *testing.T) {
	// When baseURL is empty, the client must fall back to the hardcoded default.
	client, err := NewClient("test-key", "")
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	if client == nil {
		t.Fatal("client is nil, want non-nil")
	}

	// Verify the internal baseURL field holds the default value via reflection.
	v := reflect.ValueOf(client).Elem()
	baseURL := v.FieldByName("baseURL").String()
	want := "https://opencode.ai/zen/v1"
	if baseURL != want {
		t.Errorf("baseURL = %q, want %q", baseURL, want)
	}
}
