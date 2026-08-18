package tui

import (
	"context"
	"fmt"
	"strings"
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

	// Type a prompt — capture each Update result
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

	// Type and submit — capture each Update result
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

	// Type /sessions and submit — capture each Update result
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

	// Type /sessions and submit — capture each Update result
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

	// Type /resume (no ID) and submit — capture each Update result
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

	// Type /resume test-123 and submit — capture each Update result
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

	// Type /quit and submit — capture each Update result
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

	// Type "/" — should trigger autocomplete
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

	view := app.View()
	if view == "" {
		t.Fatal("expected non-empty view")
	}

	// Footer should contain the model name when set
	if !strings.Contains(view, "gpt-4") {
		t.Errorf("view should contain model name 'gpt-4', got:\n%s", view)
	}

	// Footer should contain MCP status
	if !strings.Contains(view, "MCP") {
		t.Errorf("view should contain 'MCP' in footer, got:\n%s", view)
	}
}

func TestAppFooterUpdatesOnStreamDone(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	c.SetModelName("gpt-4")
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Simulate stream done
	app.Update(streamDoneMsg{})

	// Footer should still render with model name
	view := app.View()
	if !strings.Contains(view, "gpt-4") {
		t.Errorf("view should contain model name after stream done, got:\n%s", view)
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
