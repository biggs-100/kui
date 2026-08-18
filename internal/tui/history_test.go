package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestHistoryCreate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.jsonl")

	h, err := NewHistory(path)
	if err != nil {
		t.Fatalf("NewHistory returned error: %v", err)
	}
	if h == nil {
		t.Fatal("NewHistory returned nil history")
	}
}

func TestHistoryAppend(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.jsonl")

	h, _ := NewHistory(path)
	h.Append("first prompt")
	h.Append("second prompt")

	got := h.Previous()
	if got != "second prompt" {
		t.Errorf("Previous() = %q, want %q", got, "second prompt")
	}
}

func TestHistoryAppendDedup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.jsonl")

	h, _ := NewHistory(path)
	h.Append("same prompt")
	h.Append("same prompt")
	h.Append("same prompt")

	got := h.Previous()
	if got != "same prompt" {
		t.Errorf("Previous() = %q, want %q", got, "same prompt")
	}

	got = h.Previous()
	if got != "same prompt" {
		t.Errorf("Previous() after dedup = %q, want %q (only one entry expected)", got, "same prompt")
	}

	// Should be at the end now
	got = h.Previous()
	if got != "" {
		t.Errorf("Previous() past start = %q, want empty", got)
	}
}

func TestHistoryNav(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.jsonl")

	h, _ := NewHistory(path)
	h.Append("prompt 1")
	h.Append("prompt 2")
	h.Append("prompt 3")

	// Previous (up arrow)
	got := h.Previous()
	if got != "prompt 3" {
		t.Errorf("Previous() = %q, want %q", got, "prompt 3")
	}
	got = h.Previous()
	if got != "prompt 2" {
		t.Errorf("Previous() = %q, want %q", got, "prompt 2")
	}
	got = h.Previous()
	if got != "prompt 1" {
		t.Errorf("Previous() = %q, want %q", got, "prompt 1")
	}

	// Next (down arrow)
	got = h.Next()
	if got != "prompt 2" {
		t.Errorf("Next() = %q, want %q", got, "prompt 2")
	}
	got = h.Next()
	if got != "prompt 3" {
		t.Errorf("Next() = %q, want %q", got, "prompt 3")
	}
	got = h.Next()
	if got != "" {
		t.Errorf("Next() past end = %q, want empty", got)
	}

	// Reset should go back to end
	h.Reset()
	got = h.Previous()
	if got != "prompt 3" {
		t.Errorf("Reset + Previous() = %q, want %q", got, "prompt 3")
	}
}

func TestHistoryLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.jsonl")

	h, _ := NewHistory(path)

	// Add 55 unique entries
	for i := 0; i < 55; i++ {
		h.Append(fmt.Sprintf("prompt-%d", i))
	}

	// Read the file and count lines
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}
	if len(data) > 0 && data[len(data)-1] != '\n' {
		lines++
	}

	if lines > 50 {
		t.Errorf("history has %d entries, want at most 50", lines)
	}

	// Navigate to oldest entry — it should be "prompt-5" (entries 5-54 kept)
	h.Reset()
	for i := 0; i < 49; i++ {
		h.Previous()
	}
	got := h.Previous()
	if got != "prompt-5" {
		t.Errorf("oldest entry = %q, want %q", got, "prompt-5")
	}
}

func TestHistoryPersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.jsonl")

	// Write some history
	h1, _ := NewHistory(path)
	h1.Append("first")
	h1.Append("second")

	// Create a new history from the same file
	h2, err := NewHistory(path)
	if err != nil {
		t.Fatalf("NewHistory (reload) returned error: %v", err)
	}

	got := h2.Previous()
	if got != "second" {
		t.Errorf("reloaded Previous() = %q, want %q", got, "second")
	}
	got = h2.Previous()
	if got != "first" {
		t.Errorf("reloaded Previous() = %q, want %q", got, "first")
	}
}
