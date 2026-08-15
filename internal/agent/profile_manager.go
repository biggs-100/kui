// Package agent owns the runtime wiring between the core loop and the
// adapters. It holds the concrete profile manager implementing the
// core.ProfileManager port (D16). It performs no yaml or filesystem work
// itself — that lives in the profile adapter (REQ-PROFILE-5).
package agent

import (
	"context"
	"fmt"

	"github.com/biggs-100/kui/internal/adapters/permissions"
	"github.com/biggs-100/kui/internal/adapters/profile"
	"github.com/biggs-100/kui/internal/core"
)

// Manager is the concrete core.ProfileManager (D16, REQ-PROFILE-3).
// ApplySwitch resolves and activates the named profile, reads its system
// prompt, rebuilds the tool registry subset and the permission evaluator,
// stores the resolved model, and returns the messages (new system prompt +
// profile-context marker) the loop appends to history before the next
// provider turn.
type Manager struct {
	loader   *profile.Loader
	full     *core.Registry
	registry *core.Registry
	ruleset  *permissions.Ruleset
	active   string
	model    string
}

// NewManager creates a manager over the loader and the full tool registry.
// The full registry is never mutated; each switch derives a fresh subset.
func NewManager(loader *profile.Loader, full *core.Registry) *Manager {
	return &Manager{loader: loader, full: full}
}

// ApplySwitch implements core.ProfileManager (D16, REQ-PROFILE-3). It resolves
// and activates the named profile between turns, never during a tool call or
// mid-response, and returns the messages to append to the history: the new
// system prompt and the profile-context marker (REQ-LOOP-6).
func (m *Manager) ApplySwitch(_ context.Context, name string) ([]core.Message, error) {
	resolved, err := m.loader.Resolve(name)
	if err != nil {
		return nil, err
	}
	body, err := m.loader.SystemPrompt(resolved)
	if err != nil {
		return nil, err
	}
	m.registry = subsetRegistry(m.full, resolved.Tools)
	m.ruleset = permissions.Flatten(ruleLayer(resolved.Permissions))
	m.active = name
	m.model = resolved.Model
	return []core.Message{
		{Role: core.RoleSystem, Content: body},
		{Role: core.RoleSystem, Content: marker(name)},
	}, nil
}

// Registry returns the currently active tool registry: the subset declared by
// the active profile, or the full registry before any switch.
func (m *Manager) Registry() *core.Registry {
	if m.registry == nil {
		return m.full
	}
	return m.registry
}

// Ruleset returns the currently active permission evaluator, or nil before
// any switch.
func (m *Manager) Ruleset() *permissions.Ruleset {
	return m.ruleset
}

// Active returns the name of the currently active profile ("" before any
// switch).
func (m *Manager) Active() string {
	return m.active
}

// Model returns the resolved model of the active profile (D17). The provider
// is reconfigured with it in PR 5 via SetModel.
func (m *Manager) Model() string {
	return m.model
}

// subsetRegistry returns a registry containing only the named tools from the
// full registry, in the profile's declared order (D16). Names not registered
// in the full set are skipped, since an unregistered tool is simply not
// available (REQ-PERM-1).
func subsetRegistry(full *core.Registry, names []string) *core.Registry {
	subset := core.NewRegistry()
	for _, name := range names {
		tool, ok := full.Get(name)
		if !ok {
			continue
		}
		_ = subset.Register(tool)
	}
	return subset
}

// ruleLayer maps the profile adapter's yaml rules onto the permissions
// adapter's Rule type for Flatten (D15).
func ruleLayer(rules []profile.Rule) []permissions.Rule {
	out := make([]permissions.Rule, 0, len(rules))
	for _, r := range rules {
		out = append(out, permissions.Rule{Pattern: r.Pattern, Action: permissions.Decision(r.Action)})
	}
	return out
}

// marker builds the profile-context marker message inserted into the history
// when a switch applies (REQ-LOOP-6, D16).
func marker(name string) string {
	return fmt.Sprintf("Profile switched to %s. Continue with the existing conversation context using the new profile's system prompt, tools, and permissions.", name)
}
