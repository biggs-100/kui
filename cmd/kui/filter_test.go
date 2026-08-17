package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/biggs-100/kui/internal/core"
	"github.com/biggs-100/kui/internal/adapters/profile"
	"github.com/biggs-100/kui/internal/adapters/store"
)

// ---------------------------------------------------------------------------
// filterTools — pure function tests
// ---------------------------------------------------------------------------

// stubTool is a minimal core.Tool for test registries.
type stubTool struct {
	name string
}

func (t *stubTool) Name() string        { return t.name }
func (t *stubTool) Description() string { return "stub" }
func (t *stubTool) Schema() string      { return "{}" }
func (t *stubTool) Execute(_ context.Context, _ json.RawMessage) (string, error) {
	return "", nil
}

func buildRegistry(names ...string) *core.Registry {
	r := core.NewRegistry()
	for _, n := range names {
		_ = r.Register(&stubTool{name: n})
	}
	return r
}

func registryNames(r *core.Registry) []string {
	var names []string
	for _, t := range r.List() {
		names = append(names, t.Name())
	}
	return names
}

func TestFilterToolsEmptyReturnsFull(t *testing.T) {
	full := buildRegistry("bash", "read_file", "write_file")
	got := filterTools(full, "", "", false)
	if len(got.List()) != 3 {
		t.Errorf("filterTools() returned %d tools, want 3", len(got.List()))
	}
}

func TestFilterToolsIncludeSingle(t *testing.T) {
	full := buildRegistry("bash", "read_file", "write_file")
	got := filterTools(full, "read_file", "", false)
	names := registryNames(got)
	if len(names) != 1 || names[0] != "read_file" {
		t.Errorf("filterTools() returned %v, want [read_file]", names)
	}
}

func TestFilterToolsIncludeMultiple(t *testing.T) {
	full := buildRegistry("bash", "read_file", "write_file")
	got := filterTools(full, "read_file,write_file", "", false)
	names := registryNames(got)
	if len(names) != 2 {
		t.Errorf("filterTools() returned %d tools, want 2", len(names))
	}
}

func TestFilterToolsExcludeSingle(t *testing.T) {
	full := buildRegistry("bash", "read_file", "write_file")
	got := filterTools(full, "", "bash", false)
	names := registryNames(got)
	for _, n := range names {
		if n == "bash" {
			t.Errorf("filterTools() included excluded tool %q", n)
		}
	}
	if len(names) != 2 {
		t.Errorf("filterTools() returned %d tools, want 2", len(names))
	}
}

func TestFilterToolsExcludeMultiple(t *testing.T) {
	full := buildRegistry("bash", "read_file", "write_file")
	got := filterTools(full, "", "bash,write_file", false)
	names := registryNames(got)
	if len(names) != 1 || names[0] != "read_file" {
		t.Errorf("filterTools() returned %v, want [read_file]", names)
	}
}

func TestFilterToolsExcludeWins(t *testing.T) {
	full := buildRegistry("bash", "read_file", "write_file")
	got := filterTools(full, "read_file,bash", "bash", false)
	names := registryNames(got)
	if len(names) != 1 || names[0] != "read_file" {
		t.Errorf("filterTools() returned %v, want [read_file]", names)
	}
}

func TestFilterToolsExcludeSupersetOfInclude(t *testing.T) {
	full := buildRegistry("bash", "read_file", "write_file")
	got := filterTools(full, "read_file", "read_file,write_file", false)
	if len(got.List()) != 0 {
		t.Errorf("filterTools() returned %d tools, want 0", len(got.List()))
	}
}

func TestFilterToolsNoTools(t *testing.T) {
	full := buildRegistry("bash", "read_file", "write_file")
	got := filterTools(full, "bash", "read_file", true)
	if len(got.List()) != 0 {
		t.Errorf("filterTools() returned %d tools with noTools=true, want 0", len(got.List()))
	}
}

func TestFilterToolsEmptyRegistry(t *testing.T) {
	full := core.NewRegistry()
	got := filterTools(full, "bash", "", false)
	if len(got.List()) != 0 {
		t.Errorf("filterTools() returned %d tools from empty registry, want 0", len(got.List()))
	}
}

func TestFilterToolsNonexistentTool(t *testing.T) {
	full := buildRegistry("bash", "read_file")
	got := filterTools(full, "nonexistent", "", false)
	if len(got.List()) != 0 {
		t.Errorf("filterTools() returned %d tools for nonexistent include, want 0", len(got.List()))
	}
}

// ---------------------------------------------------------------------------
// resolveWithOverride — model override tests
// ---------------------------------------------------------------------------

// setupTestStore creates a temporary store with a saved model for the given profile.
func setupTestStore(t *testing.T, profileName, model string) *store.Store {
	t.Helper()
	dir := t.TempDir()
	st := store.New(dir)
	if err := st.Set(profileName, model); err != nil {
		t.Fatalf("setupTestStore: %v", err)
	}
	return st
}

// setupTestLoader creates a temporary loader with a profile that has the given model.
func setupTestLoader(t *testing.T, profileName, model string) *profile.Loader {
	t.Helper()
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "profiles", profileName)
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatalf("setupTestLoader: %v", err)
	}
	yaml := "name: " + profileName + "\nmodel: " + model + "\n"
	if err := os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("setupTestLoader: %v", err)
	}
	return profile.NewLoader(filepath.Join(dir, "profiles"), dir, dir)
}

func TestResolveWithOverrideTakesPrecedence(t *testing.T) {
	st := setupTestStore(t, "coder", "gpt-4o")
	loader := setupTestLoader(t, "coder", "gpt-4o-profile")

	got := resolveWithOverride("gpt-4o-mini", st, loader, "coder")
	if got != "gpt-4o-mini" {
		t.Errorf("resolveWithOverride() = %q, want %q", got, "gpt-4o-mini")
	}
}

func TestResolveWithOverrideEmptyFallsThroughToSaved(t *testing.T) {
	st := setupTestStore(t, "coder", "gpt-4o-saved")
	loader := setupTestLoader(t, "coder", "gpt-4o-profile")

	got := resolveWithOverride("", st, loader, "coder")
	if got != "gpt-4o-saved" {
		t.Errorf("resolveWithOverride() = %q, want %q", got, "gpt-4o-saved")
	}
}

func TestResolveWithOverrideEmptyFallsThroughToProfile(t *testing.T) {
	dir := t.TempDir()
	st := store.New(dir) // empty store — no saved model
	loader := setupTestLoader(t, "coder", "gpt-4o-profile")

	got := resolveWithOverride("", st, loader, "coder")
	if got != "gpt-4o-profile" {
		t.Errorf("resolveWithOverride() = %q, want %q", got, "gpt-4o-profile")
	}
}

func TestResolveWithOverrideEmptyFallsThroughToEnv(t *testing.T) {
	dir := t.TempDir()
	st := store.New(dir)
	loader := setupTestLoader(t, "coder", "") // no profile model

	t.Setenv("OPENAI_MODEL", "gpt-from-env")
	got := resolveWithOverride("", st, loader, "coder")
	if got != "gpt-from-env" {
		t.Errorf("resolveWithOverride() = %q, want %q", got, "gpt-from-env")
	}
}

func TestResolveWithOverrideEmptyFallsThroughToDefault(t *testing.T) {
	dir := t.TempDir()
	st := store.New(dir)
	loader := setupTestLoader(t, "coder", "") // no profile model

	got := resolveWithOverride("", st, loader, "coder")
	if got != defaultModel {
		t.Errorf("resolveWithOverride() = %q, want %q", got, defaultModel)
	}
}
