package theme

import "github.com/charmbracelet/lipgloss"

// Styles holds pre-computed lipgloss styles for a theme.
// Create once with NewStyles(), then pass to all views.
type Styles struct {
	// Chat view
	UserRole      lipgloss.Style
	AssistantRole lipgloss.Style
	Profile       lipgloss.Style
	Error         lipgloss.Style
	EmptyHint     lipgloss.Style

	// Header view
	ActiveTab   lipgloss.Style
	InactiveTab lipgloss.Style
	Hint        lipgloss.Style

	// Tool view
	ToolName    lipgloss.Style
	ToolResult  lipgloss.Style
	ToolPending lipgloss.Style
	ToolEmpty   lipgloss.Style

	// Status bar (footer)
	StatusLine  lipgloss.Style
	StatusOK    lipgloss.Style
	StatusError lipgloss.Style
	StatusWarn  lipgloss.Style

	// Diff view
	FileDiff    lipgloss.Style
	DiffAdded   lipgloss.Style
	DiffRemoved lipgloss.Style
	DiffContext lipgloss.Style
	DiffHunk    lipgloss.Style

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

		// Panel — rounded bordered panel uses backgroundPanel
		Panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(t.BorderSubtle)).
			Background(lipgloss.Color(t.BackgroundPanel)).
			Padding(0, 1),

		// Popup — centered overlay for palette, BackgroundPanel with subtle border
		Popup: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(t.BorderSubtle)).
			Background(lipgloss.Color(t.BackgroundPanel)).
			Padding(1, 1),

		// Sidebar — dark panel for opencode right sidebar uses backgroundPanel
		Sidebar: lipgloss.NewStyle().
			Background(lipgloss.Color(t.BackgroundPanel)).
			Foreground(lipgloss.Color(t.TextMuted)).
			Padding(0, 1),

		// InputBar — full-width bar with backgroundElement and primary accent
		InputBar: lipgloss.NewStyle().
			Background(lipgloss.Color(t.BackgroundElement)).
			Foreground(lipgloss.Color(t.FG)).
			Padding(0, 1),

		InputBarAccent: lipgloss.NewStyle().
			Border(lipgloss.Border{Left: "▏"}, false, false, false, true).
			BorderForeground(lipgloss.Color(t.Primary)).
			Background(lipgloss.Color(t.BackgroundElement)).
			Padding(0, 1),

		// CodeBlock — backgroundElement for fenced code
		CodeBlock: lipgloss.NewStyle().
			Background(lipgloss.Color(t.BackgroundElement)).
			Foreground(lipgloss.Color(t.FG)).
			Padding(0, 1),

		// Thought — warning color for "Thought:" prefix
		Thought: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Warning)).
			Bold(true),
	}
}
