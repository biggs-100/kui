package theme

import (
	"encoding/json"
	"testing"
)

// TestTheme40Fields_OpenCode tests REQ-TUI-THEME-1: Theme has all OpenCode fields.
func TestTheme40Fields_OpenCode(t *testing.T) {
	th := OpenCode()
	if th == nil {
		t.Fatal("OpenCode() returned nil")
	}
	// Check 40+ fields are non-empty
	missing := []string{}
	check := func(name, val string) {
		if val == "" {
			missing = append(missing, name)
		}
	}
	// Base
	check("BG", th.BG)
	check("FG", th.FG)
	check("Border", th.Border)
	check("BorderActive", th.BorderActive)
	check("BorderSubtle", th.BorderSubtle)
	// Semantic
	check("Primary", th.Primary)
	check("Secondary", th.Secondary)
	check("Accent", th.Accent)
	check("Error", th.Error)
	check("Warning", th.Warning)
	check("Success", th.Success)
	check("Info", th.Info)
	check("Text", th.Text)
	check("TextMuted", th.TextMuted)
	check("SelectedListItemText", th.SelectedListItemText)
	// Background tokens
	check("Background", th.Background)
	check("BackgroundPanel", th.BackgroundPanel)
	check("BackgroundElement", th.BackgroundElement)
	check("BackgroundMenu", th.BackgroundMenu)
	// Diff
	check("DiffAdded", th.DiffAdded)
	check("DiffRemoved", th.DiffRemoved)
	check("DiffContext", th.DiffContext)
	check("DiffHunkHeader", th.DiffHunkHeader)
	check("DiffHighlight", th.DiffHighlight)
	check("DiffAddedBg", th.DiffAddedBg)
	check("DiffRemovedBg", th.DiffRemovedBg)
	check("DiffContextBg", th.DiffContextBg)
	check("DiffLineNumber", th.DiffLineNumber)
	check("DiffLineNumberBg", th.DiffLineNumberBg)
	// Markdown 10
	check("MarkdownText", th.MarkdownText)
	check("MarkdownHeading", th.MarkdownHeading)
	check("MarkdownLink", th.MarkdownLink)
	check("MarkdownLinkText", th.MarkdownLinkText)
	check("MarkdownCode", th.MarkdownCode)
	check("MarkdownBlockQuote", th.MarkdownBlockQuote)
	check("MarkdownEmph", th.MarkdownEmph)
	check("MarkdownStrong", th.MarkdownStrong)
	check("MarkdownHRule", th.MarkdownHRule)
	check("MarkdownListItem", th.MarkdownListItem)
	// Syntax 9
	check("SyntaxComment", th.SyntaxComment)
	check("SyntaxKeyword", th.SyntaxKeyword)
	check("SyntaxFunction", th.SyntaxFunction)
	check("SyntaxVariable", th.SyntaxVariable)
	check("SyntaxString", th.SyntaxString)
	check("SyntaxNumber", th.SyntaxNumber)
	check("SyntaxType", th.SyntaxType)
	check("SyntaxOperator", th.SyntaxOperator)
	check("SyntaxPunctuation", th.SyntaxPunctuation)
	// Thinking
	if th.ThinkingOpacity == 0 {
		missing = append(missing, "ThinkingOpacity")
	}
	if len(missing) > 0 {
		t.Errorf("OpenCode() missing %d fields: %v", len(missing), missing)
	}
	// Count non-empty should be >=40
	count := 0
	for _, v := range []string{
		th.Primary, th.Secondary, th.Accent, th.Error, th.Warning, th.Success, th.Info,
		th.Text, th.TextMuted, th.SelectedListItemText, th.Background, th.BackgroundPanel,
		th.BackgroundElement, th.BackgroundMenu, th.Border, th.BorderActive, th.BorderSubtle,
		th.DiffAdded, th.DiffRemoved, th.DiffContext, th.DiffHunkHeader, th.DiffHighlight,
		th.DiffAddedBg, th.DiffRemovedBg, th.DiffContextBg, th.DiffLineNumber, th.DiffLineNumberBg,
		th.MarkdownText, th.MarkdownHeading, th.MarkdownLink, th.MarkdownLinkText, th.MarkdownCode,
		th.MarkdownBlockQuote, th.MarkdownEmph, th.MarkdownStrong, th.MarkdownHRule, th.MarkdownListItem,
		th.SyntaxComment, th.SyntaxKeyword, th.SyntaxFunction, th.SyntaxVariable, th.SyntaxString,
		th.SyntaxNumber, th.SyntaxType, th.SyntaxOperator, th.SyntaxPunctuation,
	} {
		if v != "" {
			count++
		}
	}
	if th.ThinkingOpacity != 0 {
		count++
	}
	if count < 40 {
		t.Errorf("expected >=40 non-empty fields, got %d", count)
	}
}

