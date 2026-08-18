package orchestration

import (
	"context"
	"fmt"
	"time"
)

// SpawnRequest defines what to spawn.
type SpawnRequest struct {
	AgentName string // agent definition name
	Task      string // task description
	Context   string // additional context (optional)
	Model     string // model override (optional)
	Thinking  string // thinking override (optional)
}

// TokenUsage tracks token accounting for a spawn.
type TokenUsage struct {
	Prompt     int
	Completion int
	Total      int
}

// SpawnResult contains the agent's output.
type SpawnResult struct {
	Output   string        // final text output
	Messages []Message     // conversation history
	Tokens   TokenUsage    // token accounting
	Duration time.Duration // execution time
	Error    error         // if failed
}

// Message is a conversation message in SpawnResult.
// Defined locally to avoid import cycles with core.
type Message struct {
	Role    string
	Content string
}

// ExecuteFunc is the function signature for task execution.
// Injected for testability; the real implementation will wrap core.Agent.
type ExecuteFunc func(ctx context.Context, def *AgentDef, req SpawnRequest) (string, error)

// Spawner creates isolated agent instances in-process.
type Spawner struct {
	registry *AgentRegistry
	execute  ExecuteFunc
}

// NewSpawner creates a spawner with the given agent registry.
func NewSpawner(registry *AgentRegistry) *Spawner {
	return &Spawner{
		registry: registry,
		execute:  defaultExecute,
	}
}

// NewSpawnerWithExecute creates a spawner with a custom execution function (for testing).
func NewSpawnerWithExecute(registry *AgentRegistry, execute ExecuteFunc) *Spawner {
	return &Spawner{
		registry: registry,
		execute:  execute,
	}
}

// Spawn creates an isolated agent and runs it.
func (s *Spawner) Spawn(ctx context.Context, req SpawnRequest) (*SpawnResult, error) {
	// Check context before doing work
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("spawn cancelled: %w", err)
	}

	// Load agent definition from registry
	def, err := s.registry.Load(req.AgentName)
	if err != nil {
		return nil, fmt.Errorf("loading agent: %w", err)
	}

	start := time.Now()

	// Execute the task
	output, execErr := s.execute(ctx, def, req)

	duration := time.Since(start)

	return &SpawnResult{
		Output:   output,
		Duration: duration,
		Error:    execErr,
	}, nil
}

// defaultExecute is the placeholder execution function.
// For now, it echoes the task back as output.
// The real implementation will wrap core.Agent in a later PR.
func defaultExecute(_ context.Context, _ *AgentDef, req SpawnRequest) (string, error) {
	return req.Task, nil
}
