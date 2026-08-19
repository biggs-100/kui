package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/biggs-100/kui/internal/credentials"
)

// ---------------------------------------------------------------------------
// Setup Integration Tests — WARNING 1 (Interactive stdin) & WARNING 2 (Stdout)
// ---------------------------------------------------------------------------

// pipeStdin replaces os.Stdin with a pipe fed by input, returns a restore func.
func pipeStdin(t *testing.T, input string) func() {
	t.Helper()
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdin = r
	// Write input in a goroutine so the reader doesn't block.
	go func() {
		defer w.Close()
		_, _ = w.WriteString(input)
	}()
	return func() {
		os.Stdin = oldStdin
		r.Close()
	}
}

// captureStdout replaces os.Stdout with a pipe, returns the restore func and a
// function that reads whatever was written.
func captureStdout(t *testing.T) (readOutput func() string, restore func()) {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	return func() string {
		w.Close()
		var buf [4096]byte
		n, _ := r.Read(buf[:])
		os.Stdout = oldStdout
		r.Close()
		return string(buf[:n])
	}, func() {
		w.Close()
		os.Stdout = oldStdout
		r.Close()
	}
}

// TestSetupInteractiveFlow covers WARNING 1 / Scenario 11: the full
// interactive stdin flow — provider selection (numeric) + API key input —
// must complete successfully and persist credentials.
func TestSetupInteractiveFlow(t *testing.T) {
	// Simulate user typing "1\n" for provider selection, then an API key.
	input := "1\nsk-integration-test-12345\n"
	restoreStdin := pipeStdin(t, input)
	defer restoreStdin()

	root := t.TempDir()
	code := runSetup(root, nil) // no --provider flag → interactive path

	if code != 0 {
		t.Fatalf("runSetup interactive = %d, want 0 (success)", code)
	}

	// Verify credential store was updated.
	cs := credentials.NewCredentialStore(root)
	if err := cs.Load(); err != nil {
		t.Fatalf("Load credentials: %v", err)
	}
	key, err := cs.GetAPIKey("openai")
	if err != nil {
		t.Fatalf("GetAPIKey(openai) after interactive setup: %v", err)
	}
	if key != "sk-integration-test-12345" {
		t.Errorf("stored key = %q, want %q", key, "sk-integration-test-12345")
	}
}

// TestSetupInteractiveDefaultSelection covers the case where the user presses
// Enter without typing a number — the default selection [1] must be used.
func TestSetupInteractiveDefaultSelection(t *testing.T) {
	// Empty line for provider (default "1"), then API key.
	input := "\nsk-default-selection-test\n"
	restoreStdin := pipeStdin(t, input)
	defer restoreStdin()

	root := t.TempDir()
	code := runSetup(root, nil)

	if code != 0 {
		t.Fatalf("runSetup default selection = %d, want 0", code)
	}

	cs := credentials.NewCredentialStore(root)
	if err := cs.Load(); err != nil {
		t.Fatalf("Load credentials: %v", err)
	}
	key, err := cs.GetAPIKey("openai")
	if err != nil {
		t.Fatalf("GetAPIKey(openai): %v", err)
	}
	if key != "sk-default-selection-test" {
		t.Errorf("stored key = %q, want %q", key, "sk-default-selection-test")
	}
}

// TestSetupInteractiveSecondProvider covers selecting the second provider
// (opencode) by typing "2".
func TestSetupInteractiveSecondProvider(t *testing.T) {
	input := "2\nopencode-test-key-abc\n"
	restoreStdin := pipeStdin(t, input)
	defer restoreStdin()

	root := t.TempDir()
	code := runSetup(root, nil)

	if code != 0 {
		t.Fatalf("runSetup second provider = %d, want 0", code)
	}

	cs := credentials.NewCredentialStore(root)
	if err := cs.Load(); err != nil {
		t.Fatalf("Load credentials: %v", err)
	}
	key, err := cs.GetAPIKey("opencode")
	if err != nil {
		t.Fatalf("GetAPIKey(opencode): %v", err)
	}
	if key != "opencode-test-key-abc" {
		t.Errorf("stored key = %q, want %q", key, "opencode-test-key-abc")
	}
}

// TestSetupInteractiveInvalidSelection covers an out-of-range provider number
// causing a usage error (exit 2).
func TestSetupInteractiveInvalidSelection(t *testing.T) {
	input := "9\nsk-valid-key\n"
	restoreStdin := pipeStdin(t, input)
	defer restoreStdin()

	root := t.TempDir()
	code := runSetup(root, nil)

	if code != 2 {
		t.Errorf("runSetup(invalid selection 9) = %d, want 2 (usage error)", code)
	}
}

