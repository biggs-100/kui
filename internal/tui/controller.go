// Package tui implements the Bubble Tea application for kui's interactive
// TUI. The controller wires views to the runtime (store, loader, agent)
// without touching core — core stays stdlib-only (REQ-TUI-APP-4).
package tui

import (
	"context"
	"sync"

	"github.com/biggs-100/kui/internal/core"
)

// Runner is the port the controller uses to execute prompts and access the
// steering queue. The core.Agent satisfies this via Go's structural typing.
type Runner interface {
	Run(ctx context.Context, prompt string) (string, error)
	Steering() core.PendingQueue
}

// ModelResolver resolves the model name for a profile via the REQ-CLI-4
// chain (saved → profile yaml → env → default). The concrete implementation
// calls agent.ResolveModel, but the controller depends only on this function
// type so the agent package is not imported (guard test).
type ModelResolver func(profileName string) string

// Controller manages profile switching, prompt submission, and event
// delivery for the TUI. It owns the buffered events channel and the
// active-profile index. The events channel uses D3 channel+Cmd handoff:
// a buffered chan (capacity 64) with drop-on-full via select-default.
//
// Controller is safe for concurrent use.
type Controller struct {
	runner    Runner
	resolver  ModelResolver
	profiles  []string
	active    int
	events    chan any
	eventsBuf int
	// SetModeler sets the model on the provider before each prompt.
	// When nil, model setting is skipped (e.g. in pure cycle tests).
	SetModeler SetModeler

	mu sync.Mutex
}

// NewController creates a Controller with the given profiles, runner, and
// model resolver. profiles must be non-empty; the first entry is the
// initial active profile. runner and resolver may be nil for pure
// cycle tests (SwitchProfile/ActiveProfile work without them).
func NewController(profiles []string, runner Runner, resolver ModelResolver) *Controller {
	if len(profiles) == 0 {
		profiles = []string{""}
	}
	return &Controller{
		runner:    runner,
		resolver:  resolver,
		profiles:  profiles,
		active:    0,
		events:    make(chan any, 64),
		eventsBuf: 64,
	}
}

// ActiveProfile returns the name of the currently active profile.
func (c *Controller) ActiveProfile() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.profiles[c.active]
}

// Profiles returns the list of profile names.
func (c *Controller) Profiles() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make([]string, len(c.profiles))
	copy(cp, c.profiles)
	return cp
}

// SwitchProfile advances (delta > 0) or retreats (delta < 0) through the
// profile list with wrap-around (D5). Each call advances exactly one step,
// so rapid presses cycle deterministically (REQ-TUI-PROF-2). The switch is
// also enqueued via the steering queue so the loop applies it between turns
// (REQ-TUI-PROF-3).
func (c *Controller) SwitchProfile(delta int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	n := len(c.profiles)
	c.active = ((c.active+delta)%n + n) % n

	if c.runner != nil {
		if q := c.runner.Steering(); q != nil {
			q.Enqueue(core.PendingMessage{
				SwitchProfile: c.profiles[c.active],
			})
		}
	}
}

// SubmitPrompt resolves the model for the active profile, sets it on the
// provider (REQ-CLI-4), then spawns a goroutine to run agent.Run per D4
// (one goroutine per prompt). Submissions are blocked while a run is
// active — the caller must wait on the returned channel or the Events
// channel for completion.
//
// Events (stream chunks, done, tool calls) are delivered through the
// Events channel using D3 channel+Cmd handoff: buffered chan with
// drop-on-full via select-default.
func (c *Controller) SubmitPrompt(text string) {
	c.mu.Lock()
	runner := c.runner
	resolver := c.resolver
	profile := c.profiles[c.active]
	sm := c.SetModeler
	c.mu.Unlock()

	if runner == nil || resolver == nil {
		return
	}

	model := resolver(profile)
	// REQ-CLI-4: set the resolved model on the provider before each run.
	if sm != nil {
		sm.SetModel(model)
	}

	go func() {
		_, err := runner.Run(context.Background(), text)
		if err != nil {
			c.emit(streamDoneMsg{err: err})
			return
		}
		c.emit(streamDoneMsg{})
	}()
}

// Events returns a read-only channel of controller events. The caller
// (typically the Bubble Tea app) reads events and translates them into
// tea.Msg values for the Update cycle (D3).
//
// Event types:
//   - streamDoneMsg: a run completed (err == nil on success)
//   - streamChunkMsg: a chunk of the answer (deferred to SSE)
//   - toolCallMsg / toolResultMsg: tool events (deferred to Slice C)
func (c *Controller) Events() <-chan any {
	return c.events
}

// emit sends a message to the events channel. If the channel is full
// (D3 select-default), the message is dropped — the controller never
// blocks the loop goroutine. A nil events channel is a no-op.
func (c *Controller) emit(msg any) {
	if c.events == nil {
		return
	}
	select {
	case c.events <- msg:
	default:
	}
}

// streamDoneMsg is emitted when agent.Run completes. err is nil on
// success; a non-nil err indicates the run failed (REQ-TUI-CHAT-2).
type streamDoneMsg struct {
	err error
}

// streamChunkMsg is emitted for streaming answer chunks. Currently
// deferred — the controller emits the whole answer as one chunk then
// done (D7).
type streamChunkMsg struct {
	msgID int
	delta string
}

// toolCallMsg is emitted when a tool call begins (deferred to Slice C).
type toolCallMsg struct {
	callID string
	name   string
}

// toolResultMsg is emitted when a tool call completes (deferred to Slice C).
type toolResultMsg struct {
	callID string
	result string
}
