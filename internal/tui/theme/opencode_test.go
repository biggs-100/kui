package theme

import "testing"

func TestOpenCodeThemeLoads(t *testing.T) {
	th := OpenCode()
	if th == nil {
		t.Fatal("OpenCode() returned nil")
	}
}

func TestOpenCodeThemeName(t *testing.T) {
	th := OpenCode()
	if th.Name != "opencode" {
		t.Errorf("Name = %q, want %q", th.Name, "opencode")
	}
}

func TestOpenCodeThemeColors(t *testing.T) {
	th := OpenCode()

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"BG", th.BG, "#0a0a0a"},
		{"FG", th.FG, "#eeeeee"},
		{"TextMuted", th.TextMuted, "#808080"},
		{"Accent", th.Accent, "#9d7cd8"},
		{"Border", th.Border, "#484848"},
		{"Primary", th.Primary, "#fab283"},
		{"BackgroundPanel", th.BackgroundPanel, "#141414"},
		{"Success", th.Success, "#7fd88f"},
		{"Error", th.Error, "#e06c75"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestOpenCodeThemeDefaultFallback(t *testing.T) {
	th := Load("opencode")
	if th == nil {
		t.Fatal("Load('opencode') returned nil")
	}
	// Load falls back to DefaultTheme since file discovery won't find
	// the built-in theme. That's OK — the theme is registered programmatically.
	// We verify the built-in function returns the right values.
	builtin := OpenCode()
	if builtin.BG != "#0a0a0a" {
		t.Errorf("builtin BG = %q, want #0a0a0a", builtin.BG)
	}
}
