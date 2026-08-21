package tui

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// newSessionApp builds a session-route App at the given terminal size using
// the standard test controller (profile "coder", no runner/resolver). A fixed
// workspace KV keeps golden dumps deterministic across machines; the real
// os.Getwd fallback is covered by TestAppWorkspaceFallbackRealPath.
func newSessionApp(t *testing.T, width, height int) *App {
	t.Helper()
	c := NewController([]string{"coder"}, nil, nil)
	c.SetKV("workspace", "~/dev-biggz/kui")
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: width, Height: height})
	app.route = "session"
	return app
}

// stripTitleSequence removes the leading OSC-0 window-title escape sequence
// so width assertions measure only visible content.
func stripTitleSequence(s string) string {
	if i := strings.Index(s, "\x07"); i >= 0 {
		return s[i+1:]
	}
	return s
}

// TestAppNarrowSidebarOverlayRenders proves REQ-TUI-APP-2 "Narrow overlays
// sidebar": GIVEN width 100 WHEN session renders THEN contentWidth is 96 and
// the sidebar renders as an overlay with backdrop.
func TestAppNarrowSidebarOverlayRenders(t *testing.T) {
	app := newSessionApp(t, 100, 30)
	if got := app.ContentWidth(); got != 96 {
		t.Fatalf("ContentWidth() = %d, want 96", got)
	}
	dump := app.View()
	// Sidebar section markers must appear in narrow mode; before the overlay
	// fix the narrow path discarded the sidebar entirely. The workspace path
	// must be the real configured value, never a fabricated literal.
	for _, marker := range []string{"Session", "~/dev-biggz/kui"} {
		if !strings.Contains(dump, marker) {
			t.Errorf("narrow session dump missing sidebar %q marker:\n%s", marker, dump)
		}
	}
}

// TestAppWorkspaceFallbackRealPath proves REQ-TUI-CHAT-6 workspace truth:
// GIVEN no workspace KV WHEN the session renders THEN the sidebar shows the
// process working directory with the home prefix shortened to ~.
func TestAppWorkspaceFallbackRealPath(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app.route = "session"
	dump := app.View()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd failed: %v", err)
	}
	want := shortenHome(wd)
	if !strings.Contains(dump, want) {
		t.Errorf("sidebar should fall back to real working directory %q, dump:\n%s", want, dump)
	}
}

// TestShortenHome proves the ~ shortening contract with table-driven cases
// built from the real user home (never a hardcoded machine-specific path).
func TestShortenHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("no user home available")
	}
	sep := string(os.PathSeparator)
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"home itself", home, "~"},
		{"direct child", home + sep + "projects", "~" + sep + "projects"},
		{"nested", home + sep + "a" + sep + "b", "~" + sep + "a" + sep + "b"},
		{"outside home unchanged", sep + "opt" + sep + "data", sep + "opt" + sep + "data"},
		{"prefix lookalike unchanged", home + "-suffix", home + "-suffix"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shortenHome(tc.in); got != tc.want {
				t.Errorf("shortenHome(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestAppNarrowSidebarOverlayWidthBounds guards against horizontal overflow
// introduced by the overlay: every rendered line stays within the terminal.
func TestAppNarrowSidebarOverlayWidthBounds(t *testing.T) {
	app := newSessionApp(t, 100, 30)
	dump := stripTitleSequence(app.View())
	for i, line := range strings.Split(dump, "\n") {
		if w := lipgloss.Width(line); w > 100 {
			t.Fatalf("line %d visible width %d exceeds terminal width 100", i, w)
		}
	}
}
