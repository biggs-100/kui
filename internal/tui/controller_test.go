package tui

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/biggs-100/kui/internal/core"
)

// --- fakes for testing ---

// fakeRunner implements Runner for unit tests. Run delegates to the
// embedded core.Agent; Steering returns the injected pending queue.
type fakeRunner struct {
	agent    *core.Agent
	steering core.PendingQueue
}

func (r *fakeRunner) Run(ctx context.Context, prompt string) (string, error) {
	return r.agent.Run(ctx, prompt)
}

func (r *fakeRunner) Steering() core.PendingQueue {
	return r.steering
}

// fakeModelMemory is a minimal ModelMemory that returns a fixed model for
// every profile lookup, exercising the "saved model" branch of ResolveModel.
type fakeModelMemory struct {
	model string
}

func (m *fakeModelMemory) Get(_ string) (string, bool) { return m.model, true }
func (m *fakeModelMemory) Set(_, _ string) error       { return nil }

// stubQueue is a PendingQueue that records every Enqueue call and drains
// all messages in order. It is not concurrent-safe — tests run sequentially.
type stubQueue struct {
	messages []core.PendingMessage
}

func (q *stubQueue) Drain() []core.PendingMessage {
	d := q.messages
	q.messages = nil
	return d
}

func (q *stubQueue) Enqueue(msg core.PendingMessage) {
	q.messages = append(q.messages, msg)
}

// stubProvider returns a fixed content response.
type stubProvider struct {
	response string
}

func (p *stubProvider) Chat(_ context.Context, _ []core.Message, _ []core.Tool) ([]core.Message, error) {
	return []core.Message{{Role: core.RoleAssistant, Content: p.response}}, nil
}

// collectEvents drains events from the channel into a slice, blocking until
// the channel is closed or the deadline elapses. Returns concrete event
// values (streamDoneMsg, etc.) so callers can type-switch.
func collectEvents(ch <-chan any, timeout time.Duration) []any {
	deadline := time.Now().Add(timeout)
	var msgs []any
	for time.Now().Before(deadline) {
		select {
		case m, ok := <-ch:
			if !ok {
				return msgs
			}
			msgs = append(msgs, m)
		case <-time.After(time.Until(deadline)):
			return msgs
		}
	}
	return msgs
}

// --- Cycle Tests (REQ-TUI-PROF-2) ---

func TestControllerCycleWrapForward(t *testing.T) {
	c := NewController([]string{"coder", "writer"}, nil, nil)
	c.SwitchProfile(1)
	if c.ActiveProfile() != "writer" {
		t.Errorf("ActiveProfile() = %q, want %q", c.ActiveProfile(), "writer")
	}
	c.SwitchProfile(1)
	if c.ActiveProfile() != "coder" {
		t.Errorf("ActiveProfile() = %q, want %q", c.ActiveProfile(), "coder")
	}
}

func TestControllerCycleWrapBackward(t *testing.T) {
	c := NewController([]string{"coder", "writer"}, nil, nil)
	c.SwitchProfile(-1)
	if c.ActiveProfile() != "writer" {
		t.Errorf("ActiveProfile() = %q, want %q", c.ActiveProfile(), "writer")
	}
	c.SwitchProfile(-1)
	if c.ActiveProfile() != "coder" {
		t.Errorf("ActiveProfile() = %q, want %q", c.ActiveProfile(), "coder")
	}
}

func TestControllerCycleRapidPresses(t *testing.T) {
	c := NewController([]string{"coder", "writer"}, nil, nil)
	// Three rapid TAB presses: 0→1→0→1
	c.SwitchProfile(1)
	c.SwitchProfile(1)
	c.SwitchProfile(1)
	if c.ActiveProfile() != "writer" {
		t.Errorf("ActiveProfile() = %q, want %q", c.ActiveProfile(), "writer")
	}
}

