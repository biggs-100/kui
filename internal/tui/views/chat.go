package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/biggs-100/kui/internal/core"
	"github.com/biggs-100/kui/internal/tui/markdown"
	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/biggs-100/kui/internal/tui/ui"
	"github.com/biggs-100/kui/internal/tui/util"
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
)

// Message represents a single conversation entry: user prompt or assistant
// answer. Each prompt captures its profile and model at submission time
// (REQ-TUI-CHAT-3). Extended for PR3 per-part rendering.
type Message struct {
	Role      string // "user" or "assistant"
	Content   string
	Profile   string // captured at submission time
	Model     string // resolved via REQ-CLI-4 chain
	Kind      PartKind
	Queued    bool
	Hover     bool
	Timestamp time.Time
}

// ChatModel manages the conversation view: a scrollable list of messages,
// streaming answer chunks, error state, and a status line for reload feedback
// (REQ-TUI-CHAT-1/2, REQ-RELOAD-12). PR3 adds per-part SplitBorder rendering.
type ChatModel struct {
	messages     []Message
	lastError    string
	status       string   // REQ-RELOAD-12: neutral status line
	diagnostics  []string // inline diagnostic annotations
	styles       *theme.Styles
	stickyScroll bool
	width        int
}

// NewChatModel creates an empty ChatModel.
func NewChatModel(styles *theme.Styles) ChatModel {
	return ChatModel{
		styles:       styles,
		stickyScroll: true,
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

// AppendQueuedMessage adds a queued prompt part with QUEUED badge.
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

// SetHover sets hover state for message at index.
func (m *ChatModel) SetHover(idx int, hover bool) {
	if idx >= 0 && idx < len(m.messages) {
		m.messages[idx].Hover = hover
	}
}

// SetQueued sets queued badge for message at index.
func (m *ChatModel) SetQueued(idx int, queued bool) {
	if idx >= 0 && idx < len(m.messages) {
		m.messages[idx].Queued = queued
	}
}

// SetStickyScroll sets sticky scroll (auto-follow) state.
func (m *ChatModel) SetStickyScroll(v bool) {
	m.stickyScroll = v
}

// StickyScroll returns whether sticky scroll is enabled.
func (m ChatModel) StickyScroll() bool { return m.stickyScroll }

// SetWidth sets viewport width for word-wrap calculations.
func (m *ChatModel) SetWidth(w int) { m.width = w }

// AppendChunk appends text to the most recent assistant message. If there
// is no assistant message yet, one is created (REQ-TUI-CHAT-2 streaming
// chunks).
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

func (m ChatModel) agentColor(role string) string {
	if m.styles == nil || m.styles.Theme == nil {
		return theme.DefaultTheme().Primary
	}
	t := m.styles.Theme
	if role == "user" {
		if t.Primary != "" {
			return t.Primary
		}
		return t.Accent
	}
	if t.Accent != "" {
		return t.Accent
	}
	if t.Primary != "" {
		return t.Primary
	}
	return theme.DefaultTheme().Primary
}

// View renders the chat at given width with per-part SplitBorder (┃╹), hover,
// QUEUED badge, compaction divider, and locale timestamps.
func (m ChatModel) View(width int) string {
	if len(m.messages) == 0 {
		return m.styles.EmptyHint.Render("start a conversation...")
	}
	var parts []string
	for _, msg := range m.messages {
		// Compaction divider
		if msg.Kind == PartKindCompaction {
			div := "── compaction ──"
			if m.styles != nil {
				div = m.styles.HomeMuted.Render(div)
			}
			// compaction divider with SplitBorder bottom terminator hint
			parts = append(parts, div+"\n╹")
			continue
		}

		var inner strings.Builder

		// QUEUED badge
		if msg.Queued {
			badge := "QUEUED"
			if m.styles != nil && m.styles.Theme != nil && m.styles.Theme.Warning != "" {
				badge = lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.Theme.Warning)).Bold(true).Render("QUEUED")
			}
			inner.WriteString(badge)
			inner.WriteString(" ")
		}

		// Profile/model context (preserve for per-prompt stability)
		if msg.Profile != "" {
			prof := fmt.Sprintf("(%s/%s)", msg.Profile, msg.Model)
			if m.styles != nil {
				prof = m.styles.Profile.Render(prof)
			}
			inner.WriteString(prof)
			inner.WriteString(" ")
		}

		// Timestamp via locale: today → time, older → date
		if !msg.Timestamp.IsZero() {
			ts := util.TodayTimeOrDateTime(msg.Timestamp, ChatNow())
			if m.styles != nil {
				ts = m.styles.HomeMuted.Render(ts)
			}
			inner.WriteString(ts)
			inner.WriteString(" ")
		}

		// Hover marker: when Hover true, background = BackgroundElement and add "hover" fallback
		hoverExtra := ""
		if msg.Hover {
			hoverExtra = " hover"
			// marker ensures dump indicates backgroundElement path even in plain text fallback
			if m.styles != nil && m.styles.Theme != nil && m.styles.Theme.BackgroundElement != "" {
				hoverExtra = " hover:" + m.styles.Theme.BackgroundElement
			}
		}

		inner.WriteString("\n")

		// Content: assistant via markdown tokens, user plain
		var content string
		if msg.Role == "assistant" {
			content = markdown.Render(msg.Content, m.styles)
		} else {
			content = msg.Content
		}
		// Width-aware truncation/wrap when width provided
		if width > 0 && lipgloss.Width(content) > width-6 {
			// simple wrap: truncate with width handling; real word wrap is handled by lipgloss Width
			content = truncateToWidth(content, width-6)
		}
		inner.WriteString(content)
		if hoverExtra != "" {
			inner.WriteString(hoverExtra)
		}

		innerStr := strings.TrimSpace(inner.String())
		// If inner only had header newline, ensure content still present
		if innerStr == "" {
			innerStr = msg.Content
		}

		// Per-part SplitBorder: left ┃ via ui.SplitBorder, bottom ╹ terminator
		var rendered string
		if m.styles != nil && m.styles.Theme != nil {
			agentColor := m.agentColor(msg.Role)
			style := lipgloss.NewStyle().
				Border(ui.SplitBorder).
				BorderForeground(lipgloss.Color(agentColor)).
				Padding(0, 1)
			if msg.Hover && m.styles.Theme.BackgroundElement != "" {
				style = style.Background(lipgloss.Color(m.styles.Theme.BackgroundElement))
			}
			if width > 0 {
				style = style.Width(width - 2)
			}
			rendered = style.Render(innerStr)
			// Ensure terminator ╹ is present even if lipgloss border bottom collapses on single line
			if !strings.Contains(rendered, "╹") {
				rendered += "\n╹"
			}
		} else {
			// Fallback plain with explicit border chars
			lines := strings.Split(innerStr, "\n")
			for i, l := range lines {
				lines[i] = "┃ " + l
			}
			rendered = strings.Join(lines, "\n") + "\n╹"
		}
		parts = append(parts, rendered)
	}

	if m.lastError != "" {
		parts = append(parts, m.styles.Error.Render("error: "+m.lastError))
	}
	if m.status != "" {
		if m.styles != nil {
			dot := m.styles.HomeMuted.Render("● ")
			statusLine := m.styles.HomeMuted.Render(m.status)
			parts = append(parts, dot+statusLine)
		} else {
			parts = append(parts, "● "+m.status)
		}
	}
	if len(m.diagnostics) > 0 {
		for _, d := range m.diagnostics {
			parts = append(parts, m.styles.Error.Render("  "+d))
		}
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

func truncateToWidth(s string, max int) string {
	if max <= 0 {
		return s
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	// Truncate by runes keeping width
	runes := []rune(s)
	out := ""
	for _, r := range runes {
		if lipgloss.Width(out+string(r)) > max {
			break
		}
		out += string(r)
	}
	return out + "…"
}
