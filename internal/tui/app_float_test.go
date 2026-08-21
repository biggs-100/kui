package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/biggs-100/kui/internal/tui/toast"
)

// TestPromptFieldPinnedBottom proves REQ-TUI-APP-2 "pinned prompt":
// GIVEN a session at 100x30 WHEN the frame renders THEN the input renders as
// the OpenCode-style raised field — a left ┃ bar over an element fill with a
// meta row inside and a half-block fade-out beneath — occupying exactly four
// rows directly above the footer row, regardless of conversation content.
func TestPromptFieldPinnedBottom(t *testing.T) {
	const (
		width  = 100
		height = 30
	)
	app := newSessionApp(t, width, height)
	dump := stripTitleSequence(app.View())
	rows := strings.Split(dump, "\n")
	if len(rows) != height {
		t.Fatalf("frame has %d rows, want exactly %d", len(rows), height)
	}
	if !strings.Contains(rows[height-5], "┃") {
		t.Errorf("input field bar missing on row %d: %q", height-5, rows[height-5])
	}
	if !strings.Contains(rows[height-4], "Ask kui") {
		t.Errorf("prompt line missing on row %d: %q", height-4, rows[height-4])
	}
	if !strings.Contains(rows[height-3], "coder") {
		t.Errorf("meta row with active profile missing on row %d: %q", height-3, rows[height-3])
	}
	if !strings.Contains(rows[height-2], "▀") {
		t.Errorf("fade-out row missing on row %d: %q", height-2, rows[height-2])
	}
	if !strings.Contains(rows[height-1], "Get started") {
		t.Errorf("footer not pinned to last row %d: %q", height-1, rows[height-1])
	}
}

// TestAutocompletePopupFloatsOverConversation proves the popup hovers over
// the frame above the prompt box without adding flow rows (frame height and
// pinning are unchanged while it is open).
func TestAutocompletePopupFloatsOverConversation(t *testing.T) {
	const (
		width  = 100
		height = 30
	)
	app := newSessionApp(t, width, height)
	fillChat(t, app, 60)
	app.autocomplete.Activate("/")

	dump := stripTitleSequence(app.View())
	rows := strings.Split(dump, "\n")
	if len(rows) != height {
		t.Fatalf("frame has %d rows with popup open, want exactly %d", len(rows), height)
	}
	if !strings.Contains(rows[height-4], "Ask kui") {
		t.Errorf("prompt must stay pinned while popup is open, row %d: %q", height-4, rows[height-4])
	}
	if !strings.Contains(dump, "/") {
		t.Errorf("popup content should be visible in the frame")
	}
}

// TestToastFloatsAboveInputWithoutShiftingRows proves the toast chip is
// pasted over the conversation above the prompt box (stacked over any
// popup) without moving the pinned regions or growing the frame.
func TestToastFloatsAboveInputWithoutShiftingRows(t *testing.T) {
	const (
		width  = 100
		height = 30
	)
	app := newSessionApp(t, width, height)
	fillChat(t, app, 5)
	app.toast.Push("layout-refactor-toast", toast.LevelSuccess, time.Second)

	dump := stripTitleSequence(app.View())
	rows := strings.Split(dump, "\n")
	if len(rows) != height {
		t.Fatalf("frame has %d rows with toast active, want exactly %d", len(rows), height)
	}
	toastRow := -1
	for i, r := range rows {
		if strings.Contains(r, "layout-refactor-toast") {
			toastRow = i
			break
		}
	}
	if toastRow < 0 {
		t.Fatalf("toast text should be visible in the frame:\n%s", dump)
	}
	inputRow := height - 4
	if toastRow >= inputRow-1 && toastRow <= inputRow+1 {
		t.Errorf("toast should float above the prompt field, found on row %d (input at %d)", toastRow, inputRow)
	}
	if !strings.Contains(rows[inputRow], "Ask kui") {
		t.Errorf("prompt must stay pinned while toast is shown, row %d: %q", inputRow, rows[inputRow])
	}
}
