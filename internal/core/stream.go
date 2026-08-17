package core

// StreamChunk represents a single chunk from a streaming provider response.
// Exactly one payload field is set per chunk (mutual exclusivity by convention).
type StreamChunk struct {
	TextDelta      string
	ReasoningDelta string
	ToolCallStart  *ToolCall
	ToolCallDelta  *ToolCallDelta
	ToolCallEnd    *ToolCall
	Error          error
	Done           bool
	Usage          *Usage
}

// ToolCallDelta represents incremental tool call arguments during streaming.
type ToolCallDelta struct {
	ID        string
	Name      string
	Arguments string
}

// Usage reports token consumption for a streaming response.
type Usage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

// IsTerminal returns true if this chunk signals the end of the stream
// (either a Done chunk or an error chunk).
func (c StreamChunk) IsTerminal() bool {
	return c.Done || c.Error != nil
}