package core

import (
	"testing"
)

func TestCacheAwareRequestBuilder(t *testing.T) {
	b := NewCacheAwareRequestBuilder()
	b.SetSystemPrompt("You are a helpful assistant.")
	b.SetHistory([]Message{
		{Role: RoleUser, Content: "hello"},
		{Role: RoleAssistant, Content: "hi there"},
	})
	b.SetCurrentTurn([]Message{
		{Role: RoleUser, Content: "what is 2+2?"},
	})

	messages := b.BuildCachePrefix()

	// System prompt should be first.
	if messages[0].Role != RoleSystem {
		t.Errorf("messages[0].Role = %q, want %q", messages[0].Role, RoleSystem)
	}

	// History should follow.
	if messages[1].Role != RoleUser {
		t.Errorf("messages[1].Role = %q, want %q", messages[1].Role, RoleUser)
	}

	// Current turn should be last.
	last := messages[len(messages)-1]
	if last.Content != "what is 2+2?" {
		t.Errorf("last message = %q, want %q", last.Content, "what is 2+2?")
	}
}

func TestEstimateCacheHit(t *testing.T) {
	b := NewCacheAwareRequestBuilder()
	b.SetSystemPrompt("You are a helpful assistant with very long instructions.") // ~15 tokens
	b.SetHistory([]Message{
		{Role: RoleUser, Content: "hello"},   // ~1 token
		{Role: RoleAssistant, Content: "hi"}, // ~1 token
		{Role: RoleUser, Content: "test"},    // ~1 token
	})
	b.SetCurrentTurn([]Message{
		{Role: RoleUser, Content: "question"}, // ~1 token
	})

	hit := b.EstimateCacheHit()
	if hit < 0.5 {
		t.Errorf("EstimateCacheHit() = %f, want > 0.5 (system prompt dominates)", hit)
	}
}

func TestSortMessagesForCache(t *testing.T) {
	messages := []Message{
		{Role: RoleUser, Content: "hello"},
		{Role: RoleAssistant, Content: "hi"},
		{Role: RoleSystem, Content: "system"},
		{Role: RoleTool, Content: "result"},
		{Role: RoleUser, Content: "question"},
	}

	sorted := SortMessagesForCache(messages)

	// System should be first.
	if sorted[0].Role != RoleSystem {
		t.Errorf("sorted[0].Role = %q, want %q", sorted[0].Role, RoleSystem)
	}

	// User messages should be last.
	lastUser := false
	for _, m := range sorted {
		if m.Role == RoleUser {
			lastUser = true
		}
		if lastUser && m.Role == RoleAssistant {
			t.Error("assistant after user in sorted order")
		}
	}
}

func TestIsCacheableRole(t *testing.T) {
	tests := []struct {
		role string
		want bool
	}{
		{RoleSystem, true},
		{RoleAssistant, true},
		{RoleUser, false},
		{RoleTool, false},
		{"unknown", false},
	}
	for _, tt := range tests {
		if got := IsCacheableRole(tt.role); got != tt.want {
			t.Errorf("IsCacheableRole(%q) = %v, want %v", tt.role, got, tt.want)
		}
	}
}

func TestCountCacheableTokens(t *testing.T) {
	messages := []Message{
		{Role: RoleSystem, Content: "system prompt"},         // ~4 tokens
		{Role: RoleUser, Content: "user message"},            // ~3 tokens (not cacheable)
		{Role: RoleAssistant, Content: "assistant response"}, // ~3 tokens (cacheable)
	}

	count := CountCacheableTokens(messages)
	if count < 5 {
		t.Errorf("CountCacheableTokens() = %d, want >= 5", count)
	}
}

func TestEstimateCacheSavings(t *testing.T) {
	messages := []Message{
		{Role: RoleSystem, Content: "very long system prompt with many words and instructions"}, // ~15 tokens
		{Role: RoleUser, Content: "short"}, // ~1 token (not cacheable)
	}

	savings := EstimateCacheSavings(messages)
	if savings < 0.8 {
		t.Errorf("EstimateCacheSavings() = %f, want >= 0.8 (system dominates)", savings)
	}
}

func TestBuildCacheKey(t *testing.T) {
	key := BuildCacheKey("coder", "gpt-4o", "openai")
	if key != "coder:gpt-4o:openai" {
		t.Errorf("BuildCacheKey() = %q, want %q", key, "coder:gpt-4o:openai")
	}

	// Empty parts should be handled.
	key = BuildCacheKey("", "", "")
	if key != "" {
		t.Errorf("BuildCacheKey(empty) = %q, want empty", key)
	}
}

// --- Phase 3: Builder Integration Tests ---

func TestBuildRequestAssemblesCacheOptimalOrder(t *testing.T) {
	b := NewCacheAwareRequestBuilder()

	protected := []Message{
		{Role: RoleSystem, Content: "system prompt"},
	}
	compacted := []Message{
		{Role: RoleUser, Content: "old question"},
		{Role: RoleAssistant, Content: "old answer"},
	}
	currentTurn := []Message{
		{Role: RoleUser, Content: "new question"},
	}

	result := b.BuildRequest(protected, compacted, currentTurn)

	// Must be: protected → compacted → currentTurn
	if len(result) != 4 {
		t.Fatalf("BuildRequest() returned %d messages, want 4", len(result))
	}
	if result[0].Role != RoleSystem || result[0].Content != "system prompt" {
		t.Errorf("result[0] = %q/%q, want system/system prompt", result[0].Role, result[0].Content)
	}
	if result[1].Content != "old question" {
		t.Errorf("result[1].Content = %q, want first compacted", result[1].Content)
	}
	if result[2].Content != "old answer" {
		t.Errorf("result[2].Content = %q, want second compacted", result[2].Content)
	}
	if result[3].Content != "new question" {
		t.Errorf("result[3].Content = %q, want current turn", result[3].Content)
	}
}

func TestBuildRequestEmptyCompacted(t *testing.T) {
	b := NewCacheAwareRequestBuilder()

	protected := []Message{
		{Role: RoleSystem, Content: "system prompt"},
	}
	currentTurn := []Message{
		{Role: RoleUser, Content: "question"},
	}

	result := b.BuildRequest(protected, nil, currentTurn)

	// Empty compacted → protected + currentTurn
	if len(result) != 2 {
		t.Fatalf("BuildRequest() returned %d messages, want 2", len(result))
	}
	if result[0].Content != "system prompt" {
		t.Errorf("result[0].Content = %q, want system prompt", result[0].Content)
	}
	if result[1].Content != "question" {
		t.Errorf("result[1].Content = %q, want question", result[1].Content)
	}
}

func TestBuildRequestEmptyProtected(t *testing.T) {
	b := NewCacheAwareRequestBuilder()

	compacted := []Message{
		{Role: RoleAssistant, Content: "summary"},
	}
	currentTurn := []Message{
		{Role: RoleUser, Content: "follow-up"},
	}

	result := b.BuildRequest(nil, compacted, currentTurn)

	// No protected → compacted + currentTurn
	if len(result) != 2 {
		t.Fatalf("BuildRequest() returned %d messages, want 2", len(result))
	}
	if result[0].Content != "summary" {
		t.Errorf("result[0].Content = %q, want summary", result[0].Content)
	}
	if result[1].Content != "follow-up" {
		t.Errorf("result[1].Content = %q, want follow-up", result[1].Content)
	}
}
