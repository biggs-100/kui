package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestInputModelCreate(t *testing.T) {
	m := NewInputModel("type here...", "")
	if m.Value() != "" {
		t.Errorf("Value() = %q, want empty", m.Value())
	}
}

func TestInputValue(t *testing.T) {
	m := NewInputModel("", "")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if m.Value() != "hi" {
		t.Errorf("Value() = %q, want %q", m.Value(), "hi")
	}
}

func TestInputClear(t *testing.T) {
	m := NewInputModel("", "")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})

	submitted := m.Submit()
	if submitted != "hi" {
		t.Errorf("Submit() = %q, want %q", submitted, "hi")
	}
	if m.Value() != "" {
		t.Errorf("Value() after Submit = %q, want empty", m.Value())
	}
}

func TestInputCursorMove(t *testing.T) {
	m := NewInputModel("", "")
	m.Focus()

	// Type "hello"
	for _, r := range "hello" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Move left twice
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})

	// Type "x" — should insert at cursor position (after "hel")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	got := m.Value()
	want := "helxlo"
	if got != want {
		t.Errorf("Value() after cursor move + insert = %q, want %q", got, want)
	}

	// Home key moves to beginning
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyHome})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	got = m.Value()
	want = "zhelxlo"
	if got != want {
		t.Errorf("Value() after Home + insert = %q, want %q", got, want)
	}

	// End key moves to end
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	got = m.Value()
	want = "zhelxloy"
	if got != want {
		t.Errorf("Value() after End + insert = %q, want %q", got, want)
	}
}

func TestInputWordNav(t *testing.T) {
	m := NewInputModel("", "")
	m.Focus()

	// Type "hello world"
	for _, r := range "hello world" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Ctrl+Left jumps to start of current/previous word (Alt+Left in textarea).
	// Cursor lands before 'w' in "world", so inserting 'x' produces "hello xworld"
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlLeft})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	got := m.Value()
	want := "hello xworld"
	if got != want {
		t.Errorf("Value() after Ctrl+Left + insert = %q, want %q", got, want)
	}
}

func TestPasteDetection(t *testing.T) {
	m := NewInputModel("", "")
	m.Focus()

	// Simulate paste: bubbletea sends pasted text as a single KeyRunes msg
	// with multiple runes (bracketed paste decoded into runes by terminal).
	pasted := []rune("hello world")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: pasted})
	if m.Value() != "hello world" {
		t.Errorf("Value() after paste = %q, want %q", m.Value(), "hello world")
	}

	// Paste more text — should append
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" foo")})
	if m.Value() != "hello world foo" {
		t.Errorf("Value() after second paste = %q, want %q", m.Value(), "hello world foo")
	}
}

func TestPasteWithCursor(t *testing.T) {
	m := NewInputModel("", "")
	m.Focus()

	// Type "hi"
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})

	// Move cursor to start
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyHome})

	// Paste at beginning
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(">>")})
	if m.Value() != ">>hi" {
		t.Errorf("Value() after paste at start = %q, want %q", m.Value(), ">>hi")
	}
}

func TestInputUndoRedo(t *testing.T) {
	m := NewInputModel("", "")
	m.Focus()

	// Type "abc"
	for _, r := range "abc" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Undo twice — should remove "c" then "b"
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
	got := m.Value()
	if got != "a" {
		t.Errorf("Value() after 2 undos = %q, want %q", got, "a")
	}

	// Redo — should restore "b"
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlY})
	got = m.Value()
	if got != "ab" {
		t.Errorf("Value() after redo = %q, want %q", got, "ab")
	}
}
