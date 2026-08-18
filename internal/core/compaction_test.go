package core

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// fakeCompactionProvider returns a fixed summary for compaction requests.
type fakeCompactionProvider struct {
	summary string
}

func (f *fakeCompactionProvider) Chat(_ context.Context, messages []Message, _ []Tool) ([]Message, error) {
	return []Message{{Role: RoleAssistant, Content: f.summary}}, nil
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"hi", 1},
		{"hello world", 3},
		{strings.Repeat("a", 100), 25},
	}
	for _, tt := range tests {
		got := estimateTokens(tt.input)
		if got != tt.want {
			t.Errorf("estimateTokens(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestCompactorNeedsCompaction(t *testing.T) {
	provider := &fakeCompactionProvider{summary: "summary"}
	c := NewCompactor(provider, WithMaxTokens(100), WithKeepTokens(20))

	// Small history — no compaction needed.
	small := []Message{
		{Role: RoleUser, Content: "hi"},
		{Role: RoleAssistant, Content: "hello"},
	}
	if c.NeedsCompaction(small) {
		t.Error("NeedsCompaction(small) = true, want false")
	}

	// Large history — compaction needed.
	big := []Message{
		{Role: RoleUser, Content: strings.Repeat("x", 500)},
	}
	if !c.NeedsCompaction(big) {
		t.Error("NeedsCompaction(big) = false, want true")
	}
}

func TestCompactorCompactWithinBudget(t *testing.T) {
	provider := &fakeCompactionProvider{summary: "summary"}
	c := NewCompactor(provider, WithMaxTokens(1000), WithKeepTokens(200))

	messages := []Message{
		{Role: RoleUser, Content: "hello"},
		{Role: RoleAssistant, Content: "hi"},
	}

	result, err := c.Compact(context.Background(), messages)
	if err != nil {
		t.Fatalf("Compact() error = %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Compact() returned %d messages, want 2 (unchanged)", len(result))
	}
}

func TestCompactorCompactSummarizes(t *testing.T) {
	provider := &fakeCompactionProvider{summary: "User discussed architecture."}
	c := NewCompactor(provider, WithMaxTokens(50), WithKeepTokens(10))

	// 4 messages totaling ~200 tokens — exceeds budget of 50.
	messages := []Message{
		{Role: RoleUser, Content: strings.Repeat("a", 100)},
		{Role: RoleAssistant, Content: strings.Repeat("b", 100)},
		{Role: RoleUser, Content: "what do you think?"},
		{Role: RoleAssistant, Content: "I agree."},
	}

	result, err := c.Compact(context.Background(), messages)
	if err != nil {
		t.Fatalf("Compact() error = %v", err)
	}

	// Should have compaction marker + tail messages.
	if len(result) < 2 {
		t.Fatalf("Compact() returned %d messages, want >= 2", len(result))
	}

	// First message should be the compaction marker.
	if result[0].Role != CompactionMessageRole {
		t.Errorf("result[0].Role = %q, want %q", result[0].Role, CompactionMessageRole)
	}
	if !strings.HasPrefix(result[0].Content, CompactionMarker) {
		t.Errorf("result[0].Content doesn't start with %q", CompactionMarker)
	}
	if !strings.Contains(result[0].Content, "User discussed architecture") {
		t.Errorf("result[0].Content doesn't contain summary")
	}
}

func TestCompactorCompactFallbackOnError(t *testing.T) {
	// Provider that returns empty — should fall back to truncation.
	provider := &fakeCompactionProvider{summary: ""}
	c := NewCompactor(provider, WithMaxTokens(50), WithKeepTokens(10))

	messages := []Message{
		{Role: RoleUser, Content: strings.Repeat("a", 100)},
		{Role: RoleAssistant, Content: strings.Repeat("b", 100)},
		{Role: RoleUser, Content: "final question"},
	}

	result, err := c.Compact(context.Background(), messages)
	if err != nil {
		t.Fatalf("Compact() error = %v", err)
	}

	// Should keep at least the tail messages.
	if len(result) < 1 {
		t.Fatal("Compact() returned 0 messages")
	}
}

func TestCompactorCompactWithToolCalls(t *testing.T) {
	provider := &fakeCompactionProvider{summary: "Tool was called."}
	c := NewCompactor(provider, WithMaxTokens(50), WithKeepTokens(10))

	args, _ := json.Marshal(map[string]string{"file": "main.go"})
	messages := []Message{
		{Role: RoleUser, Content: strings.Repeat("x", 200)},
		{Role: RoleAssistant, ToolCall: &ToolCall{ID: "1", Name: "read", Arguments: string(args)}},
		{Role: RoleTool, Content: strings.Repeat("file contents ", 10), ToolCallID: "1"},
		{Role: RoleAssistant, Content: "done"},
	}

	result, err := c.Compact(context.Background(), messages)
	if err != nil {
		t.Fatalf("Compact() error = %v", err)
	}

	// Should have compaction marker.
	if len(result) == 0 {
		t.Fatal("Compact() returned 0 messages")
	}
	if !strings.HasPrefix(result[0].Content, CompactionMarker) {
		t.Errorf("first message is not a compaction marker")
	}
}

func TestNewCompactorDefaults(t *testing.T) {
	provider := &fakeCompactionProvider{summary: "s"}
	c := NewCompactor(provider)

	if c.maxTokens != DefaultMaxInputTokens {
		t.Errorf("maxTokens = %d, want %d", c.maxTokens, DefaultMaxInputTokens)
	}
	if c.keepTokens != DefaultKeepTokens {
		t.Errorf("keepTokens = %d, want %d", c.keepTokens, DefaultKeepTokens)
	}
}
