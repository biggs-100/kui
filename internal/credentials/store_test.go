package credentials

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestNewCredentialStore verifies that NewCredentialStore resolves the
// credentials file path under .kui/credentials.json relative to the root.
func TestNewCredentialStore(t *testing.T) {
	root := t.TempDir()
	cs := NewCredentialStore(root)
	if cs == nil {
		t.Fatal("NewCredentialStore returned nil")
	}

	want := filepath.Join(root, ".kui", "credentials.json")
	got := cs.credPath()
	if got != want {
		t.Errorf("credPath() = %q, want %q", got, want)
	}
}

// TestLoadEmpty verifies that loading when the file does not exist yields
// an empty credential set with no error (REQ-CRED-1).
func TestLoadEmpty(t *testing.T) {
	root := t.TempDir()
	cs := NewCredentialStore(root)

	if err := cs.Load(); err != nil {
		t.Fatalf("Load() on missing file returned error: %v", err)
	}

	// GetAPIKey on any provider should return not-found — the store is empty.
	if _, err := cs.GetAPIKey("openai"); err == nil {
		t.Error("GetAPIKey(openai) on empty store should return error")
	}
}

// TestLoadValid verifies that a well-formed credentials file is parsed
// correctly (REQ-CRED-2).
func TestLoadValid(t *testing.T) {
	root := t.TempDir()
	// Write a valid credentials file.
	dir := filepath.Join(root, ".kui")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data := map[string]any{
		"providers": map[string]any{
			"openai": map[string]string{"api_key": "sk-123"},
		},
	}
	raw, _ := json.Marshal(data)
	path := filepath.Join(dir, "credentials.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cs := NewCredentialStore(root)
	if err := cs.Load(); err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	got, err := cs.GetAPIKey("openai")
	if err != nil {
		t.Fatalf("GetAPIKey(openai) returned error: %v", err)
	}
	if got != "sk-123" {
		t.Errorf("GetAPIKey(openai) = %q, want %q", got, "sk-123")
	}
}

// TestLoadInvalid verifies that malformed JSON returns a descriptive parse
// error (REQ-CRED-2).
func TestLoadInvalid(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".kui")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(dir, "credentials.json")
	if err := os.WriteFile(path, []byte("{invalid"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cs := NewCredentialStore(root)
	if err := cs.Load(); err == nil {
		t.Fatal("Load() on malformed JSON should return error, got nil")
	}
}

// TestGetAPIKey verifies key lookup: found returns the key, not-found returns
// an error (REQ-CRED-4).
func TestGetAPIKey(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".kui")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data := map[string]any{
		"providers": map[string]any{
			"openai": map[string]string{"api_key": "sk-123"},
		},
	}
	raw, _ := json.Marshal(data)
	path := filepath.Join(dir, "credentials.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cs := NewCredentialStore(root)
	if err := cs.Load(); err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	// Found case.
	got, err := cs.GetAPIKey("openai")
	if err != nil {
		t.Fatalf("GetAPIKey(openai) error: %v", err)
	}
	if got != "sk-123" {
		t.Errorf("GetAPIKey(openai) = %q, want %q", got, "sk-123")
	}

	// Not found case.
	_, err = cs.GetAPIKey("anthropic")
	if err == nil {
		t.Error("GetAPIKey(anthropic) should return error for missing provider")
	}
}

// TestSetAPIKey verifies saving a new key and updating an existing key
// (REQ-CRED-5).
func TestSetAPIKey(t *testing.T) {
	root := t.TempDir()
	cs := NewCredentialStore(root)
	if err := cs.Load(); err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	// New key.
	if err := cs.SetAPIKey("openai", "sk-123"); err != nil {
		t.Fatalf("SetAPIKey(openai) error: %v", err)
	}
	got, err := cs.GetAPIKey("openai")
	if err != nil {
		t.Fatalf("GetAPIKey(openai) after Set error: %v", err)
	}
	if got != "sk-123" {
		t.Errorf("GetAPIKey(openai) = %q, want %q", got, "sk-123")
	}

	// Update existing key.
	if err := cs.SetAPIKey("openai", "sk-new"); err != nil {
		t.Fatalf("SetAPIKey(openai, update) error: %v", err)
	}
	got, err = cs.GetAPIKey("openai")
	if err != nil {
		t.Fatalf("GetAPIKey(openai) after update error: %v", err)
	}
	if got != "sk-new" {
		t.Errorf("GetAPIKey(openai) = %q, want %q", got, "sk-new")
	}
}

// TestSaveAndLoad verifies round-trip persistence: a key saved through one
// store instance is readable through a fresh instance over the same root.
func TestSaveAndLoad(t *testing.T) {
	root := t.TempDir()

	first := NewCredentialStore(root)
	if err := first.Load(); err != nil {
		t.Fatalf("first Load() error: %v", err)
	}
	if err := first.SetAPIKey("openai", "sk-abc"); err != nil {
		t.Fatalf("first SetAPIKey error: %v", err)
	}

	// New instance simulates a fresh session.
	second := NewCredentialStore(root)
	if err := second.Load(); err != nil {
		t.Fatalf("second Load() error: %v", err)
	}
	got, err := second.GetAPIKey("openai")
	if err != nil {
		t.Fatalf("second GetAPIKey error: %v", err)
	}
	if got != "sk-abc" {
		t.Errorf("round-trip GetAPIKey(openai) = %q, want %q", got, "sk-abc")
	}
}

// TestFilePermissions verifies that credentials.json is written with 0600
// permissions on Unix. On Windows the check is skipped (REQ-CRED-3).
func TestFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission check skipped on Windows")
	}

	root := t.TempDir()
	cs := NewCredentialStore(root)
	if err := cs.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if err := cs.SetAPIKey("openai", "sk-123"); err != nil {
		t.Fatalf("SetAPIKey error: %v", err)
	}

	info, err := os.Stat(cs.credPath())
	if err != nil {
		t.Fatalf("stat credentials file: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("credentials file permissions = %o, want 0600", perm)
	}
}

// TestSetAPIKeyCreatesDirectory verifies that saving a key creates the .kui
// directory if it does not exist (REQ-CRED-3).
func TestSetAPIKeyCreatesDirectory(t *testing.T) {
	root := t.TempDir()
	cs := NewCredentialStore(root)
	if err := cs.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if err := cs.SetAPIKey("openai", "sk-123"); err != nil {
		t.Fatalf("SetAPIKey error: %v", err)
	}

	dir := filepath.Join(root, ".kui")
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat .kui dir: %v", err)
	}
	if !info.IsDir() {
		t.Errorf(".kui is not a directory")
	}
}
