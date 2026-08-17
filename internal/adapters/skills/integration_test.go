package skills

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestIntegrationRegistryCacheFlow(t *testing.T) {
	// 4.1 RED: Full flow: httptest registry → FetchIndex → FetchFile →
	// Cache.Store → Cache.IsCached hit → ParseFrontmatter.
	indexData := RegistryIndex{
		Skills: []IndexSkill{
			{Name: "go-testing", Version: "1.0", Files: []string{"SKILL.md", "skill.yaml"}},
		},
	}
	skillContent := []byte(`---
name: go-testing
description: Run and debug Go tests
triggers:
  - go test
  - testing
---

# go-testing

Run Go tests.`)
	yamlContent := []byte("name: go-testing\ndescription: Run Go tests\ntriggers:\n  - go test\n")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(indexData)
		case "/go-testing/SKILL.md":
			w.Write(skillContent)
		case "/go-testing/skill.yaml":
			w.Write(yamlContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Step 1: Fetch the index.
	client := NewRegistryClient(10)
	index, err := client.FetchIndex(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("FetchIndex: %v", err)
	}
	if len(index.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(index.Skills))
	}

	skill := index.Skills[0]

	// Step 2: Check cache miss.
	cache := NewCache(t.TempDir())
	skillDir := cache.Dir(srv.URL, skill.Name, skill.Version)
	if cache.IsCached(skillDir, skill.Version) {
		t.Error("IsCached should be false before Store")
	}

	// Step 3: Fetch files and store.
	files := make(map[string][]byte)
	for _, fname := range skill.Files {
		data, err := client.FetchFile(context.Background(), srv.URL, skill.Name, fname)
		if err != nil {
			t.Fatalf("FetchFile(%s): %v", fname, err)
		}
		files[fname] = data
	}

	if err := cache.Store(srv.URL, skill.Name, skill.Version, files); err != nil {
		t.Fatalf("Store: %v", err)
	}

	// Step 4: Cache hit.
	if !cache.IsCached(skillDir, skill.Version) {
		t.Error("IsCached should be true after Store")
	}

	// Step 5: Parse frontmatter from cached SKILL.md.
	cachedSKILL, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("reading cached SKILL.md: %v", err)
	}
	meta, body, err := ParseFrontmatter(cachedSKILL)
	if err != nil {
		t.Fatalf("ParseFrontmatter: %v", err)
	}
	if meta.Name != "go-testing" {
		t.Errorf("frontmatter Name = %q, want %q", meta.Name, "go-testing")
	}
	if meta.Description != "Run and debug Go tests" {
		t.Errorf("frontmatter Description = %q", meta.Description)
	}
	if len(meta.Triggers) != 2 {
		t.Errorf("frontmatter Triggers = %v, want 2 triggers", meta.Triggers)
	}
	if body == "" {
		t.Error("body should not be empty")
	}
}
