package orchestration

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// ─── Task 3.1: RED — Test tool name ───

func TestToolName(t *testing.T) {
	registry := NewAgentRegistry(nil)
	spawner := NewSpawner(registry)
	tool := NewTool(spawner)

	if tool.Name() != "orchestrate" {
		t.Errorf("expected name 'orchestrate', got %q", tool.Name())
	}
}

// ─── Task 3.1: RED — Test tool description ───

func TestToolDescription(t *testing.T) {
	registry := NewAgentRegistry(nil)
	spawner := NewSpawner(registry)
	tool := NewTool(spawner)

	if tool.Description() == "" {
		t.Error("expected non-empty description")
	}
}

// ─── Task 3.2: RED — Test tool schema is valid JSON ───

func TestToolSchema(t *testing.T) {
	registry := NewAgentRegistry(nil)
	spawner := NewSpawner(registry)
	tool := NewTool(spawner)

	schema := tool.Schema()
	if schema == "" {
		t.Fatal("expected non-empty schema")
	}

	// Must be valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(schema), &parsed); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}

	// Must have required fields
	if parsed["name"] != "orchestrate" {
		t.Errorf("schema name should be 'orchestrate', got %v", parsed["name"])
	}
	if parsed["description"] == nil {
		t.Error("schema should have description")
	}
	params, ok := parsed["parameters"].(map[string]interface{})
	if !ok {
		t.Fatal("schema should have parameters object")
	}
	if params["type"] != "object" {
		t.Error("parameters type should be 'object'")
	}
	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("parameters should have properties")
	}
	if props["operation"] == nil {
		t.Error("schema should have operation property")
	}
	if props["agents"] == nil {
		t.Error("schema should have agents property")
	}
}

// ─── Task 3.3: RED — Test spawn operation ───

