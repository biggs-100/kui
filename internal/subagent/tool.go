package subagent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Tool implements core.Tool for the subagent_run command.
type Tool struct {
	kuiBinary string
	cwd       string
	policy    Policy
}

// NewTool creates a subagent_run tool.
// kuiBinary is the path to the kui executable.
// cwd is the working directory for sub-agents.
// policy controls whether sub-agents are allowed.
func NewTool(kuiBinary, cwd string, policy Policy) *Tool {
	return &Tool{
		kuiBinary: kuiBinary,
		cwd:       cwd,
		policy:    policy,
	}
}

// Name returns the tool name.
func (t *Tool) Name() string { return "subagent_run" }

// Description returns the tool description.
func (t *Tool) Description() string {
	return "Run a sub-agent to execute a task. The sub-agent runs kui as a sub-process with the given prompt."
}

// Schema returns the JSON parameter schema.
func (t *Tool) Schema() string {
	return `{
		"type": "object",
		"properties": {
			"task": {
				"type": "string",
				"description": "The prompt/instruction for the sub-agent to execute"
			},
			"context": {
				"type": "string",
				"description": "Additional context to prepend to the task (optional)"
			}
		},
		"required": ["task"]
	}`
}

// Execute runs the sub-agent.
func (t *Tool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if t.policy != PolicyOn {
		return "", fmt.Errorf("subagent_run: policy is off (set background-subagents.json to enable)")
	}

	var input RunRequest
	if err := json.Unmarshal(args, &input); err != nil {
		return "", fmt.Errorf("subagent_run: invalid arguments: %w", err)
	}

	if input.Task == "" {
		return "", fmt.Errorf("subagent_run: task is required")
	}

	// Build command args.
	cmdArgs := []string{"--"}
	if input.Context != "" {
		cmdArgs = append(cmdArgs, input.Context+"\n\n"+input.Task)
	} else {
		cmdArgs = append(cmdArgs, input.Task)
	}

	cmd := exec.CommandContext(ctx, t.kuiBinary, cmdArgs...)
	cmd.Dir = t.cwd

	// Capture output.
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return "", fmt.Errorf("subagent_run: failed to start: %w", err)
		}
	}

	// Format result.
	var result strings.Builder
	if stdout.Len() > 0 {
		result.WriteString(stdout.String())
	}
	if stderr.Len() > 0 && exitCode != 0 {
		result.WriteString("\n--- STDERR ---\n")
		result.WriteString(stderr.String())
	}
	if exitCode != 0 {
		result.WriteString(fmt.Sprintf("\n--- EXIT CODE: %d ---\n", exitCode))
	}

	return result.String(), nil
}

// FindKuiBinary locates the kui binary. It checks:
// 1. KUI_BINARY env var
// 2. Same directory as the current executable
// 3. PATH
func FindKuiBinary() string {
	// Check env var.
	if bin := os.Getenv("KUI_BINARY"); bin != "" {
		return bin
	}

	// Check same directory as current exe.
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidate := filepath.Join(dir, "kui")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		candidate = filepath.Join(dir, "kui.exe")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// Check PATH.
	if path, err := exec.LookPath("kui"); err == nil {
		return path
	}

	return "kui" // fallback — will fail if not in PATH
}
