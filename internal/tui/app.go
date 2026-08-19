package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/biggs-100/kui/internal/tui/toast"
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
	diff   views.DiffModel
	styles *theme.Styles

	width  int
	height int
	input  InputModel
	autocomplete AutocompleteModel
	quitting bool

	// Diff view toggle: when true, the diff panel is rendered instead of chat.
	diffVisible bool

	// Session list mode: when non-nil, the session list view is active.
	sessionList *views.SessionListModel
	listMode    bool

	// Command palette mode: when true, the command palette overlay is active.
	paletteMode    bool
	commandPalette *views.CommandPaletteModel

	// Registry holds all command metadata and dispatches commands.
	registry *CommandRegistry

	// toast manages non-blocking notification overlays.
	toast *toast.Model

	// currentTheme tracks the active theme name for cycling.
	currentTheme string

	// lspPendingG tracks whether 'g' was pressed and we're waiting for the
	// second key in a gd/gr sequence. When true, the next key press is checked
	// for 'd' or 'r' to complete the LSP keybinding.
	lspPendingG bool
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
		ctrl:         ctrl,
		styles:       styles,
		input:        NewInputModel("type here...", ""),
		autocomplete: NewAutocompleteModel(),
		chat:         views.NewChatModel(styles),
		tool:         views.NewToolModel(styles),
		footer:       views.NewFooterModel(styles),
		diff:         views.NewDiffModel(styles),
		registry:     NewCommandRegistry(),
		toast:        toast.NewModel(styles),
		currentTheme: themeName,
	}
}

// Input returns the App's InputModel for inspection.
func (a *App) Input() *InputModel {
	return &a.input
}

