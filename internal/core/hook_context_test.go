package core

import (
	"testing"
)

// TestHookContextMessagesMutation verifies that SetMessages replaces the
// message slice returned by Messages (D4, REQ-EXT-4).
func TestHookContextMessagesMutation(t *testing.T) {
	original := []Message{
		{Role: RoleUser, Content: "hello"},
	}
	ctx := NewHookContext("test", original)

	got := ctx.Messages()
	if len(got) != 1 || got[0].Content != "hello" {
		t.Errorf("initial Messages() = %v, want [hello]", got)
	}

	replacement := []Message{
		{Role: RoleUser, Content: "modified"},
		{Role: RoleAssistant, Content: "response"},
	}
	ctx.SetMessages(replacement)

	got = ctx.Messages()
	if len(got) != 2 {
		t.Fatalf("after SetMessages, Messages() returned %d items, want 2", len(got))
	}
	if got[0].Content != "modified" || got[1].Content != "response" {
		t.Errorf("after SetMessages, Messages() = %v", got)
	}
}

// TestHookContextBlockAndUnblock verifies that Block sets IsBlocked and
// stores the reason, and that the initial state is unblocked.
func TestHookContextBlockAndUnblock(t *testing.T) {
	ctx := NewHookContext("test", nil)

	// Initially unblocked
	if ctx.IsBlocked() {
		t.Error("new context is blocked, want unblocked")
	}
	if ctx.BlockReason() != "" {
		t.Errorf("new context BlockReason() = %q, want empty", ctx.BlockReason())
	}

	// Block with a reason
	ctx.Block("not allowed")

	if !ctx.IsBlocked() {
		t.Error("after Block(), IsBlocked() = false, want true")
	}
	if ctx.BlockReason() != "not allowed" {
		t.Errorf("BlockReason() = %q, want %q", ctx.BlockReason(), "not allowed")
	}
}

// TestHookContextToolCall verifies SetToolCall replaces the tool call.
func TestHookContextToolCall(t *testing.T) {
	ctx := NewHookContext("test", nil)

	if ctx.ToolCall() != nil {
		t.Error("new context has non-nil ToolCall, want nil")
	}

	call := &ToolCall{ID: "call-1", Name: "read_file", Arguments: `{"path":"a.md"}`}
	ctx.SetToolCall(call)

	got := ctx.ToolCall()
	if got == nil {
		t.Fatal("ToolCall() is nil after SetToolCall")
	}
	if got.ID != "call-1" || got.Name != "read_file" {
		t.Errorf("ToolCall() = %+v, want ID=call-1, Name=read_file", got)
	}
}

// TestHookContextEventName verifies that EventName returns the event passed
// to NewHookContext.
func TestHookContextEventName(t *testing.T) {
	ctx := NewHookContext("before_tool_execution", nil)

	if ctx.EventName() != "before_tool_execution" {
		t.Errorf("EventName() = %q, want %q", ctx.EventName(), "before_tool_execution")
	}
}

// TestHookContextNilMessagesReturnsNil verifies that creating a HookContext
// with nil messages returns nil from Messages() (nil-safe).
func TestHookContextNilMessagesReturnsNil(t *testing.T) {
	ctx := NewHookContext("test", nil)

	got := ctx.Messages()
	if got != nil {
		t.Errorf("Messages() = %v, want nil", got)
	}
}

// TestHookContextSetMessagesToNil verifies that SetMessages(nil) results in
// Messages() returning nil.
func TestHookContextSetMessagesToNil(t *testing.T) {
	ctx := NewHookContext("test", []Message{{Role: RoleUser, Content: "hi"}})

	ctx.SetMessages(nil)

	if ctx.Messages() != nil {
		t.Errorf("Messages() after SetMessages(nil) = %v, want nil", ctx.Messages())
	}
}

// TestHookContextMultipleBlocksLastReasonWins verifies that calling Block
// multiple times updates the reason.
func TestHookContextMultipleBlocksLastReasonWins(t *testing.T) {
	ctx := NewHookContext("test", nil)

	ctx.Block("first reason")
	ctx.Block("second reason")

	if ctx.BlockReason() != "second reason" {
		t.Errorf("BlockReason() = %q, want %q", ctx.BlockReason(), "second reason")
	}
}
