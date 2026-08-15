package agent

import (
	"sync"

	"github.com/biggs-100/kui/internal/core"
)

// PendingMessageQueue is the concrete mutex-backed core.PendingQueue (D19,
// REQ-QUEUE-1..3). Drain releases messages according to the queue's mode: all
// drains the entire queue in order, one-at-a-time drains exactly one message
// per call. It is safe for concurrent use.
type PendingMessageQueue struct {
	mu       sync.Mutex
	mode     core.QueueMode
	messages []core.PendingMessage
}

// NewPendingMessageQueue creates an empty queue operating in the given mode.
func NewPendingMessageQueue(mode core.QueueMode) *PendingMessageQueue {
	return &PendingMessageQueue{mode: mode}
}

// Enqueue appends a message to the queue.
func (q *PendingMessageQueue) Enqueue(message core.PendingMessage) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.messages = append(q.messages, message)
}

// Drain implements core.PendingQueue (D19, REQ-QUEUE-1). In QueueModeAll it
// releases the whole queue in order and empties it; in QueueModeOneAtATime it
// releases exactly the first message. Released messages are removed from the
// queue.
func (q *PendingMessageQueue) Drain() []core.PendingMessage {
	q.mu.Lock()
	defer q.mu.Unlock()
	switch q.mode {
	case core.QueueModeOneAtATime:
		if len(q.messages) == 0 {
			return nil
		}
		message := q.messages[0]
		q.messages = q.messages[1:]
		return []core.PendingMessage{message}
	default:
		drained := q.messages
		q.messages = nil
		return drained
	}
}
