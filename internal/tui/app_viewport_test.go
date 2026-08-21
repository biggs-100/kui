package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// fillChat appends n distinct user messages so the conversation content is
// far taller than the terminal viewport.
func fillChat(t *testing.T, app *App, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		app.chat.AppendMessage("user", fmt.Sprintf("message-%03d", i), "coder", "")
	}
}

// TestViewportSticksToBottom proves REQ-TUI-APP-9 sticky-bottom scrolling:
// GIVEN a conversation taller than the viewport WHEN it renders THEN the
// newest message stays visible AND the frame still fills exactly a.height
// rows (the pinned layout budget is never broken by overflow).
func TestViewportSticksToBottom(t *testing.T) {
	const (
		width  = 100
		height = 30
	)
	app := newSessionApp(t, width, height)
	fillChat(t, app, 60)

	dump := stripTitleSequence(app.View())
	lines := strings.Split(dump, "\n")
	if len(lines) != height {
		t.Fatalf("frame has %d rows, want exactly %d", len(lines), height)
	}
	if !strings.Contains(dump, "message-059") {
		t.Errorf("viewport should stick to bottom and show the newest message, dump:\n%s", dump)
	}
}

// TestViewportScrollPositionPreservedOnPageUp proves that reading history is
// not disturbed by streaming: GIVEN the user paged up WHEN new content
// arrives THEN the viewport offset stays where the user left it.
func TestViewportScrollPositionPreservedOnPageUp(t *testing.T) {
	const (
		width  = 100
		height = 30
	)
	app := newSessionApp(t, width, height)
	fillChat(t, app, 80)
	app.View() // settle at bottom

	app.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	offset := app.scrollVP.YOffset
	if offset <= 0 {
		t.Fatalf("PgUp should scroll up, YOffset = %d", offset)
	}

	app.chat.AppendMessage("user", "arrived-while-scrolled", "coder", "")
	dump := stripTitleSequence(app.View())
	if app.scrollVP.YOffset != offset {
		t.Errorf("YOffset drifted after new content: got %d, want %d", app.scrollVP.YOffset, offset)
	}
	if strings.Contains(dump, "arrived-while-scrolled") {
		t.Errorf("newest message should stay out of view while the user is scrolled up")
	}
	lines := strings.Split(dump, "\n")
	if len(lines) != height {
		t.Fatalf("frame has %d rows after scroll, want exactly %d", len(lines), height)
	}
}
