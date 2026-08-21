package theme

import (
	"fmt"
	"testing"
)

func colorString(c interface{}) string {
	return fmt.Sprintf("%v", c)
}

func TestStylesUseTokensNotLiterals(t *testing.T) {
	th := &Theme{
		Background:        "#111111",
		BackgroundPanel:   "#222222",
		BackgroundElement: "#333333",
		BackgroundMenu:    "#444444",
		BG:                "#111111",
		BGHighlight:       "#999999", // distinct to detect misuse
		BGSidebar:         "#888888",
		BGFloat:           "#777777",
		Primary:           "#569cd6",
		Warning:           "#e0af68",
		Border:            "#333333",
		BorderSubtle:      "#222222",
		FG:                "#e0e0e0",
		TextMuted:         "#808080",
		Accent:            "#569cd6",
		DiffAdded:         "#4ec9b0",
		DiffRemoved:       "#f44747",
		DiffContext:       "#808080",
		SyntaxComment:     "#808080",
		SyntaxKeyword:     "#c586c0",
		SyntaxFunction:    "#dcdcaa",
		SyntaxString:      "#ce9178",
		SyntaxNumber:      "#b5cea8",
		SyntaxType:        "#4ec9b0",
		SyntaxVariable:    "#9cdcfe",
		SyntaxOperator:    "#569cd6",
		SyntaxPunctuation: "#808080",
		ThinkingOpacity:   0.6,
		Text:              "#e0e0e0",
		SelectedListItemText: "#ffffff",
		TabActive:         "#569cd6",
		TabInactive:       "#808080",
		TabActiveBG:       "#222222",
		UserLabel:         "#569cd6",
		AssistantLabel:    "#808080",
		ProfileText:       "#808080",
		ToolName:          "#569cd6",
		ToolResult:        "#e0e0e0",
		ToolPending:       "#e0af68",
		StatusOK:          "#4ec9b0",
		StatusError:       "#f44747",
		StatusWarn:        "#e0af68",
		DiffHunkHeader:    "#569cd6",
		DiffHighlight:     "#569cd6",
		DiffAddedBg:       "#1e3a2a",
		DiffRemovedBg:     "#3a1e1e",
		DiffContextBg:     "#222222",
		DiffLineNumber:    "#808080",
		DiffLineNumberBg:  "#222222",
		MarkdownText:      "#e0e0e0",
		MarkdownHeading:   "#569cd6",
		MarkdownLink:      "#569cd6",
		MarkdownLinkText:  "#4ec9b0",
		MarkdownCode:      "#ce9178",
		MarkdownBlockQuote: "#808080",
		MarkdownEmph:      "#e0af68",
		MarkdownStrong:    "#e0e0e0",
		MarkdownHRule:     "#333333",
		MarkdownListItem:  "#e0e0e0",
		Secondary:         "#4ec9b0",
		Success:           "#4ec9b0",
		Error:             "#f44747",
		Info:              "#569cd6",
		Hint:              "#808080",
		TextFaint:         "#333333",
		FGFloat:           "#e0e0e0",
		BGPopup:           "#111111",
		BGStatusline:      "#222222",
		BorderActive:      "#569cd6",
	}
	styles := NewStyles(th)

	// Panel should use BackgroundPanel, not BGHighlight (#999999)
	if bg := styles.Panel.GetBackground(); colorString(bg) != "#222222" {
		t.Errorf("Panel background = %q, want BackgroundPanel #222222, got %v", colorString(bg), bg)
	}
	// InputBar should use BackgroundElement #333333, not literal #2a2a2a or BGHighlight
	if bg := styles.InputBar.GetBackground(); colorString(bg) != "#333333" {
		t.Errorf("InputBar background = %q, want BackgroundElement #333333", colorString(bg))
	}
	// InputBarAccent border foreground should be Primary #569cd6
	if fg := styles.InputBarAccent.GetBorderLeftForeground(); colorString(fg) != "#569cd6" {
		t.Errorf("InputBarAccent border fg = %q, want Primary #569cd6", colorString(fg))
	}
	// CodeBlock should use BackgroundElement #333333, not #252525
	if bg := styles.CodeBlock.GetBackground(); colorString(bg) != "#333333" {
		t.Errorf("CodeBlock background = %q, want BackgroundElement #333333", colorString(bg))
	}
	// Thought should use Warning #e0af68
	if fg := styles.Thought.GetForeground(); colorString(fg) != "#e0af68" {
		t.Errorf("Thought fg = %q, want Warning #e0af68", colorString(fg))
	}
}
