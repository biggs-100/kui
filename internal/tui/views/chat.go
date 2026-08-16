package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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
// streaming answer chunks, and error state (REQ-TUI-CHAT-1/2).
type ChatModel struct {
	messages  []Message
	lastError string
}

// NewChatModel creates an empty ChatModel.
func NewChatModel() ChatModel {
	return ChatModel{}
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

// Messages returns the current message slice (for testing and inspection).
func (m ChatModel) Messages() []Message {
	return m.messages
}

// LastError returns the current error string (for testing).
func (m ChatModel) LastError() string {
	return m.lastError
}

var (
	userRoleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	assistantRoleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))

	profileStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Faint(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	emptyHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Faint(true)
)

// Render produces the full chat view string.
func (m ChatModel) Render() string {
	if len(m.messages) == 0 {
		return emptyHintStyle.Render("start a conversation...")
	}

	var parts []string
	for _, msg := range m.messages {
		var line strings.Builder

		if msg.Role == "user" {
			line.WriteString(userRoleStyle.Render("you"))
		} else {
			line.WriteString(assistantRoleStyle.Render("assistant"))
		}

		// REQ-TUI-CHAT-3: show profile and model context per prompt
		if msg.Profile != "" {
			line.WriteString(" ")
			line.WriteString(profileStyle.Render(fmt.Sprintf("(%s/%s)", msg.Profile, msg.Model)))
		}

		line.WriteString("\n")
		line.WriteString(msg.Content)

		parts = append(parts, line.String())
	}

	if m.lastError != "" {
		parts = append(parts, errorStyle.Render("error: "+m.lastError))
	}

	return strings.Join(parts, "\n\n")
}
