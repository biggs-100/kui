package theme

import (
	"testing"
)

func TestParseBytes(t *testing.T) {
	data := []byte(`{
		"name": "test-theme",
		"bg": "#000000",
		"bg_highlight": "#111111",
		"bg_popup": "#000000",
		"bg_statusline": "#111111",
		"bg_sidebar": "#000000",
		"bg_float": "#000000",
		"fg": "#ffffff",
		"fg_float": "#ffffff",
		"border": "#111111",
		"border_active": "#00ff00",
		"border_subtle": "#111111",
		"primary": "#00ff00",
		"secondary": "#00ffff",
		"accent": "#ffff00",
		"error": "#ff0000",
		"warning": "#ffff00",
		"success": "#00ff00",
		"info": "#00ff00",
		"hint": "#00ffff",
		"text": "#ffffff",
		"text_muted": "#888888",
		"text_faint": "#444444",
		"tab_active": "#ffff00",
		"tab_inactive": "#888888",
		"tab_active_bg": "#111111",
		"user_label": "#00ff00",
		"assistant_label": "#888888",
		"profile_text": "#888888",
		"tool_name": "#00ff00",
		"tool_result": "#ffffff",
		"tool_pending": "#ff8800",
		"status_ok": "#00ff00",
		"status_error": "#ff0000",
		"status_warn": "#ffff00",
		"diff_added": "#00ff00",
		"diff_removed": "#ff0000",
		"diff_context": "#888888",
		"syntax_comment": "#888888",
		"syntax_keyword": "#ff00ff",
		"syntax_function": "#00ff00",
		"syntax_string": "#00ffff",
		"syntax_number": "#ff8800",
		"syntax_type": "#ffff00",
		"syntax_variable": "#00ff00"
	}`)

	theme, err := ParseBytes(data)
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	if theme.Name != "test-theme" {
		t.Errorf("Name = %q, want %q", theme.Name, "test-theme")
	}
	if theme.BG != "#000000" {
		t.Errorf("BG = %q, want %q", theme.BG, "#000000")
	}
	if theme.FG != "#ffffff" {
		t.Errorf("FG = %q, want %q", theme.FG, "#ffffff")
	}
	if theme.Primary != "#00ff00" {
		t.Errorf("Primary = %q, want %q", theme.Primary, "#00ff00")
	}
	if theme.Error != "#ff0000" {
		t.Errorf("Error = %q, want %q", theme.Error, "#ff0000")
	}
}

