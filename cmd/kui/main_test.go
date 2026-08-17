package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// binPath is the CLI binary built once in TestMain so every case exercises the
// real process, including exit codes and output streams (D13, REQ-CLI-1..2).
var binPath string

func TestMain(m *testing.M) {
	binDir, err := os.MkdirTemp("", "kui-bin-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "make temp dir: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = os.RemoveAll(binDir) }()

	name := "kui"
	if runtime.GOOS == "windows" {
		name = "kui.exe"
	}
	binPath = filepath.Join(binDir, name)

	// go test runs with the working directory set to the package directory.
	build := exec.Command("go", "build", "-o", binPath, ".")
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build kui binary: %v\n%s", err, out)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// cleanEnv returns the current environment with every OPENAI_* variable
// removed, then applies overrides. An override mapped to "" keeps the variable
// unset, so tests never depend on the developer machine's credentials.
func cleanEnv(overrides map[string]string) []string {
	env := make([]string, 0, len(os.Environ()))
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "OPENAI_") {
			continue
		}
		env = append(env, kv)
	}
	for k, v := range overrides {
		if v != "" {
			env = append(env, k+"="+v)
		}
	}
	return env
}

// runCLI executes the built binary and returns stdout, stderr, and the exit
// code.
func runCLI(t *testing.T, env map[string]string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Env = cleanEnv(env)
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("run %s: %v", binPath, err)
		}
		code = exitErr.ExitCode()
	}
	return out.String(), errOut.String(), code
}

// TestCLINoPrompt covers REQ-CLI-1 "No prompt": without arguments the CLI
// prints usage to stderr and exits 2 (D13).
func TestCLINoPrompt(t *testing.T) {
	stdout, stderr, code := runCLI(t, nil)

	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty (usage is an error, not the answer)", stdout)
	}
	if !strings.Contains(stderr, "PROMPT") {
		t.Errorf("stderr = %q, want usage text mentioning PROMPT", stderr)
	}
}

// TestCLIMissingKey covers REQ-CLI-2 "Missing API key": without
// OPENAI_API_KEY the CLI prints an actionable error naming the variable to
// stderr and exits 1.
func TestCLIMissingKey(t *testing.T) {
	_, stderr, code := runCLI(t, map[string]string{"OPENAI_API_KEY": "", "KUI_HOME": t.TempDir()}, "hello")

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "OPENAI_API_KEY") {
		t.Errorf("stderr = %q, want an error naming OPENAI_API_KEY", stderr)
	}
}

// TestCLIProviderFailure covers REQ-CLI-2 "the loop fails": a provider error
// (HTTP 500) is reported on stderr with exit code 1.
func TestCLIProviderFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"error":{"message":"boom"}}`)
	}))
	defer srv.Close()

	_, stderr, code := runCLI(t, map[string]string{
		"OPENAI_API_KEY":  "sk-test-123",
		"OPENAI_BASE_URL": srv.URL,
		"KUI_HOME":        t.TempDir(),
	}, "hello")

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "provider server error") {
		t.Errorf("stderr = %q, want a provider failure message", stderr)
	}
}

// TestCLISuccess covers REQ-CLI-1 "Prompt with tool use" and REQ-CLI-2
// "Successful completion": with a reachable provider the CLI prints the final
// answer to stdout and exits 0. The fake provider also proves the request
// carries the prompt and an explicit model field.
func TestCLISuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		if req.Model == "" {
			t.Error("request has no model field")
		}
		if len(req.Messages) == 0 || req.Messages[0].Role != "user" || req.Messages[0].Content != "hello world" {
			t.Errorf("messages[0] = %+v, want user prompt %q", req.Messages, "hello world")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"hello from fake provider"}}]}`)
	}))
	defer srv.Close()

	stdout, stderr, code := runCLI(t, map[string]string{
		"OPENAI_API_KEY":  "sk-test-123",
		"OPENAI_BASE_URL": srv.URL,
		"KUI_HOME":        t.TempDir(),
	}, "hello", "world")

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if stdout != "hello from fake provider\n" {
		t.Errorf("stdout = %q, want the answer %q", stdout, "hello from fake provider\n")
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty on success", stderr)
	}
}

