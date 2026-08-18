// Package core is the domain heart of kui: the agent loop and the ports it
// depends on. It performs no I/O and imports only the standard library.
package core

import "context"

// Message roles exchanged with the provider.
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
	// RoleSystem marks system prompts and profile-context marker messages
	// (D16, REQ-LOOP-6). Providers serialize it verbatim as role "system".
	RoleSystem = "system"
)

// ToolCall is a provider request to invoke a tool. Arguments is the raw JSON
// argument object exactly as the provider emitted it.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Message is one entry of the conversation exchanged with the provider.
// ToolCall is set on assistant messages that request tool execution;
// ToolCallID links a tool result message back to the call it answers.
type Message struct {
	Role       string    `json:"role"`
	Content    string    `json:"content"`
	ToolCall   *ToolCall `json:"tool_call,omitempty"`
	ToolCallID string    `json:"tool_call_id,omitempty"`
}

// Provider is the port to a model backend (D2). Chat exchanges the full
// message sequence and the advertised tool set, and returns the provider's
// response messages.
type Provider interface {
	Chat(ctx context.Context, messages []Message, tools []Tool) ([]Message, error)
}