func TestToolSpawn(t *testing.T) {
	registry := NewAgentRegistry(nil)
	spawner := NewSpawner(registry)
	tool := NewTool(spawner)

	args := `{
		"operation": "spawn",
		"agents": [{"name": "explore", "task": "read all go files"}]
	}`

	result, err := tool.Execute(context.Background(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}

// ─── Task 3.5: RED — Test fan-out operation ───

func TestToolFanOut(t *testing.T) {
	registry := NewAgentRegistry(nil)
	spawner := NewSpawner(registry)
	tool := NewTool(spawner)

	args := `{
		"operation": "fan-out",
		"agents": [
			{"name": "explore", "task": "read test files"},
			{"name": "explore", "task": "read config files"}
		]
	}`

	result, err := tool.Execute(context.Background(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
	// Fan-out should contain outputs from both agents
	if !strings.Contains(result, "read test files") {
		t.Error("fan-out result should contain first agent output")
	}
	if !strings.Contains(result, "read config files") {
		t.Error("fan-out result should contain second agent output")
	}
}

// ─── Task 3.7: RED — Test chain operation ───

func TestToolChain(t *testing.T) {
	registry := NewAgentRegistry(nil)
	spawner := NewSpawner(registry)
	tool := NewTool(spawner)

	args := `{
		"operation": "chain",
		"agents": [
			{"name": "explore", "task": "read files"},
			{"name": "worker", "task": "process files"}
		]
	}`

	result, err := tool.Execute(context.Background(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}

// ─── Task 3.9: RED — Test dedup integration ───

func TestToolDedup(t *testing.T) {
	registry := NewAgentRegistry(nil)
	spawner := NewSpawner(registry)
	tool := NewTool(spawner)

	args := `{
		"operation": "spawn",
		"agents": [{"name": "explore", "task": "same task"}]
	}`

	// First call
	result1, err := tool.Execute(context.Background(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	// Second call — same task should be deduped
	result2, err := tool.Execute(context.Background(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}

	// Results should be identical (deduped)
	if result1 != result2 {
		t.Error("deduped results should be identical")
	}
}

// ─── Task 3.9: RED — Test invalid operation ───

func TestToolInvalidOperation(t *testing.T) {
	registry := NewAgentRegistry(nil)
	spawner := NewSpawner(registry)
	tool := NewTool(spawner)

	args := `{
		"operation": "invalid",
		"agents": [{"name": "explore", "task": "do something"}]
	}`

	_, err := tool.Execute(context.Background(), json.RawMessage(args))
	if err == nil {
		t.Error("expected error for invalid operation")
	}
}

// ─── Task 3.9: RED — Test empty agents list ───

func TestToolEmptyAgents(t *testing.T) {
	registry := NewAgentRegistry(nil)
	spawner := NewSpawner(registry)
	tool := NewTool(spawner)

	args := `{
		"operation": "spawn",
		"agents": []
	}`

	_, err := tool.Execute(context.Background(), json.RawMessage(args))
	if err == nil {
		t.Error("expected error for empty agents list")
	}
}

// ─── Task 3.9: RED — Test unknown agent in spawn ───

func TestToolUnknownAgent(t *testing.T) {
	registry := NewAgentRegistry(nil)
	spawner := NewSpawner(registry)
	tool := NewTool(spawner)

	args := `{
		"operation": "spawn",
		"agents": [{"name": "nonexistent", "task": "do something"}]
	}`

	_, err := tool.Execute(context.Background(), json.RawMessage(args))
	if err == nil {
		t.Error("expected error for unknown agent")
	}
}

// ─── Task 3.9: RED — Test invalid JSON ───

func TestToolInvalidJSON(t *testing.T) {
	registry := NewAgentRegistry(nil)
	spawner := NewSpawner(registry)
	tool := NewTool(spawner)

	_, err := tool.Execute(context.Background(), json.RawMessage("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// ─── Task 3.9: RED — Test chain feeds output to next agent ───

func TestToolChainFeedsOutput(t *testing.T) {
	registry := NewAgentRegistry(nil)

	// Custom executor that captures the task input
	var capturedTasks []string
	execute := func(_ context.Context, _ *AgentDef, req SpawnRequest) (string, error) {
		capturedTasks = append(capturedTasks, req.Task)
		return "output: " + req.Task, nil
	}

	spawner := NewSpawnerWithExecute(registry, execute)
	tool := NewTool(spawner)

	args := `{
		"operation": "chain",
		"agents": [
			{"name": "explore", "task": "step1"},
			{"name": "worker", "task": "step2"}
		]
	}`

	result, err := tool.Execute(context.Background(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Chain should feed output of step1 as context to step2
	if len(capturedTasks) < 2 {
		t.Fatalf("expected 2 tasks executed, got %d", len(capturedTasks))
	}

	// Second task should receive output from first
	if capturedTasks[1] == "step2" {
		t.Error("chain should feed first agent output to second agent")
	}
	if !strings.Contains(result, "output:") {
		t.Error("chain result should contain output")
	}
}

// ─── Task 3.9: RED — Test fan-out with custom executor ───

func TestToolFanOutParallel(t *testing.T) {
	registry := NewAgentRegistry(nil)

	var executionOrder []string
	execute := func(_ context.Context, _ *AgentDef, req SpawnRequest) (string, error) {
		executionOrder = append(executionOrder, req.Task)
		return "done: " + req.Task, nil
	}

	spawner := NewSpawnerWithExecute(registry, execute)
	tool := NewTool(spawner)

	args := `{
		"operation": "fan-out",
		"agents": [
			{"name": "explore", "task": "task-a"},
			{"name": "explore", "task": "task-b"},
			{"name": "explore", "task": "task-c"}
		]
	}`

	result, err := tool.Execute(context.Background(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All 3 tasks should execute
	if len(executionOrder) != 3 {
		t.Errorf("expected 3 tasks executed, got %d", len(executionOrder))
	}

	// Result should contain all outputs
	if !strings.Contains(result, "done: task-a") {
		t.Error("fan-out result should contain task-a output")
	}
	if !strings.Contains(result, "done: task-b") {
		t.Error("fan-out result should contain task-b output")
	}
	if !strings.Contains(result, "done: task-c") {
		t.Error("fan-out result should contain task-c output")
	}
}
