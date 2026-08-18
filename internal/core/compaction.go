package core

import (
	"context"
	"fmt"
	"strings"
)

const (
	// DefaultMaxInputTokens is the default context window budget for input
	// tokens. When estimated input exceeds this, compaction is triggered.
	DefaultMaxInputTokens = 120000

	// DefaultKeepTokens is the number of recent tokens to keep verbatim
	// after compaction. Older messages are summarized.
	DefaultKeepTokens = 8000

	// CompactionMessageRole is the role used for compaction summary messages.
	CompactionMessageRole = "user"

	// CompactionMarker is the prefix that identifies a compaction summary.
	CompactionMarker = "[session-compaction]"
)

// Compactor compresses conversation history when it exceeds the context window.
// It summarizes old messages via the provider and keeps recent messages verbatim.
type Compactor struct {
	provider    Provider
	maxTokens   int // max input tokens before compaction
	keepTokens  int // tokens to keep verbatim after compaction
	tokenEstFn  func(string) int // token estimation function
}

// CompactorOption configures the compactor.
type CompactorOption func(*Compactor)

// WithMaxTokens sets the maximum input token budget.
func WithMaxTokens(n int) CompactorOption {
	return func(c *Compactor) { c.maxTokens = n }
}

// WithKeepTokens sets the number of recent tokens to keep verbatim.
func WithKeepTokens(n int) CompactorOption {
	return func(c *Compactor) { c.keepTokens = n }
}

// NewCompactor creates a compactor that summarizes old messages via the provider.
func NewCompactor(provider Provider, opts ...CompactorOption) *Compactor {
	c := &Compactor{
		provider:   provider,
		maxTokens:  DefaultMaxInputTokens,
		keepTokens: DefaultKeepTokens,
		tokenEstFn: estimateTokens,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// NeedsCompaction reports whether the message history exceeds the token budget.
func (c *Compactor) NeedsCompaction(messages []Message) bool {
	total := 0
	for _, m := range messages {
		total += c.tokenEstFn(m.Content)
		if m.ToolCall != nil {
			total += c.tokenEstFn(m.ToolCall.Arguments)
		}
	}
	return total > c.maxTokens
}

// Compact compresses the message history by summarizing old messages and keeping
// recent messages verbatim. It returns the compacted message slice. If no
// compaction is needed (history is within budget), it returns the original
// messages unchanged.
//
// The compaction works by:
// 1. Estimating total tokens in the history
// 2. Splitting messages into head (old, to summarize) and tail (recent, keep)
// 3. Summarizing the head via a provider Chat call
// 4. Prepending the summary as a compaction marker message
func (c *Compactor) Compact(ctx context.Context, messages []Message) ([]Message, error) {
	if !c.NeedsCompaction(messages) {
		return messages, nil
	}

	// Find the split point: walk backward from the end, keeping messages
	// until we exceed keepTokens.
	keepCount := 0
	keepTokens := 0
	for i := len(messages) - 1; i >= 0; i-- {
		msgTokens := c.tokenEstFn(messages[i].Content)
		if messages[i].ToolCall != nil {
			msgTokens += c.tokenEstFn(messages[i].ToolCall.Arguments)
		}
		if keepTokens+msgTokens > c.keepTokens && keepCount > 0 {
			break
		}
		keepTokens += msgTokens
		keepCount++
	}

	if keepCount >= len(messages) {
		// All messages fit in the keep budget — no compaction needed.
		return messages, nil
	}

	head := messages[:len(messages)-keepCount]
	tail := messages[len(messages)-keepCount:]

	// Build a summary request: serialize head messages as text for the LLM.
	var sb strings.Builder
	sb.WriteString("Summarize the following conversation history concisely. ")
	sb.WriteString("Focus on: key decisions, current task, important context, ")
	sb.WriteString("and any pending work. Be factual and brief.\n\n")

	for _, m := range head {
		role := m.Role
		content := m.Content
		if content == "" && m.ToolCall != nil {
			content = fmt.Sprintf("[tool call: %s(%s)]", m.ToolCall.Name, truncate(m.ToolCall.Arguments, 200))
		}
		sb.WriteString(fmt.Sprintf("[%s] %s\n\n", role, content))
	}

	summaryPrompt := sb.String()

	// Call the provider to generate a summary.
	summaryMsgs, err := c.provider.Chat(ctx, []Message{
		{Role: RoleUser, Content: summaryPrompt},
	}, nil)
	if err != nil {
		// On provider error, fall back to truncation (keep tail only).
		return tail, nil
	}

	summary := ""
	if len(summaryMsgs) > 0 {
		summary = summaryMsgs[len(summaryMsgs)-1].Content
	}

	if summary == "" {
		return tail, nil
	}

	// Prepend the compaction marker.
	compacted := make([]Message, 0, 1+len(tail))
	compacted = append(compacted, Message{
		Role:    CompactionMessageRole,
		Content: CompactionMarker + "\n\n" + summary,
	})
	compacted = append(compacted, tail...)

	return compacted, nil
}

// estimateTokens provides a rough token estimate (4 chars ≈ 1 token).
func estimateTokens(s string) int {
	if len(s) == 0 {
		return 0
	}
	return (len(s) + 3) / 4
}

// truncate cuts a string to maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
