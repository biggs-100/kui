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

// --- Phase 2: Cache-Aware Compaction Tests ---

func TestCompactorNeedsCompactionExcludesProtected(t *testing.T) {
	provider := &fakeCompactionProvider{summary: "summary"}
	// Budget is 100 tokens. Protected system content is ~100 tokens alone,
	// but since protected tokens are excluded, NeedsCompaction should only
	// count compactable tokens (user messages ~10 tokens) → false.
	c := NewCompactor(provider, WithMaxTokens(100), WithKeepTokens(20))

	messages := []Message{
		{Role: RoleSystem, Content: strings.Repeat("x", 400)}, // ~100 tokens, protected
		{Role: RoleUser, Content: "hello"},                    // ~1 token, compactable
		{Role: RoleAssistant, Content: "hi"},                  // ~1 token, compactable
	}

	// The system message is protected and should NOT count toward the budget.
	// Total compactable tokens: ~2, well under budget of 100.
	if c.NeedsCompaction(messages) {
		t.Error("NeedsCompaction() = true with protected system message excluded from budget, want false")
	}
}

func TestCompactorNeedsCompactionCountsCompactableOnly(t *testing.T) {
	provider := &fakeCompactionProvider{summary: "summary"}
	// Budget is 100 tokens. System message is ~300 tokens but protected.
	// Compactable messages total ~40 tokens → under budget.
	// Currently (without fix) ALL tokens count → ~340 > 100 → triggers.
	// After fix: only compactable ~40 < 100 → does NOT trigger.
	c := NewCompactor(provider, WithMaxTokens(100), WithKeepTokens(10))

	messages := []Message{
		{Role: RoleSystem, Content: strings.Repeat("x", 1200)},  // protected, ~300 tokens
		{Role: RoleUser, Content: strings.Repeat("a", 80)},      // compactable, ~20 tokens
		{Role: RoleAssistant, Content: strings.Repeat("b", 80)}, // compactable, ~20 tokens
	}

	// Total compactable tokens: ~40, under budget of 100.
	// Without the fix, protected tokens (~300) push total over budget.
	if c.NeedsCompaction(messages) {
		t.Error("NeedsCompaction() = true with protected excluded, want false (compactable only ~40 < 100)")
	}
}

func TestCompactorCompactPreservesProtectedPrefix(t *testing.T) {
	provider := &fakeCompactionProvider{summary: "User discussed design."}
	// Small budget forces compaction. System message is protected.
	c := NewCompactor(provider, WithMaxTokens(50), WithKeepTokens(10))

	messages := []Message{
		{Role: RoleSystem, Content: "You are a helpful assistant."},
		{Role: RoleUser, Content: strings.Repeat("a", 200)},
		{Role: RoleAssistant, Content: strings.Repeat("b", 200)},
		{Role: RoleUser, Content: "final question"},
	}

	result, err := c.Compact(context.Background(), messages)
	if err != nil {
		t.Fatalf("Compact() error = %v", err)
	}

	// Output must be: [protected] → [summary] → [tail]
	// Position 0 must be the protected system message.
	if len(result) < 2 {
		t.Fatalf("Compact() returned %d messages, want >= 2", len(result))
	}
	if result[0].Role != RoleSystem {
		t.Errorf("result[0].Role = %q, want %q (protected prefix)", result[0].Role, RoleSystem)
	}
	if result[0].Content != "You are a helpful assistant." {
		t.Errorf("result[0].Content = %q, want system prompt preserved", result[0].Content)
	}

	// The summary should follow the protected messages.
	if !strings.HasPrefix(result[1].Content, CompactionMarker) {
		t.Errorf("result[1] is not a compaction marker, got %q", result[1].Content)
	}
}

func TestCompactorCompactAllProtectedNoCompaction(t *testing.T) {
	provider := &fakeCompactionProvider{summary: "should not be called"}
	// Even with tiny budget, if ALL messages are protected, no compaction runs.
	c := NewCompactor(provider, WithMaxTokens(1), WithKeepTokens(1))

	messages := []Message{
		{Role: RoleSystem, Content: "system prompt"},
		{Role: RoleSystem, Content: strings.Repeat("x", 1000)}, // also system
	}

	result, err := c.Compact(context.Background(), messages)
	if err != nil {
		t.Fatalf("Compact() error = %v", err)
	}

	// All messages are protected → original messages returned unchanged.
	if len(result) != len(messages) {
		t.Errorf("Compact() returned %d messages, want %d (unchanged)", len(result), len(messages))
	}
	for i, m := range result {
		if m.Content != messages[i].Content {
			t.Errorf("result[%d].Content changed, compacted when all protected", i)
		}
	}
}

func TestCompactorCompactProtectedPrefixOrdering(t *testing.T) {
	provider := &fakeCompactionProvider{summary: "compacted summary"}
	c := NewCompactor(provider, WithMaxTokens(50), WithKeepTokens(10))

	// Multiple protected messages: system + profile marker.
	messages := []Message{
		{Role: RoleSystem, Content: "system prompt"},
		{Role: RoleUser, Content: "Profile switched to coding"},
		{Role: RoleUser, Content: strings.Repeat("a", 200)},
		{Role: RoleAssistant, Content: strings.Repeat("b", 200)},
		{Role: RoleUser, Content: "question"},
	}

	result, err := c.Compact(context.Background(), messages)
	if err != nil {
		t.Fatalf("Compact() error = %v", err)
	}

	// Both protected messages should be at the prefix in original order.
	if len(result) < 3 {
		t.Fatalf("Compact() returned %d messages, want >= 3", len(result))
	}
	if result[0].Role != RoleSystem || result[0].Content != "system prompt" {
		t.Errorf("result[0] = %q/%q, want system/system prompt", result[0].Role, result[0].Content)
	}
	if !strings.Contains(result[1].Content, "Profile switched") {
		t.Errorf("result[1].Content = %q, want profile marker", result[1].Content)
	}
	// Summary follows the protected messages.
	if !strings.HasPrefix(result[2].Content, CompactionMarker) {
		t.Errorf("result[2] is not compaction marker, got %q", result[2].Content)
	}
}

func TestCompactorCompactMultipleSystemMessagesPrefixOrder(t *testing.T) {
	// Multiple system messages from profile switches all appear at prefix
	// in insertion order (REQ-CAC-02, REQ-CAC-05).
	provider := &fakeCompactionProvider{summary: "summary"}
	c := NewCompactor(provider, WithMaxTokens(50), WithKeepTokens(10))

	messages := []Message{
		{Role: RoleSystem, Content: "system prompt v1"},
		{Role: RoleUser, Content: strings.Repeat("a", 200)},
		{Role: RoleSystem, Content: "system prompt v2"}, // profile switch
		{Role: RoleUser, Content: strings.Repeat("b", 200)},
		{Role: RoleUser, Content: "question"},
	}

	result, err := c.Compact(context.Background(), messages)
	if err != nil {
		t.Fatalf("Compact() error = %v", err)
	}

	// Both system messages must be at prefix in insertion order.
	if len(result) < 3 {
		t.Fatalf("Compact() returned %d messages, want >= 3", len(result))
	}
	if result[0].Content != "system prompt v1" {
		t.Errorf("result[0].Content = %q, want first system prompt", result[0].Content)
	}
	if result[1].Content != "system prompt v2" {
		t.Errorf("result[1].Content = %q, want second system prompt", result[1].Content)
	}
}
