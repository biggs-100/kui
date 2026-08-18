package orchestration

import (
	"context"
	"strings"
	"testing"
	"time"
)

// ─── Task 2.1: RED — Test spawner creates result ───

func TestSpawnerCreatesResult(t *testing.T) {
	registry := NewAgentRegistry(nil)
	spawner := NewSpawner(registry)

	req := SpawnRequest{
		AgentName: "explore",
		Task:      "list all go files",
	}

	result, err := spawner.Spawn(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// ─── Task 2.2: RED — Test agent not found ───

func TestSpawnerAgentNotFound(t *testing.T) {
	registry := NewAgentRegistry(nil)
	spawner := NewSpawner(registry)

	req := SpawnRequest{
		AgentName: "nonexistent",
		Task:      "do something",
	}

	_, err := spawner.Spawn(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention agent name, got: %v", err)
	}
}

// ─── Task 2.3: RED — Test agent lookup from registry ───

func TestSpawnerAgentLookup(t *testing.T) {
	registry := NewAgentRegistry(nil)
	spawner := NewSpawner(registry)

	// Built-in "explore" agent should be found
	req := SpawnRequest{
		AgentName: "explore",
		Task:      "read files",
	}

	result, err := spawner.Spawn(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// ─── Task 2.4: RED — Test duration is measured ───

func TestSpawnResultDuration(t *testing.T) {
	registry := NewAgentRegistry(nil)

	// Use a slow executor to guarantee measurable duration
	slowExecute := func(_ context.Context, _ *AgentDef, req SpawnRequest) (string, error) {
		time.Sleep(10 * time.Millisecond)
		return req.Task, nil
	}
	spawner := NewSpawnerWithExecute(registry, slowExecute)

	req := SpawnRequest{
		AgentName: "explore",
		Task:      "list files",
	}

	result, err := spawner.Spawn(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Duration < 10*time.Millisecond {
		t.Errorf("expected duration >= 10ms, got %v", result.Duration)
	}
}

// ─── Task 2.5: RED — Test output contains task ───

func TestSpawnResultOutput(t *testing.T) {
	registry := NewAgentRegistry(nil)
	spawner := NewSpawner(registry)

	req := SpawnRequest{
		AgentName: "explore",
		Task:      "find all test files in the project",
	}

	result, err := spawner.Spawn(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Output == "" {
		t.Error("expected non-empty output")
	}
	if !strings.Contains(result.Output, "find all test files") {
		t.Errorf("output should contain task text, got: %q", result.Output)
	}
}

// ─── Task 2.6: RED — Test context cancellation ───

func TestSpawnerWithContext(t *testing.T) {
	registry := NewAgentRegistry(nil)
	spawner := NewSpawner(registry)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	req := SpawnRequest{
		AgentName: "explore",
		Task:      "do work",
	}

	_, err := spawner.Spawn(ctx, req)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

// ─── Additional: Test agent not found with context ───

func TestSpawnerAgentNotFoundWithContext(t *testing.T) {
	registry := NewAgentRegistry(nil)
	spawner := NewSpawner(registry)

	ctx := context.Background()
	req := SpawnRequest{
		AgentName: "missing-agent",
		Task:      "do something",
	}

	_, err := spawner.Spawn(ctx, req)
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
}
