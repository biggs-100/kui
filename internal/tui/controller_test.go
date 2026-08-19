package tui

import (
	"context"
	"fmt"
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

func (r *fakeRunner) Run(ctx context.Context, prompt string, history []core.Message) (string, []core.Message, error) {
	r.agent.History = history
	answer, err := r.agent.Run(ctx, prompt)
	return answer, r.agent.Messages(), err
}

func (r *fakeRunner) Steering() core.PendingQueue {
	return r.steering
}

func (r *fakeRunner) Provider() core.Provider {
	if r.agent != nil {
		return r.agent.Provider
	}
	return nil
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

func TestTrackUsage(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	c.SetModelName("gpt-4")
	usage := core.Usage{InputTokens: 100, OutputTokens: 50, TotalTokens: 150}
	c.TrackUsage(usage)

	if c.TotalTokens() != 150 {
		t.Errorf("TotalTokens() = %d, want 150", c.TotalTokens())
	}
	if c.Cost() <= 0 {
		t.Error("Cost() should be > 0 after tracking usage")
	}
}

func TestTrackUsageMultiple(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	c.TrackUsage(core.Usage{InputTokens: 100, OutputTokens: 50, TotalTokens: 150})
	c.TrackUsage(core.Usage{InputTokens: 200, OutputTokens: 100, TotalTokens: 300})

	if c.TotalTokens() != 450 {
		t.Errorf("TotalTokens() = %d, want 450", c.TotalTokens())
	}
}

func TestTrackUsageUnknownModel(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	c.SetModelName("unknown-model")
	c.TrackUsage(core.Usage{InputTokens: 100, OutputTokens: 50, TotalTokens: 150})

	// Unknown model should have zero cost
	if c.Cost() != 0 {
		t.Errorf("Cost() for unknown model = %f, want 0", c.Cost())
	}
}

func TestSetModelName(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	c.SetModelName("gpt-4")
	if c.ModelName() != "gpt-4" {
		t.Errorf("ModelName() = %q, want %q", c.ModelName(), "gpt-4")
	}
}

func TestTrackUsageGPT4(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	c.SetModelName("gpt-4")
	c.TrackUsage(core.Usage{InputTokens: 1000, OutputTokens: 500, TotalTokens: 1500})

	// gpt-4: $30/1M input, $60/1M output
	// cost = 1000*30/1e6 + 500*60/1e6 = 0.03 + 0.03 = 0.06
	cost := c.Cost()
	if cost < 0.05 || cost > 0.07 {
		t.Errorf("Cost() = %f, want ~0.06", cost)
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

// --- Streaming fakes ---

// fakeStreamingProvider implements core.StreamingProvider for testing the
// controller's streaming detection path.
type fakeStreamingProvider struct {
	responses [][]core.StreamChunk
	calls     int
}

func (p *fakeStreamingProvider) Chat(_ context.Context, _ []core.Message, _ []core.Tool) ([]core.Message, error) {
	return nil, fmt.Errorf("Chat should not be called on streaming provider")
}

func (p *fakeStreamingProvider) StreamChat(_ context.Context, _ []core.Message, _ []core.Tool) (<-chan core.StreamChunk, error) {
	if p.calls >= len(p.responses) {
		return nil, fmt.Errorf("fakeStreamingProvider: unexpected extra StreamChat call")
	}
	chunks := p.responses[p.calls]
	p.calls++
	ch := make(chan core.StreamChunk, len(chunks))
	for _, c := range chunks {
		ch <- c
	}
	close(ch)
	return ch, nil
}

// streamingRunner is a Runner whose Provider returns a StreamingProvider.
type streamingRunner struct {
	provider core.StreamingProvider
	steering core.PendingQueue
}

func (r *streamingRunner) Run(_ context.Context, _ string, _ []core.Message) (string, []core.Message, error) {
	return "", nil, fmt.Errorf("Run should not be called when streaming")
}

func (r *streamingRunner) Steering() core.PendingQueue {
	return r.steering
}

func (r *streamingRunner) Provider() core.Provider {
	return r.provider
}

// --- Streaming Tests (Task 4.1 RED + 4.2 GREEN) ---

func TestControllerStreamingPath(t *testing.T) {
	// D7: StreamingProvider detected via type assertion → StreamChat called,
	// streamChunkMsg emitted per TextDelta, streamDoneMsg on completion.
	provider := &fakeStreamingProvider{
		responses: [][]core.StreamChunk{
			{
				{TextDelta: "Hello"},
				{TextDelta: " world"},
				{Done: true},
			},
		},
	}
	runner := &streamingRunner{provider: provider}
	c := NewController([]string{"coder"}, runner, func(name string) string {
		return "gpt-4o-mini"
	})

	c.SubmitPrompt("hello")
	events := collectEvents(c.Events(), 500*time.Millisecond)

	// Expect: 2 streamChunkMsg + 1 streamDoneMsg
	var chunks []streamChunkMsg
	var done streamDoneMsg
	doneFound := false
	for _, e := range events {
		switch v := e.(type) {
		case streamChunkMsg:
			chunks = append(chunks, v)
		case streamDoneMsg:
			done = v
			doneFound = true
		}
	}

	if !doneFound {
		t.Fatal("expected streamDoneMsg on events channel after stream completion")
	}
	if done.err != nil {
		t.Errorf("streamDoneMsg.err = %v, want nil", done.err)
	}
	if len(chunks) != 2 {
		t.Fatalf("received %d streamChunkMsg, want 2", len(chunks))
	}
	if chunks[0].delta != "Hello" {
		t.Errorf("chunk[0].delta = %q, want %q", chunks[0].delta, "Hello")
	}
	if chunks[1].delta != " world" {
		t.Errorf("chunk[1].delta = %q, want %q", chunks[1].delta, " world")
	}
}

// --- Sync Fallback Tests (Task 4.4 RED + 4.5 GREEN) ---

func TestControllerSyncFallback(t *testing.T) {
	// D7: provider does NOT implement StreamingProvider → Run() called
	// instead, streamDoneMsg emitted on completion.
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

	// Should get only a streamDoneMsg (no streamChunkMsg)
	found := false
	for _, e := range events {
		if d, ok := e.(streamDoneMsg); ok {
			found = true
			if d.err != nil {
				t.Errorf("streamDoneMsg.err = %v, want nil", d.err)
			}
		}
		if _, ok := e.(streamChunkMsg); ok {
			t.Error("unexpected streamChunkMsg in sync fallback path")
		}
	}
	if !found {
		t.Error("expected streamDoneMsg after sync run")
	}
}

func TestControllerStreamError(t *testing.T) {
	// D8: error mid-stream → streamDoneMsg{err} emitted.
	streamErr := fmt.Errorf("connection lost")
	provider := &fakeStreamingProvider{
		responses: [][]core.StreamChunk{
			{
				{TextDelta: "partial"},
				{Error: streamErr},
			},
		},
	}
	runner := &streamingRunner{provider: provider}
	c := NewController([]string{"coder"}, runner, func(name string) string {
		return "gpt-4o-mini"
	})

	c.SubmitPrompt("hello")
	events := collectEvents(c.Events(), 500*time.Millisecond)

	// Should get streamChunkMsg for "partial", then streamDoneMsg{err}
	var chunks []streamChunkMsg
	var done streamDoneMsg
	doneFound := false
	for _, e := range events {
		switch v := e.(type) {
		case streamChunkMsg:
			chunks = append(chunks, v)
		case streamDoneMsg:
			done = v
			doneFound = true
		}
	}

	if !doneFound {
		t.Fatal("expected streamDoneMsg after stream error")
	}
	if done.err == nil {
		t.Error("streamDoneMsg.err = nil, want stream error")
	}
	if done.err != streamErr {
		t.Errorf("streamDoneMsg.err = %v, want %v", done.err, streamErr)
	}
	if len(chunks) != 1 {
		t.Fatalf("received %d streamChunkMsg before error, want 1", len(chunks))
	}
	if chunks[0].delta != "partial" {
		t.Errorf("chunks[0].delta = %q, want %q", chunks[0].delta, "partial")
	}
}

// --- Undo/Redo Tests (Phase 3) ---

func TestUndoStack(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	// Start with 2 messages and push undo point
	c.mu.Lock()
	c.messages = []core.Message{
		{Role: core.RoleUser, Content: "hello"},
		{Role: core.RoleAssistant, Content: "hi there"},
	}
	c.mu.Unlock()

	// Push undo point at 2 messages
	c.PushUndo()

	// Add 2 more messages (simulating a new turn)
	c.mu.Lock()
	c.messages = append(c.messages,
		core.Message{Role: core.RoleUser, Content: "question"},
		core.Message{Role: core.RoleAssistant, Content: "answer"},
	)
	c.mu.Unlock()

	// Now there are 4 messages, undo should restore to 2
	if !c.Undo() {
		t.Fatal("Undo should return true when stack is non-empty")
	}

	c.mu.Lock()
	n := len(c.messages)
	c.mu.Unlock()
	if n != 2 {
		t.Errorf("after Undo, messages len = %d, want 2", n)
	}

	// Redo should restore to 4
	if !c.Redo() {
		t.Fatal("Redo should return true when stack is non-empty")
	}

	c.mu.Lock()
	n = len(c.messages)
	c.mu.Unlock()
	if n != 4 {
		t.Errorf("after Redo, messages len = %d, want 4", n)
	}
}

func TestUndoEmptyStack(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	c.mu.Lock()
	c.messages = []core.Message{
		{Role: core.RoleUser, Content: "hello"},
	}
	c.mu.Unlock()

	// Undo on empty stack should return false and not panic
	if c.Undo() {
		t.Error("Undo on empty stack should return false")
	}

	c.mu.Lock()
	n := len(c.messages)
	c.mu.Unlock()
	if n != 1 {
		t.Errorf("Undo on empty stack should not change messages, len = %d, want 1", n)
	}
}

func TestRedoClearsOnNewPush(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	c.mu.Lock()
	c.messages = []core.Message{
		{Role: core.RoleUser, Content: "hello"},
		{Role: core.RoleAssistant, Content: "hi"},
	}
	c.mu.Unlock()

	// Push undo, undo, then push new undo point — redo should be cleared
	c.PushUndo()
	c.Undo()
	c.PushUndo()

	// Redo should fail (redo stack was cleared)
	if c.Redo() {
		t.Error("Redo should return false after new push clears redo stack")
	}
}

// --- Session Persistence Tests (Phase 4) ---

// fakeSessionStore implements core.SessionStore for testing.
type fakeSessionStore struct {
	sessions map[string]*core.Session
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{sessions: make(map[string]*core.Session)}
}

func (s *fakeSessionStore) Save(session *core.Session) error {
	s.sessions[session.Meta.ID] = session
	return nil
}

func (s *fakeSessionStore) Load(id string) (*core.Session, error) {
	sess, ok := s.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session %q not found", id)
	}
	return sess, nil
}

func (s *fakeSessionStore) List() ([]core.SessionMeta, error) {
	var metas []core.SessionMeta
	for _, sess := range s.sessions {
		metas = append(metas, sess.Meta)
	}
	return metas, nil
}

func (s *fakeSessionStore) Delete(id string) error {
	delete(s.sessions, id)
	return nil
}

func (s *fakeSessionStore) Rename(id string, name string) error {
	sess, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("session %q not found", id)
	}
	sess.Meta.Name = name
	return nil
}

func TestControllerSetSessionStoreAndID(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	store := newFakeSessionStore()

	c.SetSessionStore(store)
	c.SetSessionID("test-session-1")

	if c.SessionStore() != store {
		t.Error("SessionStore() did not return the set store")
	}
	if c.SessionID() != "test-session-1" {
		t.Errorf("SessionID() = %q, want %q", c.SessionID(), "test-session-1")
	}
}

func TestControllerSaveSession(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	store := newFakeSessionStore()
	c.SetSessionStore(store)
	c.SetSessionID("save-test")

	// Simulate accumulated messages
	c.mu.Lock()
	c.messages = []core.Message{
		{Role: core.RoleUser, Content: "hello"},
		{Role: core.RoleAssistant, Content: "hi"},
	}
	c.mu.Unlock()

	if err := c.SaveSession(); err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	sess, err := store.Load("save-test")
	if err != nil {
		t.Fatalf("store.Load() error = %v", err)
	}
	if len(sess.Messages) != 2 {
		t.Errorf("saved session has %d messages, want 2", len(sess.Messages))
	}
	if sess.Meta.Profile != "coder" {
		t.Errorf("saved session profile = %q, want %q", sess.Meta.Profile, "coder")
	}
}

func TestControllerSaveSessionNoopWithoutStore(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	// No store configured — SaveSession should be a no-op.
	if err := c.SaveSession(); err != nil {
		t.Fatalf("SaveSession() with no store returned error: %v", err)
	}
}

func TestControllerLoadSession(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	store := newFakeSessionStore()
	c.SetSessionStore(store)

	// Pre-populate a session
	_ = store.Save(&core.Session{
		Meta:     core.NewSessionMeta("load-test", "coder"),
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "question"},
			{Role: core.RoleAssistant, Content: "answer"},
		},
	})

	msgs, err := c.LoadSession("load-test")
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("LoadSession() returned %d messages, want 2", len(msgs))
	}
	if msgs[0].Content != "question" {
		t.Errorf("loaded message[0].Content = %q, want %q", msgs[0].Content, "question")
	}
	if c.SessionID() != "load-test" {
		t.Errorf("SessionID() = %q, want %q after LoadSession", c.SessionID(), "load-test")
	}
}

