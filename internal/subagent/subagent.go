package subagent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Policy represents the background subagents policy.
type Policy string

const (
	PolicyOn  Policy = "on"
	PolicyOff Policy = "off"
)

// PolicyConfig is the JSON schema for background-subagents.json.
type PolicyConfig struct {
	Schema string `json:"schema"`
	Policy Policy `json:"policy"`
}

// Capability reports whether subagent_run is available.
type Capability string

const (
	CapabilityReady  Capability = "ready"
	CapabilityAbsent Capability = "absent"
)

// RunRequest is the input to the subagent_run tool.
type RunRequest struct {
	// Task is the prompt/instruction for the sub-agent.
	Task string `json:"task"`
	// Mode is "task" (foreground, waits for result) or "background" (future).
	Mode string `json:"mode,omitempty"`
	// Agent is the agent definition name (optional, for future use).
	Agent string `json:"agent,omitempty"`
	// Context is additional context to pass to the sub-agent (optional).
	Context string `json:"context,omitempty"`
}

// RunResult is the output from a sub-agent execution.
type RunResult struct {
	// Output is the sub-agent's response text.
	Output string `json:"output"`
	// ExitCode is the process exit code.
	ExitCode int `json:"exit_code"`
	// Error is set when the sub-agent failed.
	Error string `json:"error,omitempty"`
}

// Run executes a sub-agent by running kui as a sub-process with the given task.
// It runs in foreground mode (waits for completion).
func Run(ctx context.Context, req RunRequest, kuiBinary string, cwd string) (*RunResult, error) {
	if req.Task == "" {
		return nil, fmt.Errorf("subagent_run: task is required")
	}

	// Build the command: kui --provider <provider> --model <model> <task>
	args := []string{}
	if req.Context != "" {
		args = append(args, "--")
		args = append(args, req.Context+" "+req.Task)
	} else {
		args = append(args, "--")
		args = append(args, req.Task)
	}

	cmd := exec.CommandContext(ctx, kuiBinary, args...)
	cmd.Dir = cwd

	// Capture stdout and stderr.
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("subagent_run: failed to start: %w", err)
		}
	}

	result := &RunResult{
		Output:   stdout.String(),
		ExitCode: exitCode,
	}
	if stderr.Len() > 0 && exitCode != 0 {
		result.Error = stderr.String()
	}

	return result, nil
}

// ParsePolicy parses a background-subagents.json file content.
func ParsePolicy(raw string) (Policy, error) {
	var config PolicyConfig
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return PolicyOff, fmt.Errorf("invalid JSON: %w", err)
	}
	if config.Schema != "kui.background-subagents/v1" {
		return PolicyOff, fmt.Errorf("unknown schema: %s", config.Schema)
	}
	if config.Policy != PolicyOn && config.Policy != PolicyOff {
		return PolicyOff, fmt.Errorf("invalid policy: %s", config.Policy)
	}
	// Reject extra keys (must be exactly schema + policy).
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &rawMap); err != nil {
		return PolicyOff, fmt.Errorf("invalid JSON: %w", err)
	}
	if len(rawMap) != 2 {
		return PolicyOff, fmt.Errorf("unexpected keys: got %d fields, want 2", len(rawMap))
	}
	return config.Policy, nil
}

// ResolvePolicy resolves the policy from project config, global config, or default.
// Resolution order: project file → global file → default "off".
func ResolvePolicy(projectDir string, globalDir string) (Policy, string) {
	// Try project file first.
	projectPath := projectDir + "/background-subagents.json"
	if data, err := readFile(projectPath); err == nil {
		if policy, err := ParsePolicy(data); err == nil {
			return policy, "project_file"
		}
		// Malformed file → fail closed to "off"
		return PolicyOff, "project_file_malformed"
	}

	// Try global file.
	globalPath := globalDir + "/background-subagents.json"
	if data, err := readFile(globalPath); err == nil {
		if policy, err := ParsePolicy(data); err == nil {
			return policy, "global_file"
		}
		return PolicyOff, "global_file_malformed"
	}

	// Default.
	return PolicyOff, "default"
}

// ResolveCapability checks if subagent_run is available.
// It checks the tool registry for subagent_run.
func ResolveCapability(activeTools []string) Capability {
	for _, name := range activeTools {
		if name == "subagent_run" || strings.HasSuffix(name, ".subagent_run") {
			return CapabilityReady
		}
	}
	return CapabilityAbsent
}

// readFile reads a file and returns its content.
func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
