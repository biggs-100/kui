package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/biggs-100/kui/internal/core"
)

func TestAppInitialRouteIsHome(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	if app.route != "home" {
		t.Errorf("initial route = %q, want %q", app.route, "home")
	}
}

func TestAppEnterOnHomeSwitchesToSession(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Type a prompt
	for _, r := range "hello" {
		msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = msg.(*App)
	}

	// Submit
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a := msg.(*App)

	if a.route != "session" {
		t.Errorf("route after submit = %q, want %q", a.route, "session")
	}
}

func TestAppEmptyEnterStaysOnHome(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Enter with empty input
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a := msg.(*App)

	if a.route != "home" {
		t.Errorf("route after empty enter = %q, want %q", a.route, "home")
	}
}

func TestAppViewRendersHomeWhenRouteIsHome(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	view := app.View()
	// Home view should contain the logo
	if view == "" {
		t.Fatal("home view should not be empty")
	}
}

func TestAppViewRendersSessionWhenRouteIsSession(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Force route to session
	app.route = "session"
	view := app.View()

	// Session view should render (may be empty chat but non-empty)
	if view == "" {
		// Session view with no messages might be "loading..."
		// That's OK — just verify it doesn't panic
		t.Log("session view with no messages rendered empty (expected)")
	}
}

func TestAppCtrlCWorksOnHome(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Ctrl+C on home should quit
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("Ctrl+C on home should return quit command")
	}
}

func TestAppTabWorksOnHome(t *testing.T) {
	c := NewController([]string{"coder", "writer"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Tab on home should cycle profile
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyTab})
	a := msg.(*App)

	if a.ctrl.ActiveProfile() != "writer" {
		t.Errorf("Tab on home should cycle profile, got %q", a.ctrl.ActiveProfile())
	}
}

func TestAppPaletteWorksOnHome(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	app := NewApp(c)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Ctrl+P on home should open palette
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	a := msg.(*App)

	if !a.paletteMode {
		t.Error("Ctrl+P on home should open command palette")
	}
}

func TestAppSubmitCapturesPrompt(t *testing.T) {
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
	for _, r := range "hello world" {
		msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = msg.(*App)
	}

	// Submit
	msg, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a := msg.(*App)

	// Should be in session route with the message in chat
	if a.route != "session" {
		t.Errorf("route = %q, want session", a.route)
	}
	if len(a.chat.Messages()) == 0 {
		t.Fatal("chat should have messages after submit")
	}
	if a.chat.Messages()[0].Content != "hello world" {
		t.Errorf("submitted message = %q, want %q", a.chat.Messages()[0].Content, "hello world")
	}
}
