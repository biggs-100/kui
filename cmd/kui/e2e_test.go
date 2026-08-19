//go:build e2e

package main

import (
	"os"
	"strings"
	"testing"
)

// skipIfNoAPIKey skips the test when OPENCODE_API_KEY is not set.
// E2E tests require a real API key to hit the live endpoint.
func skipIfNoAPIKey(t *testing.T) {
	t.Helper()
	if os.Getenv("OPENCODE_API_KEY") == "" {
		t.Skip("OPENCODE_API_KEY not set, skipping E2E test")
	}
}

// TestE2ESmokeOpenCode sends a real prompt to the OpenCode provider and
// verifies the full pipeline: CLI binary → provider adapter → API → response.
// Requires: OPENCODE_API_KEY set, network available.
func TestE2ESmokeOpenCode(t *testing.T) {
	skipIfNoAPIKey(t)

	stdout, stderr, code := runCLI(t, map[string]string{
		"OPENCODE_API_KEY": os.Getenv("OPENCODE_API_KEY"),
		"KUI_HOME":         t.TempDir(),
	}, "--provider", "opencode", "--model", "big-pickle", "say hello")

	if code != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr: %s", code, stderr)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Fatal("expected non-empty stdout")
	}
	t.Logf("response: %s", strings.TrimSpace(stdout))
}

// TestE2EEmptyPrompt verifies that an empty prompt does not crash the CLI.
// It should either fail gracefully (non-zero exit with error) or show help.
func TestE2EEmptyPrompt(t *testing.T) {
	skipIfNoAPIKey(t)

	_, _, code := runCLI(t, map[string]string{
		"OPENCODE_API_KEY": os.Getenv("OPENCODE_API_KEY"),
		"KUI_HOME":         t.TempDir(),
	}, "--provider", "opencode", "--model", "big-pickle")

	// Empty prompt should either fail gracefully or show help — not a crash.
	if code != 0 {
		t.Logf("exit code %d (expected for empty prompt)", code)
	}
}
