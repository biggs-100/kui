package orchestration

import (
	"sync"
	"testing"
)

// ─── Task 3.11: RED — Test dedup first call is not duplicate ───

func TestDedupFirstCall(t *testing.T) {
	dedup := NewLaunchDedup()

	if dedup.IsDuplicate("explore", "read all files") {
		t.Error("first call should not be duplicate")
	}
}

// ─── Task 3.11: RED — Test dedup second call after MarkSeen is duplicate ───

func TestDedupSecondCall(t *testing.T) {
	dedup := NewLaunchDedup()

	dedup.MarkSeen("explore", "read all files", "result") // mark as seen

	if !dedup.IsDuplicate("explore", "read all files") {
		t.Error("second call after MarkSeen should be duplicate")
	}
}

// ─── Task 3.11: RED — Test dedup reset clears cache ───

func TestDedupReset(t *testing.T) {
	dedup := NewLaunchDedup()

	dedup.MarkSeen("explore", "read all files", "result") // mark as seen
	dedup.Reset()

	if dedup.IsDuplicate("explore", "read all files") {
		t.Error("after reset, same call should not be duplicate")
	}
}

// ─── Task 3.11: RED — Test dedup different tasks are not duplicates ───

func TestDedupDifferentTasks(t *testing.T) {
	dedup := NewLaunchDedup()

	dedup.MarkSeen("explore", "read all files", "result")

	if dedup.IsDuplicate("explore", "read test files") {
		t.Error("different tasks should not be duplicate")
	}
}

// ─── Task 3.11: RED — Test dedup different agents are not duplicates ───

func TestDedupDifferentAgents(t *testing.T) {
	dedup := NewLaunchDedup()

	dedup.MarkSeen("explore", "read all files", "result")

	if dedup.IsDuplicate("worker", "read all files") {
		t.Error("different agents with same task should not be duplicate")
	}
}

// ─── Task 3.11: RED — Test dedup concurrent access ───

func TestDedupConcurrent(t *testing.T) {
	dedup := NewLaunchDedup()
	var wg sync.WaitGroup

	// 100 goroutines marking the same task
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dedup.MarkSeen("explore", "concurrent task", "result")
		}()
	}
	wg.Wait()

	// After all goroutines, it should be marked as duplicate
	if !dedup.IsDuplicate("explore", "concurrent task") {
		t.Error("after concurrent calls, should be duplicate")
	}
}

// ─── Task 3.11: RED — Test dedup caches result ───

func TestDedupCachesResult(t *testing.T) {
	dedup := NewLaunchDedup()

	dedup.MarkSeen("explore", "task1", "cached output")

	cached, ok := dedup.GetCached("explore", "task1")
	if !ok {
		t.Error("expected cached result to exist")
	}
	if cached != "cached output" {
		t.Errorf("expected cached output 'cached output', got %q", cached)
	}
}

// ─── Task 3.11: RED — Test dedup GetCached for unseen task ───

func TestDedupGetCachedUnseen(t *testing.T) {
	dedup := NewLaunchDedup()

	_, ok := dedup.GetCached("explore", "unseen")
	if ok {
		t.Error("GetCached for unseen task should return false")
	}
}

// ─── Edge case: Empty agent name ───

func TestDedupEmptyAgent(t *testing.T) {
	dedup := NewLaunchDedup()

	if dedup.IsDuplicate("", "some task") {
		t.Error("first call with empty agent should not be duplicate")
	}
	dedup.MarkSeen("", "some task", "result")
	if !dedup.IsDuplicate("", "some task") {
		t.Error("second call with empty agent should be duplicate")
	}
}

// ─── Edge case: Empty task ───

func TestDedupEmptyTask(t *testing.T) {
	dedup := NewLaunchDedup()

	if dedup.IsDuplicate("explore", "") {
		t.Error("first call with empty task should not be duplicate")
	}
	dedup.MarkSeen("explore", "", "result")
	if !dedup.IsDuplicate("explore", "") {
		t.Error("second call with empty task should be duplicate")
	}
}
