package tui

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/biggs-100/kui/internal/core"
	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/biggs-100/kui/internal/tui/toast"
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
	a := msg.(*App)

	// 'q' should NOT quit — it should be typed into input
	if cmd != nil {
		// tea.Quit is a func() tea.Msg; check if cmd would quit
		// We treat non-nil quit as failure for this test
		if msg2 := cmd(); msg2 == tea.Quit() {
			t.Fatal("typing 'q' should not quit")
		}
	}
	if a.quitting {
		t.Fatal("typing 'q' should not set quitting")
	}
	if a.input.Value() != "q" {
		t.Errorf("input after 'q' = %q, want %q", a.input.Value(), "q")
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

	// Type a prompt - capture each Update result
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	app = msg.(*App)
	msg, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	app = msg.(*App)

	// Submit
	msg, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
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

	// Submit with nil runner - should be no-op, no panic
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

	// Type and submit - capture each Update result
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	app = msg.(*App)
	msg, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	app = msg.(*App)
	app.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Simulate the done event
	msg, _ = app.Update(streamDoneMsg{})
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

// --- Slash Command Tests (Phase 4: Session Lifecycle) ---

func TestAppSessionsCommandNoStore(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Type /sessions and submit - capture each Update result
	for _, r := range "/sessions" {
		msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = msg.(*App)
	}
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a := msg.(*App)

	if a.chat.Status() == "" {
		t.Error("/sessions with no store should set a status message")
	}
}

func TestAppSessionsCommandWithStore(t *testing.T) {
	store := newFakeSessionStore()
	c := NewController([]string{"coder"}, nil, nil)
	c.SetSessionStore(store)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Type /sessions and submit - capture each Update result
	for _, r := range "/sessions" {
		msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = msg.(*App)
	}
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a := msg.(*App)

	if a.chat.Status() == "" {
		t.Error("/sessions with empty store should set a status message")
	}
}

func TestAppResumeCommandNoID(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Type /resume (no ID) and submit - capture each Update result
	for _, r := range "/resume" {
		msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = msg.(*App)
	}
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a := msg.(*App)

	if a.chat.Status() == "" {
		t.Error("/resume with no ID should set a usage status message")
	}
}

func TestAppResumeCommandWithSession(t *testing.T) {
	store := newFakeSessionStore()
	_ = store.Save(&core.Session{
		Meta:     core.NewSessionMeta("test-123", "coder"),
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "hello"},
			{Role: core.RoleAssistant, Content: "hi there"},
		},
	})
	c := NewController([]string{"coder"}, nil, nil)
	c.SetSessionStore(store)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Type /resume test-123 and submit - capture each Update result
	for _, r := range "/resume test-123" {
		msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = msg.(*App)
	}
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a := msg.(*App)

	if a.chat.Status() == "" {
		t.Error("/resume with valid session should set a status message")
	}
}

func TestAppQuitCommandSavesSession(t *testing.T) {
	store := newFakeSessionStore()
	c := NewController([]string{"coder"}, nil, nil)
	c.SetSessionStore(store)
	c.SetSessionID("quit-save-test")
	c.mu.Lock()
	c.messages = []core.Message{{Role: core.RoleUser, Content: "test"}}
	c.mu.Unlock()

	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Type /quit and submit - capture each Update result
	for _, r := range "/quit" {
		msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = msg.(*App)
	}
	msg, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = msg

	if cmd == nil {
		t.Fatal("/quit should return tea.Quit command")
	}
	if !app.quitting {
		t.Error("/quit should set quitting flag")
	}

	// Verify session was saved
	sess, err := store.Load("quit-save-test")
	if err != nil {
		t.Fatalf("/quit did not save session: %v", err)
	}
	if len(sess.Messages) == 0 {
		t.Error("saved session should have messages")
	}
}

