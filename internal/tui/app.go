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
	footer views.FooterModel
	styles *theme.Styles

	width  int
	height int
	input  InputModel
	autocomplete AutocompleteModel
	quitting bool

	// Session list mode: when non-nil, the session list view is active.
	sessionList *views.SessionListModel
	listMode    bool
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
		ctrl:        ctrl,
		styles:      styles,
		input:       NewInputModel("type here...", ""),
		autocomplete: NewAutocompleteModel(),
		chat:        views.NewChatModel(styles),
		tool:        views.NewToolModel(styles),
		footer:      views.NewFooterModel(styles),
	}
}

// Input returns the App's InputModel for inspection.
func (a *App) Input() *InputModel {
	return &a.input
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
		a.ctrl.TrackUsage(msg.usage)
		a.rebuildViews()
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

// handleKey processes keyboard input. App-level keys (Tab, Ctrl+C, Enter)
// are intercepted before delegating to InputModel (REQ-TUI-APP-1).
func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// --- Session list mode: delegate all keys to list ---
	if a.listMode && a.sessionList != nil {
		updated, cmd := a.sessionList.Update(msg)
		a.sessionList = &updated
		if a.sessionList.Selected() != "" {
			id := a.sessionList.Selected()
			a.sessionList = nil
			a.listMode = false
			a.handleResumeCommand(id)
			return a, cmd
		}
		if a.sessionList == nil || a.sessionList.View() == "" {
			a.sessionList = nil
			a.listMode = false
			return a, cmd
		}
		return a, cmd
	}

	// --- App-level interceptions (never reach textarea) ---

	switch msg.Type {
	case tea.KeyTab:
		a.ctrl.SwitchProfile(1)
		a.rebuildViews()
		return a, nil

	case tea.KeyShiftTab:
		a.ctrl.SwitchProfile(-1)
		a.rebuildViews()
		return a, nil

	case tea.KeyCtrlC:
		_ = a.ctrl.SaveSession()
		a.quitting = true
		return a, tea.Quit
	}

	// --- Autocomplete-aware keys ---
	if a.autocomplete.IsActive() {
		switch msg.Type {
		case tea.KeyUp:
			a.autocomplete.MoveUp()
			return a, nil
		case tea.KeyDown:
			a.autocomplete.MoveDown()
			return a, nil
		case tea.KeyEnter:
			completed := a.autocomplete.Accept(a.input.Value())
			a.input.SetValue(completed)
			a.autocomplete.Deactivate()
			// Accept + submit in one step for slash commands
			if strings.TrimSpace(completed) != "" {
				submitted := a.input.Submit()
				if strings.HasPrefix(submitted, "/") {
					return a.handleCommand(submitted)
				}
				a.chat.AppendMessage("user", submitted, a.ctrl.ActiveProfile(), "")
				a.ctrl.SubmitPrompt(submitted)
			}
			return a, nil
		case tea.KeyEscape:
			a.autocomplete.Deactivate()
			return a, nil
		case tea.KeyTab:
			completed := a.autocomplete.Accept(a.input.Value())
			a.input.SetValue(completed)
			a.autocomplete.Deactivate()
			return a, nil
		}
	}

	// --- Enter: submit or command ---
	if msg.Type == tea.KeyEnter {
		text := a.input.Value()
		if strings.TrimSpace(text) == "" {
			return a, nil
		}
		submitted := a.input.Submit()
		a.autocomplete.Deactivate()
		// REQ-RELOAD-11: handle slash commands before submitting.
		if strings.HasPrefix(submitted, "/") {
			return a.handleCommand(submitted)
		}
		a.chat.AppendMessage("user", submitted, a.ctrl.ActiveProfile(), "")
		a.ctrl.SubmitPrompt(submitted)
		return a, nil
	}

	// --- 'q' to quit (only when input is empty) ---
	if msg.Type == tea.KeyRunes {
		runes := msg.Runes
		if len(runes) == 1 && runes[0] == 'q' && a.input.Value() == "" {
			_ = a.ctrl.SaveSession()
			a.quitting = true
			return a, tea.Quit
		}
	}

	// --- Delegate everything else to InputModel ---
	var cmd tea.Cmd
	a.input, cmd = a.input.Update(msg)

	// After input update: check if we should trigger autocomplete
	val := a.input.Value()
	if strings.HasPrefix(val, "/") {
		if !a.autocomplete.IsActive() {
			a.autocomplete.Activate(val)
		} else {
			a.autocomplete.Filter(val)
		}
	} else if a.autocomplete.IsActive() {
		a.autocomplete.Deactivate()
	}

	return a, cmd
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
		a.chat.SetStatus("commands: /sessions, /resume <id>, /rename <name>, /undo, /redo, /reload, /quit, /exit, /help")
	case "/theme":
		a.handleThemeCommand(parts)
	case "/status":
		a.handleStatusCommand()
	case "/clear":
		a.chat.SetStatus("chat cleared")
	case "/rename":
		a.handleRenameCommand(parts)
	case "/undo":
		a.handleUndoCommand()
	case "/redo":
		a.handleRedoCommand()
	default:
		a.chat.SetStatus("unknown command: " + text + " (try /help)")
	}
	return a, nil
}

