package tui

import (
	"context"
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/biggs-100/kui/internal/core"
)

// --- fakeAppRunner implements Runner for app-level tests ---

type fakeAppRunner struct {
	response string
}

func (r *fakeAppRunner) Run(_ context.Context, prompt string, _ []core.Message) (string, []core.Message, error) {
	return r.response, nil, nil
}

func (r *fakeAppRunner) Steering() core.PendingQueue {
	return nil
}

func (r *fakeAppRunner) Provider() core.Provider {
	return nil
}

// --- Model.Update tests (REQ-TUI-APP-1/2/3/4) ---

func TestAppUpdateChunkMsgGrowsAnswer(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)

	// Send a chunk message
	msg := streamChunkMsg{delta: "hello "}
	updated, _ := app.Update(msg)
	a := updated.(*App)

	// Chat should have an assistant message with the chunk content
	chat := a.chat
	if len(chat.Messages()) == 0 {
		t.Fatal("expected messages after chunk")
	}
	last := chat.Messages()[len(chat.Messages())-1]
	if last.Content != "hello " {
		t.Errorf("message content = %q, want %q", last.Content, "hello ")
	}
}

func TestAppUpdateMultipleChunksGrowAnswer(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)

	app.Update(streamChunkMsg{delta: "hello"})
	msg, _ := app.Update(streamChunkMsg{delta: " world"})
	a := msg.(*App)

	chat := a.chat
	if len(chat.Messages()) == 0 {
		t.Fatal("expected messages after chunks")
	}
	last := chat.Messages()[len(chat.Messages())-1]
	if last.Content != "hello world" {
		t.Errorf("message content = %q, want %q", last.Content, "hello world")
	}
}

func TestAppUpdateTabCyclesProfileForward(t *testing.T) {
	c := NewController([]string{"coder", "writer"}, nil, nil)
	app := NewApp(c)

	// Tab should cycle forward
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyTab})
	a := msg.(*App)

	if a.ctrl.ActiveProfile() != "writer" {
		t.Errorf("active profile after Tab = %q, want %q", a.ctrl.ActiveProfile(), "writer")
	}
}

func TestAppUpdateShiftTabCyclesProfileBackward(t *testing.T) {
	c := NewController([]string{"coder", "writer"}, nil, nil)
	app := NewApp(c)

	// Shift+Tab should cycle backward
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	a := msg.(*App)

	if a.ctrl.ActiveProfile() != "writer" {
		t.Errorf("active profile after Shift+Tab = %q, want %q", a.ctrl.ActiveProfile(), "writer")
	}
}

func TestAppUpdateTabWrapsAround(t *testing.T) {
	c := NewController([]string{"coder", "writer"}, nil, nil)
	app := NewApp(c)

	// Tab from coder → writer → coder (wrap)
	app.Update(tea.KeyMsg{Type: tea.KeyTab})
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyTab})
	a := msg.(*App)

	if a.ctrl.ActiveProfile() != "coder" {
		t.Errorf("active profile after two Tabs = %q, want %q", a.ctrl.ActiveProfile(), "coder")
	}
}

func TestAppUpdateResizeReflows(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)

	// Send initial size
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Resize
	msg, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a := msg.(*App)

	// View should render without panic and include all regions
	view := a.View()
	if view == "" {
		t.Error("expected non-empty view after resize")
	}
}

func TestAppUpdateQuitOnQ(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)

	// Send initial size so View works
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	msg, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	_ = msg

	// cmd should return tea.Quit
	if cmd == nil {
		t.Fatal("expected quit command on 'q' key")
	}
}

func TestAppUpdateQuitOnCtrlC(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)

	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	msg, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	_ = msg

	if cmd == nil {
		t.Fatal("expected quit command on ctrl+c")
	}
}

func TestAppUpdateEmptyInputIgnored(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)

	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Enter with empty input should not submit
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a := msg.(*App)

	// Chat should have no messages (empty input ignored)
	chat := a.chat
	if len(chat.Messages()) != 0 {
		t.Errorf("expected no messages on empty enter, got %d", len(chat.Messages()))
	}
}

func TestAppUpdateSubmitPrompt(t *testing.T) {
	queue := &stubQueue{}
	runner := &fakeRunner{
		agent: &core.Agent{
			Provider:      &stubProvider{response: "ok"},
			Tools:         core.NewRegistry(),
			MaxIterations: 1,
		},
		steering: queue,
	}
	c := NewController([]string{"coder"}, runner, func(name string) string {
		return "gpt-4o-mini"
	})
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Type a prompt
	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})

	// Submit
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a := msg.(*App)

	// Chat should have a user message
	chat := a.chat
	if len(chat.Messages()) == 0 {
		t.Fatal("expected user message after submit")
	}
	if chat.Messages()[0].Role != "user" {
		t.Errorf("first message role = %q, want %q", chat.Messages()[0].Role, "user")
	}
	if chat.Messages()[0].Content != "hi" {
		t.Errorf("first message content = %q, want %q", chat.Messages()[0].Content, "hi")
	}
}

func TestAppUpdateNilObserverDoesNotPanic(t *testing.T) {
	// App with nil runner/resolver should not panic on operations
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Submit with nil runner — should be no-op, no panic
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = msg
}

func TestAppViewThreeRegions(t *testing.T) {
	c := NewController([]string{"coder", "writer"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	view := app.View()
	if view == "" {
		t.Fatal("expected non-empty view")
	}
}

func TestAppStreamDoneMsgSetsAnswer(t *testing.T) {
	queue := &stubQueue{}
	runner := &fakeRunner{
		agent: &core.Agent{
			Provider:      &stubProvider{response: "done"},
			Tools:         core.NewRegistry(),
			MaxIterations: 1,
		},
		steering: queue,
	}
	c := NewController([]string{"coder"}, runner, func(name string) string {
		return "gpt-4o-mini"
	})
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Type and submit — each character arrives as a separate KeyMsg
	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	app.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Simulate the done event
	msg, _ := app.Update(streamDoneMsg{})
	a := msg.(*App)

	// The chat should reflect the completed run
	chat := a.chat
	if len(chat.Messages()) == 0 {
		t.Error("expected messages after stream done")
	}
}

func TestAppStreamDoneMsgWithErrorSetsErrorState(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Simulate a stream done with error
	msg, _ := app.Update(streamDoneMsg{err: fmt.Errorf("provider error")})
	a := msg.(*App)

	// Chat should show the error
	chat := a.chat
	if chat.LastError() == "" {
		t.Error("expected error state after stream done with error")
	}
	_ = a
}
