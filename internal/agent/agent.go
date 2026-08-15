package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/biggs-100/kui/internal/adapters/skills"
	"github.com/biggs-100/kui/internal/core"
)

// Agent is the runtime wrapper (D19, D21): it owns the active profile manager,
// the steering and follow-up queues, and the skills index, and it wires the
// core loop on each Run. It performs no yaml or filesystem work itself — all
// IO lives in the adapters (REQ-PROFILE-5, guard test). SystemMessages seeds
// the skills index (names, descriptions, triggers — never bodies, REQ-SKILL-3)
// and LoadSkill reads a body only when the skill is invoked.
type Agent struct {
	manager       *Manager
	skills        *skills.Index
	provider      core.Provider
	steering      *PendingMessageQueue
	followUp      *PendingMessageQueue
	maxIterations int
}

// NewAgent builds the wrapper over the profile manager, the skills index, the
// provider, and the loop iteration budget. The steering queue drains the
// whole queue after each turn (REQ-QUEUE-1) and the follow-up queue releases
// one message when the loop would otherwise stop (REQ-QUEUE-2).
func NewAgent(manager *Manager, skillsIndex *skills.Index, provider core.Provider, maxIterations int) *Agent {
	return &Agent{
		manager:       manager,
		skills:        skillsIndex,
		provider:      provider,
		steering:      NewPendingMessageQueue(core.QueueModeAll),
		followUp:      NewPendingMessageQueue(core.QueueModeOneAtATime),
		maxIterations: maxIterations,
	}
}

// Run executes one session through the core loop, wiring the manager's active
// registry and ruleset and the agent's queues as the steering and follow-up
// ports (D14, D19). It returns the provider's final answer.
//
// Permissions is wired only when the manager holds a ruleset: assigning a nil
// *Ruleset to the interface field would defeat the loop's nil-safe port
// contract (a typed nil inside an interface is non-nil).
func (a *Agent) Run(ctx context.Context, prompt string) (string, error) {
	loop := &core.Agent{
		Provider:      a.provider,
		Tools:         a.manager.Registry(),
		MaxIterations: a.maxIterations,
		Steering:      a.steering,
		FollowUp:      a.followUp,
		Profiles:      a.manager,
	}
	if ruleset := a.manager.Ruleset(); ruleset != nil {
		loop.Permissions = ruleset
	}
	return loop.Run(ctx, prompt)
}

// SystemMessages seeds the session with the available skills from the index —
// names, descriptions and triggers only, never any body text (REQ-SKILL-3,
// D21). With no indexed skills it returns nothing.
func (a *Agent) SystemMessages() []core.Message {
	if a.skills == nil {
		return nil
	}
	skills := a.skills.List()
	if len(skills) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString("Available skills:\n")
	for _, s := range skills {
		b.WriteString("- " + s.Name + ": " + s.Description + " (triggers: " + strings.Join(s.Triggers, ", ") + ")\n")
	}
	return []core.Message{{Role: core.RoleSystem, Content: b.String()}}
}

// LoadSkill reads the named skill's SKILL.md body at invocation time
// (D21, REQ-SKILL-3). An unindexed skill or a missing body returns a typed
// error naming it.
func (a *Agent) LoadSkill(name string) (string, error) {
	skill, ok := a.skills.Get(name)
	if !ok {
		return "", &core.SkillLoadError{Name: name, Err: fmt.Errorf("skill %q not found", name)}
	}
	return a.skills.Load(skill)
}

// Manager returns the wrapped profile manager.
func (a *Agent) Manager() *Manager {
	return a.manager
}

// Steering returns the steering queue for enqueuing between-turn messages.
func (a *Agent) Steering() *PendingMessageQueue {
	return a.steering
}

// FollowUp returns the follow-up queue for enqueuing stop-delaying messages.
func (a *Agent) FollowUp() *PendingMessageQueue {
	return a.followUp
}
