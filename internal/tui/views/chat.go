package views

import (
	"strings"
	"time"

	"github.com/biggs-100/kui/internal/core"
	"github.com/biggs-100/kui/internal/tui/markdown"
	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/biggs-100/kui/internal/tui/ui"
	"github.com/biggs-100/kui/internal/tui/util"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
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
	messages    []Message
	lastError   string
	status      string   // REQ-RELOAD-12: neutral status line
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

// View renders the chat following the upstream message language: user
// prompts are panel blocks behind a left ┃ bar in the agent color, assistant
// answers are naked indented text closed by an ▣ end-cap carrying identity
// (profile · model · locale timestamp). QUEUED badges, hover fills, the
// compaction divider and diagnostics are preserved.
func (m ChatModel) View(width int) string {
	if len(m.messages) == 0 {
		return m.styles.EmptyHint.Render("start a conversation...")
	}
	var parts []string
	for _, msg := range m.messages {
		// Real local shell execution: plain muted block, no border and no
		// agent identity — it is utility output, not a conversation turn.
		if msg.Kind == PartKindShell {
			text := msg.Content
			if m.styles != nil {
				text = m.styles.HomeMuted.Render(text)
			}
			parts = append(parts, text)
			continue
		}

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

		// Hover marker: when Hover true, background lifts to BackgroundElement
		// and a "hover" fallback marker is appended.
		hoverExtra := ""
		if msg.Hover {
			hoverExtra = " hover"
			// marker ensures dump indicates backgroundElement path even in plain text fallback
			if m.styles != nil && m.styles.Theme != nil && m.styles.Theme.BackgroundElement != "" {
				hoverExtra = " hover:" + m.styles.Theme.BackgroundElement
			}
		}

		// Content: assistant via markdown tokens, everything else plain.
		var content string
		if msg.Role == "assistant" {
			content = markdown.Render(msg.Content, m.styles)
		} else {
			content = msg.Content
		}

		switch msg.Role {
		case "user":
			// User prompt: left bar over a panel fill; inline identity is
			// gone — only the QUEUED badge may precede the content.
			var head strings.Builder
			if msg.Queued {
				badge := "QUEUED"
				if m.styles != nil && m.styles.Theme != nil && m.styles.Theme.Warning != "" {
					badge = lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.Theme.Warning)).Bold(true).Render("QUEUED")
				}
				head.WriteString(badge)
				head.WriteString("\n")
			}
			head.WriteString(content)
			head.WriteString(hoverExtra)

			var rendered string
			if m.styles != nil && m.styles.Theme != nil {
				agentColor := m.agentColor(msg.Role)
				style := lipgloss.NewStyle().
					Border(ui.SplitBorder).
					BorderForeground(lipgloss.Color(agentColor)).
					BorderBottom(false).
					Padding(1, 0, 1, 2)
				if !msg.Hover && m.styles.Theme.BackgroundPanel != "" {
					style = style.Background(lipgloss.Color(m.styles.Theme.BackgroundPanel))
				}
				if msg.Hover && m.styles.Theme.BackgroundElement != "" {
					style = style.Background(lipgloss.Color(m.styles.Theme.BackgroundElement))
				}
				if width > 0 {
					style = style.Width(width - 2)
				}
				rendered = style.Render(head.String())
				rendered += "\n" + lipgloss.NewStyle().
					Foreground(lipgloss.Color(agentColor)).
					Render("╹")
			} else {
				lines := strings.Split(head.String(), "\n")
				for i, l := range lines {
					lines[i] = "┃ " + l
				}
				rendered = strings.Join(lines, "\n") + "\n╹"
			}
			parts = append(parts, rendered)

		default:
			// Assistant answer: naked indented text (no border, no fill),
			// closed by an ▣ end-cap with profile · model · timestamp.
			body := content + hoverExtra
			if width > 0 {
				if wrapped := wordwrap.String(body, width-5); wrapped != "" || body == "" {
					body = wrapped
				}
			}
			rendered := indentLines(body, 3)
			capLine := endCap(m, msg)
			if capLine != "" {
				rendered += "\n" + capLine
			}
			parts = append(parts, "\n"+rendered)
		}
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

// endCap renders the ▣ identity line closing an assistant turn: mark in the
// agent color, name in text, model and locale timestamp in muted.
func endCap(m ChatModel, msg Message) string {
	name := msg.Role
	if msg.Profile != "" {
		name = msg.Profile
	}
	if m.styles == nil || m.styles.Theme == nil {
		segs := []string{name}
		if msg.Model != "" {
			segs = append(segs, msg.Model)
		}
		if !msg.Timestamp.IsZero() {
			segs = append(segs, util.TodayTimeOrDateTime(msg.Timestamp, ChatNow()))
		}
		return "▣ " + strings.Join(segs, " · ")
	}
	mark := lipgloss.NewStyle().Foreground(lipgloss.Color(m.agentColor(msg.Role))).Render("▣")
	nameSeg := lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.Theme.Text)).Render(name)
	dot := lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.Theme.TextMuted)).Render(" · ")
	out := mark + " " + nameSeg
	if msg.Model != "" {
		out += dot + lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.Theme.TextMuted)).Render(msg.Model)
	}
	if !msg.Timestamp.IsZero() {
		out += dot + m.styles.HomeMuted.Render(util.TodayTimeOrDateTime(msg.Timestamp, ChatNow()))
	}
	return out
}

// indentLines prefixes every non-empty line of s with n spaces.
func indentLines(s string, n int) string {
	pad := strings.Repeat(" ", n)
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = pad + l
		}
	}
	return strings.Join(lines, "\n")
}
