package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/biggs-100/kui/internal/tui/views"
)

// App is the root Bubble Tea model that composes the header, chat, and tool
// views into a three-region layout (REQ-TUI-APP-2). It delegates profile
// switching and prompt submission to the Controller, and translates controller
// events into tea.Msg values for the Update cycle (REQ-TUI-APP-3).
//
// App never runs UI work on the agent loop's goroutine — all UI updates flow
// through tea.Cmd (D3 channel+Cmd handoff).
type App struct {
	ctrl   *Controller
	header views.HeaderModel
	chat   views.ChatModel
	tool   views.ToolModel
	styles *theme.Styles

	width  int
	height int
	input  string
	quitting bool
}

// NewApp creates an App wrapping the given Controller. The Controller must be
// created before the App so that profile names and runner are available.
func NewApp(ctrl *Controller) *App {
	return NewAppWithTheme(ctrl, "")
}

// NewAppWithTheme creates an App with a specific theme name.
// If name is empty, the default theme is used.
func NewAppWithTheme(ctrl *Controller, themeName string) *App {
	t := theme.Load(themeName)
	styles := theme.NewStyles(t)

	return &App{
		ctrl:   ctrl,
		styles: styles,
		chat:   views.NewChatModel(styles),
		tool:   views.NewToolModel(styles),
	}
}

// Init returns the initial command. The controller's event pump is started
// externally (by Run) so Init returns nil.
func (a *App) Init() tea.Cmd {
	return nil
}

// Update handles incoming messages: key events, window resize, and controller
// events (stream chunks, done, tool events). It returns the updated model and
// an optional tea.Cmd.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.rebuildViews()
		return a, nil

	case tea.KeyMsg:
		return a.handleKey(msg)

	case streamChunkMsg:
		a.chat.AppendChunk(msg.delta)
		return a, nil

	case streamDoneMsg:
		if msg.err != nil {
			a.chat.SetError(msg.err.Error())
		}
		return a, nil

	case toolCallMsg:
		a.tool.AppendCall(msg.callID, msg.name)
		return a, nil

	case toolResultMsg:
		a.tool.AppendResult(msg.callID, msg.result)
		return a, nil

	case reloadStartMsg:
		a.chat.SetStatus("reloading…")
		return a, nil

	case reloadDoneMsg:
		if msg.err != nil {
			a.chat.SetStatus("reload failed: " + msg.err.Error())
		} else {
			a.chat.SetStatus(fmt.Sprintf("reload complete"))
		}
		a.rebuildViews()
		return a, nil
	}

	return a, nil
}

// handleKey processes keyboard input. Tab and Shift+Tab cycle profiles;
// Enter submits the current input; q and Ctrl+C quit (REQ-TUI-APP-1).
func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab:
		a.ctrl.SwitchProfile(1)
		a.rebuildViews()
		return a, nil

	case tea.KeyShiftTab:
		a.ctrl.SwitchProfile(-1)
		a.rebuildViews()
		return a, nil

	case tea.KeyEnter:
		if strings.TrimSpace(a.input) == "" {
			return a, nil
		}
		text := a.input
		a.input = ""
		// REQ-RELOAD-11: handle slash commands before submitting.
		if strings.HasPrefix(text, "/") {
			return a.handleCommand(text)
		}
		a.chat.AppendMessage("user", text, a.ctrl.ActiveProfile(), "")
		a.ctrl.SubmitPrompt(text)
		return a, nil

	case tea.KeyRunes:
		runes := msg.Runes
		if len(runes) == 1 {
			r := runes[0]
			if r == 'q' && a.input == "" {
				_ = a.ctrl.SaveSession()
				a.quitting = true
				return a, tea.Quit
			}
			a.input += string(r)
		}
		return a, nil

	case tea.KeyBackspace, tea.KeyDelete:
		if len(a.input) > 0 {
			a.input = a.input[:len(a.input)-1]
		}
		return a, nil

	case tea.KeyCtrlC:
		_ = a.ctrl.SaveSession()
		a.quitting = true
		return a, tea.Quit
	}

	return a, nil
}

