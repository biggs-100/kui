package core

import (
	"errors"
	"testing"
)

// TestStreamChunkTextDeltaOnly verifies that a StreamChunk with only TextDelta
// set has all other payload fields at their zero values.
func TestStreamChunkTextDeltaOnly(t *testing.T) {
	chunk := StreamChunk{
		TextDelta: "hello",
	}

	if chunk.TextDelta != "hello" {
		t.Errorf("TextDelta = %q, want %q", chunk.TextDelta, "hello")
	}
	if chunk.ReasoningDelta != "" {
		t.Errorf("ReasoningDelta = %q, want empty", chunk.ReasoningDelta)
	}
	if chunk.ToolCallStart != nil {
		t.Errorf("ToolCallStart = %v, want nil", chunk.ToolCallStart)
	}
	if chunk.ToolCallDelta != nil {
		t.Errorf("ToolCallDelta = %v, want nil", chunk.ToolCallDelta)
	}
	if chunk.ToolCallEnd != nil {
		t.Errorf("ToolCallEnd = %v, want nil", chunk.ToolCallEnd)
	}
	if chunk.Error != nil {
		t.Errorf("Error = %v, want nil", chunk.Error)
	}
	if chunk.Done {
		t.Errorf("Done = true, want false")
	}
	if chunk.Usage != nil {
		t.Errorf("Usage = %v, want nil", chunk.Usage)
	}
}

// TestStreamChunkToolCallStart verifies ToolCallStart field population.
func TestStreamChunkToolCallStart(t *testing.T) {
	call := &ToolCall{
		ID:   "call_123",
		Name: "get_weather",
	}
	chunk := StreamChunk{
		ToolCallStart: call,
	}

	if chunk.ToolCallStart == nil {
		t.Fatal("ToolCallStart is nil")
	}
	if chunk.ToolCallStart.ID != "call_123" {
		t.Errorf("ToolCallStart.ID = %q, want %q", chunk.ToolCallStart.ID, "call_123")
	}
	if chunk.ToolCallStart.Name != "get_weather" {
		t.Errorf("ToolCallStart.Name = %q, want %q", chunk.ToolCallStart.Name, "get_weather")
	}
}

// TestStreamChunkToolCallDelta verifies ToolCallDelta field population.
func TestStreamChunkToolCallDelta(t *testing.T) {
	delta := &ToolCallDelta{
		ID:        "call_123",
		Name:      "get_weather",
		Arguments: `{"location":"NYC"}`,
	}
	chunk := StreamChunk{
		ToolCallDelta: delta,
	}

	if chunk.ToolCallDelta == nil {
		t.Fatal("ToolCallDelta is nil")
	}
	if chunk.ToolCallDelta.ID != "call_123" {
		t.Errorf("ToolCallDelta.ID = %q, want %q", chunk.ToolCallDelta.ID, "call_123")
	}
	if chunk.ToolCallDelta.Arguments != `{"location":"NYC"}` {
		t.Errorf("ToolCallDelta.Arguments = %q, want %q", chunk.ToolCallDelta.Arguments, `{"location":"NYC"}`)
	}
}

// TestStreamChunkToolCallEnd verifies ToolCallEnd field population.
func TestStreamChunkToolCallEnd(t *testing.T) {
	call := &ToolCall{
		ID:   "call_123",
		Name: "get_weather",
	}
	chunk := StreamChunk{
		ToolCallEnd: call,
	}

	if chunk.ToolCallEnd == nil {
		t.Fatal("ToolCallEnd is nil")
	}
	if chunk.ToolCallEnd.ID != "call_123" {
		t.Errorf("ToolCallEnd.ID = %q, want %q", chunk.ToolCallEnd.ID, "call_123")
	}
}

// TestStreamChunkError verifies Error field population.
func TestStreamChunkError(t *testing.T) {
	err := errors.New("network timeout")
	chunk := StreamChunk{
		Error: err,
	}

	if chunk.Error == nil {
		t.Fatal("Error is nil")
	}
	if chunk.Error.Error() != "network timeout" {
		t.Errorf("Error = %q, want %q", chunk.Error.Error(), "network timeout")
	}
}

