package core

import "testing"

// mockStreamingObserver implements StreamingObserver for testing.
type mockStreamingObserver struct {
	deltas []string
}

func (m *mockStreamingObserver) OnTurnStart() {}

func (m *mockStreamingObserver) OnTurnEnd() {}

func (m *mockStreamingObserver) OnToolCall(call ToolCall) {}

func (m *mockStreamingObserver) OnToolResult(callID, result string) {}

func (m *mockStreamingObserver) OnTextDelta(delta string) {
	m.deltas = append(m.deltas, delta)
}

// mockNonStreamingObserver implements only Observer (no OnTextDelta).
type mockNonStreamingObserver struct{}

func (m mockNonStreamingObserver) OnTurnStart()                 {}
func (m mockNonStreamingObserver) OnTurnEnd()                   {}
func (m mockNonStreamingObserver) OnToolCall(call ToolCall)     {}
func (m mockNonStreamingObserver) OnToolResult(callID, result string) {}

// TestStreamingObserverSatisfaction verifies that a type implementing
// Observer + OnTextDelta satisfies StreamingObserver.
func TestStreamingObserverSatisfaction(t *testing.T) {
	var _ StreamingObserver = &mockStreamingObserver{}
}

// TestNonStreamingObserverDoesNotSatisfyStreamingObserver verifies that
// a type implementing only Observer does not satisfy StreamingObserver.
func TestNonStreamingObserverDoesNotSatisfyStreamingObserver(t *testing.T) {
	var _ Observer = mockNonStreamingObserver{}

	var o Observer = mockNonStreamingObserver{}
	if _, ok := o.(StreamingObserver); ok {
		t.Error("non-streaming observer should not satisfy StreamingObserver")
	}
}

// TestStreamingObserverTypeAssertion verifies type assertion works correctly.
func TestStreamingObserverTypeAssertion(t *testing.T) {
	// Concrete streaming observer
	so := &mockStreamingObserver{}
	if _, ok := Observer(so).(StreamingObserver); !ok {
		t.Error("mockStreamingObserver should satisfy StreamingObserver")
	}

	// Concrete non-streaming observer
	nso := mockNonStreamingObserver{}
	if _, ok := Observer(nso).(StreamingObserver); ok {
		t.Error("mockNonStreamingObserver should not satisfy StreamingObserver")
	}
}

// TestEmitTextDeltaNilObserver verifies that emitTextDelta with nil observer
// does not panic.
func TestEmitTextDeltaNilObserver(t *testing.T) {
	emitTextDelta(nil, "hello")
	// No panic = pass
}

// TestEmitTextDeltaNonNilObserver verifies that emitTextDelta calls OnTextDelta
// on a non-nil observer.
func TestEmitTextDeltaNonNilObserver(t *testing.T) {
	obs := &mockStreamingObserver{}
	emitTextDelta(obs, "hello")
	emitTextDelta(obs, " world")

	if len(obs.deltas) != 2 {
		t.Fatalf("expected 2 deltas, got %d", len(obs.deltas))
	}
	if obs.deltas[0] != "hello" {
		t.Errorf("delta[0] = %q, want %q", obs.deltas[0], "hello")
	}
	if obs.deltas[1] != " world" {
		t.Errorf("delta[1] = %q, want %q", obs.deltas[1], " world")
	}
}

// TestEmitTextDeltaEmptyDelta verifies that emitTextDelta delivers empty deltas.
func TestEmitTextDeltaEmptyDelta(t *testing.T) {
	obs := &mockStreamingObserver{}
	emitTextDelta(obs, "")

	if len(obs.deltas) != 1 {
		t.Fatalf("expected 1 delta, got %d", len(obs.deltas))
	}
	if obs.deltas[0] != "" {
		t.Errorf("delta[0] = %q, want empty", obs.deltas[0])
	}
}

// TestEmitTextDeltaPanicRecovery verifies that a panicking observer's
// panic is recovered.
func TestEmitTextDeltaPanicRecovery(t *testing.T) {
	obs := &panicDeltaObserver{}
	// Should not panic
	emitTextDelta(obs, "trigger")
}

// panicDeltaObserver panics on OnTextDelta.
type panicDeltaObserver struct{}

func (p *panicDeltaObserver) OnTurnStart()                           {}
func (p *panicDeltaObserver) OnTurnEnd()                             {}
func (p *panicDeltaObserver) OnToolCall(call ToolCall)               {}
func (p *panicDeltaObserver) OnToolResult(callID, result string)     {}
func (p *panicDeltaObserver) OnTextDelta(delta string)               { panic("observer panic") }