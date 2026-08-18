package core

import "testing"

func TestClassifySystemMessagesProtected(t *testing.T) {
	tests := []struct {
		name     string
		messages []Message
		wantProt []Message
		wantComp []Message
	}{
		{
			name: "single system message is protected",
			messages: []Message{
				{Role: RoleSystem, Content: "You are a helpful assistant."},
			},
			wantProt: []Message{
				{Role: RoleSystem, Content: "You are a helpful assistant."},
			},
			wantComp: nil,
		},
		{
			name: "system message among user messages",
			messages: []Message{
				{Role: RoleSystem, Content: "sys"},
				{Role: RoleUser, Content: "hello"},
				{Role: RoleAssistant, Content: "hi"},
			},
			wantProt: []Message{
				{Role: RoleSystem, Content: "sys"},
			},
			wantComp: []Message{
				{Role: RoleUser, Content: "hello"},
				{Role: RoleAssistant, Content: "hi"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProt, gotComp := ClassifyMessages(tt.messages)
			assertMessagesEqual(t, "protected", gotProt, tt.wantProt)
			assertMessagesEqual(t, "compactable", gotComp, tt.wantComp)
		})
	}
}

func TestClassifyProfileMarkersProtected(t *testing.T) {
	tests := []struct {
		name     string
		messages []Message
		wantProt []Message
		wantComp []Message
	}{
		{
			name: "user message with Profile switched is protected",
			messages: []Message{
				{Role: RoleUser, Content: "Profile switched to coding"},
				{Role: RoleAssistant, Content: "Got it."},
			},
			wantProt: []Message{
				{Role: RoleUser, Content: "Profile switched to coding"},
			},
			wantComp: []Message{
				{Role: RoleAssistant, Content: "Got it."},
			},
		},
		{
			name: "assistant message with Profile switched is protected",
			messages: []Message{
				{Role: RoleUser, Content: "hello"},
				{Role: RoleAssistant, Content: "Profile switched to research"},
			},
			wantProt: []Message{
				{Role: RoleAssistant, Content: "Profile switched to research"},
			},
			wantComp: []Message{
				{Role: RoleUser, Content: "hello"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProt, gotComp := ClassifyMessages(tt.messages)
			assertMessagesEqual(t, "protected", gotProt, tt.wantProt)
			assertMessagesEqual(t, "compactable", gotComp, tt.wantComp)
		})
	}
}

func TestClassifyToolPairPreservation(t *testing.T) {
	tests := []struct {
		name     string
		messages []Message
		wantProt []Message
		wantComp []Message
	}{
		{
			name: "protected tool call protects matching result",
			messages: []Message{
				{Role: RoleAssistant, ToolCall: &ToolCall{ID: "tc1", Name: "read", Arguments: "{}"}, Content: "Profile switched to coding"},
				{Role: RoleTool, Content: "file contents", ToolCallID: "tc1"},
				{Role: RoleUser, Content: "next question"},
			},
			wantProt: []Message{
				{Role: RoleAssistant, ToolCall: &ToolCall{ID: "tc1", Name: "read", Arguments: "{}"}, Content: "Profile switched to coding"},
				{Role: RoleTool, Content: "file contents", ToolCallID: "tc1"},
			},
			wantComp: []Message{
				{Role: RoleUser, Content: "next question"},
			},
		},
		{
			name: "unprotected tool call stays compactable with result",
			messages: []Message{
				{Role: RoleUser, Content: "read file"},
				{Role: RoleAssistant, ToolCall: &ToolCall{ID: "tc2", Name: "read", Arguments: "{}"}},
				{Role: RoleTool, Content: "file contents", ToolCallID: "tc2"},
			},
			wantProt: nil,
			wantComp: []Message{
				{Role: RoleUser, Content: "read file"},
				{Role: RoleAssistant, ToolCall: &ToolCall{ID: "tc2", Name: "read", Arguments: "{}"}},
				{Role: RoleTool, Content: "file contents", ToolCallID: "tc2"},
			},
		},
		{
			name: "orphaned tool result stays compactable",
			messages: []Message{
				{Role: RoleUser, Content: "hello"},
				{Role: RoleTool, Content: "orphan result", ToolCallID: "tc-missing"},
			},
			wantProt: nil,
			wantComp: []Message{
				{Role: RoleUser, Content: "hello"},
				{Role: RoleTool, Content: "orphan result", ToolCallID: "tc-missing"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProt, gotComp := ClassifyMessages(tt.messages)
			assertMessagesEqual(t, "protected", gotProt, tt.wantProt)
			assertMessagesEqual(t, "compactable", gotComp, tt.wantComp)
		})
	}
}

func TestClassifyEmptyHistory(t *testing.T) {
	gotProt, gotComp := ClassifyMessages([]Message{})
	if len(gotProt) != 0 {
		t.Errorf("protected: got %d messages, want 0", len(gotProt))
	}
	if len(gotComp) != 0 {
		t.Errorf("compactable: got %d messages, want 0", len(gotComp))
	}
}

func TestClassifyAllSystemMessages(t *testing.T) {
	messages := []Message{
		{Role: RoleSystem, Content: "You are helpful."},
		{Role: RoleSystem, Content: "Profile switched to coding"},
		{Role: RoleSystem, Content: "Additional context."},
	}
	gotProt, gotComp := ClassifyMessages(messages)
	if len(gotProt) != 3 {
		t.Errorf("protected: got %d messages, want 3", len(gotProt))
	}
	if len(gotComp) != 0 {
		t.Errorf("compactable: got %d messages, want 0", len(gotComp))
	}
}

func assertMessagesEqual(t *testing.T, label string, got, want []Message) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: got %d messages, want %d", label, len(got), len(want))
		return
	}
	for i := range got {
		if got[i].Role != want[i].Role || got[i].Content != want[i].Content {
			t.Errorf("%s[%d]: got %+v, want %+v", label, i, got[i], want[i])
		}
	}
}
