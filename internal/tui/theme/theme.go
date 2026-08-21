package theme

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/charmbracelet/lipgloss"
)

// Theme defines the color palette for kui's TUI.
type Theme struct {
	// Identity
	Name string `json:"name"`

	// Base colors
	BG           string `json:"bg"`
	BGHighlight  string `json:"bg_highlight"`
	BGPopup      string `json:"bg_popup"`
	BGStatusline string `json:"bg_statusline"`
	BGSidebar    string `json:"bg_sidebar"`
	BGFloat      string `json:"bg_float"`
	FG           string `json:"fg"`
	FGFloat      string `json:"fg_float"`

	// Background tokens (REQ-TUI-THEME-5)
	Background        string `json:"background"`
	BackgroundPanel   string `json:"background_panel"`
	BackgroundElement string `json:"background_element"`
	BackgroundMenu    string `json:"background_menu"`

	// Borders
	Border       string `json:"border"`
	BorderActive string `json:"border_active"`
	BorderSubtle string `json:"border_subtle"`

	// Semantic colors
	Primary   string `json:"primary"`
	Secondary string `json:"secondary"`
	Accent    string `json:"accent"`
	Error     string `json:"error"`
	Warning   string `json:"warning"`
	Success   string `json:"success"`
	Info      string `json:"info"`
	Hint      string `json:"hint"`

	// Text variants
	Text                 string `json:"text"`
	TextMuted            string `json:"text_muted"`
	TextFaint            string `json:"text_faint"`
	SelectedListItemText string `json:"selected_list_item_text"`

	// UI elements
	TabActive      string `json:"tab_active"`
	TabInactive    string `json:"tab_inactive"`
	TabActiveBG    string `json:"tab_active_bg"`
	UserLabel      string `json:"user_label"`
	AssistantLabel string `json:"assistant_label"`
	ProfileText    string `json:"profile_text"`

	// Tool panel
	ToolName    string `json:"tool_name"`
	ToolResult  string `json:"tool_result"`
	ToolPending string `json:"tool_pending"`

	// Status
	StatusOK    string `json:"status_ok"`
	StatusError string `json:"status_error"`
	StatusWarn  string `json:"status_warn"`

	// Diff
	DiffAdded        string `json:"diff_added"`
	DiffRemoved      string `json:"diff_removed"`
	DiffContext      string `json:"diff_context"`
	DiffHunkHeader   string `json:"diff_hunk_header"`
	DiffHighlight    string `json:"diff_highlight"`
	DiffAddedBg      string `json:"diff_added_bg"`
	DiffRemovedBg    string `json:"diff_removed_bg"`
	DiffContextBg    string `json:"diff_context_bg"`
	DiffLineNumber   string `json:"diff_line_number"`
	DiffLineNumberBg string `json:"diff_line_number_bg"`

	// Markdown tokens
	MarkdownText       string `json:"markdown_text"`
	MarkdownHeading    string `json:"markdown_heading"`
	MarkdownLink       string `json:"markdown_link"`
	MarkdownLinkText   string `json:"markdown_link_text"`
	MarkdownCode       string `json:"markdown_code"`
	MarkdownBlockQuote string `json:"markdown_block_quote"`
	MarkdownEmph       string `json:"markdown_emph"`
	MarkdownStrong     string `json:"markdown_strong"`
	MarkdownHRule      string `json:"markdown_h_rule"`
	MarkdownListItem   string `json:"markdown_list_item"`

	// Syntax highlighting
	SyntaxComment     string `json:"syntax_comment"`
	SyntaxKeyword     string `json:"syntax_keyword"`
	SyntaxFunction    string `json:"syntax_function"`
	SyntaxString      string `json:"syntax_string"`
	SyntaxNumber      string `json:"syntax_number"`
	SyntaxType        string `json:"syntax_type"`
	SyntaxVariable    string `json:"syntax_variable"`
	SyntaxOperator    string `json:"syntax_operator"`
	SyntaxPunctuation string `json:"syntax_punctuation"`

	// Thinking
	ThinkingOpacity float64 `json:"thinking_opacity"`
}

// StyleFG returns a lipgloss.Style with the given foreground color.
func (t *Theme) StyleFG(color string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color))
}

// StyleBG returns a lipgloss.Style with the given background color.
func (t *Theme) StyleBG(color string) lipgloss.Style {
	return lipgloss.NewStyle().Background(lipgloss.Color(color))
}

// StyleColors returns a lipgloss.Style with foreground and background colors.
func (t *Theme) StyleColors(fg, bg string) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(fg)).
		Background(lipgloss.Color(bg))
}

