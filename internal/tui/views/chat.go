package views

import (
	"strings"
	"time"

	"github.com/biggs-100/kui/internal/core"
	"github.com/biggs-100/kui/internal/tui/markdown"
	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// ChatNow is used for timestamps; override in tests for determinism.
var ChatNow = time.Now

// PartKind enumerates per-part types for streaming answer rendering.
type PartKind string

const (
	PartKindText       PartKind = "text"
	PartKindReasoning  PartKind = "reasoning"
	PartKindTool       PartKind = "tool"
	PartKindFile       PartKind = "file"
	PartKindCompaction PartKind = "compaction"
	PartKindShell      PartKind = "shell" // real local shell execution + output
)

// Message represents a single conversation entry: user prompt or assistant
// answer. Each prompt captures its profile and model at submission time.
// Extended for per-part rendering (reasoning/compaction/shell kinds).
type Message struct {
	Role      string // "user", "assistant", "system" or "compaction"
	Content   string
	Profile   string // captured at submission time
	Model     string // resolved via REQ-CLI-4 chain
	Kind      PartKind
	Queued    bool // retained for compat; QUEUED badge is REMOVED (pi parity)
	Hover     bool // retained for compat; hover fill is REMOVED (pi parity)
	Timestamp time.Time
}

// ChatModel manages the conversation view: a scrollable list of messages,
// streaming answer chunks, error state, and a status line for reload feedback
// (REQ-TUI-CHAT-1/2/7). User prompts render as a UserMessageBg Box;
// assistant answers render as plain markdown followed by Spacer(1).
type ChatModel struct {
	messages    []Message
	lastError   string
	status      string   // neutral status line (transient, never a toast)
	diagnostics []string // inline diagnostic annotations
	styles      *theme.Styles
	width       int
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
		Role:      role,
		Content:   content,
		Profile:   profile,
		Model:     model,
		Kind:      PartKindText,
		Timestamp: ChatNow(),
	})
}

// AppendPart adds a typed part (text/reasoning/tool/file/compaction).
func (m *ChatModel) AppendPart(kind PartKind, content, profile, model string) {
	if kind == "" {
		kind = PartKindText
	}
	role := "assistant"
	if kind == PartKindCompaction {
		role = "compaction"
	}
	m.messages = append(m.messages, Message{
		Role:      role,
		Content:   content,
		Profile:   profile,
		Model:     model,
		Kind:      kind,
		Timestamp: ChatNow(),
	})
}

// AppendShell records a REAL local shell execution and its combined output.
// The entry renders as a plain muted block — it is not a conversation turn
// and carries no agent identity.
func (m *ChatModel) AppendShell(script, output string, execErr error) {
	var b strings.Builder
	b.WriteString("$ ")
	b.WriteString(script)
	if execErr != nil {
		b.WriteString("\n(error: ")
		b.WriteString(execErr.Error())
		b.WriteString(")")
	}
	if output != "" {
		b.WriteString("\n")
		b.WriteString(output)
	}
	m.messages = append(m.messages, Message{
		Role:      "system",
		Kind:      PartKindShell,
		Content:   b.String(),
		Timestamp: ChatNow(),
	})
}

// Clear removes all rendered conversation state: messages, error, status and
// diagnostics. It clears the display only — persisted session history is
// untouched.
func (m *ChatModel) Clear() {
	m.messages = nil
	m.lastError = ""
	m.status = ""
	m.diagnostics = nil
}

// AppendQueuedMessage adds a queued prompt part. The QUEUED badge is removed
// (pi parity); the message renders as a normal user Box. Retained for API
// compat with callers and tests.
func (m *ChatModel) AppendQueuedMessage(role, content, profile, model string) {
	m.messages = append(m.messages, Message{
		Role:      role,
		Content:   content,
		Profile:   profile,
		Model:     model,
		Kind:      PartKindText,
		Queued:    true,
		Timestamp: ChatNow(),
	})
}

// SetTimestamp sets timestamp for message at index (for tests).
func (m *ChatModel) SetTimestamp(idx int, t time.Time) {
	if idx >= 0 && idx < len(m.messages) {
		m.messages[idx].Timestamp = t
	}
}

// SetHover sets hover state for message at index. Retained for API compat;
// hover fill is removed (pi parity) and has no visual effect.
func (m *ChatModel) SetHover(idx int, hover bool) {
	if idx >= 0 && idx < len(m.messages) {
		m.messages[idx].Hover = hover
	}
}

// SetQueued sets queued badge for message at index. Retained for API compat;
// the badge is removed (pi parity) and has no visual effect.
func (m *ChatModel) SetQueued(idx int, queued bool) {
	if idx >= 0 && idx < len(m.messages) {
		m.messages[idx].Queued = queued
	}
}

