package theme

import (
	"strings"
	"testing"
)

func TestTintProducesShadow(t *testing.T) {
	bg := "#1a1a1a"
	fg := "#e0e0e0"
	got := Tint(bg, fg, 0.25)
	if got == "" {
		t.Fatal("Tint returned empty")
	}
	if !strings.HasPrefix(got, "#") {
		t.Errorf("Tint = %q, want hex starting with #", got)
	}
	if strings.EqualFold(got, bg) {
		t.Errorf("Tint result %q should be distinct from background %q", got, bg)
	}
	if strings.EqualFold(got, fg) {
		t.Errorf("Tint result %q should be distinct from foreground %q", got, fg)
	}
	// Triangulate: 0 should return bg, 1 should return fg
	if got0 := Tint(bg, fg, 0); !strings.EqualFold(got0, bg) {
		t.Errorf("Tint(bg,fg,0) = %q, want %q", got0, bg)
	}
	if got1 := Tint(bg, fg, 1); !strings.EqualFold(got1, fg) {
		t.Errorf("Tint(bg,fg,1) = %q, want %q", got1, fg)
	}
	// 0.5 should be mid blend distinct
	mid := Tint(bg, fg, 0.5)
	if strings.EqualFold(mid, bg) || strings.EqualFold(mid, fg) {
		t.Errorf("Tint 0.5 should be distinct, got %q", mid)
	}
	if strings.EqualFold(mid, got) {
		t.Errorf("Tint 0.25 (%q) and 0.5 (%q) should differ", got, mid)
	}
}

func TestTintInvalidHexFallsBack(t *testing.T) {
	// Invalid hex should not panic
	got := Tint("nothex", "#ffffff", 0.25)
	if got == "" {
		t.Error("Tint with invalid bg should return fallback, not empty")
	}
}

func TestGetSyntaxRules(t *testing.T) {
	th := OpenCode()
	rules := GetSyntaxRules(th)
	if len(rules) < 9 {
		t.Errorf("GetSyntaxRules returned %d rules, want >=9", len(rules))
	}
	checks := map[string]string{
		"comment":     th.SyntaxComment,
		"keyword":     th.SyntaxKeyword,
		"function":    th.SyntaxFunction,
		"string":      th.SyntaxString,
		"number":      th.SyntaxNumber,
		"type":        th.SyntaxType,
		"variable":    th.SyntaxVariable,
		"operator":    th.SyntaxOperator,
		"punctuation": th.SyntaxPunctuation,
	}
	for k, want := range checks {
		if got := rules[k]; got != want {
			t.Errorf("rule %q = %q, want %q", k, got, want)
		}
	}
}

func TestSelectedForeground(t *testing.T) {
	th := OpenCode()
	// Selected foreground should be either SelectedListItemText or fallback
	got := SelectedForeground(th)
	if got == "" {
		t.Error("SelectedForeground returned empty")
	}
	if got != th.SelectedListItemText && got != th.Text {
		t.Errorf("SelectedForeground = %q, want SelectedListItemText %q or Text %q", got, th.SelectedListItemText, th.Text)
	}
}
