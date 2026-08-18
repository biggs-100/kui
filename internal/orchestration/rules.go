package orchestration

import "github.com/biggs-100/kui/internal/adapters/profile"

// Rules define when to delegate work vs execute inline.
type Rules struct {
	ExploreThreshold int  `yaml:"explore_threshold"` // files to read before delegating
	WriteThreshold   int  `yaml:"write_threshold"`   // files to write before delegating
	ContextRule      bool `yaml:"context_rule"`      // delegate reading that prepares a write
}

// DefaultRules returns the default delegation rules.
func DefaultRules() *Rules {
	return &Rules{
		ExploreThreshold: 4,
		WriteThreshold:   2,
		ContextRule:      true,
	}
}

// ShouldDelegate decides if work should be delegated based on action type and file count.
func (r *Rules) ShouldDelegate(action string, fileCount int) bool {
	switch action {
	case "explore":
		return fileCount >= r.ExploreThreshold
	case "write":
		return fileCount >= r.WriteThreshold
	case "context":
		return r.ContextRule
	default:
		return false
	}
}

// RulesFromConfig creates Rules from an OrchestrationConfig, falling back to defaults.
func RulesFromConfig(config *profile.OrchestrationConfig) *Rules {
	if config == nil || config.Delegation == nil {
		return DefaultRules()
	}
	d := config.Delegation
	return &Rules{
		ExploreThreshold: d.ExploreThreshold,
		WriteThreshold:   d.WriteThreshold,
		ContextRule:      d.ContextRule,
	}
}