// ParseFile loads a theme from a JSON file.
func ParseFile(path string) (*Theme, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read theme file: %w", err)
	}
	return ParseBytes(data)
}

// builtinThemes holds built-in themes that don't require file discovery.
var builtinThemes = map[string]*Theme{
	"kui-default": DefaultTheme(),
	"opencode":    OpenCodeTheme,
}

// DefaultTheme returns the built-in default theme (matches original hardcoded colors).
func DefaultTheme() *Theme {
	return &Theme{
		Name:                 "kui-default",
		BG:                   "#1a1b26",
		BGHighlight:          "#24283b",
		BGPopup:              "#1a1b26",
		BGStatusline:         "#24283b",
		BGSidebar:            "#1a1b26",
		BGFloat:              "#1a1b26",
		FG:                   "#a9b1d6",
		FGFloat:              "#a9b1d6",
		Background:           "#1a1b26",
		BackgroundPanel:      "#24283b",
		BackgroundElement:    "#24283b",
		BackgroundMenu:       "#24283b",
		Border:               "#24283b",
		BorderActive:         "#7aa2f7",
		BorderSubtle:         "#24283b",
		Primary:              "#7aa2f7",
		Secondary:            "#9ece6a",
		Accent:               "#ff9e64",
		Error:                "#f7768e",
		Warning:              "#e0af68",
		Success:              "#9ece6a",
		Info:                 "#7aa2f7",
		Hint:                 "#7dcfff",
		Text:                 "#a9b1d6",
		TextMuted:            "#565f89",
		TextFaint:            "#24283b",
		SelectedListItemText: "#a9b1d6",
		TabActive:            "#ff9e64",
		TabInactive:          "#565f89",
		TabActiveBG:          "#24283b",
		UserLabel:            "#7aa2f7",
		AssistantLabel:       "#565f89",
		ProfileText:          "#565f89",
		ToolName:             "#7aa2f7",
		ToolResult:           "#a9b1d6",
		ToolPending:          "#e0af68",
		StatusOK:             "#9ece6a",
		StatusError:          "#f7768e",
		StatusWarn:           "#e0af68",
		DiffAdded:            "#9ece6a",
		DiffRemoved:          "#f7768e",
		DiffContext:          "#565f89",
		DiffHunkHeader:       "#7aa2f7",
		DiffHighlight:        "#7aa2f7",
		DiffAddedBg:          "#1e3a2a",
		DiffRemovedBg:        "#3a1e1e",
		DiffContextBg:        "#24283b",
		DiffLineNumber:       "#565f89",
		DiffLineNumberBg:     "#24283b",
		MarkdownText:         "#a9b1d6",
		MarkdownHeading:      "#7aa2f7",
		MarkdownLink:         "#7aa2f7",
		MarkdownLinkText:     "#9ece6a",
		MarkdownCode:         "#9ece6a",
		MarkdownBlockQuote:   "#565f89",
		MarkdownEmph:         "#e0af68",
		MarkdownStrong:       "#a9b1d6",
		MarkdownHRule:        "#24283b",
		MarkdownListItem:     "#a9b1d6",
		SyntaxComment:        "#565f89",
		SyntaxKeyword:        "#bb9af7",
		SyntaxFunction:       "#7aa2f7",
		SyntaxString:         "#9ece6a",
		SyntaxNumber:         "#ff9e64",
		SyntaxType:           "#2ac3de",
		SyntaxVariable:       "#a9b1d6",
		SyntaxOperator:       "#7aa2f7",
		SyntaxPunctuation:    "#565f89",
		ThinkingOpacity:      0.6,
	}
}

