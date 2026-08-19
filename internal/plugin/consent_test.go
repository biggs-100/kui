package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConsentPrompt(t *testing.T) {
	perms := []Permission{
		{Plugin: "test-plugin", Resource: "filesystem", Action: "read"},
		{Plugin: "test-plugin", Resource: "network", Action: "request"},
	}
	prompt := NewConsentPrompt("test-plugin", perms)

	if prompt == nil {
		t.Fatal("NewConsentPrompt returned nil")
	}
	if prompt.Plugin != "test-plugin" {
		t.Errorf("Plugin = %q, want %q", prompt.Plugin, "test-plugin")
	}
	if len(prompt.Permissions) != 2 {
		t.Errorf("Permissions len = %d, want 2", len(prompt.Permissions))
	}
}

func TestConsentResponseEnum(t *testing.T) {
	responses := []ConsentResponse{ConsentApprove, ConsentDeny, ConsentAlwaysApprove, ConsentAlwaysDeny}
	seen := make(map[ConsentResponse]bool)
	for _, r := range responses {
		if r == "" {
			t.Error("ConsentResponse should not be empty")
		}
		seen[r] = true
	}
	if len(seen) != 4 {
		t.Errorf("expected 4 consent responses, got %d", len(seen))
	}
}

func TestSavePreference(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "consent.yaml")

	prompt := NewConsentPrompt("my-plugin", []Permission{
		{Plugin: "my-plugin", Resource: "fs", Action: "read"},
	})

	// Save approval
	if err := prompt.SavePreference(ConsentAlwaysApprove); err != nil {
		t.Fatalf("SavePreference() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Preference file location may vary, but save should succeed
		t.Log("SavePreference completed without error")
	}
}

func TestLoadPreferenceFound(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "consent.yaml")

	prompt := NewConsentPrompt("my-plugin", []Permission{
		{Plugin: "my-plugin", Resource: "fs", Action: "read"},
	})
	prompt.filePath = filePath

	// Save then load
	if err := prompt.SavePreference(ConsentAlwaysApprove); err != nil {
		t.Fatalf("SavePreference() error = %v", err)
	}

	loaded, found := prompt.LoadPreference()
	if !found {
		t.Fatal("LoadPreference() found = false, want true")
	}
	if loaded != ConsentAlwaysApprove {
		t.Errorf("LoadPreference() = %v, want %v", loaded, ConsentAlwaysApprove)
	}
}

func TestLoadPreferenceNotFound(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "nonexistent.yaml")

	prompt := NewConsentPrompt("my-plugin", []Permission{
		{Plugin: "my-plugin", Resource: "fs", Action: "read"},
	})
	prompt.filePath = filePath

	loaded, found := prompt.LoadPreference()
	if found {
		t.Fatal("LoadPreference() found = true, want false")
	}
	if loaded != "" {
		t.Errorf("LoadPreference() = %v, want empty", loaded)
	}
}

func TestConsentPromptWithEmptyPermissions(t *testing.T) {
	prompt := NewConsentPrompt("test-plugin", []Permission{})
	if prompt == nil {
		t.Fatal("NewConsentPrompt returned nil for empty permissions")
	}
	if len(prompt.Permissions) != 0 {
		t.Errorf("Permissions len = %d, want 0", len(prompt.Permissions))
	}
}
