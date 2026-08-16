package core

// QueueMode controls how many pending messages a queue releases per drain
// (D19). Core only consumes the PendingQueue port; the concrete mutex-backed
// queue implementing a mode lives outside the core (internal/agent, PR 4).
type QueueMode int

const (
	// QueueModeAll drains the entire queue in order on every drain.
	QueueModeAll QueueMode = iota
	// QueueModeOneAtATime drains exactly one message per drain.
	QueueModeOneAtATime
)

// PendingMessage is a message queued for injection between turns. Content is
// injected as a user message; SwitchProfile requests a profile change that
// applies between turns, never during a tool call (REQ-LOOP-5, D16).
type PendingMessage struct {
	Content       string
	SwitchProfile string
}

// PendingQueue is the port the loop uses to drain queued messages between
// turns (REQ-QUEUE-1/2, D19). Steering drains after each completed turn;
// FollowUp drains only when the loop would otherwise stop. Drain returns the
// released messages and removes them from the queue. Enqueue appends a
// message for injection between turns — the TUI controller uses this to
// queue profile switches (REQ-TUI-PROF-3).
type PendingQueue interface {
	Enqueue(message PendingMessage)
	Drain() []PendingMessage
}