// TestCLIPromptDashDash covers REQ-CLI-1 "Prompt given as --": the "--"
// separator runs the remaining words as the prompt (the separator is
// consumed, not part of the prompt), so a prompt that starts with "profile"
// or a dash is still reachable.
func TestCLIPromptDashDash(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		if len(req.Messages) == 0 || req.Messages[0].Content != "hello world" {
			t.Errorf("messages[0] = %+v, want the prompt %q without the -- separator", req.Messages, "hello world")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"dash answer"}}]}`)
	}))
	defer srv.Close()

	stdout, stderr, code := runCLI(t, map[string]string{
		"OPENAI_API_KEY":  "sk-test-123",
		"OPENAI_BASE_URL": srv.URL,
		"KUI_HOME":        t.TempDir(),
	}, "--", "hello", "world")

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if stdout != "dash answer\n" {
		t.Errorf("stdout = %q, want the answer %q", stdout, "dash answer\n")
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty on success", stderr)
	}
}

// ---------------------------------------------------------------------------
// Profile subcommands (REQ-PCLI-1..3, REQ-CLI-3/4, D18).
// All cases run the real binary with a hermetic KUI_HOME so no state leaks
// from the developer machine.
// ---------------------------------------------------------------------------

// writeProfileDir creates <home>/profiles/<name>/profile.yaml with the given
// content.
func writeProfileDir(t *testing.T, home, name, yamlContent string) {
	t.Helper()
	dir := filepath.Join(home, "profiles", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "profile.yaml"), []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("WriteFile(profile.yaml): %v", err)
	}
}

// writeHomeFile writes <home>/<relPath> (creating parents) with the given
// content, for .kui/active, .kui/models.json, and SYSTEM.md fixtures.
func writeHomeFile(t *testing.T, home, relPath, content string) {
	t.Helper()
	path := filepath.Join(home, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

// recordingServer captures every request body and answers with "final",
// letting smoke tests assert what the loop actually sent to the provider.
func recordingServer(t *testing.T, bodies *[]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		*bodies = append(*bodies, string(body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"final"}}]}`)
	}))
}