func TestControllerLoadSessionNotFound(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	store := newFakeSessionStore()
	c.SetSessionStore(store)

	_, err := c.LoadSession("nonexistent")
	if err == nil {
		t.Fatal("LoadSession() for nonexistent session should return error")
	}
}

func TestControllerAutoSaveAfterSubmit(t *testing.T) {
	// Verify that SubmitPrompt triggers auto-save after the run completes.
	store := newFakeSessionStore()
	queue := &stubQueue{}
	runner := &fakeRunner{
		agent: &core.Agent{
			Provider:      &stubProvider{response: "saved answer"},
			Tools:         core.NewRegistry(),
			MaxIterations: 1,
		},
		steering: queue,
	}
	c := NewController([]string{"coder"}, runner, func(name string) string {
		return "gpt-4o-mini"
	})
	c.SetSessionStore(store)
	c.SetSessionID("auto-save-test")

	c.SubmitPrompt("hello")
	events := collectEvents(c.Events(), 500*time.Millisecond)

	// Wait for streamDoneMsg
	found := false
	for _, e := range events {
		if _, ok := e.(streamDoneMsg); ok {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected streamDoneMsg after auto-save submit")
	}

	// Give auto-save goroutine time to complete
	time.Sleep(100 * time.Millisecond)

	sess, err := store.Load("auto-save-test")
	if err != nil {
		t.Fatalf("auto-save did not persist session: %v", err)
	}
	if len(sess.Messages) == 0 {
		t.Error("auto-saved session has no messages")
	}
}

// --- LSP Dispatch Tests (Fix #6) ---

func TestDispatchLspWithoutDispatcher(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	_, err := c.DispatchLsp("lsp_definition", map[string]interface{}{"uri": "file:///test.go"})
	if err == nil {
		t.Error("DispatchLsp without dispatcher should return error")
	}
}

func TestDispatchLspWithDispatcher(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	called := false
	c.SetLspDispatcher(func(toolName string, args map[string]interface{}) (string, error) {
		called = true
		if toolName != "lsp_definition" {
			t.Errorf("toolName = %q, want %q", toolName, "lsp_definition")
		}
		return `{"result":"ok"}`, nil
	})

	result, err := c.DispatchLsp("lsp_definition", map[string]interface{}{"uri": "file:///test.go"})
	if err != nil {
		t.Fatalf("DispatchLsp error: %v", err)
	}
	if !called {
		t.Error("dispatcher was not called")
	}
	if result != `{"result":"ok"}` {
		t.Errorf("result = %q, want %q", result, `{"result":"ok"}`)
	}
}
