package plugin

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ConsentResponse represents the user's consent decision.
type ConsentResponse string

const (
	// ConsentApprove grants permission for this session only.
	ConsentApprove ConsentResponse = "approve"
	// ConsentDeny denies permission for this session only.
	ConsentDeny ConsentResponse = "deny"
	// ConsentAlwaysApprove grants permission permanently.
	ConsentAlwaysApprove ConsentResponse = "always_approve"
	// ConsentAlwaysDeny denies permission permanently.
	ConsentAlwaysDeny ConsentResponse = "always_deny"
)

// ConsentPrompt manages the interactive consent flow for plugin permissions.
type ConsentPrompt struct {
	Plugin      string
	Permissions []Permission
	Response    ConsentResponse
	filePath    string
}

// NewConsentPrompt creates a new consent prompt for the given plugin and permissions.
func NewConsentPrompt(plugin string, permissions []Permission) *ConsentPrompt {
	return &ConsentPrompt{
		Plugin:      plugin,
		Permissions: permissions,
	}
}

// Ask presents the consent prompt to the user and returns their response.
// This is a placeholder for the interactive TUI prompt.
func (cp *ConsentPrompt) Ask() (ConsentResponse, error) {
	// In a real implementation, this would use Bubble Tea to present
	// an interactive prompt. For now, return the stored response.
	return cp.Response, nil
}

// SavePreference persists the user's consent decision for a plugin.
func (cp *ConsentPrompt) SavePreference(response ConsentResponse) error {
	cp.Response = response

	if cp.filePath == "" {
		// Default location
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		cp.filePath = filepath.Join(home, ".config", "kui", "consent.yaml")
	}

	// Ensure directory exists
	dir := filepath.Dir(cp.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	type consentEntry struct {
		Plugin   string          `yaml:"plugin"`
		Response ConsentResponse `yaml:"response"`
	}

	type config struct {
		Preferences []consentEntry `yaml:"preferences"`
	}

	// Load existing config or create new
	c := config{}
	if data, err := os.ReadFile(cp.filePath); err == nil {
		_ = yaml.Unmarshal(data, &c)
	}

	// Update or add entry
	found := false
	for i, entry := range c.Preferences {
		if entry.Plugin == cp.Plugin {
			c.Preferences[i].Response = response
			found = true
			break
		}
	}
	if !found {
		c.Preferences = append(c.Preferences, consentEntry{
			Plugin:   cp.Plugin,
			Response: response,
		})
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(cp.filePath, data, 0644)
}

// LoadPreference retrieves the stored consent decision for a plugin.
// Returns the response and true if found, empty string and false otherwise.
func (cp *ConsentPrompt) LoadPreference() (ConsentResponse, bool) {
	if cp.filePath == "" {
		return "", false
	}

	data, err := os.ReadFile(cp.filePath)
	if err != nil {
		return "", false
	}

	type consentEntry struct {
		Plugin   string          `yaml:"plugin"`
		Response ConsentResponse `yaml:"response"`
	}

	type config struct {
		Preferences []consentEntry `yaml:"preferences"`
	}

	var c config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return "", false
	}

	for _, entry := range c.Preferences {
		if entry.Plugin == cp.Plugin {
			return entry.Response, true
		}
	}

	return "", false
}