// SetWidth sets viewport width for block calculations.
func (m *ChatModel) SetWidth(w int) { m.width = w }

// AppendChunk appends text to the most recent assistant message. If there
// is no assistant message yet, one is created (REQ-TUI-CHAT-2 streaming
// chunks append incrementally to the same block).
func (m *ChatModel) AppendChunk(delta string) {
	if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "assistant" {
		m.messages[len(m.messages)-1].Content += delta
	} else {
		m.messages = append(m.messages, Message{
			Role:      "assistant",
			Content:   delta,
			Kind:      PartKindText,
			Timestamp: ChatNow(),
		})
	}
}

// SetError records a stream error for display (REQ-TUI-CHAT-7: plain muted
// transcript line, never a toast).
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
				Role:      "user",
				Content:   msg.Content,
				Kind:      PartKindText,
				Timestamp: ChatNow(),
			})
		case core.RoleAssistant:
			if msg.Content != "" {
				m.messages = append(m.messages, Message{
					Role:      "assistant",
					Content:   msg.Content,
					Kind:      PartKindText,
					Timestamp: ChatNow(),
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

// SetStatus sets a neutral status line for reload feedback (REQ-TUI-CHAT-7).
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

// userBox renders a user prompt as a full-width Box with UserMessageBg
// (#343541) and horizontal padding — no SplitBorder, no "you:" label
// (REQ-TUI-CHAT-1).
func (m ChatModel) userBox(content string, width int) string {
	if m.styles != nil {
		style := m.styles.UserMessage
		if width > 0 {
			style = style.Width(width - 4)
		}
		return style.Render(content)
	}
	return content
}

// muted renders a plain muted transcript line (shell/compaction/status).
func (m ChatModel) muted(s string) string {
	if m.styles != nil {
		return m.styles.HomeMuted.Render(s)
	}
	return s
}

// View renders the transcript: user Boxes, plain assistant markdown blocks
// separated by one blank line (Spacer(1)), muted shell blocks, full-width
// compaction dividers, dim-italic thinking, and plain muted error/status
// lines (REQ-TUI-CHAT-1/2/7). Empty transcript renders "" — the frame shows
// only editor + footer.
func (m ChatModel) View(width int) string {
	if width <= 0 {
		width = m.width
	}
	if width <= 0 {
		width = 120
	}
	if len(m.messages) == 0 && m.lastError == "" && m.status == "" && len(m.diagnostics) == 0 {
		return ""
	}
	var parts []string
	for _, msg := range m.messages {
		// Real local shell execution: plain muted block, no border and no
		// agent identity — it is utility output, not a conversation turn.
		if msg.Kind == PartKindShell {
			parts = append(parts, m.muted(msg.Content))
			continue
		}

		// Compaction: full-width divider line.
		if msg.Kind == PartKindCompaction || msg.Role == "compaction" {
			parts = append(parts, m.muted(compactionDivider(width)))
			continue
		}

		// Thinking / reasoning: dim italic.
		if msg.Kind == PartKindReasoning {
			parts = append(parts, m.thinking(msg.Content))
			continue
		}

		switch msg.Role {
		case "user":
			parts = append(parts, m.userBox(msg.Content, width))
		default:
			// Assistant answer: plain markdown, no background, no border,
			// followed by Spacer(1) — one blank line between blocks.
			parts = append(parts, markdown.Render(msg.Content, m.styles))
			parts = append(parts, "")
		}
	}

	if m.lastError != "" {
		parts = append(parts, m.muted("error: "+m.lastError))
	}
	if m.status != "" {
		parts = append(parts, m.muted("● "+m.status))
	}
	for _, d := range m.diagnostics {
		parts = append(parts, m.muted("  "+d))
	}
	return strings.Join(parts, "\n")
}

// Render produces the full chat view string (width-agnostic).
func (m ChatModel) Render() string {
	w := m.width
	if w == 0 {
		w = 120
	}
	return m.View(w)
}

// thinking renders reasoning content as dim italic (pi parity: thinking
// levels fold to TextMuted, opacity keeps 0.6).
func (m ChatModel) thinking(s string) string {
	if m.styles != nil && m.styles.Theme != nil {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.styles.Theme.TextMuted)).
			Italic(true).
			Render(s)
	}
	return s
}

// compactionDivider returns a full-width divider line labeling the
// compaction boundary.
func compactionDivider(width int) string {
	label := "── compaction ──"
	if width <= 0 || len(label) >= width {
		return label
	}
	fill := width - len(label) - 1
	if fill < 0 {
		fill = 0
	}
	return label + " " + strings.Repeat("─", fill)
}
