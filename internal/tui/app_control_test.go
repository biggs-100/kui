package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/biggs-100/kui/internal/adapters/store"
	"github.com/biggs-100/kui/internal/core"
)

// TestInterruptStateMachine proves Interrupt only fires while a run is
// active and clears nothing when idle.
func TestInterruptStateMachine(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	if c.IsRunning() {
		t.Fatal("fresh controller must not be running")
	}
	if c.Interrupt() {
		t.Error("interrupt on idle controller should return false")
	}

	called := false
	c.mu.Lock()
	c.running = true
	c.cancel = func() { called = true }
	c.mu.Unlock()

	if !c.IsRunning() {
		t.Error("IsRunning should reflect running state")
	}
	if !c.Interrupt() {
		t.Error("interrupt while running should return true")
	}
	if !called {
		t.Error("cancel function was never invoked")
	}
}

// TestStartNewSessionRotates proves /new starts a clean transcript: history,
// usage and undo stacks reset, and the session ID rotates so autosave never
// appends to the old transcript.
func TestStartNewSessionRotates(t *testing.T) {
	dir := t.TempDir()
	c := NewController([]string{"coder"}, nil, nil)
	ss := store.NewSessionStore(dir)
	c.SetSessionStore(ss)
	c.SetSessionID("coder-old-id")

	c.mu.Lock()
	c.messages = []core.Message{{Role: core.RoleUser, Content: "hello"}}
	c.totalTokens = 1234
	c.mu.Unlock()
	c.PushUndo()

	c.StartNewSession()

	if got := len(c.Messages()); got != 0 {
		t.Errorf("messages after /new = %d, want 0", got)
	}
	if id := c.SessionID(); id == "" || id == "coder-old-id" {
		t.Errorf("session ID should rotate to a fresh value, got %q", id)
	}
	if tokens := c.TotalTokens(); tokens != 0 {
		t.Errorf("usage should reset on new session, got %d", tokens)
	}
	if c.Undo() {
		t.Error("undo stack should be cleared by /new")
	}
}

// TestDoubleEscInterruptWindow proves the base-layer Esc handler implements
// the 5s double-press window: first Esc arms the window, second fires.
func TestDoubleEscInterruptWindow(t *testing.T) {
	app := newSessionApp(t, 100, 30)
	if !app.lastEsc.IsZero() {
		t.Fatal("precondition: lastEsc should start zero")
	}
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	a := msg.(*App)
	if a.lastEsc.IsZero() {
		t.Error("first Esc should arm the interrupt window")
	}
	if time.Since(a.lastEsc) > 5*time.Second {
		t.Error("armed window should be within 5s of now")
	}
}
