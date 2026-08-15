package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestBashHelperProcess is the re-executed helper subprocess: when
// GO_WANT_BASH_HELPER is set, it runs the mode requested in
// GO_BASH_HELPER_MODE and exits, exercising the real subprocess boundary of
// the bash tool (D12) without requiring a bash installation.
func TestBashHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_BASH_HELPER") != "1" {
		return
	}
	switch os.Getenv("GO_BASH_HELPER_MODE") {
	case "sleep":
		time.Sleep(10 * time.Second)
		os.Exit(0)
	case "exit1":
		fmt.Fprintln(os.Stderr, "boom")
		os.Exit(1)
	case "stderr":
		fmt.Fprintln(os.Stderr, "to stderr")
		os.Exit(0)
	default:
		os.Exit(0)
	}
}

func setHelperEnv(t *testing.T, mode string) {
	t.Helper()
	t.Setenv("GO_WANT_BASH_HELPER", "1")
	t.Setenv("GO_BASH_HELPER_MODE", mode)
}

func helperExe(t *testing.T) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable failed: %v", err)
	}
	return exe
}

// TestBashTimeoutKill covers REQ-TOOLS-3 "Command timeout" at the subprocess
// boundary: a command running longer than the timeout is terminated and a
// TimeoutError is returned. It uses a Go helper subprocess so it does not
// depend on a bash installation (design open question: bash on Windows dev
// machines).
func TestBashTimeoutKill(t *testing.T) {
	setHelperEnv(t, "sleep")
	exe := helperExe(t)

	start := time.Now()
	stdout, stderr, exitCode, err := runWithTimeout(context.Background(), time.Second, exe, "-test.run=TestBashHelperProcess")
	elapsed := time.Since(start)

	var timeoutErr *TimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("error = %v, want *TimeoutError (the command must be terminated)", err)
	}
	if timeoutErr.Timeout != time.Second {
		t.Errorf("TimeoutError.Timeout = %v, want 1s", timeoutErr.Timeout)
	}
	if elapsed > 3*time.Second {
		t.Errorf("command was not terminated promptly: took %v with a 1s timeout", elapsed)
	}
	if exitCode != 0 || stdout != "" || stderr != "" {
		t.Errorf("on timeout got exitCode=%d stdout=%q stderr=%q, want 0 and empty output", exitCode, stdout, stderr)
	}
}

// TestBashExitCodeMapping covers REQ-TOOLS-3 "Non-zero exit" at the
// subprocess boundary: a failing command maps to its exit code and captured
// stderr without being a tool error.
func TestBashExitCodeMapping(t *testing.T) {
	setHelperEnv(t, "exit1")
	exe := helperExe(t)

	stdout, stderr, exitCode, err := runWithTimeout(context.Background(), 10*time.Second, exe, "-test.run=TestBashHelperProcess")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr, "boom") {
		t.Errorf("stderr = %q, want captured %q", stderr, "boom")
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
}

// TestBashCapturesStderrSeparately triangulates output capture: stdout and
// stderr are returned as separate streams.
func TestBashCapturesStderrSeparately(t *testing.T) {
	setHelperEnv(t, "stderr")
	exe := helperExe(t)

	stdout, stderr, exitCode, err := runWithTimeout(context.Background(), 10*time.Second, exe, "-test.run=TestBashHelperProcess")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if stderr != "to stderr\n" {
		t.Errorf("stderr = %q, want %q", stderr, "to stderr\n")
	}
}

