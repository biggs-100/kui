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
	_, stderr, code := runCLI(t, map[string]string{"OPENAI_API_KEY": ""}, "hello")

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