// TestSetupSuccessOutput covers WARNING 2 / Scenario 16: the success message
// printed to stdout must contain "Credentials saved" and "Next step".
func TestSetupSuccessOutput(t *testing.T) {
	input := "sk-success-output-test\n"
	restoreStdin := pipeStdin(t, input)
	defer restoreStdin()

	readStdout, _ := captureStdout(t)

	root := t.TempDir()
	code := runSetup(root, []string{"--provider", "openai"})

	if code != 0 {
		t.Fatalf("runSetup = %d, want 0; stdout = %q", code, readStdout())
	}

	stdout := readStdout()
	if !strings.Contains(stdout, "Credentials saved") {
		t.Errorf("stdout = %q, want it to contain 'Credentials saved'", stdout)
	}
	if !strings.Contains(stdout, "Next step") {
		t.Errorf("stdout = %q, want it to contain 'Next step'", stdout)
	}
}

// TestSetupSuccessOutputInteractive covers the success output for the
// interactive path (no --provider flag).
func TestSetupSuccessOutputInteractive(t *testing.T) {
	input := "1\nsk-interactive-output-test\n"
	restoreStdin := pipeStdin(t, input)
	defer restoreStdin()

	readStdout, _ := captureStdout(t)

	root := t.TempDir()
	code := runSetup(root, nil)

	if code != 0 {
		t.Fatalf("runSetup interactive = %d, want 0", code)
	}

	stdout := readStdout()
	if !strings.Contains(stdout, "Credentials saved for openai") {
		t.Errorf("stdout = %q, want it to contain 'Credentials saved for openai'", stdout)
	}
	if !strings.Contains(stdout, "kui tui") {
		t.Errorf("stdout = %q, want it to mention 'kui tui' as next step", stdout)
	}
}

// TestSetupNonInteractiveWithFlag covers the non-interactive path: when
// --provider is given, stdin is only read once (for the API key), not twice.
func TestSetupNonInteractiveWithFlag(t *testing.T) {
	input := "sk-noninteractive-key\n"
	restoreStdin := pipeStdin(t, input)
	defer restoreStdin()

	root := t.TempDir()
	code := runSetup(root, []string{"--provider", "opencode"})

	if code != 0 {
		t.Fatalf("runSetup --provider opencode = %d, want 0", code)
	}

	cs := credentials.NewCredentialStore(root)
	if err := cs.Load(); err != nil {
		t.Fatalf("Load credentials: %v", err)
	}
	key, err := cs.GetAPIKey("opencode")
	if err != nil {
		t.Fatalf("GetAPIKey(opencode): %v", err)
	}
	if key != "sk-noninteractive-key" {
		t.Errorf("stored key = %q, want %q", key, "sk-noninteractive-key")
	}
}

// TestSetupCredentialFileExists covers the scenario where a credentials file
// already exists — setup must update it without clobbering other providers.
func TestSetupCredentialFileExists(t *testing.T) {
	root := t.TempDir()

	// Pre-seed with an existing provider.
	cs := credentials.NewCredentialStore(root)
	if err := cs.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := cs.SetAPIKey("opencode", "existing-key"); err != nil {
		t.Fatalf("SetAPIKey: %v", err)
	}

	// Now run setup for openai.
	input := "sk-preexisting-openai-key\n"
	restoreStdin := pipeStdin(t, input)
	defer restoreStdin()

	code := runSetup(root, []string{"--provider", "openai"})
	if code != 0 {
		t.Fatalf("runSetup = %d, want 0", code)
	}

	// Verify both providers exist.
	if err := cs.Load(); err != nil {
		t.Fatalf("Load after setup: %v", err)
	}
	openaiKey, err := cs.GetAPIKey("openai")
	if err != nil {
		t.Fatalf("GetAPIKey(openai): %v", err)
	}
	if openaiKey != "sk-preexisting-openai-key" {
		t.Errorf("openai key = %q, want %q", openaiKey, "sk-preexisting-openai-key")
	}

	opencodeKey, err := cs.GetAPIKey("opencode")
	if err != nil {
		t.Fatalf("GetAPIKey(opencode): %v", err)
	}
	if opencodeKey != "existing-key" {
		t.Errorf("opencode key = %q, want %q (must not be clobbered)", opencodeKey, "existing-key")
	}
}

// TestSetupCredentialFileOnDisk verifies the credentials file is written to
// the expected path and is valid JSON.
func TestSetupCredentialFileOnDisk(t *testing.T) {
	input := "sk-file-path-test\n"
	restoreStdin := pipeStdin(t, input)
	defer restoreStdin()

	root := t.TempDir()
	code := runSetup(root, []string{"--provider", "openai"})
	if code != 0 {
		t.Fatalf("runSetup = %d, want 0", code)
	}

	credPath := filepath.Join(root, ".kui", "credentials.json")
	data, err := os.ReadFile(credPath)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", credPath, err)
	}
	if !strings.Contains(string(data), "sk-file-path-test") {
		t.Errorf("credentials.json = %q, want it to contain the API key", string(data))
	}
	if !strings.Contains(string(data), "openai") {
		t.Errorf("credentials.json = %q, want it to contain 'openai'", string(data))
	}
}
