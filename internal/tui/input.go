package tui

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// InputModel wraps textarea.Model with history and undo/redo support.
type InputModel struct {
	textarea textarea.Model
	history  *History

	// Undo/redo stacks
	undoStack []string
	redoStack []string
}

// NewInputModel creates an InputModel with placeholder and history path.
func NewInputModel(placeholder, historyPath string) InputModel {
	ta := textarea.New()
	ta.Placeholder = placeholder
	ta.Focus()
	ta.CharLimit = 0
	ta.SetHeight(1)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	var h *History
	if historyPath != "" {
		var err error
		h, err = NewHistory(historyPath)
		if err != nil {
			h = &History{index: -1}
		}
	} else {
		h = &History{index: -1}
	}

	return InputModel{
		textarea: ta,
		history:  h,
	}
}

// Value returns the current input text.
func (m InputModel) Value() string {
	return m.textarea.Value()
}

// Submit clears the textarea and returns the submitted text.
func (m *InputModel) Submit() string {
	text := m.textarea.Value()
	if text != "" && m.history != nil {
		m.history.Append(text)
	}
	m.textarea.Reset()
	m.undoStack = nil
	m.redoStack = nil
	return text
}

// Clear clears the input text without recording history.
func (m *InputModel) Clear() {
	m.textarea.Reset()
	m.undoStack = nil
	m.redoStack = nil
}

// Update handles key messages, intercepting special keys before delegating
// to textarea:
//   - Ctrl+Z → undo (textarea has no built-in undo)
//   - Ctrl+Y → redo
//   - Ctrl+Left → word backward (translated to Alt+Left for textarea)
//   - Ctrl+Right → word forward (translated to Alt+Right for textarea)
//   - Character/backspace/delete → push undo state before modification
func (m InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Undo: Ctrl+Z
		if msg.Type == tea.KeyCtrlZ {
			return m.undo()
		}
		// Redo: Ctrl+Y
		if msg.Type == tea.KeyCtrlY {
			return m.redo()
		}
		// Word backward: Ctrl+Left → translate to Alt+Left for textarea
		if msg.Type == tea.KeyCtrlLeft {
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyLeft, Alt: true})
			return m, cmd
		}
		// Word forward: Ctrl+Right → translate to Alt+Right for textarea
		if msg.Type == tea.KeyCtrlRight {
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRight, Alt: true})
			return m, cmd
		}
		// For character input, push undo state before modification
		if msg.Type == tea.KeyRunes || msg.Type == tea.KeyBackspace || msg.Type == tea.KeyDelete {
			m.pushUndo()
		}
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// View renders the textarea.
func (m InputModel) View() string {
	return m.textarea.View()
}

// Focus gives focus to the textarea.
func (m *InputModel) Focus() tea.Cmd {
	return m.textarea.Focus()
}

// Blur removes focus from the textarea.
func (m *InputModel) Blur() {
	m.textarea.Blur()
}

// SetValue sets the textarea content (for history recall).
func (m *InputModel) SetValue(s string) {
	m.pushUndo()
	m.textarea.SetValue(s)
}

// History returns the underlying History (for navigation).
func (m *InputModel) History() *History {
	return m.history
}

// pushUndo saves the current state to the undo stack.
func (m *InputModel) pushUndo() {
	m.undoStack = append(m.undoStack, m.textarea.Value())
	m.redoStack = nil
}

// undo restores the previous state.
func (m InputModel) undo() (InputModel, tea.Cmd) {
	if len(m.undoStack) == 0 {
		return m, nil
	}
	m.redoStack = append(m.redoStack, m.textarea.Value())
	n := len(m.undoStack)
	prev := m.undoStack[n-1]
	m.undoStack = m.undoStack[:n-1]
	m.textarea.SetValue(prev)
	return m, nil
}

// redo restores the next state.
func (m InputModel) redo() (InputModel, tea.Cmd) {
	if len(m.redoStack) == 0 {
		return m, nil
	}
	m.undoStack = append(m.undoStack, m.textarea.Value())
	n := len(m.redoStack)
	next := m.redoStack[n-1]
	m.redoStack = m.redoStack[:n-1]
	m.textarea.SetValue(next)
	return m, nil
}
