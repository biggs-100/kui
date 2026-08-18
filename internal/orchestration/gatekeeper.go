package orchestration

import (
	"errors"

	"github.com/biggs-100/kui/internal/adapters/profile"
)

// Sentinel errors for gatekeeper validation.
var (
	ErrEmptyOutput = errors.New("agent produced empty output")
	ErrAgentError  = errors.New("agent returned an error")
)

// Gatekeeper validates agent results in auto mode.
type Gatekeeper struct {
	maxRetries int
}

// NewGatekeeper creates a gatekeeper with the given max retry count.
func NewGatekeeper(maxRetries int) *Gatekeeper {
	return &Gatekeeper{maxRetries: maxRetries}
}

// Validate checks if a result meets quality gates.
func (g *Gatekeeper) Validate(result *SpawnResult) error {
	if result.Error != nil {
		return ErrAgentError
	}
	if result.Output == "" {
		return ErrEmptyOutput
	}
	return nil
}

// ShouldRetry decides if a failed result should be retried.
func (g *Gatekeeper) ShouldRetry(result *SpawnResult, attempt int) bool {
	return result.Error != nil && attempt < g.maxRetries
}

// GatekeeperFromConfig creates a Gatekeeper from an OrchestrationConfig, falling back to defaults.
func GatekeeperFromConfig(config *profile.OrchestrationConfig) *Gatekeeper {
	if config == nil || config.Gatekeeper == nil {
		return NewGatekeeper(1)
	}
	return NewGatekeeper(config.Gatekeeper.MaxRetries)
}