func TestAppCtrlCSavesSession(t *testing.T) {
	store := newFakeSessionStore()
	c := NewController([]string{"coder"}, nil, nil)
	c.SetSessionStore(store)
	c.SetSessionID("ctrlc-save-test")
	c.mu.Lock()
	c.messages = []core.Message{{Role: core.RoleUser, Content: "test"}}
	c.mu.Unlock()

	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	_ = cmd

	if !app.quitting {
		t.Error("Ctrl+C should set quitting flag")
	}

	sess, err := store.Load("ctrlc-save-test")
	if err != nil {
		t.Fatalf("Ctrl+C did not save session: %v", err)
	}
	if len(sess.Messages) == 0 {
		t.Error("saved session should have messages")
	}
}

// --- PR2: App Integration Tests ---

func TestAppInputModel(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)

	// App should expose an InputModel
	input := app.Input()
	if input == nil {
		t.Fatal("expected non-nil InputModel from App")
	}
}

func TestAppKeybindings(t *testing.T) {
	c := NewController([]string{"coder", "writer"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Tab should still cycle profiles
	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyTab})
	a := updated.(*App)
	if a.ctrl.ActiveProfile() != "writer" {
		t.Errorf("Tab should cycle to writer, got %q", a.ctrl.ActiveProfile())
	}

	// Ctrl+C should quit
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("Ctrl+C should return quit command")
	}
}