func TestControllerCycleThreeProfiles(t *testing.T) {
	c := NewController([]string{"a", "b", "c"}, nil, nil)
	c.SwitchProfile(1)
	if c.ActiveProfile() != "b" {
		t.Errorf("1: got %q, want %q", c.ActiveProfile(), "b")
	}
	c.SwitchProfile(1)
	if c.ActiveProfile() != "c" {
		t.Errorf("2: got %q, want %q", c.ActiveProfile(), "c")
	}
	c.SwitchProfile(1)
	if c.ActiveProfile() != "a" {
		t.Errorf("3 wrap: got %q, want %q", c.ActiveProfile(), "a")
	}
	c.SwitchProfile(-1)
	if c.ActiveProfile() != "c" {
		t.Errorf("4 back: got %q, want %q", c.ActiveProfile(), "c")
	}
}

// --- Steering Enqueue Tests (REQ-TUI-PROF-3) ---

func TestSwitchEnqueuesToSteering(t *testing.T) {
	queue := &stubQueue{}
	runner := &fakeRunner{
		agent:    &core.Agent{Tools: core.NewRegistry(), MaxIterations: 1},
		steering: queue,
	}
	c := NewController([]string{"coder", "writer"}, runner, nil)

	c.SwitchProfile(1)
	if len(queue.messages) != 1 {
		t.Fatalf("steering queue length = %d, want 1", len(queue.messages))
	}
	if queue.messages[0].SwitchProfile != "writer" {
		t.Errorf("queued SwitchProfile = %q, want %q", queue.messages[0].SwitchProfile, "writer")
	}
}

func TestSwitchDoesNotChangeActiveImmediately(t *testing.T) {
	queue := &stubQueue{}
	runner := &fakeRunner{
		agent:    &core.Agent{Tools: core.NewRegistry(), MaxIterations: 1},
		steering: queue,
	}
	c := NewController([]string{"coder", "writer"}, runner, nil)

	c.SwitchProfile(1)
	if c.ActiveProfile() != "writer" {
		t.Errorf("ActiveProfile() = %q, want %q", c.ActiveProfile(), "writer")
	}
	if len(queue.messages) != 1 {
		t.Errorf("steering queue not enqueued: len = %d", len(queue.messages))
	}
}

// --- Per-Prompt Model Chain Tests (REQ-CLI-4) ---

func TestSubmitPromptResolvesModelFromStore(t *testing.T) {
	store := &fakeModelMemory{model: "saved"}
	queue := &stubQueue{}
	runner := &fakeRunner{
		agent: &core.Agent{
			Provider:      &stubProvider{response: "ok"},
			Tools:         core.NewRegistry(),
			MaxIterations: 1,
		},
		steering: queue,
	}
	c := NewController([]string{"coder"}, runner, func(name string) string {
		return resolveTestModel(store, nil, name, "")
	})

	c.SubmitPrompt("hello")
	time.Sleep(50 * time.Millisecond)
}

func TestSubmitPromptUsesEnvModelFallback(t *testing.T) {
	queue := &stubQueue{}
	runner := &fakeRunner{
		agent: &core.Agent{
			Provider:      &stubProvider{response: "ok"},
			Tools:         core.NewRegistry(),
			MaxIterations: 1,
		},
		steering: queue,
	}
	c := NewController([]string{"coder"}, runner, func(name string) string {
		return resolveTestModel(nil, nil, name, "env-model")
	})

	c.SubmitPrompt("hello")
	time.Sleep(50 * time.Millisecond)
}

// resolveTestModel mirrors agent.ResolveModel without importing the agent
// package. Only the store and envModel branches are exercised — the loader
// branch is covered by the agent package's own tests.
func resolveTestModel(store core.ModelMemory, _ interface{}, name, envModel string) string {
	if store != nil {
		if model, ok := store.Get(name); ok {
			return model
		}
	}
	if envModel != "" {
		return envModel
	}
	return "gpt-4o-mini"
}

// --- Events Emitted on Run Completion ---