func TestParseBytesInvalid(t *testing.T) {
	_, err := ParseBytes([]byte(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseFile(t *testing.T) {
	theme, err := ParseFile("../../../themes/solarized-osaka.json")
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if theme.Name != "solarized-osaka" {
		t.Errorf("Name = %q, want %q", theme.Name, "solarized-osaka")
	}
	if theme.BG != "#002b36" {
		t.Errorf("BG = %q, want %q", theme.BG, "#002b36")
	}
	if theme.Primary != "#268bd2" {
		t.Errorf("Primary = %q, want %q", theme.Primary, "#268bd2")
	}
}

func TestFooterStylesExist(t *testing.T) {
	styles := NewStyles(DefaultTheme())

	// All four footer styles must be non-zero (properly initialized).
	// Render with actual text to verify they produce output.
	if styles.StatusLine.Render("test") == "" {
		t.Error("StatusLine rendered empty")
	}
	if styles.StatusOK.Render("●") == "" {
		t.Error("StatusOK rendered empty")
	}
	if styles.StatusError.Render("●") == "" {
		t.Error("StatusError rendered empty")
	}
	if styles.StatusWarn.Render("●") == "" {
		t.Error("StatusWarn rendered empty")
	}
}

func TestFooterStylesUseThemeColors(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	// StatusOK foreground should use the theme's StatusOK color.
	okFg := styles.StatusOK.GetForeground()
	if okFg == nil {
		t.Error("StatusOK foreground is nil")
	}

	errFg := styles.StatusError.GetForeground()
	if errFg == nil {
		t.Error("StatusError foreground is nil")
	}
}

func TestSolarizedOsakaColors(t *testing.T) {
	theme, err := ParseFile("../../../themes/solarized-osaka.json")
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Verify key solarized-osaka colors
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"bg", theme.BG, "#002b36"},
		{"fg", theme.FG, "#839496"},
		{"primary", theme.Primary, "#268bd2"},
		{"error", theme.Error, "#dc322f"},
		{"success", theme.Success, "#859900"},
		{"warning", theme.Warning, "#b58900"},
		{"info", theme.Info, "#268bd2"},
		{"hint", theme.Hint, "#2aa198"},
		{"accent", theme.Accent, "#b58900"},
		{"secondary", theme.Secondary, "#2aa198"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}

// TestPiDarkHexes guards the pi-dark port (REQ-TUI-THEME-1): every field
// MUST equal the design color mapping resolved from pi dark.json exactly.
// Hex comparison is case-insensitive (lipgloss treats hex case-insensitively).
func TestPiDarkHexes(t *testing.T) {
	th := Load("pi-dark")
	if th == nil {
		t.Fatal("Load(pi-dark) returned nil")
	}
	if th.Name != "pi-dark" {
		t.Errorf("Name = %q, want %q", th.Name, "pi-dark")
	}
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"BG", th.BG, "#18181e"},
		{"Background", th.Background, "#18181e"},
		{"BGStatusline", th.BGStatusline, "#18181e"},
		{"BGSidebar", th.BGSidebar, "#18181e"},
		{"BGHighlight", th.BGHighlight, "#1e1e24"},
		{"BackgroundPanel", th.BackgroundPanel, "#1e1e24"},
		{"BackgroundElement", th.BackgroundElement, "#1e1e24"},
		{"BackgroundMenu", th.BackgroundMenu, "#1e1e24"},
		{"BGPopup", th.BGPopup, "#1e1e24"},
		{"BGFloat", th.BGFloat, "#1e1e24"},
		{"Border", th.Border, "#5f87ff"},
		{"BorderActive", th.BorderActive, "#00d7ff"},
		{"BorderSubtle", th.BorderSubtle, "#505050"},
		{"Primary", th.Primary, "#00d7ff"},
		{"Secondary", th.Secondary, "#81a2be"},
		{"Info", th.Info, "#81a2be"},
		{"Accent", th.Accent, "#8abeb7"},
		{"Success", th.Success, "#b5bd68"},
		{"StatusOK", th.StatusOK, "#b5bd68"},
		{"Error", th.Error, "#cc6666"},
		{"StatusError", th.StatusError, "#cc6666"},
		{"Warning", th.Warning, "#ffff00"},
		{"StatusWarn", th.StatusWarn, "#ffff00"},
		{"ToolPending", th.ToolPending, "#ffff00"},
		{"Hint", th.Hint, "#666666"},
		{"Text", th.Text, "#d4d4d4"},
		{"FG", th.FG, "#d4d4d4"},
		{"FGFloat", th.FGFloat, "#d4d4d4"},
		{"SelectedListItemText", th.SelectedListItemText, "#d4d4d4"},
		{"UserLabel", th.UserLabel, "#d4d4d4"},
		{"ToolName", th.ToolName, "#d4d4d4"},
		{"TextMuted", th.TextMuted, "#808080"},
		{"AssistantLabel", th.AssistantLabel, "#808080"},
		{"ProfileText", th.ProfileText, "#808080"},
		{"ToolResult", th.ToolResult, "#808080"},
		{"TextFaint", th.TextFaint, "#666666"},
		{"TabInactive", th.TabInactive, "#666666"},
		{"TabActive", th.TabActive, "#808080"},
		{"TabActiveBG", th.TabActiveBG, "#1e1e24"},
		{"DiffAdded", th.DiffAdded, "#b5bd68"},
		{"DiffRemoved", th.DiffRemoved, "#cc6666"},
		{"DiffContext", th.DiffContext, "#808080"},
		{"DiffHunkHeader", th.DiffHunkHeader, "#8abeb7"},
		{"DiffHighlight", th.DiffHighlight, "#f0c674"},
		{"DiffAddedBg", th.DiffAddedBg, "#283228"},
		{"DiffRemovedBg", th.DiffRemovedBg, "#3c2828"},
		{"DiffContextBg", th.DiffContextBg, "#282832"},
		{"DiffLineNumber", th.DiffLineNumber, "#666666"},
		{"DiffLineNumberBg", th.DiffLineNumberBg, "#18181e"},
		{"MarkdownText", th.MarkdownText, "#d4d4d4"},
		{"MarkdownHeading", th.MarkdownHeading, "#f0c674"},
		{"MarkdownLink", th.MarkdownLink, "#81a2be"},
		{"MarkdownLinkText", th.MarkdownLinkText, "#666666"},
		{"MarkdownCode", th.MarkdownCode, "#8abeb7"},
		{"MarkdownBlockQuote", th.MarkdownBlockQuote, "#808080"},
		{"MarkdownHRule", th.MarkdownHRule, "#808080"},
		{"MarkdownListItem", th.MarkdownListItem, "#8abeb7"},
		{"MarkdownEmph", th.MarkdownEmph, "#d4d4d4"},
		{"MarkdownStrong", th.MarkdownStrong, "#d4d4d4"},
		{"SyntaxComment", th.SyntaxComment, "#6a9955"},
		{"SyntaxKeyword", th.SyntaxKeyword, "#569cd6"},
		{"SyntaxFunction", th.SyntaxFunction, "#dcdcaa"},
		{"SyntaxVariable", th.SyntaxVariable, "#9cdcfe"},
		{"SyntaxString", th.SyntaxString, "#ce9178"},
		{"SyntaxNumber", th.SyntaxNumber, "#b5cea8"},
		{"SyntaxType", th.SyntaxType, "#4ec9b0"},
		{"SyntaxOperator", th.SyntaxOperator, "#d4d4d4"},
		{"SyntaxPunctuation", th.SyntaxPunctuation, "#d4d4d4"},
		{"UserMessageBg", th.UserMessageBg, "#343541"},
		{"ToolPendingBg", th.ToolPendingBg, "#282832"},
		{"ToolSuccessBg", th.ToolSuccessBg, "#283228"},
		{"ToolErrorBg", th.ToolErrorBg, "#3c2828"},
		{"CustomMessageBg", th.CustomMessageBg, "#2d2838"},
		{"SelectedBg", th.SelectedBg, "#3a3a4a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !equalHex(tt.got, tt.expected) {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
	if th.ThinkingOpacity != 0.6 {
		t.Errorf("ThinkingOpacity = %v, want 0.6", th.ThinkingOpacity)
	}
}

func equalHex(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'F' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'F' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// TestPiDarkIsDefault guards REQ-TUI-THEME-1: no override means pi-dark.
func TestPiDarkIsDefault(t *testing.T) {
	th := Load("")
	if th == nil {
		t.Fatal("Load empty returned nil")
	}
	if th.Name != "pi-dark" {
		t.Errorf("default theme Name = %q, want %q", th.Name, "pi-dark")
	}
}

// TestThemeSwitchKeepsWorking guards REQ-TUI-THEME-6: pi-dark plus another
// theme load and apply without panic.
func TestThemeSwitchKeepsWorking(t *testing.T) {
	pi := Load("pi-dark")
	other := Load("opencode")
	if pi == nil || other == nil {
		t.Fatal("pi-dark and opencode themes must both load")
	}
	for _, th := range []*Theme{pi, other} {
		s := NewStyles(th)
		if s.Theme != th {
			t.Error("NewStyles did not keep theme reference")
		}
		if s.StatusLine.Render("x") == "" {
			t.Errorf("theme %q StatusLine rendered empty", th.Name)
		}
	}
	if Load("nonexistent-theme-xyz").Name == "nonexistent-theme-xyz" {
		t.Error("unknown theme name must not become active; want fallback")
	}
}