// ParseBytes parses a theme from JSON bytes.
func ParseBytes(data []byte) (*Theme, error) {
	var t Theme
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parse theme JSON: %w", err)
	}
	// Backward compat: fill new tokens from old fields if empty.
	if t.Background == "" {
		t.Background = t.BG
	}
	if t.BackgroundPanel == "" {
		t.BackgroundPanel = t.BGHighlight
		if t.BackgroundPanel == "" {
			t.BackgroundPanel = t.BGSidebar
		}
	}
	if t.BackgroundElement == "" {
		t.BackgroundElement = t.BGHighlight
	}
	if t.BackgroundMenu == "" {
		t.BackgroundMenu = t.BGHighlight
	}
	if t.SelectedListItemText == "" {
		t.SelectedListItemText = t.Text
		if t.SelectedListItemText == "" {
			t.SelectedListItemText = t.FG
		}
	}
	if t.DiffHunkHeader == "" {
		t.DiffHunkHeader = t.Accent
	}
	if t.DiffHighlight == "" {
		t.DiffHighlight = t.Accent
	}
	if t.DiffAddedBg == "" {
		t.DiffAddedBg = t.BGHighlight
	}
	if t.DiffRemovedBg == "" {
		t.DiffRemovedBg = t.BGHighlight
	}
	if t.DiffContextBg == "" {
		t.DiffContextBg = t.BGHighlight
	}
	if t.DiffLineNumber == "" {
		t.DiffLineNumber = t.TextMuted
	}
	if t.DiffLineNumberBg == "" {
		t.DiffLineNumberBg = t.BGHighlight
	}
	if t.MarkdownText == "" {
		t.MarkdownText = t.Text
	}
	if t.MarkdownHeading == "" {
		t.MarkdownHeading = t.Primary
	}
	if t.MarkdownLink == "" {
		t.MarkdownLink = t.Primary
	}
	if t.MarkdownLinkText == "" {
		t.MarkdownLinkText = t.Secondary
	}
	if t.MarkdownCode == "" {
		t.MarkdownCode = t.SyntaxString
	}
	if t.MarkdownBlockQuote == "" {
		t.MarkdownBlockQuote = t.TextMuted
	}
	if t.MarkdownEmph == "" {
		t.MarkdownEmph = t.Warning
	}
	if t.MarkdownStrong == "" {
		t.MarkdownStrong = t.Text
	}
	if t.MarkdownHRule == "" {
		t.MarkdownHRule = t.Border
	}
	if t.MarkdownListItem == "" {
		t.MarkdownListItem = t.Text
	}
	if t.SyntaxOperator == "" {
		t.SyntaxOperator = t.SyntaxKeyword
	}
	if t.SyntaxPunctuation == "" {
		t.SyntaxPunctuation = t.TextMuted
	}
	if t.ThinkingOpacity == 0 {
		t.ThinkingOpacity = 0.6
	}
	return &t, nil
}

// DefaultDirs returns the default theme discovery directories.
// Order: project .kui/themes → user ~/.config/kui/themes
func DefaultDirs() []string {
	var dirs []string

	// Project themes (cwd)
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, cwd)
	}

	// User config themes
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".config", "kui"))
	}

	return dirs
}

// Load finds a theme by name from built-in themes or the default directories.
// Returns the default theme if name is empty or not found.
func Load(name string) *Theme {
	if name == "" {
		// Default to the OpenCode-style gray theme so the TUI matches OpenCode out of the box.
		return OpenCodeTheme
	}

	// Check built-in themes first
	if t, ok := builtinThemes[name]; ok {
		return t
	}

	themes := Discover(DefaultDirs())
	if t, ok := themes[name]; ok {
		return t
	}

	// Fallback to default
	return DefaultTheme()
}

// List returns all available theme names from the default directories.
func List() []string {
	themes := Discover(DefaultDirs())
	names := make([]string, 0, len(themes)+1)
	names = append(names, "kui-default")
	for name := range themes {
		if name != "kui-default" {
			names = append(names, name)
		}
	}
	return names
}

// ThemeNames returns all available theme names sorted alphabetically.
// This is used by the /theme next|prev command for cycling.
func ThemeNames() []string {
	themes := Discover(DefaultDirs())
	names := make([]string, 0, len(themes)+len(builtinThemes))
	for name := range builtinThemes {
		names = append(names, name)
	}
	for name := range themes {
		if _, builtin := builtinThemes[name]; !builtin {
			names = append(names, name)
		}
	}
	// Sort for deterministic cycling order
	sort.Strings(names)
	return names
}

// Discover finds theme files in the given directories.
// Later directories override earlier ones.
func Discover(dirs []string) map[string]*Theme {
	themes := make(map[string]*Theme)
	for _, dir := range dirs {
		themesFromDir := discoverDir(dir)
		for name, t := range themesFromDir {
			themes[name] = t
		}
	}
	return themes
}

func discoverDir(dir string) map[string]*Theme {
	themes := make(map[string]*Theme)
	entries, err := os.ReadDir(filepath.Join(dir, "themes"))
	if err != nil {
		return themes
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, "themes", entry.Name())
		t, err := ParseFile(path)
		if err != nil {
			continue
		}
		name := t.Name
		if name == "" {
			name = entry.Name()[:len(entry.Name())-5] // strip .json
		}
		themes[name] = t
	}
	return themes
}
