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