// TestCLIProfileListMarksActive covers REQ-PCLI-1 "List with active marker":
// both profiles print, the active one is marked, and the exit is zero.
func TestCLIProfileListMarksActive(t *testing.T) {
	home := t.TempDir()
	writeProfileDir(t, home, "coder", "name: coder\nsystem_prompt: SYSTEM.md\ntools: [read_file]\n")
	writeProfileDir(t, home, "ops", "name: ops\nsystem_prompt: SYSTEM.md\ntools: [bash]\n")
	writeHomeFile(t, home, filepath.Join(".kui", "active"), "coder")

	stdout, stderr, code := runCLI(t, map[string]string{"KUI_HOME": home}, "profile", "list")

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if want := "* coder\n  ops\n"; stdout != want {
		t.Errorf("stdout = %q, want %q (coder marked active)", stdout, want)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

// TestCLIProfileListNoProfiles covers REQ-PCLI-1 "No profiles": an empty
// profile root prints an empty list and exits zero.
func TestCLIProfileListNoProfiles(t *testing.T) {
	home := t.TempDir()

	stdout, stderr, code := runCLI(t, map[string]string{"KUI_HOME": home}, "profile", "list")

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want an empty list", stdout)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

// TestCLIProfileSwitchUnknown covers REQ-PCLI-2 "Switch to unknown profile":
// stderr names the profile and the exit is non-zero.
func TestCLIProfileSwitchUnknown(t *testing.T) {
	home := t.TempDir()

	_, stderr, code := runCLI(t, map[string]string{"KUI_HOME": home}, "profile", "switch", "nope")

	if code == 0 {
		t.Error("exit code = 0, want non-zero for an unknown profile")
	}
	if !strings.Contains(stderr, "nope") {
		t.Errorf("stderr = %q, want it to name the unknown profile %q", stderr, "nope")
	}
}

// TestCLIProfileSwitchKnown covers REQ-PCLI-2 "Switch to known profile": the
// profile is persisted to .kui/active and the exit is zero.
func TestCLIProfileSwitchKnown(t *testing.T) {
	home := t.TempDir()
	writeProfileDir(t, home, "coder", "name: coder\n")

	stdout, stderr, code := runCLI(t, map[string]string{"KUI_HOME": home}, "profile", "switch", "coder")

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if stdout != "" || stderr != "" {
		t.Errorf("stdout = %q, stderr = %q, want both empty on success", stdout, stderr)
	}
	data, err := os.ReadFile(filepath.Join(home, ".kui", "active"))
	if err != nil {
		t.Fatalf("read .kui/active: %v", err)
	}
	if string(data) != "coder" {
		t.Errorf(".kui/active = %q, want %q", string(data), "coder")
	}
}

// TestCLIProfileSwitchDashPrompt covers D18 "dual path": `switch <name> --
// <prompt>` persists the activation AND runs a session whose steering switch
// applies the profile mid-run with the profile-context marker.
func TestCLIProfileSwitchDashPrompt(t *testing.T) {
	var bodies []string
	srv := recordingServer(t, &bodies)
	defer srv.Close()

	home := t.TempDir()
	writeProfileDir(t, home, "coder", "name: coder\nsystem_prompt: SYSTEM.md\n")
	writeHomeFile(t, home, filepath.Join("profiles", "coder", "SYSTEM.md"), "You are the coder profile.\n")

	_, stderr, code := runCLI(t, map[string]string{
		"OPENAI_API_KEY":  "sk-test-123",
		"OPENAI_BASE_URL": srv.URL,
		"KUI_HOME":        home,
	}, "profile", "switch", "coder", "--", "hello")

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if len(bodies) < 2 {
		t.Fatalf("provider received %d requests, want 2 (initial turn + steering-drained turn)", len(bodies))
	}
	if !strings.Contains(bodies[1], "You are the coder profile.") {
		t.Errorf("second request %q does not contain the profile system prompt", bodies[1])
	}
	if !strings.Contains(bodies[1], "Profile switched to coder") {
		t.Errorf("second request %q does not contain the profile-context marker", bodies[1])
	}
	data, err := os.ReadFile(filepath.Join(home, ".kui", "active"))
	if err != nil {
		t.Fatalf("read .kui/active: %v", err)
	}
	if string(data) != "coder" {
		t.Errorf(".kui/active = %q, want %q (dual-path activation persists)", string(data), "coder")
	}
}

// TestCLIProfileModelPersists covers REQ-PCLI-3 "Model set persists": the
// model is written to .kui/models.json and the exit is zero.
func TestCLIProfileModelPersists(t *testing.T) {
	home := t.TempDir()
	writeProfileDir(t, home, "coder", "name: coder\n")

	stdout, stderr, code := runCLI(t, map[string]string{"KUI_HOME": home}, "profile", "model", "coder", "gpt-4o")

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	data, err := os.ReadFile(filepath.Join(home, ".kui", "models.json"))
	if err != nil {
		t.Fatalf("read .kui/models.json: %v", err)
	}
	models := map[string]string{}
	if err := json.Unmarshal(data, &models); err != nil {
		t.Fatalf("models.json is not a valid map: %v", err)
	}
	if models["coder"] != "gpt-4o" {
		t.Errorf("models.json = %v, want coder -> gpt-4o", models)
	}
}

// TestCLIProfileModelUnknownProfile covers REQ-CLI-3: setting a model for an
// unknown profile names the profile and exits non-zero.
func TestCLIProfileModelUnknownProfile(t *testing.T) {
	home := t.TempDir()

	_, stderr, code := runCLI(t, map[string]string{"KUI_HOME": home}, "profile", "model", "nope", "gpt-4o")

	if code == 0 {
		t.Error("exit code = 0, want non-zero for an unknown profile")
	}
	if !strings.Contains(stderr, "nope") {
		t.Errorf("stderr = %q, want it to name the unknown profile %q", stderr, "nope")
	}
}

// TestCLIProfileModelMissingArgument covers REQ-PCLI-3 "Missing arguments":
// usage prints and the exit is non-zero (2, D13).
func TestCLIProfileModelMissingArgument(t *testing.T) {
	home := t.TempDir()

	_, stderr, code := runCLI(t, map[string]string{"KUI_HOME": home}, "profile", "model", "coder")

	if code != 2 {
		t.Errorf("exit code = %d, want 2 (usage)", code)
	}
	if !strings.Contains(stderr, "profile") {
		t.Errorf("stderr = %q, want usage text for the profile subcommands", stderr)
	}
}

// TestCLIProfileNoSubcommand covers REQ-CLI-1 "No prompt" for the profile
// group: `kui profile` without a subcommand prints usage and exits 2.
func TestCLIProfileNoSubcommand(t *testing.T) {
	home := t.TempDir()

	_, stderr, code := runCLI(t, map[string]string{"KUI_HOME": home}, "profile")

	if code != 2 {
		t.Errorf("exit code = %d, want 2 (usage)", code)
	}
	if !strings.Contains(stderr, "profile") {
		t.Errorf("stderr = %q, want usage text mentioning profile subcommands", stderr)
	}
}

// TestCLIProfileModelResolutionSavedWins covers REQ-CLI-4 "Saved model wins":
// the saved model from .kui/models.json wins over the profile.yaml model, and
// the provider is configured with it at session start.
func TestCLIProfileModelResolutionSavedWins(t *testing.T) {
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		bodies = append(bodies, req.Model)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"final"}}]}`)
	}))
	defer srv.Close()

	home := t.TempDir()
	writeProfileDir(t, home, "coder", "name: coder\nmodel: gpt-4o-mini\nsystem_prompt: SYSTEM.md\n")
	writeHomeFile(t, home, filepath.Join("profiles", "coder", "SYSTEM.md"), "You are the coder profile.\n")
	writeHomeFile(t, home, filepath.Join(".kui", "active"), "coder")
	writeHomeFile(t, home, filepath.Join(".kui", "models.json"), `{"coder":"gpt-4o"}`)

	_, stderr, code := runCLI(t, map[string]string{
		"OPENAI_API_KEY":  "sk-test-123",
		"OPENAI_BASE_URL": srv.URL,
		"KUI_HOME":        home,
	}, "hello")

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if len(bodies) == 0 {
		t.Fatal("provider received no requests")
	}
	if bodies[0] != "gpt-4o" {
		t.Errorf("request model = %q, want %q (saved model wins over profile.yaml)", bodies[0], "gpt-4o")
	}
}

// TestCLIProfileModelFallbackToGlobal covers REQ-CLI-4 "Fallback chain": with
// no saved model and no profile.yaml model, the global OPENAI_MODEL is used.
func TestCLIProfileModelFallbackToGlobal(t *testing.T) {
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		bodies = append(bodies, req.Model)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"final"}}]}`)
	}))
	defer srv.Close()

	home := t.TempDir()
	writeProfileDir(t, home, "coder", "name: coder\nsystem_prompt: SYSTEM.md\n")
	writeHomeFile(t, home, filepath.Join("profiles", "coder", "SYSTEM.md"), "You are the coder profile.\n")
	writeHomeFile(t, home, filepath.Join(".kui", "active"), "coder")

	_, stderr, code := runCLI(t, map[string]string{
		"OPENAI_API_KEY":  "sk-test-123",
		"OPENAI_BASE_URL": srv.URL,
		"OPENAI_MODEL":    "env-model",
		"KUI_HOME":        home,
	}, "hello")

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if len(bodies) == 0 {
		t.Fatal("provider received no requests")
	}
	if bodies[0] != "env-model" {
		t.Errorf("request model = %q, want %q (OPENAI_MODEL fallback)", bodies[0], "env-model")
	}
}

