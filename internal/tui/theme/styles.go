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

	// Status bar (footer)
	StatusLine   lipgloss.Style
	StatusOK     lipgloss.Style
	StatusError  lipgloss.Style
	StatusWarn   lipgloss.Style

	// Diff view
	FileDiff     lipgloss.Style
	DiffAdded    lipgloss.Style
	DiffRemoved  lipgloss.Style
	DiffContext  lipgloss.Style
	DiffHunk     lipgloss.Style

	// Home screen
	LogoAccent lipgloss.Style
	HomeBorder lipgloss.Style
	HomeMuted  lipgloss.Style

	// Panel / popup (opencode session)
	Panel lipgloss.Style
	Popup lipgloss.Style

	// Opencode session sidebar + input bar
	Sidebar        lipgloss.Style
	InputBar       lipgloss.Style
	InputBarAccent lipgloss.Style
	CodeBlock      lipgloss.Style
	Thought        lipgloss.Style
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

		// Header — opencode minimal: active blue bold no strong BG, inactive dim faint
		ActiveTab: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(t.TabActive)).
			Padding(0, 1),

		InactiveTab: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.TextMuted)).
			Faint(true).
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

		// Status bar (footer)
		StatusLine: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.FG)).
			Background(lipgloss.Color(t.BGStatusline)),

		StatusOK: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.StatusOK)),

		StatusError: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.StatusError)),

		StatusWarn: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.StatusWarn)),

		// Home screen
		LogoAccent: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Accent)).
			Bold(true),

		HomeBorder: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Border)),

		HomeMuted: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.TextMuted)).
			Faint(true),

		// Diff
		FileDiff: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Primary)),

		DiffAdded: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.DiffAdded)),

		DiffRemoved: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.DiffRemoved)),

		DiffContext: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.DiffContext)),

		DiffHunk: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Accent)).
			Bold(true),

		// Panel — rounded bordered panel BGHighlight #252525 Border #333333 for tool/diff
		Panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(t.BorderSubtle)).
			Background(lipgloss.Color(t.BGHighlight)).
			Padding(0, 1),

		// Popup — centered overlay for palette, BGFloat Border subtle
		Popup: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(t.BorderSubtle)).
			Background(lipgloss.Color(t.BGFloat)).
			Padding(1, 1),

		// Sidebar — dark panel for opencode right sidebar (#1a1a1a bg, muted text)
		Sidebar: lipgloss.NewStyle().
			Background(lipgloss.Color(t.BGSidebar)).
			Foreground(lipgloss.Color(t.TextMuted)).
			Padding(0, 1),

		// InputBar — full-width dark gray bar #2a2a2a with left blue accent
		InputBar: lipgloss.NewStyle().
			Background(lipgloss.Color("#2a2a2a")).
			Foreground(lipgloss.Color(t.FG)).
			Padding(0, 1),

		InputBarAccent: lipgloss.NewStyle().
			Border(lipgloss.Border{Left: "▏"}, false, false, false, true).
			BorderForeground(lipgloss.Color("#569cd6")).
			Background(lipgloss.Color("#2a2a2a")).
			Padding(0, 1),

		// CodeBlock — dark #252525 background for fenced code
		CodeBlock: lipgloss.NewStyle().
			Background(lipgloss.Color("#252525")).
			Foreground(lipgloss.Color(t.FG)).
			Padding(0, 1),

		// Thought — orange #e0af68 for "Thought:" prefix
		Thought: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e0af68")).
			Bold(true),

	}
}
