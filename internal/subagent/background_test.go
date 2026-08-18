package subagent

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBackgroundManagerCanLaunch(t *testing.T) {
	m := NewBackgroundManager(2)

	if !m.CanLaunch() {
		t.Error("CanLaunch() = false, want true (empty)")
	}

	// Fill capacity.
	m.Launch("1", "task1", func(ctx context.Context) (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "done", nil
	})
	m.Launch("2", "task2", func(ctx context.Context) (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "done", nil
	})

	if m.CanLaunch() {
		t.Error("CanLaunch() = true, want false (at capacity)")
	}
}

func TestBackgroundManagerRejectsOverCapacity(t *testing.T) {
	m := NewBackgroundManager(1)

	m.Launch("1", "task1", func(ctx context.Context) (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "done", nil
	})

	err := m.Launch("2", "task2", func(ctx context.Context) (string, error) {
		return "done", nil
	})
	if err == nil {
		t.Error("Launch() error = nil, want error (at capacity)")
	}
}

func TestBackgroundManagerActiveCount(t *testing.T) {
	m := NewBackgroundManager(2)

	if m.ActiveCount() != 0 {
		t.Errorf("ActiveCount() = %d, want 0", m.ActiveCount())
	}

	m.Launch("1", "task1", func(ctx context.Context) (string, error) {
		time.Sleep(50 * time.Millisecond)
		return "done", nil
	})

	// Wait for task to complete.
	time.Sleep(100 * time.Millisecond)

	if m.ActiveCount() != 0 {
		t.Errorf("ActiveCount() = %d after completion, want 0", m.ActiveCount())
	}
}

func TestBackgroundManagerCancel(t *testing.T) {
	m := NewBackgroundManager(2)

	m.Launch("1", "task1", func(ctx context.Context) (string, error) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(1 * time.Second):
			return "done", nil
		}
	})

	if !m.Cancel("1") {
		t.Error("Cancel() = false, want true")
	}

	// Should complete quickly after cancel.
	time.Sleep(50 * time.Millisecond)
	if m.ActiveCount() != 0 {
		t.Errorf("ActiveCount() = %d after cancel, want 0", m.ActiveCount())
	}
}

func TestBackgroundManagerList(t *testing.T) {
	m := NewBackgroundManager(2)

	m.Launch("1", "task1", func(ctx context.Context) (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "done", nil
	})

	tasks := m.List()
	if len(tasks) != 1 {
		t.Errorf("List() returned %d tasks, want 1", len(tasks))
	}
	if tasks[0].ID != "1" {
		t.Errorf("List()[0].ID = %q, want %q", tasks[0].ID, "1")
	}
}

func TestBackgroundManagerOnChange(t *testing.T) {
	m := NewBackgroundManager(2)

	var count atomic.Int32
	m.SetOnChange(func() {
		count.Add(1)
	})

	m.Launch("1", "task1", func(ctx context.Context) (string, error) {
		time.Sleep(50 * time.Millisecond)
		return "done", nil
	})

	// Wait for task to complete and callback to fire.
	time.Sleep(100 * time.Millisecond)

	if count.Load() < 2 {
		t.Errorf("onChange called %d times, want >= 2 (launch + complete)", count.Load())
	}
}

func TestBackgroundManagerConcurrent(t *testing.T) {
	m := NewBackgroundManager(2)

	var started sync.WaitGroup
	started.Add(2)

	m.Launch("1", "task1", func(ctx context.Context) (string, error) {
		started.Done()
		time.Sleep(50 * time.Millisecond)
		return "done", nil
	})
	m.Launch("2", "task2", func(ctx context.Context) (string, error) {
		started.Done()
		time.Sleep(50 * time.Millisecond)
		return "done", nil
	})

	// Wait for both to start.
	started.Wait()

	tasks := m.List()
	if len(tasks) != 2 {
		t.Errorf("List() returned %d tasks during execution, want 2", len(tasks))
	}
}

func TestBackgroundManagerWait(t *testing.T) {
	m := NewBackgroundManager(2)

	m.Launch("1", "task1", func(ctx context.Context) (string, error) {
		time.Sleep(50 * time.Millisecond)
		return "done", nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := m.Wait(ctx)
	if err != nil {
		t.Errorf("Wait() error = %v", err)
	}

	if m.ActiveCount() != 0 {
		t.Errorf("ActiveCount() = %d after Wait, want 0", m.ActiveCount())
	}
}
