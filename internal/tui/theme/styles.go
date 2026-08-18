package theme

import "github.com/charmbracelet/lipgloss"

// Styles holds pre-computed lipgloss styles for a theme.
// Create once with NewStyles(), then pass to all views.
type Styles struct {
	// Chat view
	UserRole       lipgloss.Style
	AssistantRole  lipgloss.Style
	Profile        lipgloss.Style
	Error          lipgloss.Style
	EmptyHint      lipgloss.Style

	// Header view
	ActiveTab      lipgloss.Style
	InactiveTab    lipgloss.Style
	Hint           lipgloss.Style

	// Tool view
	ToolName       lipgloss.Style
	ToolResult     lipgloss.Style
	ToolPending    lipgloss.Style
	ToolEmpty      lipgloss.Style
}

// NewStyles creates a Styles from a Theme.
func NewStyles(t *Theme) *Styles {
	return &Styles{
		// Chat
		UserRole: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(t.UserLabel)),

		AssistantRole: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.AssistantLabel)),

		Profile: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.ProfileText)).
			Faint(true),

		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Error)).
			Bold(true),

		EmptyHint: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.TextMuted)).
			Faint(true),

		// Header
		ActiveTab: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(t.TabActive)).
			Background(lipgloss.Color(t.TabActiveBG)).
			Padding(0, 1),

		InactiveTab: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.TabInactive)).
			Padding(0, 1),

		Hint: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.TextMuted)).
			Faint(true),

		// Tool
		ToolName: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(t.ToolName)),

		ToolResult: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.ToolResult)),

		ToolPending: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.ToolPending)).
			Faint(true),

		ToolEmpty: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.TextMuted)).
			Faint(true),
	}
}