func TestAppSubmit(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Type into the input via the textarea
	for _, r := range "hello" {
		app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Verify input has content
	if app.input.Value() == "" {
		t.Fatal("input should have content after typing")
	}

	// Submit via Enter
	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a := updated.(*App)

	// Chat should have a user message
	chat := a.chat
	if len(chat.Messages()) == 0 {
		t.Fatal("expected user message after submit")
	}
	if chat.Messages()[0].Content != "hello" {
		t.Errorf("message content = %q, want %q", chat.Messages()[0].Content, "hello")
	}

	// Input should be cleared
	if a.input.Value() != "" {
		t.Errorf("input should be empty after submit, got %q", a.input.Value())
	}
}

func TestAppAutocompleteTrigger(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Type "/" - should trigger autocomplete
	for _, r := range "/" {
		app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Check autocomplete state via view
	view := app.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestAppViewIncludesFooter(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	c.SetModelName("gpt-4")
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Switch to session mode by typing a prompt and pressing Enter
	for _, r := range "hello" {
		msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = msg.(*App)
	}
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = msg.(*App)

	view := app.View()
	if view == "" {
		t.Fatal("expected non-empty view")
	}

	// Session footer in welcome state (no sync.data) should show Get started /connect tick, not fabricated tokens
	if !strings.Contains(view, "Get started") && !strings.Contains(view, "/connect") && !strings.Contains(view, "/status") && !strings.Contains(view, "•") {
		t.Errorf("view should contain welcome footer 'Get started'/'/connect' or connected dots, got:\n%s", view)
	}
	// Chat should contain the user message
	if !strings.Contains(view, "hello") {
		t.Errorf("view should contain user message 'hello', got:\n%s", view)
	}
}

func TestAppFooterUpdatesOnStreamDone(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	c.SetModelName("gpt-4")
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Switch to session mode by typing a prompt and pressing Enter
	for _, r := range "hello" {
		msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = msg.(*App)
	}
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = msg.(*App)

	// Simulate stream done
	app.Update(streamDoneMsg{})

	// Footer should still render (welcome tick or connected dots) after stream done
	view := app.View()
	if !strings.Contains(view, "Get started") && !strings.Contains(view, "/connect") && !strings.Contains(view, "/status") && !strings.Contains(view, "•") {
		t.Errorf("view should contain footer after stream done, got:\n%s", view)
	}
}

func TestAppSessionsInteractive(t *testing.T) {
	store := newFakeSessionStore()
	_ = store.Save(&core.Session{
		Meta:     core.NewSessionMeta("s1", "coder"),
		Messages: []core.Message{{Role: core.RoleUser, Content: "hello"}},
	})
	c := NewController([]string{"coder"}, nil, nil)
	c.SetSessionStore(store)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Type /sessions and submit
	for _, r := range "/sessions" {
		msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = msg.(*App)
	}
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a := msg.(*App)

	// Session list should be active
	if a.sessionList == nil {
		t.Fatal("expected sessionList to be non-nil after /sessions with sessions")
	}
}

func TestAppSessionsInteractiveEmpty(t *testing.T) {
	store := newFakeSessionStore()
	c := NewController([]string{"coder"}, nil, nil)
	c.SetSessionStore(store)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Type /sessions and submit
	for _, r := range "/sessions" {
		msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = msg.(*App)
	}
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a := msg.(*App)

	// Empty store should show status, not open list
	if a.sessionList != nil {
		t.Error("sessionList should be nil for empty store")
	}
	if a.chat.Status() == "" {
		t.Error("expected status message for empty store")
	}
}

func TestAppUndoCommand(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	c.mu.Lock()
	c.messages = []core.Message{
		{Role: core.RoleUser, Content: "hello"},
		{Role: core.RoleAssistant, Content: "hi"},
	}
	c.mu.Unlock()

	c.PushUndo()
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Type /undo and submit
	for _, r := range "/undo" {
		msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = msg.(*App)
	}
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = msg

	// After undo, state should be restored from push (2 messages)
	c.mu.Lock()
	n := len(c.messages)
	c.mu.Unlock()
	if n != 2 {
		t.Errorf("after /undo, messages len = %d, want 2", n)
	}
}

func TestAppRedoCommand(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	c.mu.Lock()
	c.messages = []core.Message{
		{Role: core.RoleUser, Content: "hello"},
		{Role: core.RoleAssistant, Content: "hi"},
	}
	c.mu.Unlock()

	c.PushUndo()
	c.Undo()
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Type /redo and submit
	for _, r := range "/redo" {
		msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = msg.(*App)
	}
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = msg

	c.mu.Lock()
	n := len(c.messages)
	c.mu.Unlock()
	if n != 2 {
		t.Errorf("after /redo, messages len = %d, want 2", n)
	}
}

func TestAppRenameCommand(t *testing.T) {
	store := newFakeSessionStore()
	_ = store.Save(&core.Session{
		Meta:     core.NewSessionMeta("rename-test", "coder"),
		Messages: []core.Message{{Role: core.RoleUser, Content: "hello"}},
	})
	c := NewController([]string{"coder"}, nil, nil)
	c.SetSessionStore(store)
	c.SetSessionID("rename-test")
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Type /rename MyName and submit
	for _, r := range "/rename MyName" {
		msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = msg.(*App)
	}
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = msg

	sess, err := store.Load("rename-test")
	if err != nil {
		t.Fatalf("store.Load() error = %v", err)
	}
	if sess.Meta.Name != "MyName" {
		t.Errorf("Meta.Name = %q, want %q", sess.Meta.Name, "MyName")
	}
}

func TestAppRenameCommandNoSession(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Type /rename and submit
	for _, r := range "/rename" {
		msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = msg.(*App)
	}
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a := msg.(*App)

	if a.chat.Status() == "" {
		t.Error("/rename with no session should set a status message")
	}
}

func TestAppRenameCommandNoName(t *testing.T) {
	store := newFakeSessionStore()
	_ = store.Save(&core.Session{
		Meta:     core.NewSessionMeta("rename-test2", "coder"),
		Messages: []core.Message{{Role: core.RoleUser, Content: "hello"}},
	})
	c := NewController([]string{"coder"}, nil, nil)
	c.SetSessionStore(store)
	c.SetSessionID("rename-test2")
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Type /rename (no name) and submit
	for _, r := range "/rename" {
		msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = msg.(*App)
	}
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a := msg.(*App)

	if a.chat.Status() == "" {
		t.Error("/rename without name should set a usage status message")
	}
}

func TestStreamDoneMsgWiresUsageToController(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	c.SetModelName("gpt-4o-mini")
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	usage := core.Usage{
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
	}

	// Simulate stream done with usage data
	msg, _ := app.Update(streamDoneMsg{usage: usage})
	a := msg.(*App)

	// Controller should have accumulated the tokens
	if got := a.ctrl.TotalTokens(); got != 150 {
		t.Errorf("TotalTokens() = %d, want 150 after streamDoneMsg with usage", got)
	}
}

// --- Diff View Toggle Tests ---

func TestAppDiffToggle(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Initially diff should not be visible
	if app.diffVisible {
		t.Error("diff should not be visible initially")
	}

	// Press 'd' to toggle diff view
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	a := msg.(*App)

	if !a.diffVisible {
		t.Error("diff should be visible after pressing 'd'")
	}

	// Press 'd' again to hide
	msg, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	a = msg.(*App)

	if a.diffVisible {
		t.Error("diff should be hidden after pressing 'd' again")
	}
}

func TestAppDiffViewDoesNotAffectInput(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Type some text, then 'd' should not toggle diff when input has content
	for _, r := range "hello" {
		msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = msg.(*App)
	}

	// Press 'd' - should be typed into input, not toggle diff
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	a := msg.(*App)

	if a.diffVisible {
		t.Error("diff should not toggle when input has content")
	}
	if a.input.Value() != "hellod" {
		t.Errorf("input should contain 'hellod', got %q", a.input.Value())
	}
}

func TestAppDiffViewRendered(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Toggle diff view
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	a := msg.(*App)

	// View should render without panic
	view := a.View()
	if view == "" {
		t.Error("expected non-empty view with diff visible")
	}
}

// --- LSP Keybinding Tests (Fix #6) ---

func TestAppLspKeybindingGdWithoutDispatcher(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Press 'g' then 'd' - should not panic even without dispatcher
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	a := msg.(*App)
	if !a.lspPendingG {
		t.Error("pressing 'g' should set lspPendingG")
	}

	msg, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	_ = msg

	// lspPendingG should be cleared
	a2 := msg.(*App)
	if a2.lspPendingG {
		t.Error("lspPendingG should be false after 'gd' sequence")
	}

	// Status should show an error (no dispatcher configured)
	if a2.chat.Status() == "" {
		t.Error("gd without dispatcher should set error status")
	}
}

func TestAppLspKeybindingGrWithoutDispatcher(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Press 'g' then 'r'
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	a := msg.(*App)
	msg, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	a = msg.(*App)

	if a.chat.Status() == "" {
		t.Error("gr without dispatcher should set error status")
	}
}

func TestAppLspKeybindingKWithoutDispatcher(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Press 'K' (uppercase)
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}})
	a := msg.(*App)

	if a.chat.Status() == "" {
		t.Error("K without dispatcher should set error status")
	}
}

