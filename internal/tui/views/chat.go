package views

import (
	"fmt"
	"strings"

	"github.com/biggs-100/kui/internal/core"
	"github.com/biggs-100/kui/internal/tui/markdown"
	"github.com/biggs-100/kui/internal/tui/theme"
)

// Message represents a single conversation entry: user prompt or assistant
// answer. Each prompt captures its profile and model at submission time
// (REQ-TUI-CHAT-3).
type Message struct {
	Role    string // "user" or "assistant"
	Content string
	Profile string // captured at submission time
	Model   string // resolved via REQ-CLI-4 chain
}

// ChatModel manages the conversation view: a scrollable list of messages,
// streaming answer chunks, error state, and a status line for reload feedback
// (REQ-TUI-CHAT-1/2, REQ-RELOAD-12).
type ChatModel struct {
	messages    []Message
	lastError   string
	status      string   // REQ-RELOAD-12: neutral status line
	diagnostics []string // inline diagnostic annotations
	styles      *theme.Styles
}

// NewChatModel creates an empty ChatModel.
func NewChatModel(styles *theme.Styles) ChatModel {
	return ChatModel{
		styles: styles,
	}
}

// AppendMessage adds a completed message to the conversation.
func (m *ChatModel) AppendMessage(role, content, profile, model string) {
	m.messages = append(m.messages, Message{
		Role:    role,
		Content: content,
		Profile: profile,
		Model:   model,
	})
}

// AppendChunk appends text to the most recent assistant message. If there
// is no assistant message yet, one is created (REQ-TUI-CHAT-2 streaming
// chunks).
func (m *ChatModel) AppendChunk(delta string) {
	if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "assistant" {
		m.messages[len(m.messages)-1].Content += delta
	} else {
		m.messages = append(m.messages, Message{
			Role:    "assistant",
			Content: delta,
		})
	}
}

// SetError records a stream error for display (REQ-TUI-CHAT-2).
func (m *ChatModel) SetError(msg string) {
	m.lastError = msg
}

// LoadHistory populates the chat view with messages from a restored session.
// Core messages are mapped to view messages for rendering. System and tool
// messages are skipped since they are internal to the agent loop.
func (m *ChatModel) LoadHistory(msgs []core.Message) {
	m.messages = nil
	m.lastError = ""
	for _, msg := range msgs {
		switch msg.Role {
		case core.RoleUser:
			m.messages = append(m.messages, Message{
				Role:    "user",
				Content: msg.Content,
			})
		case core.RoleAssistant:
			if msg.Content != "" {
				m.messages = append(m.messages, Message{
					Role:    "assistant",
					Content: msg.Content,
				})
			}
		}
	}
}

// Messages returns the current message slice (for testing and inspection).
func (m ChatModel) Messages() []Message {
	return m.messages
}

// LastError returns the current error string (for testing).
func (m ChatModel) LastError() string {
	return m.lastError
}

// SetStatus sets a neutral status line for reload feedback (REQ-RELOAD-12).
func (m *ChatModel) SetStatus(s string) {
	m.status = s
}

// Status returns the current status string (for testing).
func (m ChatModel) Status() string {
	return m.status
}

// SetDiagnostics sets inline diagnostic annotations displayed below messages.
func (m *ChatModel) SetDiagnostics(diags []string) {
	m.diagnostics = diags
}

// Diagnostics returns the current diagnostic annotations (for testing).
func (m ChatModel) Diagnostics() []string {
	return m.diagnostics
}

// Render produces the full chat view string.
func (m ChatModel) Render() string {
	if len(m.messages) == 0 {
		return m.styles.EmptyHint.Render("start a conversation...")
	}

	var parts []string
	for _, msg := range m.messages {
		var line strings.Builder

		if msg.Role == "user" {
			line.WriteString(m.styles.UserRole.Render("you"))
		} else {
			line.WriteString(m.styles.AssistantRole.Render("assistant"))
		}

		// REQ-TUI-CHAT-3: show profile and model context per prompt
		if msg.Profile != "" {
			line.WriteString(" ")
			line.WriteString(m.styles.Profile.Render(fmt.Sprintf("(%s/%s)", msg.Profile, msg.Model)))
		}

		line.WriteString("\n")

		// Assistant messages go through markdown rendering
		if msg.Role == "assistant" {
			line.WriteString(markdown.Render(msg.Content, m.styles))
		} else {
			line.WriteString(msg.Content)
		}

		parts = append(parts, line.String())
	}

	if m.lastError != "" {
		parts = append(parts, m.styles.Error.Render("error: "+m.lastError))
	}

	if m.status != "" {
		parts = append(parts, m.styles.EmptyHint.Render(m.status))
	}

	// Inline diagnostic annotations.
	if len(m.diagnostics) > 0 {
		for _, d := range m.diagnostics {
			parts = append(parts, m.styles.Error.Render("  "+d))
		}
	}

	return strings.Join(parts, "\n\n")
}
