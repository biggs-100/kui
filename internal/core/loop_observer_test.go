package core

import (
	"context"
	"sync"
	"testing"
)

// loopSpyObserver records every event with thread safety so loop tests can
// assert event delivery across goroutine boundaries.
type loopSpyObserver struct {
	mu          sync.Mutex
	turnStarts  int
	turnEnds    int
	toolCalls   []ToolCall
	toolResults []struct {
		callID string
		result string
	}
}

func (s *loopSpyObserver) OnTurnStart() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.turnStarts++
}

func (s *loopSpyObserver) OnTurnEnd() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.turnEnds++
}

func (s *loopSpyObserver) OnToolCall(call ToolCall) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.toolCalls = append(s.toolCalls, call)
}

func (s *loopSpyObserver) OnToolResult(callID, result string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.toolResults = append(s.toolResults, struct {
		callID string
		result string
	}{callID, result})
}

func (s *loopSpyObserver) snapshot() (turnStarts, turnEnds int, calls []ToolCall, results []struct {
	callID string
	result string
}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.turnStarts, s.turnEnds,
		append([]ToolCall(nil), s.toolCalls...),
		append([]struct {
			callID string
			result string
		}(nil), s.toolResults...)
}

func TestLoopWithObserverGetsToolEvents(t *testing.T) {
	// REQ-LOOP-7, "Tool events published": a loop with an observer attached
	// must deliver OnToolCall and OnToolResult for each tool execution.
	readFile := &fakeTool{name: "read_file", result: "contents"}
	registry := NewRegistry()
	if err := registry.Register(readFile); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	provider := &fakeProvider{responses: [][]Message{
		{toolCall("call-1", "read_file", `{"path":"a"}`)},
		{{Role: RoleAssistant, Content: "done"}},
	}}
	spy := &loopSpyObserver{}
	agent := &Agent{
		Provider: provider, Tools: registry, MaxIterations: 5,
		Observer: spy,
	}

	answer, err := agent.Run(context.Background(), "read a")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "done" {
		t.Errorf("answer = %q, want %q", answer, "done")
	}

	_, _, calls, results := spy.snapshot()
	if len(calls) != 1 {
		t.Fatalf("toolCalls = %d, want 1", len(calls))
	}
	if calls[0].ID != "call-1" || calls[0].Name != "read_file" {
		t.Errorf("toolCall = %+v, want ID=c1 Name=read_file", calls[0])
	}
	if len(results) != 1 {
		t.Fatalf("toolResults = %d, want 1", len(results))
	}
	if results[0].callID != "call-1" || results[0].result != "contents" {
		t.Errorf("toolResult = %+v, want callID=c1 result=contents", results[0])
	}
}

func TestLoopWithObserverGetsTurnEvents(t *testing.T) {
	// REQ-LOOP-7, "Turn events published": a loop with an observer must
	// deliver OnTurnStart and OnTurnEnd for each turn.
	readFile := &fakeTool{name: "read_file", result: "ok"}
	registry := NewRegistry()
	if err := registry.Register(readFile); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	// Two turns: turn 1 calls a tool, turn 2 returns a direct answer.
	provider := &fakeProvider{responses: [][]Message{
		{toolCall("call-1", "read_file", `{}`)},
		{{Role: RoleAssistant, Content: "done"}},
	}}
	spy := &loopSpyObserver{}
	agent := &Agent{
		Provider: provider, Tools: registry, MaxIterations: 5,
		Observer: spy,
	}

	_, err := agent.Run(context.Background(), "start")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	starts, ends, _, _ := spy.snapshot()
	// Two provider calls = two turns.
	if starts != 2 {
		t.Errorf("turnStarts = %d, want 2", starts)
	}
	if ends != 2 {
		t.Errorf("turnEnds = %d, want 2", ends)
	}
}

func TestLoopWithNilObserverUnchanged(t *testing.T) {
	// REQ-LOOP-7, "Nil observer is a no-op": a loop with nil observer must
	// behave identically to the existing tests.
	provider := &fakeProvider{responses: [][]Message{
		{{Role: RoleAssistant, Content: "hello there"}},
	}}
	agent := &Agent{Provider: provider, Tools: NewRegistry(), MaxIterations: 5}

	answer, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "hello there" {
		t.Errorf("answer = %q, want %q", answer, "hello there")
	}
	if provider.calls != 1 {
		t.Errorf("provider called %d times, want 1", provider.calls)
	}
}

func TestLoopWithObserverPanicDoesNotCrash(t *testing.T) {
	// REQ-LOOP-7, "Observer failure is contained": a panicking observer
	// must not crash the loop.
	provider := &fakeProvider{responses: [][]Message{
		{{Role: RoleAssistant, Content: "answer"}},
	}}
	agent := &Agent{
		Provider: provider, Tools: NewRegistry(), MaxIterations: 5,
		Observer: &panicObserver{},
	}

	answer, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "answer" {
		t.Errorf("answer = %q, want %q", answer, "answer")
	}
}