func TestAppLspKeybindingWithDispatcher(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	c.SetLspDispatcher(func(toolName string, args map[string]interface{}) (string, error) {
		return `{"locations":[{"uri":"file:///main.go","range":{"start":{"line":10,"character":0}}}]}`, nil
	})
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Press 'g' then 'd' - should dispatch and add result to chat
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	a := msg.(*App)
	msg, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	a = msg.(*App)

	// Chat should have an assistant message with the result
	if len(a.chat.Messages()) == 0 {
		t.Fatal("gd with dispatcher should add result to chat")
	}
	last := a.chat.Messages()[len(a.chat.Messages())-1]
	if last.Role != "assistant" {
		t.Errorf("expected assistant message, got role %q", last.Role)
	}
}

func TestAppLspKeybindingCancelled(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Press 'g' then something other than 'd' or 'r' - should cancel
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	a := msg.(*App)
	if !a.lspPendingG {
		t.Error("pressing 'g' should set lspPendingG")
	}

	// Press 'x' - should cancel the gd/gr sequence
	msg, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	a = msg.(*App)
	if a.lspPendingG {
		t.Error("lspPendingG should be false after cancelled sequence")
	}
}

func TestAppLspKeybindingIgnoredWithInput(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Type some text first
	for _, r := range "hello" {
		msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = msg.(*App)
	}

	// Now press 'g' - should NOT trigger lspPendingG because input is non-empty
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	a := msg.(*App)
	if a.lspPendingG {
		t.Error("lspPendingG should not be set when input is non-empty")
	}

	// 'K' should also be ignored when input is non-empty
	msg, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}})
	a = msg.(*App)
	// Should have typed 'K' into input, not dispatched LSP
	if a.input.Value() != "hellogK" {
		t.Errorf("K with non-empty input should type into input, got %q", a.input.Value())
	}
}

