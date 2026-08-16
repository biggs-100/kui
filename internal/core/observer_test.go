package core

import (
	"sync"
	"testing"
)

// spyObserver records every call so tests can assert event delivery.
type spyObserver struct {
	mu          sync.Mutex
	turnStarts  int
	turnEnds    int
	toolCalls   []ToolCall
	toolResults []struct {
		callID string
		result string
	}
}

func (s *spyObserver) OnTurnStart() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.turnStarts++
}

func (s *spyObserver) OnTurnEnd() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.turnEnds++
}

func (s *spyObserver) OnToolCall(call ToolCall) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.toolCalls = append(s.toolCalls, call)
}

func (s *spyObserver) OnToolResult(callID, result string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.toolResults = append(s.toolResults, struct {
		callID string
		result string
	}{callID, result})
}

func (s *spyObserver) snapshot() (turnStarts, turnEnds int, calls []ToolCall, results []struct {
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

// panicObserver panics on every call — tests that emit recovers.
type panicObserver struct{}

func (p *panicObserver) OnTurnStart()              { panic("observer boom") }
func (p *panicObserver) OnTurnEnd()                { panic("observer boom") }
func (p *panicObserver) OnToolCall(_ ToolCall)     { panic("observer boom") }
func (p *panicObserver) OnToolResult(_, _ string)  { panic("observer boom") }

func TestEmitNilObserverNoOp(t *testing.T) {
	// REQ-LOOP-7, "Nil observer is a no-op": calling emit with a nil
	// observer must not panic or change behavior.
	emitObserver(nil, func() { t.Fatal("should not be called") })
	// Also test the typed helpers.
	emitTurnStart(nil)
	emitTurnEnd(nil)
	emitToolCall(nil, ToolCall{})
	emitToolResult(nil, "call-1", "ok")
}

func TestEmitTurnStartDelivered(t *testing.T) {
	spy := &spyObserver{}
	emitTurnStart(spy)
	starts, ends, _, _ := spy.snapshot()
	if starts != 1 {
		t.Errorf("turnStarts = %d, want 1", starts)
	}
	if ends != 0 {
		t.Errorf("turnEnds = %d, want 0", ends)
	}
}

func TestEmitTurnEndDelivered(t *testing.T) {
	spy := &spyObserver{}
	emitTurnEnd(spy)
	starts, ends, _, _ := spy.snapshot()
	if starts != 0 {
		t.Errorf("turnStarts = %d, want 0", starts)
	}
	if ends != 1 {
		t.Errorf("turnEnds = %d, want 1", ends)
	}
}

func TestEmitToolCallDelivered(t *testing.T) {
	spy := &spyObserver{}
	call := ToolCall{ID: "c1", Name: "read_file", Arguments: `{"path":"x"}`}
	emitToolCall(spy, call)
	_, _, calls, _ := spy.snapshot()
	if len(calls) != 1 {
		t.Fatalf("toolCalls = %d, want 1", len(calls))
	}
	if calls[0].ID != "c1" || calls[0].Name != "read_file" {
		t.Errorf("toolCall = %+v, want ID=c1 Name=read_file", calls[0])
	}
}

func TestEmitToolResultDelivered(t *testing.T) {
	spy := &spyObserver{}
	emitToolResult(spy, "c1", "file contents")
	_, _, _, results := spy.snapshot()
	if len(results) != 1 {
		t.Fatalf("toolResults = %d, want 1", len(results))
	}
	if results[0].callID != "c1" || results[0].result != "file contents" {
		t.Errorf("toolResult = %+v, want callID=c1 result=file contents", results[0])
	}
}

func TestEmitMultipleEvents(t *testing.T) {
	spy := &spyObserver{}
	emitTurnStart(spy)
	emitToolCall(spy, ToolCall{ID: "c1", Name: "bash"})
	emitToolResult(spy, "c1", "output")
	emitTurnEnd(spy)
	starts, ends, calls, results := spy.snapshot()
	if starts != 1 || ends != 1 {
		t.Errorf("turn counts = (%d, %d), want (1, 1)", starts, ends)
	}
	if len(calls) != 1 || len(results) != 1 {
		t.Errorf("tool counts = (%d, %d), want (1, 1)", len(calls), len(results))
	}
}

func TestEmitPanicRecoveryTurnStart(t *testing.T) {
	// REQ-LOOP-7, "Observer failure is contained": a panicking observer
	// must not crash the loop.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("emit panicked through recovery: %v", r)
		}
	}()
	emitTurnStart(&panicObserver{})
}

func TestEmitPanicRecoveryTurnEnd(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("emit panicked through recovery: %v", r)
		}
	}()
	emitTurnEnd(&panicObserver{})
}

func TestEmitPanicRecoveryToolCall(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("emit panicked through recovery: %v", r)
		}
	}()
	emitToolCall(&panicObserver{}, ToolCall{})
}

func TestEmitPanicRecoveryToolResult(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("emit panicked through recovery: %v", r)
		}
	}()
	emitToolResult(&panicObserver{}, "c1", "ok")
}

func TestEmitPanicRecoveryNilSafe(t *testing.T) {
	// Nil observer + panic recovery must not interfere.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("emit panicked through recovery: %v", r)
		}
	}()
	emitTurnStart(nil)
	emitTurnEnd(nil)
	emitToolCall(nil, ToolCall{})
	emitToolResult(nil, "c1", "ok")
}

func TestEmitObserverDirectly(t *testing.T) {
	// Direct emitObserver call — verify the panic-recovery wrapper works
	// end-to-end with a real panicking function.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("emitObserver panicked through recovery: %v", r)
		}
	}()
	emitObserver(&panicObserver{}, func() { panic("boom") })
}