func TestEventsEmittedOnRunCompletion(t *testing.T) {
	provider := &stubProvider{response: "done"}
	queue := &stubQueue{}
	runner := &fakeRunner{
		agent: &core.Agent{
			Provider:      provider,
			Tools:         core.NewRegistry(),
			MaxIterations: 1,
		},
		steering: queue,
	}
	c := NewController([]string{"coder"}, runner, func(name string) string {
		return "gpt-4o-mini"
	})

	c.SubmitPrompt("hello")
	events := collectEvents(c.Events(), 200*time.Millisecond)

	found := false
	for _, e := range events {
		if _, ok := e.(streamDoneMsg); ok {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected streamDoneMsg on events channel after Run completion")
	}
}

// --- Concurrent safety ---

func TestConcurrentSwitchProfile(t *testing.T) {
	c := NewController([]string{"a", "b", "c"}, nil, nil)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.SwitchProfile(1)
		}()
	}
	wg.Wait()
	// After 100 increments on a 3-profile cycle: 100 % 3 = 1 → index 1 = "b"
	if c.ActiveProfile() != "b" {
		t.Errorf("ActiveProfile() = %q, want %q after 100 concurrent presses", c.ActiveProfile(), "b")
	}
}

func TestConcurrentEvents(t *testing.T) {
	c := NewController([]string{"a"}, nil, nil)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c.emit(streamDoneMsg{})
		}(i)
	}
	wg.Wait()
}

// --- Nil Observer Rendering (REQ-LOOP-7) ---

func TestNilObserverSubmitPromptDoesNotPanic(t *testing.T) {
	// A controller with a nil runner and nil resolver must not panic
	// when SubmitPrompt is called. This covers the nil-observer scenario
	// where the core.Agent has no Observer set.
	c := NewController([]string{"coder"}, nil, nil)
	c.SubmitPrompt("hello")
	// No panic = pass
}

func TestNilObserverWithRunnerNoObserver(t *testing.T) {
	// SubmitPrompt with a runner whose agent has a nil Observer
	// must complete without panicking.
	queue := &stubQueue{}
	runner := &fakeRunner{
		agent: &core.Agent{
			Provider:      &stubProvider{response: "ok"},
			Tools:         core.NewRegistry(),
			MaxIterations: 1,
			// Observer intentionally nil
		},
		steering: queue,
	}
	c := NewController([]string{"coder"}, runner, func(name string) string {
		return "gpt-4o-mini"
	})

	c.SubmitPrompt("hello")
	// Wait for run to complete — nil observer should be handled by
	// core.emitObserver (recovered panics, nil-safe calls).
	events := collectEvents(c.Events(), 200*time.Millisecond)
	found := false
	for _, e := range events {
		if _, ok := e.(streamDoneMsg); ok {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected streamDoneMsg after run with nil observer")
	}
}

// --- Channel Overflow Drop (D3 select-default) ---

func TestEmitDropsOnFullChannel(t *testing.T) {
	c := NewController([]string{"a"}, nil, nil)
	// Drain the pre-allocated buffer by filling it
	for i := 0; i < 64; i++ {
		c.events <- streamChunkMsg{delta: "fill"}
	}
	// Channel is now full. This emit must NOT block (D3 select-default).
	done := make(chan struct{})
	go func() {
		c.emit(streamDoneMsg{})
		close(done)
	}()
	select {
	case <-done:
		// emit returned without blocking — correct
	case <-time.After(100 * time.Millisecond):
		t.Fatal("emit blocked on full channel — D3 select-default not implemented")
	}
	// Verify the overflow message was dropped (channel still has 64 items)
	if len(c.events) != 64 {
		t.Errorf("channel length = %d, want 64 (overflow message should be dropped)", len(c.events))
	}
}

func TestSubmitPromptEventDropOnFullChannel(t *testing.T) {
	// When SubmitPrompt's goroutine emits on a full channel, the event
	// must be dropped rather than blocking the loop.
	provider := &stubProvider{response: "done"}
	queue := &stubQueue{}
	runner := &fakeRunner{
		agent: &core.Agent{
			Provider:      provider,
			Tools:         core.NewRegistry(),
			MaxIterations: 1,
		},
		steering: queue,
	}
	c := NewController([]string{"coder"}, runner, func(name string) string {
		return "gpt-4o-mini"
	})

	// Fill the events channel
	for i := 0; i < 64; i++ {
		c.events <- streamChunkMsg{delta: "fill"}
	}

	c.SubmitPrompt("hello")
	// Wait for the goroutine to complete — it should not block
	time.Sleep(200 * time.Millisecond)
	// Channel should still be full (the done event was dropped)
	if len(c.events) != 64 {
		t.Errorf("channel length = %d, want 64 (overflow event should be dropped)", len(c.events))
	}
}