// handleCommand dispatches slash commands. /reload triggers a hot-reload
// of the runtime; /help shows available commands; unknown commands show an
// error hint (REQ-RELOAD-11).
func (a *App) handleCommand(text string) (tea.Model, tea.Cmd) {
	parts := strings.SplitN(text, " ", 2)
	cmd := parts[0]

	switch cmd {
	case "/reload":
		a.ctrl.Reload()
	case "/sessions":
		a.handleSessionsCommand()
	case "/resume":
		if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
			a.chat.SetStatus("usage: /resume <session-id>")
		} else {
			a.handleResumeCommand(strings.TrimSpace(parts[1]))
		}
	case "/quit", "/exit":
		_ = a.ctrl.SaveSession()
		a.quitting = true
		return a, tea.Quit
	case "/help":
		a.chat.SetStatus("commands: /sessions, /resume <id>, /reload, /quit, /exit, /help")
	default:
		a.chat.SetStatus("unknown command: " + text + " (try /help)")
	}
	return a, nil
}

// handleSessionsCommand lists all saved sessions in the chat view.
func (a *App) handleSessionsCommand() {
	store := a.ctrl.SessionStore()
	if store == nil {
		a.chat.SetStatus("session persistence not configured")
		return
	}

	metas, err := store.List()
	if err != nil {
		a.chat.SetStatus("error listing sessions: " + err.Error())
		return
	}

	if len(metas) == 0 {
		a.chat.SetStatus("no saved sessions")
		return
	}

	var b strings.Builder
	b.WriteString("saved sessions:\n")
	for _, m := range metas {
		b.WriteString(fmt.Sprintf("  %s  profile=%s  %s\n", m.ID, m.Profile, m.CreatedAt))
	}
	a.chat.SetStatus(b.String())
}

// handleResumeCommand loads a session and injects its history into the controller.
func (a *App) handleResumeCommand(id string) {
	msgs, err := a.ctrl.LoadSession(id)
	if err != nil {
		a.chat.SetStatus("error loading session: " + err.Error())
		return
	}

	// Populate the chat view with loaded history for rendering.
	a.chat.LoadHistory(msgs)
	a.chat.SetStatus("session " + id + " restored (" + fmt.Sprintf("%d", len(msgs)) + " messages)")
}

// View renders the three-region layout: header (profile tabs), chat
// (messages + input), and tool view (REQ-TUI-APP-2). Resize reflows all
// three regions.
func (a *App) View() string {
	if a.quitting {
		return ""
	}

	if a.width == 0 || a.height == 0 {
		return "loading..."
	}

	// Rebuild views with current state
	a.rebuildViews()

	// Header: 1 line
	header := a.header.Render()

	// Tool view: up to 10 lines, but no more than 1/4 of height
	toolMax := a.height / 4
	if toolMax < 3 {
		toolMax = 3
	}
	toolStr := a.tool.Render()

	// Chat: fills remaining space
	chatStr := a.chat.Render()

	// Input line at bottom
	inputLine := "> " + a.input

	// Compose regions with newlines between them
	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(chatStr)
	b.WriteString("\n")
	if toolStr != "" {
		b.WriteString(toolStr)
		b.WriteString("\n")
	}
	b.WriteString(inputLine)

	return b.String()
}

// rebuildViews synchronizes the view models with the controller state.
func (a *App) rebuildViews() {
	profile := a.ctrl.ActiveProfile()
	profiles := a.ctrl.Profiles()
	active := 0
	for i, p := range profiles {
		if p == profile {
			active = i
			break
		}
	}
	a.header = views.NewHeaderModel(profiles, active, a.styles)
}

// chat returns the chat model for inspection.
func (a *App) chatView() *views.ChatModel {
	return &a.chat
}
