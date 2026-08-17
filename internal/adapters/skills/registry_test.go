package skills

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchIndexValid(t *testing.T) {
	// REQ-RS-1, REQ-RS-5: FetchIndex parses a valid index.json from a registry.
	index := RegistryIndex{
		Skills: []IndexSkill{
			{Name: "go-testing", Version: "1.0", Files: []string{"SKILL.md", "skill.yaml"}},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/index.json" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(index)
	}))
	defer srv.Close()

	client := NewRegistryClient(10)
	got, err := client.FetchIndex(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("FetchIndex returned error: %v", err)
	}
	if len(got.Skills) != 1 {
		t.Fatalf("FetchIndex returned %d skills, want 1", len(got.Skills))
	}
	if got.Skills[0].Name != "go-testing" {
		t.Errorf("skill Name = %q, want %q", got.Skills[0].Name, "go-testing")
	}
	if got.Skills[0].Version != "1.0" {
		t.Errorf("skill Version = %q, want %q", got.Skills[0].Version, "1.0")
	}
}

func TestFetchIndex404(t *testing.T) {
	// REQ-RS-1: FetchIndex returns an error when index.json is not found.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	client := NewRegistryClient(10)
	_, err := client.FetchIndex(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("FetchIndex on 404 should return error, got nil")
	}
}

func TestFetchIndexMalformedJSON(t *testing.T) {
	// REQ-RS-1: FetchIndex returns a parse error for malformed JSON.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json`))
	}))
	defer srv.Close()

	client := NewRegistryClient(10)
	_, err := client.FetchIndex(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("FetchIndex on malformed JSON should return error, got nil")
	}
}

func TestFetchFile(t *testing.T) {
	// REQ-RS-2: FetchFile downloads a skill file from the registry.
	expected := []byte("# go-testing\n\nRun Go tests.")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/go-testing/SKILL.md" {
			http.NotFound(w, r)
			return
		}
		w.Write(expected)
	}))
	defer srv.Close()

	client := NewRegistryClient(10)
	got, err := client.FetchFile(context.Background(), srv.URL, "go-testing", "SKILL.md")
	if err != nil {
		t.Fatalf("FetchFile returned error: %v", err)
	}
	if string(got) != string(expected) {
		t.Errorf("FetchFile = %q, want %q", got, expected)
	}
}

func TestFetchFile404(t *testing.T) {
	// REQ-RS-2: FetchFile returns an error when the file is not found.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	client := NewRegistryClient(10)
	_, err := client.FetchFile(context.Background(), srv.URL, "ghost", "SKILL.md")
	if err == nil {
		t.Fatal("FetchFile on 404 should return error, got nil")
	}
}

func TestFetchContextCancellation(t *testing.T) {
	// REQ-RS-8: FetchIndex respects context cancellation.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until context is cancelled.
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	client := NewRegistryClient(10)
	_, err := client.FetchIndex(ctx, srv.URL)
	if err == nil {
		t.Fatal("FetchIndex with cancelled context should return error, got nil")
	}
}
