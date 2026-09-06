package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func newMouseApp(t *testing.T) (*App, *[]string) {
	t.Helper()
	app := newSessionApp(t, 100, 30)
	captured := &[]string{}
	app.copySink = func(s string) error {
		*captured = append(*captured, s)
		return nil
	}
	return app, captured
}

// TestSelectionExtractSingleLine proves a same-row drag extracts exactly the
// covered columns from the visible-text snapshot.
func TestSelectionExtractSingleLine(t *testing.T) {
	app, _ := newMouseApp(t)
	app.lastRows = []string{
		strings.Repeat(" ", 100),
		"the quick brown fox jumps over the lazy dog",
		strings.Repeat(" ", 100),
	}
	app.selStartX, app.selStartY = 4, 1
	app.selEndX, app.selEndY = 14, 1 // "quick brown"

	if got := app.extractSelectedText(); got != "quick brown" {
		t.Errorf("extract = %q, want %q", got, "quick brown")
	}
}

// TestSelectionExtractMultiRow proves terminal line semantics: first row
// runs to end-of-line, middle rows fully, last row stops at its column —
// including the upward-drag swap.
func TestSelectionExtractMultiRow(t *testing.T) {
	rows := []string{
		"alpha beta gamma",
		"delta epsilon zeta",
		"eta theta iota",
	}

	app, _ := newMouseApp(t)
	app.lastRows = rows
	// Downward: start row0 col6 ("beta..."), end row2 col9.
	app.selStartX, app.selStartY = 6, 0
	app.selEndX, app.selEndY = 9, 2
	want := "beta gamma\ndelta epsilon zeta\neta theta"
	if got := app.extractSelectedText(); got != want {
		t.Errorf("downward extract = %q, want %q", got, want)
	}

	// Upward drag swaps anchor/head: same visual region, opposite direction.
	app.selStartX, app.selStartY = 9, 2
	app.selEndX, app.selEndY = 6, 0
	if got := app.extractSelectedText(); got != want {
		t.Errorf("upward extract = %q, want %q", got, want)
	}
}

// TestMouseDragCopiesOnRelease proves the full flow: press anchors, motion
// extends, release copies through the clipboard sink and reports success.
func TestMouseDragCopiesOnRelease(t *testing.T) {
	app, captured := newMouseApp(t)
	app.lastRows = []string{"hello selectable world", strings.Repeat(" ", 100)}

	app.Update(tea.MouseMsg{Type: tea.MouseLeft, X: 0, Y: 0})
	app.Update(tea.MouseMsg{Type: tea.MouseMotion, X: 18, Y: 0})
	if !app.selActive {
		t.Fatal("selection should be active after press")
	}

	release := tea.MouseMsg{Type: tea.MouseRelease, X: 18, Y: 0}
	_, cmd := app.Update(release)
	if cmd == nil {
		t.Fatal("release should schedule the clipboard write")
	}
	done := cmd()
	cd, ok := done.(copyDoneMsg)
	if !ok {
		t.Fatalf("expected copyDoneMsg, got %T", done)
	}
	if cd.err != nil {
		t.Fatalf("clipboard sink error: %v", cd.err)
	}
	if len(*captured) != 1 || (*captured)[0] != "hello selectable wo" {
		t.Errorf("captured = %#v, want the dragged span", *captured)
	}

	// The status feedback flows through the returned model on delivery.
	final, _ := app.Update(done)
	fa := final.(*App)
	if !strings.Contains(fa.chat.Status(), "copied") {
		t.Errorf("status should confirm the copy, got %q", fa.chat.Status())
	}
}

// TestApplySelectionHighlightReversesSpan proves the highlight pass wraps
// exactly the selected span with reverse video and leaves the rest intact.
func TestApplySelectionHighlightReversesSpan(t *testing.T) {
	rows := []string{"hello world"}
	out := applySelectionHighlight(rows, 0, 0, 4, 0)
	got := out[0]
	if !strings.HasPrefix(got, "\x1b[7m") || !strings.Contains(got, "hello") {
		t.Errorf("selected span should be reversed, got %q", got)
	}
	if strings.Contains(got, "\x1b[7mworld") || strings.HasSuffix(got, "\x1b[7mworld\x1b[0m") {
		t.Errorf("unselected tail must stay untouched, got %q", got)
	}
}
