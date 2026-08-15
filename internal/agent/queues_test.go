package agent

import (
	"fmt"
	"sync"
	"testing"

	"github.com/biggs-100/kui/internal/core"
)

// contents extracts the Content field of drained messages for assertions.
func contents(messages []core.PendingMessage) []string {
	out := make([]string, 0, len(messages))
	for _, m := range messages {
		out = append(out, m.Content)
	}
	return out
}

func TestDrainAllMode(t *testing.T) {
	// REQ-QUEUE-1, drain all: the entire queue drains in order on one drain
	// and the queue empties.
	q := NewPendingMessageQueue(core.QueueModeAll)
	q.Enqueue(core.PendingMessage{Content: "one"})
	q.Enqueue(core.PendingMessage{Content: "two"})
	q.Enqueue(core.PendingMessage{Content: "three"})

	drained := q.Drain()
	if got, want := contents(drained), []string{"one", "two", "three"}; !equalStrings(got, want) {
		t.Errorf("Drain() = %v, want %v (all, in order)", got, want)
	}
	if got := q.Drain(); len(got) != 0 {
		t.Errorf("second Drain() = %v, want empty", got)
	}
}

func TestDrainOneAtATimeMode(t *testing.T) {
	// REQ-QUEUE-1, drain one per turn: exactly one message drains per call,
	// in order.
	q := NewPendingMessageQueue(core.QueueModeOneAtATime)
	q.Enqueue(core.PendingMessage{Content: "one"})
	q.Enqueue(core.PendingMessage{Content: "two"})
	q.Enqueue(core.PendingMessage{Content: "three"})

	for i, want := range []string{"one", "two", "three"} {
		drained := q.Drain()
		if got := contents(drained); len(got) != 1 || got[0] != want {
			t.Errorf("drain %d = %v, want exactly [%s]", i+1, got, want)
		}
	}
	if got := q.Drain(); len(got) != 0 {
		t.Errorf("drain after exhaustion = %v, want empty", got)
	}
}

func TestDrainEmptyQueue(t *testing.T) {
	// An empty queue drains nothing in either mode.
	all := NewPendingMessageQueue(core.QueueModeAll)
	if got := all.Drain(); len(got) != 0 {
		t.Errorf("empty all-mode Drain() = %v, want empty", got)
	}
	one := NewPendingMessageQueue(core.QueueModeOneAtATime)
	if got := one.Drain(); len(got) != 0 {
		t.Errorf("empty one-at-a-time Drain() = %v, want empty", got)
	}
}

func TestEnqueuePreservesOrderAcrossDrains(t *testing.T) {
	// FIFO: messages enqueued after a drain drain after the earlier ones,
	// preserving global order.
	q := NewPendingMessageQueue(core.QueueModeAll)
	q.Enqueue(core.PendingMessage{Content: "one"})
	if got := contents(q.Drain()); len(got) != 1 || got[0] != "one" {
		t.Fatalf("first Drain() = %v, want [one]", got)
	}
	q.Enqueue(core.PendingMessage{Content: "two"})
	q.Enqueue(core.PendingMessage{Content: "three"})
	if got, want := contents(q.Drain()), []string{"two", "three"}; !equalStrings(got, want) {
		t.Errorf("second Drain() = %v, want %v", got, want)
	}
}

func TestQueueConcurrentEnqueueDrain(t *testing.T) {
	// D19: the mutex makes the queue safe for concurrent Enqueue; every
	// enqueued message must appear exactly once after a drain. Run with
	// -race in verification to prove no data race.
	const goroutines = 8
	const perGoroutine = 50
	q := NewPendingMessageQueue(core.QueueModeAll)

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				q.Enqueue(core.PendingMessage{Content: fmt.Sprintf("g%d-%d", g, i)})
			}
		}(g)
	}
	wg.Wait()

	drained := q.Drain()
	if len(drained) != goroutines*perGoroutine {
		t.Fatalf("Drain() returned %d messages, want %d", len(drained), goroutines*perGoroutine)
	}
	seen := map[string]bool{}
	for _, m := range drained {
		if seen[m.Content] {
			t.Errorf("duplicate message %q", m.Content)
		}
		seen[m.Content] = true
	}
	for g := 0; g < goroutines; g++ {
		if !seen[fmt.Sprintf("g%d-%d", g, perGoroutine-1)] {
			t.Errorf("missing tail message for goroutine %d", g)
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
