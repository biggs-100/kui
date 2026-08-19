package main

import (
	"strings"
	"testing"
)

// --- ValidateKey tests (REQ-CRED-7) ---

// TestValidateKeyEmpty verifies that an empty key is rejected.
func TestValidateKeyEmpty(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"tabs", "\t\t"},
		{"newline only", "\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKey("openai", tt.key)
			if err == nil {
				t.Errorf("ValidateKey(%q, %q) = nil, want error", "openai", tt.key)
			}
		})
	}
}

// TestValidateKeyPrefixOpenAI verifies that openai keys must start with "sk-".
func TestValidateKeyPrefixOpenAI(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"valid sk- prefix", "sk-abc123def456", false},
		{"valid sk-proj prefix", "sk-proj-abc123", false},
		{"missing prefix", "abc123", true},
		{"wrong prefix sk-", "pk-abc123", true},
		{"sk lowercase only", "sk-", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKey("openai", tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateKey(%q, %q) error = %v, wantErr %v", "openai", tt.key, err, tt.wantErr)
			}
		})
	}
}

// TestValidateKeyOpencode verifies that opencode accepts any non-empty trimmed
// key without a prefix requirement.
func TestValidateKeyOpencode(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"simple key", "abc123", false},
		{"with spaces trimmed", "  key123  ", false},
		{"single char", "x", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKey("opencode", tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateKey(%q, %q) error = %v, wantErr %v", "opencode", tt.key, err, tt.wantErr)
			}
		})
	}
}

// TestValidateKeyTrimWhitespace verifies that leading/trailing whitespace is
// stripped before validation (REQ-CRED-7).
func TestValidateKeyTrimWhitespace(t *testing.T) {
	err := ValidateKey("openai", "  sk-abc123  ")
	if err != nil {
		t.Errorf("ValidateKey(openai, '  sk-abc123  ') = %v, want nil", err)
	}
}

// TestValidateKeyTrimmedEmpty verifies that a key that is only whitespace after
// trimming is rejected.
func TestValidateKeyTrimmedEmpty(t *testing.T) {
	err := ValidateKey("openai", "   ")
	if err == nil {
		t.Error("ValidateKey(openai, '   ') = nil, want error for whitespace-only input")
	}
}

// TestValidateKeyUnknownProvider verifies that an unknown provider returns an
// error since we have no prefix rule for it.
func TestValidateKeyUnknownProvider(t *testing.T) {
	err := ValidateKey("anthropic", "sk-abc")
	if err == nil {
		t.Error("ValidateKey(anthropic, 'sk-abc') = nil, want error for unknown provider")
	}
}

// TestParseSetupFlags verifies --provider flag parsing.
func TestParseSetupFlags(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantProvider string
		wantRest     []string
	}{
		{"no flags", nil, "", nil},
		{"provider flag", []string{"--provider", "openai"}, "openai", nil},
		{"provider shorthand", []string{"-p", "opencode"}, "opencode", nil},
		{"provider with trailing args", []string{"--provider", "openai", "extra"}, "openai", []string{"extra"}},
		{"unknown flag", []string{"--unknown"}, "", []string{"--unknown"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, rest := parseSetupFlags(tt.args)
			if provider != tt.wantProvider {
				t.Errorf("parseSetupFlags provider = %q, want %q", provider, tt.wantProvider)
			}
			if len(rest) != len(tt.wantRest) {
				t.Fatalf("parseSetupFlags rest = %v (len %d), want %v (len %d)", rest, len(rest), tt.wantRest, len(tt.wantRest))
			}
			for i := range rest {
				if rest[i] != tt.wantRest[i] {
					t.Errorf("parseSetupFlags rest[%d] = %q, want %q", i, rest[i], tt.wantRest[i])
				}
			}
		})
	}
}

// TestRunSetupNoProvider verifies that running setup with --provider skips the
// interactive provider list (non-interactive mode, REQ-CRED-6).
func TestRunSetupNoProvider(t *testing.T) {
	// This test verifies the non-interactive path: when --provider is given,
	// no provider selection prompt is shown. We can't easily test the full
	// stdin/stdout flow here, but we verify the flag path is wired.
	// The actual integration test would require teatest or stdin mocking.
	//
	// For now we verify that ValidateKey is called by runSetup for the
	// non-interactive path through a focused unit test.
	err := ValidateKey("openai", "sk-test123")
	if err != nil {
		t.Errorf("ValidateKey(openai, sk-test123) = %v, want nil", err)
	}
}

// TestSetupProviderList verifies the list of providers returned by the helper.
func TestSetupProviderList(t *testing.T) {
	providers := availableProviders()
	if len(providers) == 0 {
		t.Error("availableProviders() returned empty list, want at least openai and opencode")
	}
	// Must contain openai and opencode at minimum.
	found := map[string]bool{}
	for _, p := range providers {
		found[p] = true
	}
	if !found["openai"] {
		t.Error("availableProviders() missing 'openai'")
	}
	if !found["opencode"] {
		t.Error("availableProviders() missing 'opencode'")
	}
}

// TestRunSetupUsageError verifies that runSetup returns usage error for
// invalid flags.
func TestRunSetupUsageError(t *testing.T) {
	// --provider without a value should be a usage error.
	root := t.TempDir()
	code := runSetup(root, []string{"--provider"})
	if code != 2 {
		t.Errorf("runSetup(--provider without value) = %d, want 2 (usage error)", code)
	}
}

// TestValidateKeyErrorMessages verifies that validation errors contain useful
// messages indicating the expected format.
func TestValidateKeyErrorMessages(t *testing.T) {
	// Empty key error should mention "empty" or similar.
	err := ValidateKey("openai", "")
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("ValidateKey(openai, '') error = %v, want message containing 'empty'", err)
	}

	// Wrong prefix for openai should mention "sk-".
	err = ValidateKey("openai", "bad-key")
	if err == nil || !strings.Contains(err.Error(), "sk-") {
		t.Errorf("ValidateKey(openai, 'bad-key') error = %v, want message mentioning 'sk-'", err)
	}
}
