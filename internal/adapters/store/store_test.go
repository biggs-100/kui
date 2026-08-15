package store

import (
	"os"
	"path/filepath"
	"testing"
)

// TestStorePersistAndRestore covers REQ-PROFILE-4, persist and restore: the
// model saved in session one is restored by a fresh store over the same root
// in session two, and the file lives under .kui/models.json.
func TestStorePersistAndRestore(t *testing.T) {
	root := t.TempDir()

	first := New(root)
	if err := first.Set("coder", "gpt-4o"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".kui", "models.json")); err != nil {
		t.Fatalf("models.json not written under .kui: %v", err)
	}

	// A fresh store in "session two" reads the persisted file.
	second := New(root)
	model, ok := second.Get("coder")
	if !ok {
		t.Fatal("Get(coder) found=false, want the model restored from session one")
	}
	if model != "gpt-4o" {
		t.Errorf("Get(coder) = %q, want %q", model, "gpt-4o")
	}
}

// TestStoreGetNoSavedModelFallback covers REQ-PROFILE-4, no saved model: an
// unsaved profile resolves to found=false so the caller falls back to the
// layered config.
func TestStoreGetNoSavedModelFallback(t *testing.T) {
	root := t.TempDir()
	s := New(root)
	if model, ok := s.Get("nobody"); ok || model != "" {
		t.Errorf("Get(nobody) = (%q, %v), want (\"\", false)", model, ok)
	}
}

// TestStoreSetPreservesOtherProfiles triangulates persistence: saving one
// profile must not clobber models saved for other profiles, and empty roots
// stay absent.
func TestStoreSetPreservesOtherProfiles(t *testing.T) {
	root := t.TempDir()
	s := New(root)
	if err := s.Set("coder", "gpt-4o"); err != nil {
		t.Fatalf("Set(coder) returned error: %v", err)
	}
	if err := s.Set("writer", "gpt-4o-mini"); err != nil {
		t.Fatalf("Set(writer) returned error: %v", err)
	}
	reloaded := New(root)
	if model, ok := reloaded.Get("coder"); !ok || model != "gpt-4o" {
		t.Errorf("Get(coder) = (%q, %v), want (\"gpt-4o\", true)", model, ok)
	}
	if model, ok := reloaded.Get("writer"); !ok || model != "gpt-4o-mini" {
		t.Errorf("Get(writer) = (%q, %v), want (\"gpt-4o-mini\", true)", model, ok)
	}
	if model, ok := reloaded.Get("nobody"); ok || model != "" {
		t.Errorf("Get(nobody) = (%q, %v), want (\"\", false)", model, ok)
	}
}

// TestStoreActiveRoundtrip covers the .kui/active text persistence used by
// session-start activation (D18): the saved active profile survives a fresh
// store.
func TestStoreActiveRoundtrip(t *testing.T) {
	root := t.TempDir()
	first := New(root)
	if err := first.SetActive("coder"); err != nil {
		t.Fatalf("SetActive returned error: %v", err)
	}
	second := New(root)
	active, err := second.Active()
	if err != nil {
		t.Fatalf("Active returned error: %v", err)
	}
	if active != "coder" {
		t.Errorf("Active() = %q, want %q", active, "coder")
	}
}

// TestStoreActiveMissing covers the no-saved-active case: a store without a
// .kui/active file yields "" with no error.
func TestStoreActiveMissing(t *testing.T) {
	root := t.TempDir()
	s := New(root)
	active, err := s.Active()
	if err != nil {
		t.Fatalf("Active returned error: %v", err)
	}
	if active != "" {
		t.Errorf("Active() = %q, want empty", active)
	}
}

// TestStoreKUIHomeOverride covers D18's hermetic KUI_HOME override: an empty
// root resolves to KUI_HOME, and two stores built over the same environment
// share the same .kui state.
func TestStoreKUIHomeOverride(t *testing.T) {
	home := t.TempDir()
	t.Setenv("KUI_HOME", home)

	first := New("")
	if err := first.Set("coder", "gpt-4o"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".kui", "models.json")); err != nil {
		t.Fatalf("models.json not written under KUI_HOME/.kui: %v", err)
	}

	second := New("")
	model, ok := second.Get("coder")
	if !ok || model != "gpt-4o" {
		t.Errorf("Get(coder) = (%q, %v), want (\"gpt-4o\", true) across stores via KUI_HOME", model, ok)
	}
}
