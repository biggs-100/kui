package tui

// PR4 integration coverage (tasks 5.2): width-60 survival, overlay
// centering, viewport stick-to-bottom, and TAB→footer profile surfacing.
//
// NOTE: the task plan calls for a teatest integration, but teatest
// (charmbracelet/x/exp/teatest) is not vendored in go.mod and adding a
// dependency is outside the PR4 surface. These four scenarios drive the
// App directly through Update/View — same assertions, no new deps.

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// typeRunes feeds printable runes through Update like a real keyboard.
func typeRunes(t *testing.T, app *App, s string) *App {
	t.Helper()
	for _, r := range s {
		msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = msg.(*App)
	}
	return app
}

// submitPrompt types s and presses Enter (nil-runner submit is a no-op
// backend-side but still appends the user message to the transcript).
func submitPrompt(t *testing.T, app *App, s string) *App {
	t.Helper()
	app = typeRunes(t, app, s)
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return msg.(*App)
}

// TestTeatestNarrow60Survives: at 60 cols the frame reflows without panic
// or overflow; transcript, editor and footer all fit (REQ-TUI-APP-2).
func TestTeatestNarrow60Survives(t *testing.T) {
	app := newSessionApp(t, 60, 20)
	app = submitPrompt(t, app, "hello narrow")
	got := stripTitleSequence(app.View())
	if got == "" {
		t.Fatal("expected non-empty view at 60 cols")
	}
	for i, line := range strings.Split(got, "\n") {
		if w := lipgloss.Width(line); w > 61 {
			t.Errorf("line %d visible width %d exceeds 60 (+1 tolerance): %q", i, w, line)
		}
	}
	if !strings.Contains(got, "hello narrow") {
		t.Error("view should contain the submitted prompt at 60 cols")
	}
	if !strings.Contains(got, "Ask kui...") {
		t.Error("view should contain the bordered editor at 60 cols")
	}
}

// TestTeatestOverlayCentered: the palette renders as a centered dialog —
// the box is inset from both edges, never fullscreen (REQ-TUI-DLG-1).
func TestTeatestOverlayCentered(t *testing.T) {
	app := newSessionApp(t, 120, 30)
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	a := msg.(*App)
	if !a.paletteMode {
		t.Fatal("palette should be open after Ctrl+P")
	}
	got := stripTitleSequence(a.View())
	found := false
	for _, line := range strings.Split(got, "\n") {
		plain := stripVisibleANSI(line)
		idx := strings.Index(plain, "╭")
		if idx < 0 {
			continue
		}
		found = true
		left := lipgloss.Width(plain[:idx])
		if left == 0 {
			t.Errorf("dialog box is left-docked, want centered overlay: %q", plain)
		}
		if end := strings.LastIndex(plain, "╮"); end >= 0 {
			right := lipgloss.Width(plain[end+len("╮"):])
			if right == 0 {
				t.Errorf("dialog box is right-docked, want centered overlay: %q", plain)
			}
			if d := left - right; d < -2 || d > 2 {
				t.Errorf("dialog off-center: left pad %d, right pad %d: %q", left, right, plain)
			}
		}
		break
	}
	if !found {
		t.Errorf("overlay should render a dialog box, got:\n%s", got)
	}
	for i, line := range strings.Split(got, "\n") {
		if w := lipgloss.Width(line); w > 121 {
			t.Errorf("overlay line %d width %d exceeds 120 (+1): %q", i, w, line)
		}
	}
}

// TestTeatestStickBottom: with more transcript than the viewport holds,
// the view tail-follows — the latest prompt stays visible (REQ-TUI-APP-2).
func TestTeatestStickBottom(t *testing.T) {
	app := newSessionApp(t, 80, 14)
	for _, p := range []string{"first", "second", "third", "fourth", "fifth-prompt"} {
		app = submitPrompt(t, app, p)
	}
	got := stripTitleSequence(app.View())
	if !strings.Contains(got, "fifth-prompt") {
		t.Errorf("viewport should stick to bottom; latest prompt must be visible, got:\n%s", got)
	}
}

// TestTeatestTabSurfacesInFooter: TAB cycles the profile and the footer
// advertises the switcher while several profiles exist (REQ-TUI-APP-6).
func TestTeatestTabSurfacesInFooter(t *testing.T) {
	c := NewController([]string{"coder", "writer"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyTab})
	a := msg.(*App)
	if got := a.ctrl.ActiveProfile(); got != "writer" {
		t.Fatalf("active profile after TAB = %q, want writer", got)
	}
	if title := a.Title(); title != "kui | writer" {
		t.Errorf("header title = %q, want %q", title, "kui | writer")
	}
	a.rebuildViews()
	if foot := a.footer.Render(); !strings.Contains(foot, "TAB") {
		t.Errorf("footer with several profiles should hint TAB, got: %q", foot)
	}
}