// requireBash returns a usable bash executable or skips the test. Candidates
// are probed with a short timeout because on Windows the WSL launcher
// (System32\bash.exe) is a stub that hangs without a default distro; a real
// Git Bash or MSYS2 installation is required (design open question).
func requireBash(t *testing.T) string {
	t.Helper()
	seen := make(map[string]bool)
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		if runtime.GOOS == "windows" {
			lower := strings.ToLower(dir)
			if strings.HasSuffix(lower, "system32") || strings.Contains(lower, "windowsapps") {
				continue // WSL launcher stubs, not usable shells
			}
		}
		names := []string{"bash"}
		if runtime.GOOS == "windows" {
			names = []string{"bash.exe", "bash"}
		}
		for _, name := range names {
			candidate := filepath.Join(dir, name)
			if seen[candidate] {
				continue
			}
			seen[candidate] = true
			if _, err := os.Stat(candidate); err != nil {
				continue
			}
			probeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := exec.CommandContext(probeCtx, candidate, "-c", "exit 0").Run()
			cancel()
			if err == nil {
				return candidate
			}
		}
	}
	t.Skip("no usable bash executable in PATH (Git Bash or MSYS2 required on Windows; see design.md open questions)")
	return ""
}

func bashToolForTest(t *testing.T, timeout time.Duration) *Bash {
	t.Helper()
	return &Bash{shell: requireBash(t), timeout: timeout}
}

// TestBashEchoExitZero covers REQ-TOOLS-3 "Successful command" through the
// real tool: exit code 0 and stdout "hello".
func TestBashEchoExitZero(t *testing.T) {
	tool := bashToolForTest(t, 5*time.Second)

	out, err := tool.Execute(context.Background(), argsFor(t, map[string]string{"command": "echo hello"}))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	var result bashResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("result is not valid JSON: %v\n%s", err, out)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", result.ExitCode)
	}
	if !strings.Contains(result.Stdout, "hello") {
		t.Errorf("stdout = %q, want it to contain %q", result.Stdout, "hello")
	}
}

// TestBashNonZeroExit covers REQ-TOOLS-3 "Non-zero exit" through the real
// tool: exit code 1 is reported in the result, not as a tool error.
func TestBashNonZeroExit(t *testing.T) {
	tool := bashToolForTest(t, 5*time.Second)

	out, err := tool.Execute(context.Background(), argsFor(t, map[string]string{"command": "exit 1"}))
	if err != nil {
		t.Fatalf("non-zero exit must not be a tool error: %v", err)
	}
	var result bashResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("result is not valid JSON: %v\n%s", err, out)
	}
	if result.ExitCode != 1 {
		t.Errorf("exit code = %d, want 1", result.ExitCode)
	}
}

// TestBashNoInteractiveInput verifies commands never accept interactive
// input (REQ-TOOLS-3): cat with a nil stdin hits EOF immediately and exits 0.
func TestBashNoInteractiveInput(t *testing.T) {
	tool := bashToolForTest(t, 5*time.Second)

	out, err := tool.Execute(context.Background(), argsFor(t, map[string]string{"command": "cat"}))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	var result bashResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("result is not valid JSON: %v\n%s", err, out)
	}
	if result.ExitCode != 0 {
		t.Errorf("cat with nil stdin: exit code = %d, want 0 (EOF)", result.ExitCode)
	}
	if result.Stdout != "" {
		t.Errorf("stdout = %q, want empty", result.Stdout)
	}
}

// TestBashTimeoutThroughTool is the full REQ-TOOLS-3 timeout scenario through
// the real tool: a 10s sleep under a 1s timeout is terminated.
func TestBashTimeoutThroughTool(t *testing.T) {
	tool := bashToolForTest(t, time.Second)

	start := time.Now()
	_, err := tool.Execute(context.Background(), argsFor(t, map[string]string{"command": "sleep 10"}))
	elapsed := time.Since(start)

	var timeoutErr *TimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("error = %v, want *TimeoutError", err)
	}
	if elapsed > 3*time.Second {
		t.Errorf("tool took %v to terminate the command, want termination near the 1s timeout", elapsed)
	}
}

// TestBashInvalidArguments rejects malformed arguments and empty commands.
func TestBashInvalidArguments(t *testing.T) {
	tool := &Bash{shell: "bash", timeout: time.Second}

	if _, err := tool.Execute(context.Background(), json.RawMessage(`not json`)); err == nil {
		t.Error("Execute accepted invalid JSON arguments")
	}
	if _, err := tool.Execute(context.Background(), argsFor(t, map[string]string{"command": ""})); err == nil {
		t.Error("Execute accepted an empty command")
	}
}
