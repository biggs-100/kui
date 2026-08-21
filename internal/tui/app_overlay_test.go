package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// newSessionApp builds a session-route App at the given terminal size using
// the standard test controller (profile "coder", no runner/resolver).
func newSessionApp(t *testing.T, width, height int) *App {
	t.Helper()
	c := NewController([]string{"coder"}, nil, nil)
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
	// fix the narrow path discarded the sidebar entirely.
	for _, marker := range []string{"Workspace", "NotAvailable"} {
		if !strings.Contains(dump, marker) {
			t.Errorf("narrow session dump missing sidebar %q marker:\n%s", marker, dump)
		}
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
