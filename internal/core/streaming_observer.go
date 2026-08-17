package core

// StreamingObserver extends Observer with a text-delta method for real-time
// streaming events (REQ-OBS-STREAM-1). Callers detect this interface via
// type assertion to opt into streaming events.
type StreamingObserver interface {
	Observer
	OnTextDelta(delta string)
}

// emitTextDelta calls OnTextDelta on the given observer if it implements
// StreamingObserver. The call is nil-safe and panic-recovered (REQ-OBS-STREAM-2,
// REQ-OBS-STREAM-3).
func emitTextDelta(obs Observer, delta string) {
	if obs == nil {
		return
	}
	so, ok := obs.(StreamingObserver)
	if !ok {
		return
	}
	defer func() { recover() }()
	so.OnTextDelta(delta)
}