// TestCLIProfileSystemPromptInjected covers the PR 4 note: the CLI combines
// the active profile's SYSTEM.md and the skills index system messages and
// seeds them through the steering queue, so the provider sees the profile
// system context from the second request.
func TestCLIProfileSystemPromptInjected(t *testing.T) {
	var bodies []string
	srv := recordingServer(t, &bodies)
	defer srv.Close()

	home := t.TempDir()
	writeProfileDir(t, home, "coder", "name: coder\nsystem_prompt: SYSTEM.md\nskills: []\n")
	writeHomeFile(t, home, filepath.Join("profiles", "coder", "SYSTEM.md"), "You are the coder profile.\n")
	writeHomeFile(t, home, filepath.Join(".kui", "active"), "coder")

	_, stderr, code := runCLI(t, map[string]string{
		"OPENAI_API_KEY":  "sk-test-123",
		"OPENAI_BASE_URL": srv.URL,
		"KUI_HOME":        home,
	}, "hello")

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if len(bodies) < 2 {
		t.Fatalf("provider received %d requests, want 2 (initial turn + steering-drained turn)", len(bodies))
	}
	if !strings.Contains(bodies[1], "You are the coder profile.") {
		t.Errorf("second request %q does not contain the profile system prompt", bodies[1])
	}
}

// ---------------------------------------------------------------------------
// TUI subcommand (REQ-CLI-5, REQ-TUI-APP-1).
// ---------------------------------------------------------------------------

