package orchestration

import (
	"testing"

	"github.com/biggs-100/kui/internal/adapters/profile"
	"gopkg.in/yaml.v3"
)

// ─── Task 4.7: RED — Test profile YAML integration ───

func TestProfileOrchestrationConfig(t *testing.T) {
	yamlData := `
orchestration:
  delegation:
    explore_threshold: 8
    write_threshold: 4
    context_rule: false
  gatekeeper:
    enabled: true
    max_retries: 3
`
	var config profile.Config
	if err := yaml.Unmarshal([]byte(yamlData), &config); err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}

	if config.Orchestration == nil {
		t.Fatal("Orchestration config is nil")
	}
	if config.Orchestration.Delegation == nil {
		t.Fatal("Delegation config is nil")
	}
	if config.Orchestration.Delegation.ExploreThreshold != 8 {
		t.Errorf("ExploreThreshold = %d, want 8", config.Orchestration.Delegation.ExploreThreshold)
	}
	if config.Orchestration.Delegation.WriteThreshold != 4 {
		t.Errorf("WriteThreshold = %d, want 4", config.Orchestration.Delegation.WriteThreshold)
	}
	if config.Orchestration.Delegation.ContextRule {
		t.Error("ContextRule = true, want false")
	}
	if config.Orchestration.Gatekeeper == nil {
		t.Fatal("Gatekeeper config is nil")
	}
	if !config.Orchestration.Gatekeeper.Enabled {
		t.Error("Gatekeeper.Enabled = false, want true")
	}
	if config.Orchestration.Gatekeeper.MaxRetries != 3 {
		t.Errorf("Gatekeeper.MaxRetries = %d, want 3", config.Orchestration.Gatekeeper.MaxRetries)
	}
}

func TestProfileDefaultOrchestration(t *testing.T) {
	yamlData := `
name: test-profile
model: balanced
`
	var config profile.Config
	if err := yaml.Unmarshal([]byte(yamlData), &config); err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}

	if config.Orchestration != nil {
		t.Error("Orchestration should be nil when not specified in YAML")
	}
}

func TestRulesFromOrchestrationConfig(t *testing.T) {
	config := &profile.OrchestrationConfig{
		Delegation: &profile.DelegationConfig{
			ExploreThreshold: 6,
			WriteThreshold:   3,
			ContextRule:      false,
		},
	}

	r := RulesFromConfig(config)
	if r.ExploreThreshold != 6 {
		t.Errorf("ExploreThreshold = %d, want 6", r.ExploreThreshold)
	}
	if r.WriteThreshold != 3 {
		t.Errorf("WriteThreshold = %d, want 3", r.WriteThreshold)
	}
	if r.ContextRule {
		t.Error("ContextRule = true, want false")
	}
}

func TestRulesFromNilConfig(t *testing.T) {
	r := RulesFromConfig(nil)
	defaults := DefaultRules()
	if r.ExploreThreshold != defaults.ExploreThreshold {
		t.Errorf("ExploreThreshold = %d, want %d (default)", r.ExploreThreshold, defaults.ExploreThreshold)
	}
	if r.WriteThreshold != defaults.WriteThreshold {
		t.Errorf("WriteThreshold = %d, want %d (default)", r.WriteThreshold, defaults.WriteThreshold)
	}
}

func TestGatekeeperFromConfig(t *testing.T) {
	config := &profile.OrchestrationConfig{
		Gatekeeper: &profile.GatekeeperConfig{
			Enabled:    true,
			MaxRetries: 5,
		},
	}

	gk := GatekeeperFromConfig(config)
	if gk.maxRetries != 5 {
		t.Errorf("maxRetries = %d, want 5", gk.maxRetries)
	}
}

func TestGatekeeperFromNilConfig(t *testing.T) {
	gk := GatekeeperFromConfig(nil)
	if gk.maxRetries != 1 {
		t.Errorf("maxRetries = %d, want 1 (default)", gk.maxRetries)
	}
}
