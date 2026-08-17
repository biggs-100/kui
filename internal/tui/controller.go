// Package tui implements the Bubble Tea application for kui's interactive
// TUI. The controller wires views to the runtime (store, loader, agent)
// without touching core — core stays stdlib-only (REQ-TUI-APP-4).
package tui

import (
	"context"
	"errors"
	"sync"

	"github.com/biggs-100/kui/internal/core"
)

// Runner is the port the controller uses to execute prompts and access the
// steering queue. The core.Agent satisfies this via Go's structural typing.
// Provider exposes the underlying core.Provider so the controller can detect
// StreamingProvider via type assertion for real-time token streaming (D7, D8).
type Runner interface {
	Run(ctx context.Context, prompt string) (string, error)
	Steering() core.PendingQueue
	Provider() core.Provider
}

// ModelResolver resolves the model name for a profile via the REQ-CLI-4
// chain (saved → profile yaml → env → default). The concrete implementation
// calls agent.ResolveModel, but the controller depends only on this function
// type so the agent package is not imported (guard test).
type ModelResolver func(profileName string) string

// Reloader is the port the controller uses to trigger a hot-reload of the
// runtime (REQ-RELOAD-14). The concrete runtime.Runtime satisfies this.
type Reloader interface {
	Reload(ctx context.Context) error
}

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
	reloader   Reloader // REQ-RELOAD-14

	// Run tracking (REQ-RELOAD-13): guards cancel-and-wait for /reload.
	running bool
	cancel  context.CancelFunc
	runDone chan struct{}

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
// If the runner's provider implements StreamingProvider, the controller
// consumes the stream channel directly and emits streamChunkMsg for each
// TextDelta, giving the TUI real-time token-by-token rendering (D7, D8).
// On stream completion, streamDoneMsg is emitted. Error chunks emit
// streamDoneMsg{err}. If the provider does not implement StreamingProvider,
// the synchronous runner.Run path is used unchanged.
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

	ctx, cancel := context.WithCancel(context.Background())
	c.mu.Lock()
	c.running = true
	c.cancel = cancel
	done := make(chan struct{})
	c.runDone = done
	c.mu.Unlock()

	// D7: detect StreamingProvider for real-time token streaming.
	if sp, ok := runner.Provider().(core.StreamingProvider); ok {
		go func() {
			c.runStreamingPrompt(sp, text)
			c.finishRun(done)
		}()
		return
	}

	// Fallback: synchronous run via agent.Run.
	go func() {
		defer c.finishRun(done)
		_, err := runner.Run(ctx, text)
		if err != nil {
			if ctx.Err() != nil {
				return // REQ-RELOAD-8: suppress cancel error display
			}
			c.emit(streamDoneMsg{err: err})
			return
		}
		c.emit(streamDoneMsg{})
	}()
}

// finishRun clears the running flag and closes the done channel so Reload()
// can proceed after cancelling an active run (REQ-RELOAD-13).
func (c *Controller) finishRun(done chan struct{}) {
	c.mu.Lock()
	c.running = false
	c.cancel = nil
	c.mu.Unlock()
	close(done)
}

// runStreamingPrompt consumes a StreamChat channel and emits streamChunkMsg
// for each TextDelta, then streamDoneMsg on completion or error. It runs in
// a goroutine (D4).
func (c *Controller) runStreamingPrompt(sp core.StreamingProvider, text string) {
	stream, err := sp.StreamChat(context.Background(), []core.Message{
		{Role: core.RoleUser, Content: text},
	}, nil)
	if err != nil {
		c.emit(streamDoneMsg{err: err})
		return
	}

	for chunk := range stream {
		if chunk.Error != nil {
			c.emit(streamDoneMsg{err: chunk.Error})
			return
		}
		if chunk.TextDelta != "" {
			c.emit(streamChunkMsg{delta: chunk.TextDelta})
		}
	}
	c.emit(streamDoneMsg{})
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

// ── Reload support (REQ-RELOAD-13/14/15) ───────────────────────────────────

// reloadStartMsg signals the start of a hot-reload.
type reloadStartMsg struct{}

// reloadDoneMsg signals completion of a hot-reload with optional error and
// counts for status display (REQ-RELOAD-12).
type reloadDoneMsg struct {
	err      error
	skills   int
	profiles int
}

// SetReloader attaches the runtime to the controller for /reload support.
func (c *Controller) SetReloader(r Reloader) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reloader = r
}

// ReloadProfiles replaces the profile list, preserving the active profile
// when it still exists in the new list (REQ-RELOAD-15). If the active
// profile was removed, index 0 becomes active.
func (c *Controller) ReloadProfiles(names []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(names) == 0 {
		names = []string{""}
	}
	// Find if current active still exists.
	activeName := c.profiles[c.active]
	c.profiles = names
	c.active = 0
	for i, p := range names {
		if p == activeName {
			c.active = i
			return
		}
	}
}

// Reload triggers a cancel-and-wait hot-reload (REQ-RELOAD-6/7/8). It
// cancels any active run, waits for it to finish, then delegates to the
// Reloader port. The controller must not be used by SubmitPrompt while
// reload is in progress — the running flag prevents interleaving.
func (c *Controller) Reload() {
	c.mu.Lock()
	running, cancel, done, reloader :=
		c.running, c.cancel, c.runDone, c.reloader
	c.mu.Unlock()

	c.emit(reloadStartMsg{})

	if running && cancel != nil {
		cancel() // REQ-RELOAD-7: cancel active run
		if done != nil {
			<-done // REQ-RELOAD-8: wait for run to exit
		}
	}

	if reloader == nil {
		c.emit(reloadDoneMsg{err: errors.New("reload not configured")})
		return
	}

	if err := reloader.Reload(context.Background()); err != nil {
		c.emit(reloadDoneMsg{err: err})
		return
	}

	c.emit(reloadDoneMsg{})
}