// TestCLITUIDispatchStartupFailure covers REQ-CLI-5 "Startup validation
// failure": `kui tui` without a valid provider prints an actionable error
// to stderr and exits non-zero.
func TestCLITUIDispatchStartupFailure(t *testing.T) {
	_, stderr, code := runCLI(t, map[string]string{
		"OPENAI_API_KEY": "",
		"KUI_HOME":       t.TempDir(),
	}, "tui")

	if code != 1 {
		t.Errorf("exit code = %d, want 1 (startup validation failure)", code)
	}
	if !strings.Contains(stderr, "OPENAI_API_KEY") {
		t.Errorf("stderr = %q, want an error naming OPENAI_API_KEY", stderr)
	}
}

// TestCLIUsageIncludesTuiSubcommand covers REQ-CLI-5: the usage text must
// mention `kui tui` as a subcommand.
func TestCLIUsageIncludesTuiSubcommand(t *testing.T) {
	_, stderr, code := runCLI(t, nil)

	if code != 2 {
		t.Errorf("exit code = %d, want 2 (usage)", code)
	}
	if !strings.Contains(stderr, "kui tui") {
		t.Errorf("stderr = %q, want usage text mentioning 'kui tui'", stderr)
	}
}

// ---------------------------------------------------------------------------
// Remote Skills Wiring (REQ-RS-13, Slice C).
// ---------------------------------------------------------------------------

// TestCLIProfileWithSkillsURL covers REQ-RS-13: a profile declaring a skills
// registry URL must wire it through to NewIndex. The CLI must succeed (exit 0)
// even when the remote registry is unreachable — registry failures are logged
// as warnings, not fatal errors (REQ-RS-18).
func TestCLIProfileWithSkillsURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"remote-skills-ok"}}]}`)
	}))
	defer srv.Close()

	home := t.TempDir()
	// Profile declares a remote skills URL — classifySkillsPaths must extract
	// it and pass it to NewIndex (REQ-RS-13).
	writeProfileDir(t, home, "remote", `name: remote
system_prompt: SYSTEM.md
skills:
  - "https://example.com/skills/index.json"
`)
	writeHomeFile(t, home, filepath.Join("profiles", "remote", "SYSTEM.md"), "You are remote.\n")
	writeHomeFile(t, home, filepath.Join(".kui", "active"), "remote")

	stdout, stderr, code := runCLI(t, map[string]string{
		"OPENAI_API_KEY":  "sk-test-123",
		"OPENAI_BASE_URL": srv.URL,
		"KUI_HOME":        home,
	}, "hello")

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if stdout != "remote-skills-ok\n" {
		t.Errorf("stdout = %q, want the answer", stdout)
	}
}

// TestCLIProfileWithMixedSkills covers REQ-RS-13/REQ-RS-14: a profile with
// both local skill names and remote registry URLs must classify them correctly.
// Local names are directory names; URLs are passed to NewIndex as registries.
func TestCLIProfileWithMixedSkills(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"mixed-ok"}}]}`)
	}))
	defer srv.Close()

	home := t.TempDir()
	writeProfileDir(t, home, "mixed", `name: mixed
system_prompt: SYSTEM.md
skills:
  - "go-testing"
  - "https://example.com/skills/index.json"
`)
	writeHomeFile(t, home, filepath.Join("profiles", "mixed", "SYSTEM.md"), "You are mixed.\n")
	writeHomeFile(t, home, filepath.Join(".kui", "active"), "mixed")

	stdout, stderr, code := runCLI(t, map[string]string{
		"OPENAI_API_KEY":  "sk-test-123",
		"OPENAI_BASE_URL": srv.URL,
		"KUI_HOME":        home,
	}, "hello")

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if stdout != "mixed-ok\n" {
		t.Errorf("stdout = %q, want the answer", stdout)
	}
}

// TestCLIOneShotPromptUnchanged covers REQ-CLI-5: the existing one-shot
// prompt behavior must remain unchanged after adding kui tui.
func TestCLIOneShotPromptUnchanged(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"hello from fake"}}]}`)
	}))
	defer srv.Close()

	stdout, stderr, code := runCLI(t, map[string]string{
		"OPENAI_API_KEY":  "sk-test-123",
		"OPENAI_BASE_URL": srv.URL,
		"KUI_HOME":        t.TempDir(),
	}, "hello")

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if stdout != "hello from fake\n" {
		t.Errorf("stdout = %q, want the answer", stdout)
	}
}

// TestCLIModelFlagOverride verifies that --model flag overrides the resolved
// model in the REQ-CLI-4 chain. This tests the integration of parseFlags
// with the CLI runtime.
func TestCLIModelFlagOverride(t *testing.T) {
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		bodies = append(bodies, req.Model)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"model override ok"}}]}`)
	}))
	defer srv.Close()

	home := t.TempDir()
	// No saved model, no profile.yaml model — only --model flag.
	_, stderr, code := runCLI(t, map[string]string{
		"OPENAI_API_KEY":  "sk-test-123",
		"OPENAI_BASE_URL": srv.URL,
		"KUI_HOME":        home,
	}, "--model", "gpt-4o", "hello")

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if len(bodies) == 0 {
		t.Fatal("provider received no requests")
	}
	if bodies[0] != "gpt-4o" {
		t.Errorf("request model = %q, want %q (flag override)", bodies[0], "gpt-4o")
	}
}

