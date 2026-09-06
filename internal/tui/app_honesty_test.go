package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/biggs-100/kui/internal/tui/views"
)

// TestShellExecRunsLocally proves the "!" affordance is real: submitting
// "!echo <marker>" executes LOCALLY and the combined output lands in the
// conversation as a shell block — it is never sent to the LLM.
func TestShellExecRunsLocally(t *testing.T) {
	app := newSessionApp(t, 100, 30)

	app.input.SetValue("!echo kui-shell-marker")
	msg, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a := msg.(*App)
	if cmd == nil {
		t.Fatal("shell submit must produce an exec command")
	}

	done := cmd()
	shellMsg, ok := done.(shellDoneMsg)
	if !ok {
		t.Fatalf("expected shellDoneMsg, got %T", done)
	}
	if !strings.Contains(shellMsg.output, "kui-shell-marker") {
		t.Fatalf("shell output should contain marker, got %q", shellMsg.output)
	}
	if strings.Contains(shellMsg.output, "cmd /C") || strings.Contains(shellMsg.output, "sh -c") {
		t.Errorf("output should not echo the shell invocation wrapper")
	}

	a.Update(done)
	last := a.chat.Messages()[len(a.chat.Messages())-1]
	if last.Kind != views.PartKindShell {
		t.Errorf("last message kind = %q, want %q", last.Kind, views.PartKindShell)
	}
	if !strings.Contains(last.Content, "$ echo kui-shell-marker") {
		t.Errorf("shell block should show the executed script, got %q", last.Content)
	}
}

// TestClearActuallyClears proves "/clear" no longer lies: after the command,
// the conversation view holds no messages.
func TestClearActuallyClears(t *testing.T) {
	app := newSessionApp(t, 100, 30)
	fillChat(t, app, 5)
	if len(app.chat.Messages()) == 0 {
		t.Fatal("precondition: chat should have messages")
	}

	app.handleCommand("/clear")

	if got := len(app.chat.Messages()); got != 0 {
		t.Errorf("/clear left %d messages in the view", got)
	}
}