// handleThemeCommand switches the active theme.
func (a *App) handleThemeCommand(parts []string) {
	if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
		a.chat.SetStatus("usage: /theme <name>")
		return
	}
	themeName := strings.TrimSpace(parts[1])
	t := theme.Load(themeName)
	a.styles = theme.NewStyles(t)
	a.chat.SetStatus("theme changed to: " + themeName)
	a.rebuildViews()
}

// handleStatusCommand shows current app status.
func (a *App) handleStatusCommand() {
	profile := a.ctrl.ActiveProfile()
	a.chat.SetStatus("profile: " + profile)
}

// handleRenameCommand renames the active session.
func (a *App) handleRenameCommand(parts []string) {
	if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
		a.chat.SetStatus("usage: /rename <name>")
		return
	}
	name := strings.TrimSpace(parts[1])
	if err := a.ctrl.RenameSession(name); err != nil {
		a.chat.SetStatus("error renaming session: " + err.Error())
		return
	}
	a.chat.SetStatus("session renamed to: " + name)
}

// handleUndoCommand undoes the last message pair.
func (a *App) handleUndoCommand() {
	if !a.ctrl.Undo() {
		a.chat.SetStatus("nothing to undo")
		return
	}
	a.chat.SetStatus("undid last turn")
}

// handleRedoCommand redoes the last undone message pair.
func (a *App) handleRedoCommand() {
	if !a.ctrl.Redo() {
		a.chat.SetStatus("nothing to redo")
		return
	}
	a.chat.SetStatus("redid last turn")
}

// handleSessionsCommand lists all saved sessions. When sessions exist,
// it opens an interactive list view; otherwise it shows a status message.
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

	// Open interactive session list view
	list := views.NewSessionListModel(metas, a.width, a.height-4)
	a.sessionList = &list
	a.listMode = true
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

	// Session list mode: render the interactive list instead of the normal layout
	if a.listMode && a.sessionList != nil {
		return a.sessionList.View()
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

	// Input line at bottom — with autocomplete popup if active
	inputLine := a.input.View()
	if a.autocomplete.IsActive() {
		popup := a.autocomplete.View()
		if popup != "" {
			inputLine = popup + "\n" + inputLine
		}
	}

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
	b.WriteString("\n")
	b.WriteString(a.footer.Render())

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

	// Populate footer from controller state
	a.footer.SetModel(a.ctrl.ModelName())
	a.footer.SetTokens(a.ctrl.TotalTokens(), a.ctrl.ContextWindow())
	a.footer.SetCost(a.ctrl.Cost())
}

// chat returns the chat model for inspection.
func (a *App) chatView() *views.ChatModel {
	return &a.chat
}