// TestThemeJSONRoundTrip tests JSON round-trip for 40+ fields.
func TestThemeJSONRoundTrip(t *testing.T) {
	orig := OpenCode()
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	var decoded Theme
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if decoded.Primary != orig.Primary {
		t.Errorf("Primary round-trip: got %q want %q", decoded.Primary, orig.Primary)
	}
	if decoded.BackgroundPanel != orig.BackgroundPanel {
		t.Errorf("BackgroundPanel round-trip: got %q want %q", decoded.BackgroundPanel, orig.BackgroundPanel)
	}
	if decoded.MarkdownHeading != orig.MarkdownHeading {
		t.Errorf("MarkdownHeading round-trip: got %q want %q", decoded.MarkdownHeading, orig.MarkdownHeading)
	}
	if decoded.SyntaxOperator != orig.SyntaxOperator {
		t.Errorf("SyntaxOperator round-trip: got %q want %q", decoded.SyntaxOperator, orig.SyntaxOperator)
	}
	if decoded.ThinkingOpacity != orig.ThinkingOpacity {
		t.Errorf("ThinkingOpacity round-trip: got %v want %v", decoded.ThinkingOpacity, orig.ThinkingOpacity)
	}
}

// TestParseBytes_OpenCodeJSON tests loading opencode.json-like bytes.
func TestParseBytes_OpenCodeJSON(t *testing.T) {
	data := []byte(`{
		"name":"opencode",
		"bg":"#1a1a1a",
		"background":"#1a1a1a",
		"background_panel":"#252525",
		"background_element":"#2a2a2a",
		"background_menu":"#252525",
		"primary":"#569cd6",
		"secondary":"#4ec9b0",
		"accent":"#569cd6",
		"error":"#f44747",
		"warning":"#e0af68",
		"success":"#4ec9b0",
		"info":"#569cd6",
		"text":"#e0e0e0",
		"text_muted":"#808080",
		"selected_list_item_text":"#e0e0e0",
		"border":"#333333",
		"border_active":"#569cd6",
		"border_subtle":"#333333",
		"diff_added":"#4ec9b0",
		"diff_removed":"#f44747",
		"diff_context":"#808080",
		"diff_hunk_header":"#569cd6",
		"diff_highlight":"#569cd6",
		"diff_added_bg":"#1e3a2a",
		"diff_removed_bg":"#3a1e1e",
		"diff_context_bg":"#252525",
		"diff_line_number":"#808080",
		"diff_line_number_bg":"#252525",
		"markdown_text":"#e0e0e0",
		"markdown_heading":"#569cd6",
		"markdown_link":"#569cd6",
		"markdown_link_text":"#4ec9b0",
		"markdown_code":"#ce9178",
		"markdown_block_quote":"#808080",
		"markdown_emph":"#e0af68",
		"markdown_strong":"#e0e0e0",
		"markdown_h_rule":"#333333",
		"markdown_list_item":"#e0e0e0",
		"syntax_comment":"#808080",
		"syntax_keyword":"#c586c0",
		"syntax_function":"#dcdcaa",
		"syntax_variable":"#9cdcfe",
		"syntax_string":"#ce9178",
		"syntax_number":"#b5cea8",
		"syntax_type":"#4ec9b0",
		"syntax_operator":"#569cd6",
		"syntax_punctuation":"#808080",
		"thinking_opacity":0.6
	}`)
	th, err := ParseBytes(data)
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}
	if th.BackgroundPanel != "#252525" {
		t.Errorf("BackgroundPanel = %q, want #252525", th.BackgroundPanel)
	}
	if th.DiffAddedBg != "#1e3a2a" {
		t.Errorf("DiffAddedBg = %q, want #1e3a2a", th.DiffAddedBg)
	}
	if th.MarkdownCode != "#ce9178" {
		t.Errorf("MarkdownCode = %q, want #ce9178", th.MarkdownCode)
	}
	if th.SyntaxOperator != "#569cd6" {
		t.Errorf("SyntaxOperator = %q, want #569cd6", th.SyntaxOperator)
	}
	if th.ThinkingOpacity != 0.6 {
		t.Errorf("ThinkingOpacity = %v, want 0.6", th.ThinkingOpacity)
	}
	// Ensure no field lost: re-marshal and check hex equality
	out, _ := json.Marshal(th)
	var m map[string]interface{}
	_ = json.Unmarshal(out, &m)
	if m["background_panel"] != "#252525" {
		t.Errorf("re-marshaled background_panel mismatch: %v", m["background_panel"])
	}
}