// --- Command Palette Tests (Phase 5) ---

func TestAppPaletteToggle(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Palette should not be active initially
	if app.paletteMode {
		t.Error("paletteMode should be false initially")
	}

	// Press Ctrl+P to open palette
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	a := msg.(*App)

	if !a.paletteMode {
		t.Error("paletteMode should be true after Ctrl+P")
	}
	if a.commandPalette == nil {
		t.Fatal("commandPalette should be non-nil after Ctrl+P")
	}

	// View should show the palette
	view := a.View()
	if !strings.Contains(view, "Command Palette") {
		t.Errorf("view should contain 'Command Palette' when palette is active, got:\n%s", view)
	}
}

func TestAppPaletteEscape(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Open palette
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	a := msg.(*App)

	// Escape should close palette
	msg, _ = a.Update(tea.KeyMsg{Type: tea.KeyEscape})
	a = msg.(*App)

	if a.paletteMode {
		t.Error("paletteMode should be false after Escape")
	}
}

func TestAppHelpCategorized(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Type /help and submit
	for _, r := range "/help" {
		msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = msg.(*App)
	}
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a := msg.(*App)

	status := a.chat.Status()
	if status == "" {
		t.Fatal("/help should set a status message")
	}

	// Should contain category headers
	if !strings.Contains(status, "Session") {
		t.Errorf("/help output missing 'Session' category, got: %s", status)
	}
	if !strings.Contains(status, "System") {
		t.Errorf("/help output missing 'System' category, got: %s", status)
	}

	// Should contain command descriptions
	if !strings.Contains(status, "/reload") {
		t.Errorf("/help output missing /reload, got: %s", status)
	}
}

func TestAppPaletteDoesNotInterfereWithInput(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Type some text
	for _, r := range "hello" {
		msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = msg.(*App)
	}

	// Input should have the text
	if app.input.Value() != "hello" {
		t.Errorf("input should contain 'hello', got %q", app.input.Value())
	}

	// Palette should not be active
	if app.paletteMode {
		t.Error("paletteMode should be false when typing normally")
	}
}

// --- Toast Integration Tests (Phase 2) ---

func TestAppToast(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Push a toast via the app
	app.toast.Push("config reloaded", toast.LevelInfo, 3*time.Second)

	// View should contain the toast text
	view := app.View()
	if !strings.Contains(view, "config reloaded") {
		t.Errorf("View() should contain toast text, got:\n%s", view)
	}
}

func TestAppToastDismissOnTick(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Push a toast with zero duration
	app.toast.Push("expired", toast.LevelInfo, 0)

	// Tick to dismiss
	app.Update(toast.TickMsg{})

	// View should NOT contain the expired toast
	view := app.View()
	if strings.Contains(view, "expired") {
		t.Errorf("View() should not contain expired toast, got:\n%s", view)
	}
}

// --- Theme Cycling Tests (Phase 5) ---

func TestThemeCycling(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Record initial theme
	initial := app.currentTheme

	// Type /theme next and submit
	for _, r := range "/theme next" {
		msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = msg.(*App)
	}
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a := msg.(*App)

	// Theme should have changed
	if a.currentTheme == initial {
		t.Errorf("theme should change after /theme next, still %q", a.currentTheme)
	}
}