// Registry returns the App's CommandRegistry for inspection.
func (a *App) Registry() *CommandRegistry {
	return a.registry
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
	// Forward TickMsg to toast model
	if _, ok := msg.(toast.TickMsg); ok {
		updated, cmd := a.toast.Update(msg)
		a.toast = updated
		return a, cmd
	}

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
	// --- Command palette mode: delegate all keys to palette ---
	if a.paletteMode && a.commandPalette != nil {
		updated, cmd := a.commandPalette.Update(msg)
		a.commandPalette = &updated
		if a.commandPalette.Selected() != "" {
			name := a.commandPalette.Selected()
			a.commandPalette = nil
			a.paletteMode = false
			return a.executeCommandByName(name)
		}
		if a.commandPalette == nil || a.commandPalette.View() == "" {
			a.commandPalette = nil
			a.paletteMode = false
			return a, cmd
		}
		return a, cmd
	}

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
	case tea.KeyCtrlP:
		// Open command palette
		a.registry = NewCommandRegistry() // refresh registry
		cmds := a.registry.All()
		palette := views.NewCommandPaletteModel(cmds, a.width, a.height-4)
		a.commandPalette = &palette
		a.paletteMode = true
		return a, nil

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

		// --- LSP keybindings (only when input is empty) ---
		// Must be checked BEFORE single-key handlers like 'd' to avoid conflicts.
		if a.input.Value() == "" && len(runes) == 1 {
			r := runes[0]
			if a.lspPendingG {
				// Complete the gd/gr sequence.
				a.lspPendingG = false
				switch r {
				case 'd':
					return a.dispatchLspTool("lsp_definition")
				case 'r':
					return a.dispatchLspTool("lsp_references")
				default:
					return a, nil
				}
			}
			// Start of gd/gr sequence — intercept 'g' so it doesn't type into input.
			if r == 'g' {
				a.lspPendingG = true
				return a, nil
			}
			// 'K' for hover — intercept so it doesn't type into input.
			if r == 'K' {
				return a.dispatchLspTool("lsp_hover")
			}
		}

		// --- 'd' to toggle diff view (only when input is empty) ---
		if len(runes) == 1 && runes[0] == 'd' && a.input.Value() == "" {
			a.diffVisible = !a.diffVisible
			return a, nil
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

// executeCommandByName dispatches a command selected from the palette.
func (a *App) executeCommandByName(name string) (tea.Model, tea.Cmd) {
	cmd := a.registry.Lookup(name)
	if cmd == nil {
		a.chat.SetStatus("unknown command: " + name)
		return a, nil
	}

	// Handle built-in commands that need app context
	switch name {
	case "/quit", "/exit":
		_ = a.ctrl.SaveSession()
		a.quitting = true
		return a, tea.Quit
	case "/reload":
		a.ctrl.Reload()
		return a, nil
	case "/sessions":
		a.handleSessionsCommand()
		return a, nil
	case "/resume":
		a.chat.SetStatus("usage: /resume <session-id> (use /sessions to browse)")
		return a, nil
	case "/rename":
		a.chat.SetStatus("usage: /rename <name>")
		return a, nil
	case "/theme":
		a.chat.SetStatus("usage: /theme <name|next|prev>")
		return a, nil
	case "/status":
		a.handleStatusCommand()
		return a, nil
	case "/clear":
		a.chat.SetStatus("chat cleared")
		return a, nil
	case "/undo":
		a.handleUndoCommand()
		return a, nil
	case "/redo":
		a.handleRedoCommand()
		return a, nil
	case "/help":
		a.chat.SetStatus(a.registry.HelpText())
		return a, nil
	}

	return a, nil
}

// handleCommand dispatches slash commands. Uses the registry for help text
// and dispatches commands via handler functions (REQ-RELOAD-11).
func (a *App) handleCommand(text string) (tea.Model, tea.Cmd) {
	parts := strings.SplitN(text, " ", 2)
	cmd := parts[0]

	// Use registry for /help
	if cmd == "/help" {
		a.chat.SetStatus(a.registry.HelpText())
		return a, nil
	}

	// Dispatch remaining commands
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
		a.chat.SetStatus("usage: /theme <name|next|prev>")
		return
	}
	sub := strings.TrimSpace(parts[1])

	switch sub {
	case "next", "prev":
		a.cycleTheme(sub == "next")
	default:
		a.switchTheme(sub)
	}
}

// cycleTheme moves to the next or previous theme in the list.
func (a *App) cycleTheme(forward bool) {
	names := theme.ThemeNames()
	if len(names) == 0 {
		a.chat.SetStatus("no themes found")
		return
	}

	// Find current index
	idx := 0
	for i, name := range names {
		if name == a.currentTheme {
			idx = i
			break
		}
	}

	// Cycle
	if forward {
		idx = (idx + 1) % len(names)
	} else {
		idx = (idx - 1 + len(names)) % len(names)
	}

	a.switchTheme(names[idx])
}

// switchTheme loads a theme by name and updates all views.
func (a *App) switchTheme(name string) {
	t := theme.Load(name)
	a.styles = theme.NewStyles(t)
	a.currentTheme = name
	a.toast.Push("theme: "+name, toast.LevelSuccess, 3*time.Second)
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

	// Command palette mode: render the palette instead of the normal layout
	if a.paletteMode && a.commandPalette != nil {
		return a.commandPalette.View()
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

	// Chat or Diff view: fills remaining space
	var mainStr string
	if a.diffVisible {
		mainStr = a.diff.View()
	} else {
		mainStr = a.chat.Render()
	}

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
	b.WriteString(mainStr)
	b.WriteString("\n")
	if toolStr != "" {
		b.WriteString(toolStr)
		b.WriteString("\n")
	}

	// Toast overlay: rendered between tool view and input
	toastStr := a.toast.View()
	if toastStr != "" {
		b.WriteString(toastStr)
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

// dispatchLspTool dispatches an LSP tool call via the controller and shows
// the result in the chat. When the dispatcher is not configured, it shows
// an error status. The file URI and cursor position are extracted from the
// last user message context (line 0, character 0 defaults).
func (a *App) dispatchLspTool(toolName string) (tea.Model, tea.Cmd) {
	// Default to a placeholder URI — real implementation would extract
	// from the editor context or last file read.
	uri := "file:///."
	line := 0
	character := 0

	args := map[string]interface{}{
		"uri":       uri,
		"line":      line,
		"character": character,
	}

	result, err := a.ctrl.DispatchLsp(toolName, args)
	if err != nil {
		a.chat.SetStatus(toolName + ": " + err.Error())
		return a, nil
	}

	a.chat.AppendMessage("assistant", result, a.ctrl.ActiveProfile(), "")
	return a, nil
}