// TestStreamChunkDone verifies Done flag.
func TestStreamChunkDone(t *testing.T) {
	chunk := StreamChunk{
		Done: true,
	}

	if !chunk.Done {
		t.Error("Done = false, want true")
	}
}

// TestStreamChunkUsage verifies Usage field population.
func TestStreamChunkUsage(t *testing.T) {
	usage := &Usage{
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
	}
	chunk := StreamChunk{
		Usage: usage,
	}

	if chunk.Usage == nil {
		t.Fatal("Usage is nil")
	}
	if chunk.Usage.InputTokens != 100 {
		t.Errorf("Usage.InputTokens = %d, want %d", chunk.Usage.InputTokens, 100)
	}
	if chunk.Usage.OutputTokens != 50 {
		t.Errorf("Usage.OutputTokens = %d, want %d", chunk.Usage.OutputTokens, 50)
	}
	if chunk.Usage.TotalTokens != 150 {
		t.Errorf("Usage.TotalTokens = %d, want %d", chunk.Usage.TotalTokens, 150)
	}
}

// TestStreamChunkEmpty verifies that a zero-value StreamChunk has no active fields.
func TestStreamChunkEmpty(t *testing.T) {
	var chunk StreamChunk

	if chunk.TextDelta != "" {
		t.Errorf("TextDelta = %q, want empty", chunk.TextDelta)
	}
	if chunk.ReasoningDelta != "" {
		t.Errorf("ReasoningDelta = %q, want empty", chunk.ReasoningDelta)
	}
	if chunk.ToolCallStart != nil {
		t.Errorf("ToolCallStart = %v, want nil", chunk.ToolCallStart)
	}
	if chunk.ToolCallDelta != nil {
		t.Errorf("ToolCallDelta = %v, want nil", chunk.ToolCallDelta)
	}
	if chunk.ToolCallEnd != nil {
		t.Errorf("ToolCallEnd = %v, want nil", chunk.ToolCallEnd)
	}
	if chunk.Error != nil {
		t.Errorf("Error = %v, want nil", chunk.Error)
	}
	if chunk.Done {
		t.Error("Done = true, want false")
	}
	if chunk.Usage != nil {
		t.Errorf("Usage = %v, want nil", chunk.Usage)
	}
}

// TestStreamChunkReasoningDelta verifies ReasoningDelta field.
func TestStreamChunkReasoningDelta(t *testing.T) {
	chunk := StreamChunk{
		ReasoningDelta: "thinking step",
	}

	if chunk.ReasoningDelta != "thinking step" {
		t.Errorf("ReasoningDelta = %q, want %q", chunk.ReasoningDelta, "thinking step")
	}
}

// TestIsTerminal verifies that IsTerminal returns true for Done or Error chunks.
func TestIsTerminal(t *testing.T) {
	tests := []struct {
		name     string
		chunk    StreamChunk
		expected bool
	}{
		{
			name:     "done chunk",
			chunk:    StreamChunk{Done: true},
			expected: true,
		},
		{
			name:     "error chunk",
			chunk:    StreamChunk{Error: errors.New("fail")},
			expected: true,
		},
		{
			name:     "text delta chunk",
			chunk:    StreamChunk{TextDelta: "hello"},
			expected: false,
		},
		{
			name:     "tool call start chunk",
			chunk:    StreamChunk{ToolCallStart: &ToolCall{ID: "1"}},
			expected: false,
		},
		{
			name:     "tool call delta chunk",
			chunk:    StreamChunk{ToolCallDelta: &ToolCallDelta{ID: "1"}},
			expected: false,
		},
		{
			name:     "tool call end chunk",
			chunk:    StreamChunk{ToolCallEnd: &ToolCall{ID: "1"}},
			expected: false,
		},
		{
			name:     "empty chunk",
			chunk:    StreamChunk{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.chunk.IsTerminal(); got != tt.expected {
				t.Errorf("IsTerminal() = %v, want %v", got, tt.expected)
			}
		})
	}
}