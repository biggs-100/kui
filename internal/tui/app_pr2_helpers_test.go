package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// newSessionApp builds an App at the given terminal size using the standard
// test controller (profile "coder", no runner/resolver). A fixed workspace KV
// keeps dumps deterministic across machines; the real os.Getwd fallback is
// covered by footer/app tests.
func newSessionApp(t *testing.T, width, height int) *App {
	t.Helper()
	c := NewController([]string{"coder"}, nil, nil)
	c.SetKV("workspace", "~/dev-biggz/kui")
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: width, Height: height})
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
