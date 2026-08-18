package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Tool is the orchestration tool exposed to the parent agent.
type Tool struct {
	name       string
	spawner    *Spawner
	aggregator *ResultAggregator
	dedup      *LaunchDedup
}

// NewTool creates a new orchestrator tool.
func NewTool(spawner *Spawner) *Tool {
	return &Tool{
		name:       "orchestrate",
		spawner:    spawner,
		aggregator: &ResultAggregator{},
		dedup:      NewLaunchDedup(),
	}
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return t.name
}

// Description returns the tool description.
func (t *Tool) Description() string {
	return "Spawn specialized agents for focused tasks"
}

// Schema returns the JSON schema for the tool.
func (t *Tool) Schema() string {
	return `{
  "name": "orchestrate",
  "description": "Spawn specialized agents for focused tasks",
  "parameters": {
    "type": "object",
    "properties": {
      "operation": {
        "type": "string",
        "enum": ["spawn", "fan-out", "chain"]
      },
      "agents": {
        "type": "array",
        "items": {
          "type": "object",
          "properties": {
            "name": {"type": "string"},
            "task": {"type": "string"}
          },
          "required": ["name", "task"]
        }
      },
      "aggregate": {
        "type": "string",
        "enum": ["merge", "summary", "select"]
      }
    },
    "required": ["operation", "agents"]
  }
}`
}

// AgentDef represents an agent in the orchestration request.
type AgentDef_request struct {
	Name string `json:"name"`
	Task string `json:"task"`
}

// OrchestratorRequest represents the tool input.
type OrchestratorRequest struct {
	Operation string             `json:"operation"`
	Agents    []AgentDef_request `json:"agents"`
	Aggregate string             `json:"aggregate"`
}

// Execute runs the orchestration operation.
func (t *Tool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var req OrchestratorRequest
	if err := json.Unmarshal(args, &req); err != nil {
		return "", fmt.Errorf("parsing orchestration args: %w", err)
	}

	// Validate operation
	switch req.Operation {
	case "spawn", "fan-out", "chain":
		// valid
	default:
		return "", fmt.Errorf("unknown operation: %q", req.Operation)
	}

	// Validate agents list
	if len(req.Agents) == 0 {
		return "", fmt.Errorf("agents list is empty")
	}

	// Check dedup for each agent
	for _, agent := range req.Agents {
		if cached, ok := t.dedup.GetCached(agent.Name, agent.Task); ok {
			return cached, nil
		}
	}

	// Execute operation
	var result string
	var execErr error
	switch req.Operation {
	case "spawn":
		result, execErr = t.executeSpawn(ctx, req.Agents[0])
	case "fan-out":
		result, execErr = t.executeFanOut(ctx, req.Agents, req.Aggregate)
	case "chain":
		result, execErr = t.executeChain(ctx, req.Agents)
	default:
		return "", fmt.Errorf("unreachable")
	}

	if execErr != nil {
		return "", execErr
	}

	// Cache result for dedup
	for _, agent := range req.Agents {
		t.dedup.MarkSeen(agent.Name, agent.Task, result)
	}

	return result, nil
}

// executeSpawn runs a single agent.
func (t *Tool) executeSpawn(ctx context.Context, agent AgentDef_request) (string, error) {
	result, err := t.spawner.Spawn(ctx, SpawnRequest{
		AgentName: agent.Name,
		Task:      agent.Task,
	})
	if err != nil {
		return "", fmt.Errorf("spawning agent %q: %w", agent.Name, err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("agent %q failed: %w", agent.Name, result.Error)
	}
	return result.Output, nil
}

// executeFanOut runs multiple agents in parallel and aggregates results.
func (t *Tool) executeFanOut(ctx context.Context, agents []AgentDef_request, aggregateMode string) (string, error) {
	results := make([]*SpawnResult, len(agents))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, agent := range agents {
		wg.Add(1)
		go func(i int, agent AgentDef_request) {
			defer wg.Done()

			result, err := t.spawner.Spawn(ctx, SpawnRequest{
				AgentName: agent.Name,
				Task:      agent.Task,
			})

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				results[i] = &SpawnResult{Error: err}
			} else {
				results[i] = result
			}
		}(i, agent)
	}

	wg.Wait()

	// Filter out errors
	var validResults []*SpawnResult
	for _, r := range results {
		if r != nil && r.Error == nil {
			validResults = append(validResults, r)
		}
	}

	if len(validResults) == 0 {
		return "", fmt.Errorf("all agents failed")
	}

	// Default aggregate mode
	if aggregateMode == "" {
		aggregateMode = "merge"
	}

	return t.aggregator.Aggregate(validResults, aggregateMode), nil
}

// executeChain runs agents sequentially, feeding output of one as context to the next.
func (t *Tool) executeChain(ctx context.Context, agents []AgentDef_request) (string, error) {
	var previousOutput string

	for _, agent := range agents {
		// Build task with context from previous agent
		task := agent.Task
		if previousOutput != "" {
			task = fmt.Sprintf("Previous agent output:\n%s\n\nYour task: %s", previousOutput, agent.Task)
		}

		result, err := t.spawner.Spawn(ctx, SpawnRequest{
			AgentName: agent.Name,
			Task:      task,
		})
		if err != nil {
			return "", fmt.Errorf("spawning agent %q in chain: %w", agent.Name, err)
		}
		if result.Error != nil {
			return "", fmt.Errorf("agent %q failed in chain: %w", agent.Name, result.Error)
		}

		previousOutput = result.Output
	}

	return previousOutput, nil
}
