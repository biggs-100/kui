package orchestration

import (
	"testing"
)

// ─── Task 4.5: RED — Test engine wiring ───

func TestEngineWiresComponents(t *testing.T) {
	engine := NewEngine(nil)

	if engine.registry == nil {
		t.Error("registry is nil")
	}
	if engine.spawner == nil {
		t.Error("spawner is nil")
	}
	if engine.tool == nil {
		t.Error("tool is nil")
	}
	if engine.rules == nil {
		t.Error("rules is nil")
	}
	if engine.gatekeeper == nil {
		t.Error("gatekeeper is nil")
	}
}

func TestEngineToolAccessible(t *testing.T) {
	engine := NewEngine(nil)
	tool := engine.Tool()
	if tool == nil {
		t.Fatal("Tool() returned nil")
	}
	if tool.Name() != "orchestrate" {
		t.Errorf("Tool().Name() = %q, want %q", tool.Name(), "orchestrate")
	}
}

func TestEngineRulesAccessible(t *testing.T) {
	engine := NewEngine(nil)
	rules := engine.Rules()
	if rules == nil {
		t.Fatal("Rules() returned nil")
	}
	if rules.ExploreThreshold != 4 {
		t.Errorf("Rules().ExploreThreshold = %d, want 4", rules.ExploreThreshold)
	}
	if rules.WriteThreshold != 2 {
		t.Errorf("Rules().WriteThreshold = %d, want 2", rules.WriteThreshold)
	}
}

func TestEngineGatekeeperAccessible(t *testing.T) {
	engine := NewEngine(nil)
	gk := engine.Gatekeeper()
	if gk == nil {
		t.Fatal("Gatekeeper() returned nil")
	}
	if gk.maxRetries != 1 {
		t.Errorf("Gatekeeper().maxRetries = %d, want 1", gk.maxRetries)
	}
}
