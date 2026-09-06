package tui

import (
	"os"
	"path/filepath"
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
	// Hermetic git: footer renders live `gitBranch()` at View() time
	// (rebuildViews refreshes it on every render), which would lock the
	// current branch name into goldens (REQ-TUI-APP-10 locks layout, not
	// dynamic git state). Ceiling discovery above the package dir (the
	// ceiling itself must be a parent: git never excludes the cwd) so
	// `git rev-parse` fails and the branch is omitted (never fabricated).
	if wd, err := os.Getwd(); err == nil {
		t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(wd))
	}
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
