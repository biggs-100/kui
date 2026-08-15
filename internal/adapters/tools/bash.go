package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// defaultBashTimeout bounds every command when no explicit timeout is given.
const defaultBashTimeout = 30 * time.Second

// bashWaitDelay bounds the pipe drain after the main process exits. On
// Windows, killing bash leaves its grandchildren holding the stdout/stderr
// pipes open, which would otherwise block Wait until they exit.
const bashWaitDelay = 250 * time.Millisecond

// TimeoutError reports a command that exceeded its deadline and was
// terminated (D12, REQ-TOOLS-3).
type TimeoutError struct {
	Command string
	Timeout time.Duration
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("command %q exceeded the %s timeout and was terminated", e.Command, e.Timeout)
}

// Bash executes shell commands synchronously with a mandatory timeout and
// returns stdout, stderr, and the exit code (REQ-TOOLS-3). Commands never
// receive interactive input (D12).
type Bash struct {
	shell   string
	timeout time.Duration
}

// NewBash returns a bash tool that runs commands through the "bash"
// executable found in PATH. A zero timeout selects the default of 30s.
func NewBash(timeout time.Duration) *Bash {
	if timeout <= 0 {
		timeout = defaultBashTimeout
	}
	return &Bash{shell: "bash", timeout: timeout}
}

// Name returns the stable tool name (REQ-TOOLS-4).
func (b *Bash) Name() string { return "bash" }

// Description returns the tool description (REQ-TOOLS-4).
func (b *Bash) Description() string {
	return "Run a shell command with a mandatory timeout and return stdout, stderr, and the exit code"
}

// Schema returns the raw JSON parameter schema (D3, REQ-TOOLS-4).
func (b *Bash) Schema() string {
	return `{"type":"object","properties":{"command":{"type":"string"}},"required":["command"]}`
}

// bashResult is the structured outcome every command returns so the loop can
// feed it back to the provider.
type bashResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

// Execute runs the command synchronously. A non-zero exit code is reported in
// the result, not as an error; timeouts and execution failures are errors.
func (b *Bash) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if in.Command == "" {
		return "", errors.New("command must not be empty")
	}

	stdout, stderr, exitCode, err := runWithTimeout(ctx, b.timeout, b.shell, "-c", in.Command)
	if err != nil {
		return "", err
	}
	result, err := json.Marshal(bashResult{Stdout: stdout, Stderr: stderr, ExitCode: exitCode})
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// runWithTimeout runs name with args under a hard deadline. It is the shared
// subprocess boundary (D12): the child gets nil stdin (never interactive),
// and on expiry the process is killed and a TimeoutError is returned.
func runWithTimeout(ctx context.Context, timeout time.Duration, name string, args ...string) (stdout, stderr string, exitCode int, err error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	// Stdin stays nil: the child reads from the null device (REQ-TOOLS-3).
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	cmd.WaitDelay = bashWaitDelay

	runErr := cmd.Run()
	// The context is the authoritative timeout signal: on Windows the kill
	// surfaces as a plain ExitError (exit code 1 from TerminateProcess), so
	// error inspection alone cannot classify a deadline (D12).
	if ctx.Err() != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", "", 0, &TimeoutError{Command: strings.Join(args, " "), Timeout: timeout}
		}
		return "", "", 0, ctx.Err()
	}
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			return out.String(), errOut.String(), exitErr.ExitCode(), nil
		}
		return "", "", 0, runErr
	}
	return out.String(), errOut.String(), 0, nil
}