// TestCLIThinkingFlagSendsReasoningEffort verifies that --thinking medium sends
// reasoning_effort in the request body.
func TestCLIThinkingFlagSendsReasoningEffort(t *testing.T) {
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		bodies = append(bodies, string(body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"done"}}]}`)
	}))
	defer srv.Close()

	home := t.TempDir()
	_, stderr, code := runCLI(t, map[string]string{
		"OPENAI_API_KEY":  "sk-test-123",
		"OPENAI_BASE_URL": srv.URL,
		"KUI_HOME":        home,
	}, "--thinking", "medium", "hello")

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if len(bodies) == 0 {
		t.Fatal("provider received no requests")
	}
	if !strings.Contains(bodies[0], `"reasoning_effort":"medium"`) {
		t.Errorf("request body %q does not contain reasoning_effort:medium", bodies[0])
	}
}

// TestCLIThinkingOffOmitsReasoningEffort verifies that --thinking off (or no
// thinking flag) omits reasoning_effort from the request body.
func TestCLIThinkingOffOmitsReasoningEffort(t *testing.T) {
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		bodies = append(bodies, string(body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"done"}}]}`)
	}))
	defer srv.Close()

	home := t.TempDir()
	_, stderr, code := runCLI(t, map[string]string{
		"OPENAI_API_KEY":  "sk-test-123",
		"OPENAI_BASE_URL": srv.URL,
		"KUI_HOME":        home,
	}, "--thinking", "off", "hello")

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if len(bodies) == 0 {
		t.Fatal("provider received no requests")
	}
	if strings.Contains(bodies[0], "reasoning_effort") {
		t.Errorf("request body %q must not contain reasoning_effort for thinking=off", bodies[0])
	}
}

// TestCLIThinkingInvalidLevel verifies that --thinking banana prints an error
// and exits 2 (usage error).
func TestCLIThinkingInvalidLevel(t *testing.T) {
	_, stderr, code := runCLI(t, map[string]string{
		"KUI_HOME": t.TempDir(),
	}, "--thinking", "banana", "hello")

	if code != 2 {
		t.Errorf("exit code = %d, want 2 (usage error)", code)
	}
	if !strings.Contains(stderr, "banana") {
		t.Errorf("stderr = %q, want it to mention the invalid level", stderr)
	}
}

// TestCLIProfileThinkingSubcommand verifies that `kui profile thinking <name>
// <level>` persists the thinking level in profile.yaml.
func TestCLIProfileThinkingSubcommand(t *testing.T) {
	home := t.TempDir()
	writeProfileDir(t, home, "coder", "name: coder\nsystem_prompt: SYSTEM.md\n")

	_, stderr, code := runCLI(t, map[string]string{"KUI_HOME": home}, "profile", "thinking", "coder", "high")

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr = %q", code, stderr)
	}
	data, err := os.ReadFile(filepath.Join(home, "profiles", "coder", "profile.yaml"))
	if err != nil {
		t.Fatalf("read profile.yaml: %v", err)
	}
	if !strings.Contains(string(data), "thinking: high") {
		t.Errorf("profile.yaml = %q, want it to contain 'thinking: high'", string(data))
	}
}

// TestCLIProfileThinkingInvalidLevel verifies that `kui profile thinking <name>
// banana` prints an error and exits non-zero.
func TestCLIProfileThinkingInvalidLevel(t *testing.T) {
	home := t.TempDir()
	writeProfileDir(t, home, "coder", "name: coder\n")

	_, stderr, code := runCLI(t, map[string]string{"KUI_HOME": home}, "profile", "thinking", "coder", "banana")

	if code == 0 {
		t.Error("exit code = 0, want non-zero for an invalid thinking level")
	}
	if !strings.Contains(stderr, "banana") {
		t.Errorf("stderr = %q, want it to mention the invalid level", stderr)
	}
}
