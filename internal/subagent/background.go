package subagent

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	// MaxConcurrentBackground is the maximum number of concurrent background tasks.
	MaxConcurrentBackground = 2
)

// BackgroundTask represents a running background sub-agent.
type BackgroundTask struct {
	ID        string
	Task      string
	StartedAt time.Time
	Cancel    context.CancelFunc
	Done      chan struct{}
	Output    string
	Error     error
}

// BackgroundManager manages concurrent background sub-agent executions.
type BackgroundManager struct {
	mu       sync.Mutex
	tasks    map[string]*BackgroundTask
	maxConc  int
	onChange func() // called when task count changes
}

// NewBackgroundManager creates a manager with the given concurrency limit.
func NewBackgroundManager(maxConcurrent int) *BackgroundManager {
	return &BackgroundManager{
		tasks:   make(map[string]*BackgroundTask),
		maxConc: maxConcurrent,
	}
}

// SetOnChange sets a callback that fires when the task count changes.
func (m *BackgroundManager) SetOnChange(fn func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onChange = fn
}

// CanLaunch reports whether a new background task can be launched.
func (m *BackgroundManager) CanLaunch() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.tasks) < m.maxConc
}

// ActiveCount returns the number of active background tasks.
func (m *BackgroundManager) ActiveCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.tasks)
}

// Launch starts a background task. Returns an error if at capacity.
// The task runs fn in a goroutine and tracks its lifecycle.
func (m *BackgroundManager) Launch(id string, task string, fn func(ctx context.Context) (string, error)) error {
	m.mu.Lock()
	if len(m.tasks) >= m.maxConc {
		m.mu.Unlock()
		return fmt.Errorf("at capacity (%d/%d background tasks)", len(m.tasks), m.maxConc)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t := &BackgroundTask{
		ID:        id,
		Task:      task,
		StartedAt: time.Now(),
		Cancel:    cancel,
		Done:      make(chan struct{}),
	}
	m.tasks[id] = t
	m.mu.Unlock()

	// Fire change callback.
	if m.onChange != nil {
		m.onChange()
	}

	// Run in goroutine.
	go func() {
		defer close(t.Done)
		defer func() {
			m.mu.Lock()
			delete(m.tasks, id)
			m.mu.Unlock()
			if m.onChange != nil {
				m.onChange()
			}
		}()

		output, err := fn(ctx)
		t.Output = output
		t.Error = err
	}()

	return nil
}

// Cancel stops a background task by ID.
func (m *BackgroundManager) Cancel(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[id]
	if !ok {
		return false
	}
	t.Cancel()
	return true
}

// CancelAll stops all running background tasks.
func (m *BackgroundManager) CancelAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.tasks {
		t.Cancel()
	}
}

// List returns all active background tasks.
func (m *BackgroundManager) List() []BackgroundTask {
	m.mu.Lock()
	defer m.mu.Unlock()
	tasks := make([]BackgroundTask, 0, len(m.tasks))
	for _, t := range m.tasks {
		tasks = append(tasks, *t)
	}
	return tasks
}

// Wait blocks until all background tasks complete or the context is cancelled.
func (m *BackgroundManager) Wait(ctx context.Context) error {
	for {
		m.mu.Lock()
		count := len(m.tasks)
		if count == 0 {
			m.mu.Unlock()
			return nil
		}
		// Wait for any task to finish.
		var doneCh <-chan struct{}
		for _, t := range m.tasks {
			doneCh = t.Done
			break
		}
		m.mu.Unlock()

		select {
		case <-doneCh:
			// Task finished, check again.
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