func TestThemeCyclingPrev(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	initial := app.currentTheme

	// Type /theme prev and submit
	for _, r := range "/theme prev" {
		msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = msg.(*App)
	}
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a := msg.(*App)

	if a.currentTheme == initial {
		t.Errorf("theme should change after /theme prev, still %q", a.currentTheme)
	}
}

func TestThemeCyclingWraps(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	names := theme.ThemeNames()
	if len(names) < 2 {
		// In test env, themes may not be discoverable from package dir.
		// Verify wrap logic with a direct approach.
		app.currentTheme = "nonexistent-A"
		// When theme is not in list, cycling should still work (falls back to first)
		for _, r := range "/theme next" {
			msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			app = msg.(*App)
		}
		app.Update(tea.KeyMsg{Type: tea.KeyEnter})
		// Should land on "kui-default" (first in list)
		if app.currentTheme != "kui-default" {
			t.Errorf("cycling from unknown theme should land on kui-default, got %q", app.currentTheme)
		}
		return
	}

	// Cycle forward through all themes - should return to start
	// Note: original may be "" (default) which is not in the theme list,
	// so we track where we started and verify we cycle through all themes.
	seen := make(map[string]bool)
	for i := 0; i < len(names); i++ {
		for _, r := range "/theme next" {
			msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			app = msg.(*App)
		}
		app.Update(tea.KeyMsg{Type: tea.KeyEnter})
		seen[app.currentTheme] = true
	}

	// Verify we saw all themes
	for _, name := range names {
		if !seen[name] {
			t.Errorf("cycling did not visit theme %q", name)
		}
	}

	// After cycling through all themes, we should be back to the first theme in the list
	if app.currentTheme != names[0] {
		t.Errorf("cycling through all themes should wrap back to %q, got %q", names[0], app.currentTheme)
	}
}

func TestAppIsWide(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	tests := []struct {
		width int
		want  bool
	}{
		{100, false},
		{120, false},
		{121, true},
		{130, true},
		{160, true},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			app.Update(tea.WindowSizeMsg{Width: tt.width, Height: 24})
			if got := app.IsWide(); got != tt.want {
				t.Errorf("IsWide at %d = %v, want %v", tt.width, got, tt.want)
			}
		})
	}
}

func TestAppContentWidth(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 130, Height: 24})
	if got := app.ContentWidth(); got != 84 {
		t.Errorf("ContentWidth at 130 wide should be 84, got %d", got)
	}
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	if got := app.ContentWidth(); got != 96 {
		t.Errorf("ContentWidth at 100 narrow should be 96, got %d", got)
	}
}

func TestAppTitle(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	// Home route title is OpenCode
	if got := app.Title(); got != "OpenCode" {
		t.Errorf("Title on home should be 'OpenCode', got %q", got)
	}
	// Switch to session
	for _, r := range "hello" {
		msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = msg.(*App)
	}
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = msg.(*App)
	if got := app.Title(); got != "OC | coder" {
		t.Errorf("Title on session should be 'OC | coder', got %q", got)
	}
}

func TestAppHomeHasNoHeader(t *testing.T) {
	c := NewController([]string{"coder", "writer"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	view := app.View()
	// Home route must not render header tabs
	if strings.Contains(view, "coder") && strings.Contains(view, "writer") && strings.Contains(view, "tab") {
		// header contains profile tabs; home should not
		t.Errorf("home view should not contain header tabs, got:\n%s", view)
	}
	// More precise: header would contain profile names as tabs with ActiveTab styling
	// But in test env without ANSI, they appear as plain names; ensure home view does not contain both profiles as header
	// For now, check that home view does not contain the header's hint or tab pattern
	if strings.Contains(view, "no profiles") {
		t.Errorf("home view should not contain header hint")
	}
}
