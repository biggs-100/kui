package theme

// OpenCode returns the OpenCode-inspired dark theme.
// Palette: BG #1a1a1a, Text #e0e0e0, Muted #808080,
// Accent #569cd6, Border #333333, Success #4ec9b0, Error #f44747.
func OpenCode() *Theme {
	return &Theme{
		Name:          "opencode",
		BG:            "#1a1a1a",
		BGHighlight:   "#252525",
		BGPopup:       "#1a1a1a",
		BGStatusline:  "#252525",
		BGSidebar:     "#1a1a1a",
		BGFloat:       "#1a1a1a",
		FG:            "#e0e0e0",
		FGFloat:       "#e0e0e0",
		Border:        "#333333",
		BorderActive:  "#569cd6",
		BorderSubtle:  "#333333",
		Primary:       "#569cd6",
		Secondary:     "#4ec9b0",
		Accent:        "#569cd6",
		Error:         "#f44747",
		Warning:       "#e0af68",
		Success:       "#4ec9b0",
		Info:          "#569cd6",
		Hint:          "#808080",
		Text:          "#e0e0e0",
		TextMuted:     "#808080",
		TextFaint:     "#333333",
		TabActive:     "#569cd6",
		TabInactive:   "#808080",
		TabActiveBG:   "#252525",
		UserLabel:     "#569cd6",
		AssistantLabel: "#808080",
		ProfileText:   "#808080",
		ToolName:      "#569cd6",
		ToolResult:    "#e0e0e0",
		ToolPending:   "#e0af68",
		StatusOK:      "#4ec9b0",
		StatusError:   "#f44747",
		StatusWarn:    "#e0af68",
		DiffAdded:     "#4ec9b0",
		DiffRemoved:   "#f44747",
		DiffContext:   "#808080",
		SyntaxComment: "#808080",
		SyntaxKeyword: "#c586c0",
		SyntaxFunction: "#dcdcaa",
		SyntaxString:  "#ce9178",
		SyntaxNumber:  "#b5cea8",
		SyntaxType:    "#4ec9b0",
		SyntaxVariable: "#9cdcfe",
	}
}

// OpenCodeTheme is a convenience alias for registration.
var OpenCodeTheme = OpenCode()
