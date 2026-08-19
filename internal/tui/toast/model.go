package toast

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/biggs-100/kui/internal/tui/theme"
)

// Level represents the severity of a toast notification.
type Level int

const (
	LevelInfo    Level = iota // informational feedback
	LevelSuccess              // success confirmation
	LevelWarn                 // warning
	LevelError                // error condition
)

// Toast is a single notification with text, level, and auto-dismiss duration.
type Toast struct {
	Text     string
	Level    Level
	Duration time.Duration
	created  time.Time
}

// TickMsg is an exported message used to schedule dismiss checks.
// Send this via tea.Cmd or directly in tests.
type TickMsg struct{}

// Model manages a queue of toast notifications, rendering them as an overlay.
type Model struct {
	toasts []Toast
	styles *theme.Styles
}

// NewModel creates a toast Model with the given styles.
func NewModel(styles *theme.Styles) *Model {
	return &Model{styles: styles}
}

// Push adds a toast to the queue.
func (m *Model) Push(text string, level Level, duration time.Duration) {
	m.toasts = append(m.toasts, Toast{
		Text:     text,
		Level:    level,
		Duration: duration,
		created:  time.Now(),
	})
}

// Toasts returns the current toast slice (for testing).
func (m *Model) Toasts() []Toast {
	return m.toasts
}

// Update handles tick messages and dismisses expired toasts.
func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg.(type) {
	case TickMsg:
		now := time.Now()
		var active []Toast
		for _, t := range m.toasts {
			if t.Duration > 0 && now.Sub(t.created) < t.Duration {
				active = append(active, t)
			}
			// Duration == 0: expired immediately on first tick
		}
		m.toasts = active
	}
	return m, nil
}

// View renders the active toasts as a styled overlay, or empty string if none.
func (m *Model) View() string {
	if len(m.toasts) == 0 {
		return ""
	}

	var b strings.Builder
	for i, t := range m.toasts {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(m.renderToast(t))
	}
	return b.String()
}

// renderToast styles a single toast based on its level.
func (m *Model) renderToast(t Toast) string {
	switch t.Level {
	case LevelInfo:
		return m.styles.StatusLine.Render("ℹ " + t.Text)
	case LevelSuccess:
		return m.styles.StatusOK.Render("✔ " + t.Text)
	case LevelWarn:
		return m.styles.StatusWarn.Render("⚠ " + t.Text)
	case LevelError:
		return m.styles.StatusError.Render("✖ " + t.Text)
	default:
		return m.styles.StatusLine.Render("● " + t.Text)
	}
}
