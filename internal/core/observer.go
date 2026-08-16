package core

// Observer is an optional port through which the loop publishes tool-call,
// tool-result, and turn events (REQ-LOOP-7). All methods use stdlib types
// only. When nil, the loop behaves identically to today — every emit call
// is a no-op.
type Observer interface {
	OnTurnStart()
	OnTurnEnd()
	OnToolCall(call ToolCall)
	OnToolResult(callID, result string)
}

// emitObserver calls the given function on the observer, recovering from any
// panic so that a misbehaving observer can never crash the loop (REQ-LOOP-7).
// When obs is nil the call is a no-op.
func emitObserver(obs Observer, fn func()) {
	if obs == nil {
		return
	}
	defer func() { recover() }()
	fn()
}

// emitTurnStart notifies the observer that a new turn is beginning.
func emitTurnStart(obs Observer) {
	emitObserver(obs, func() { obs.OnTurnStart() })
}

// emitTurnEnd notifies the observer that the current turn has completed.
func emitTurnEnd(obs Observer) {
	emitObserver(obs, func() { obs.OnTurnEnd() })
}

// emitToolCall notifies the observer that a tool is about to execute.
func emitToolCall(obs Observer, call ToolCall) {
	emitObserver(obs, func() { obs.OnToolCall(call) })
}

// emitToolResult notifies the observer that a tool has finished executing.
func emitToolResult(obs Observer, callID, result string) {
	emitObserver(obs, func() { obs.OnToolResult(callID, result) })
}
