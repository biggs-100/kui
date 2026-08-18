package core

import (
	"encoding/json"
	"testing"
)

// TestMessageJSONRoundTrip verifies that Message (including embedded ToolCall)
// serializes to JSON and deserializes back with all fields preserved.
// JSON keys MUST be lowercase (role, content, tool_call, etc.) for clean
// session persistence files — no capitalized Go field names leaked.
func TestMessageJSONRoundTrip(t *testing.T) {
	original := Message{
		Role:    RoleAssistant,
		Content: "Let me call a tool.",
		ToolCall: &ToolCall{
			ID:        "call_abc123",
			Name:      "web_search",
			Arguments: `{"query":"golang generics"}`,
		},
		ToolCallID: "",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Verify JSON keys are lowercase — fails without json tags.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal to map failed: %v", err)
	}
	for _, key := range []string{"role", "content", "tool_call"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("JSON missing lowercase key %q; raw JSON: %s", key, data)
		}
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Role != original.Role {
		t.Errorf("Role = %q, want %q", decoded.Role, original.Role)
	}
	if decoded.Content != original.Content {
		t.Errorf("Content = %q, want %q", decoded.Content, original.Content)
	}
	if decoded.ToolCall == nil {
		t.Fatal("ToolCall is nil after round-trip")
	}
	if decoded.ToolCall.ID != original.ToolCall.ID {
		t.Errorf("ToolCall.ID = %q, want %q", decoded.ToolCall.ID, original.ToolCall.ID)
	}
	if decoded.ToolCall.Name != original.ToolCall.Name {
		t.Errorf("ToolCall.Name = %q, want %q", decoded.ToolCall.Name, original.ToolCall.Name)
	}
	if decoded.ToolCall.Arguments != original.ToolCall.Arguments {
		t.Errorf("ToolCall.Arguments = %q, want %q", decoded.ToolCall.Arguments, original.ToolCall.Arguments)
	}
}

// TestSessionStoreInterface is a compile-time assertion that SessionStore,
// Session, and SessionMeta exist with the correct shape. This test will fail
// to compile until session.go defines those types (RED phase).
func TestSessionStoreInterface(t *testing.T) {
	// Compile-time check: SessionStore must be an interface with Save, Load,
	// List, and Delete methods. The concrete implementation (FileSessionStore)
	// lives in the adapters layer; this test only validates the port contract.
	var _ SessionStore = (SessionStore)(nil)

	// Verify SessionMeta and Session exist and carry expected fields.
	meta := SessionMeta{
		ID:        "test-001",
		Profile:   "coder",
		CreatedAt: "2026-01-01T00:00:00Z",
	}
	if meta.ID != "test-001" {
		t.Errorf("SessionMeta.ID = %q, want %q", meta.ID, "test-001")
	}

	sess := Session{
		Meta:     meta,
		Messages: []Message{{Role: RoleUser, Content: "hello"}},
	}
	if len(sess.Messages) != 1 {
		t.Errorf("Session.Messages len = %d, want 1", len(sess.Messages))
	}
}

// TestSessionMetaExtended verifies the extended SessionMeta fields
// (Name, Model, Summary, UpdatedAt, MessageCount) survive JSON round-trip.
func TestSessionMetaExtended(t *testing.T) {
	original := SessionMeta{
		ID:           "test-ext-001",
		Profile:      "coder",
		Name:         "My Session",
		Model:        "gpt-4o",
		Summary:      "What is Go generics?",
		CreatedAt:    "2026-01-01T00:00:00Z",
		UpdatedAt:    "2026-01-01T01:00:00Z",
		MessageCount: 4,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded SessionMeta
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Name != original.Name {
		t.Errorf("Name = %q, want %q", decoded.Name, original.Name)
	}
	if decoded.Model != original.Model {
		t.Errorf("Model = %q, want %q", decoded.Model, original.Model)
	}
	if decoded.Summary != original.Summary {
		t.Errorf("Summary = %q, want %q", decoded.Summary, original.Summary)
	}
	if decoded.UpdatedAt != original.UpdatedAt {
		t.Errorf("UpdatedAt = %q, want %q", decoded.UpdatedAt, original.UpdatedAt)
	}
	if decoded.MessageCount != original.MessageCount {
		t.Errorf("MessageCount = %d, want %d", decoded.MessageCount, original.MessageCount)
	}
}

// TestNewSessionMetaDefaults verifies that NewSessionMeta produces a meta
// with zero-value fields that serialize cleanly (no errors, no surprises).
func TestNewSessionMetaDefaults(t *testing.T) {
	meta := NewSessionMeta("test-id", "coder")

	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded SessionMeta
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.ID != "test-id" {
		t.Errorf("ID = %q, want %q", decoded.ID, "test-id")
	}
	if decoded.Profile != "coder" {
		t.Errorf("Profile = %q, want %q", decoded.Profile, "coder")
	}
	// New fields should be zero-value on default construction
	if decoded.Name != "" {
		t.Errorf("Name should be empty on default, got %q", decoded.Name)
	}
	if decoded.Model != "" {
		t.Errorf("Model should be empty on default, got %q", decoded.Model)
	}
	if decoded.Summary != "" {
		t.Errorf("Summary should be empty on default, got %q", decoded.Summary)
	}
	if decoded.UpdatedAt != "" {
		t.Errorf("UpdatedAt should be empty on default, got %q", decoded.UpdatedAt)
	}
	if decoded.MessageCount != 0 {
		t.Errorf("MessageCount should be 0 on default, got %d", decoded.MessageCount)
	}
}

// TestMessageJSONToolResultRoundTrip verifies tool-result messages (role=tool)
// round-trip correctly, including the ToolCallID link back to the call.
func TestMessageJSONToolResultRoundTrip(t *testing.T) {
	original := Message{
		Role:       RoleTool,
		Content:    `{"status":"ok","results":[]}`,
		ToolCall:   nil,
		ToolCallID: "call_abc123",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Role != original.Role {
		t.Errorf("Role = %q, want %q", decoded.Role, original.Role)
	}
	if decoded.ToolCallID != original.ToolCallID {
		t.Errorf("ToolCallID = %q, want %q", decoded.ToolCallID, original.ToolCallID)
	}
	if decoded.ToolCall != nil {
		t.Errorf("ToolCall should be nil for tool result, got %+v", decoded.ToolCall)
	}
}
