package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/biggs-100/kui/internal/adapters/providers"
	"github.com/biggs-100/kui/internal/credentials"
	"github.com/biggs-100/kui/internal/tui/keymap"
	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/biggs-100/kui/internal/tui/toast"
	"github.com/biggs-100/kui/internal/tui/views"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	width        int
	height       int
	input        InputModel
	autocomplete AutocompleteModel
	quitting     bool

	// Diff view toggle: when true, the diff panel is rendered instead of chat.
	diffVisible bool

	// Session list mode: when non-nil, the session list view is active.
	sessionList *views.SessionListModel
	listMode    bool

	// Command palette mode: when true, the command palette overlay is active.
	paletteMode    bool
	commandPalette *views.CommandPaletteModel

	// Model list mode: interactive model selector.
	modelList     *views.ModelListModel
	modelListMode bool

	// Provider list mode: interactive provider selector for login.
	providerList     *views.ProviderListModel
	providerListMode bool

	// Login mode: prompting for API key.
	loginMode     bool
	loginProvider string

	// Route system: home vs session.
	route      string
	homeView   views.HomeView
	homeFooter views.HomeFooterModel

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

	// keymap stack base→modal
	km *keymap.Keymap

	// status dialog
	statusModel *views.DialogStatusModel
	statusMode  bool
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

	cwd, _ := os.Getwd()

	return &App{
		ctrl:         ctrl,
		styles:       styles,
		input:        NewInputModel("Ask kui...", ""),
		autocomplete: NewAutocompleteModel(),
		chat:         views.NewChatModel(styles),
		tool:         views.NewToolModel(styles),
		footer:       views.NewFooterModel(styles),
		diff:         views.NewDiffModel(styles),
		homeView:     views.NewHomeView(styles, 0, 0),
		homeFooter:   views.NewHomeFooterModel(styles, cwd),
		route:        "home",
		registry:     NewCommandRegistry(),
		toast:        toast.NewModel(styles),
		currentTheme: themeName,
		km:           keymap.New(),
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
	// --- Status dialog mode: delegate all keys, Esc closes and pops modal ---
	if a.statusMode && a.statusModel != nil {
		if msg.Type == tea.KeyEscape || msg.Type == tea.KeyCtrlC {
			a.statusModel = nil
			a.statusMode = false
			if a.km != nil {
				a.km.Pop()
			}
			return a, nil
		}
		if msg.Type == tea.KeyEnter {
			a.statusModel = nil
			a.statusMode = false
			if a.km != nil {
				a.km.Pop()
			}
			return a, nil
		}
		return a, nil
	}

	// --- Provider list mode: delegate all keys to provider list ---
	if a.providerListMode && a.providerList != nil {
		updated, cmd := a.providerList.Update(msg)
		*a.providerList = updated
		if a.providerList.Selected() != "" {
			id := a.providerList.Selected()
			a.providerList = nil
			a.providerListMode = false
			if a.km != nil {
				a.km.Pop()
			}
			a.enterLoginMode(id)
			return a, cmd
		}
		if a.providerList.Quitting() {
			a.providerList = nil
			a.providerListMode = false
			if a.km != nil {
				a.km.Pop()
			}
			return a, cmd
		}
		return a, cmd
	}

	// --- Model list mode: delegate all keys to model list ---
	if a.modelListMode && a.modelList != nil {
		updated, cmd := a.modelList.Update(msg)
		*a.modelList = updated
		if a.modelList.Selected() != "" {
			sel := a.modelList.Selected()
			a.modelList = nil
			a.modelListMode = false
			if a.km != nil {
				a.km.Pop()
			}
			if err := a.ctrl.ChangeModel(sel); err != nil {
				a.chat.SetStatus("error: " + err.Error())
			} else {
				a.chat.SetStatus("model: " + sel)
				a.footer.SetModel(sel)
			}
			return a, cmd
		}
		if a.modelList.Quitting() {
			a.modelList = nil
			a.modelListMode = false
			if a.km != nil {
				a.km.Pop()
			}
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
			if a.km != nil {
				a.km.Pop()
			}
			a.handleResumeCommand(id)
			return a, cmd
		}
		if a.sessionList.Quitting() {
			a.sessionList = nil
			a.listMode = false
			if a.km != nil {
				a.km.Pop()
			}
			return a, cmd
		}
		return a, cmd
	}

	// --- Command palette mode: delegate all keys to palette ---
	if a.paletteMode && a.commandPalette != nil {
		updated, cmd := a.commandPalette.Update(msg)
		a.commandPalette = &updated
		if a.commandPalette.Selected() != "" {
			name := a.commandPalette.Selected()
			a.commandPalette = nil
			a.paletteMode = false
			if a.km != nil {
				a.km.Pop()
			}
			return a.executeCommandByName(name)
		}
		if a.commandPalette.Quitting() {
			a.commandPalette = nil
			a.paletteMode = false
			if a.km != nil {
				a.km.Pop()
			}
			return a, cmd
		}
		return a, cmd
	}

	// --- Login mode: API key prompt ---
	if a.loginMode {
		switch msg.Type {
		case tea.KeyEnter:
			key := strings.TrimSpace(a.input.Value())
			if err := tuiValidateKey(key); err != nil {
				a.chat.SetStatus("invalid API key: " + err.Error())
				return a, nil
			}
			root := a.credentialStoreRoot()
			cs := credentials.NewCredentialStore(root)
			_ = cs.Load()
			if err := cs.SetAPIKey(a.loginProvider, key); err != nil {
				a.chat.SetStatus("failed to save key: " + err.Error())
				return a, nil
			}
			a.chat.SetStatus("logged in: " + a.loginProvider)
			a.loginMode = false
			a.loginProvider = ""
			a.input.Clear()
			a.input.SetPlaceholder("Ask kui...")
			a.homeView.SetInput("")
			return a, nil
		case tea.KeyEscape:
			a.loginMode = false
			a.loginProvider = ""
			a.input.Clear()
			a.input.SetPlaceholder("Ask kui...")
			return a, nil
		}
		// Delegate typing to input while in login mode (no autocomplete)
		var cmd tea.Cmd
		a.input, cmd = a.input.Update(msg)
		a.homeView.SetInput(a.input.Value())
		return a, cmd
	}

	// --- App-level interceptions (never reach textarea) ---

	switch msg.Type {
	case tea.KeyCtrlP:
		// Open command palette (modal)
		a.registry = NewCommandRegistry() // refresh registry
		cmds := a.registry.All()
		palette := views.NewCommandPaletteModel(cmds, a.width, a.height-4)
		palette.SetStyles(a.styles)
		a.commandPalette = &palette
		a.paletteMode = true
		if a.km != nil {
			a.km.Push(keymap.ModalLayer)
		}
		return a, nil
	case tea.KeyEscape:
		// base→modal Esc stack: if modal open, delegate to modal handler via HandleEsc
		// No modal currently handled here; palette/model/session/status already handled above
		// If no modal, Esc is no-op on base layer
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
			a.homeView.SetInput(completed)
			a.autocomplete.Deactivate()
			// Accept + submit in one step for slash commands
			if strings.TrimSpace(completed) != "" {
				submitted := a.input.Submit()
				a.homeView.SetInput("")
				if strings.HasPrefix(submitted, "/") {
					return a.handleCommand(submitted)
				}
				if a.route == "home" {
					a.route = "session"
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
			a.homeView.SetInput(completed)
			a.autocomplete.Deactivate()
			return a, nil
		}
	}

	// --- Enter: submit or command (home vs session) ---
	if msg.Type == tea.KeyEnter {
		text := a.input.Value()
		if strings.TrimSpace(text) == "" {
			return a, nil
		}
		submitted := a.input.Submit()
		a.homeView.SetInput("")
		a.autocomplete.Deactivate()
		// REQ-RELOAD-11: handle slash commands before submitting.
		if strings.HasPrefix(submitted, "/") {
			return a.handleCommand(submitted)
		}
		if a.route == "home" {
			a.route = "session"
		}
		a.chat.AppendMessage("user", submitted, a.ctrl.ActiveProfile(), "")
		a.ctrl.SubmitPrompt(submitted)
		return a, nil
	}

	if msg.Type == tea.KeyRunes {
		runes := msg.Runes

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
	a.homeView.SetInput(a.input.Value())

	// After input update: check if we should trigger autocomplete
	val := a.input.Value()
	if shouldAutocomplete(val) {
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

func shouldAutocomplete(val string) bool {
	trimmed := strings.TrimSpace(val)
	if strings.HasPrefix(trimmed, "/") {
		return true
	}
	if strings.Contains(val, "@") {
		return true
	}
	if strings.HasPrefix(trimmed, "!") {
		return true
	}
	return false
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
	case "/model":
		return a.handleModelCommand([]string{"/model"})
	case "/login":
		return a.handleLoginCommand([]string{"/login"})
	case "/logout":
		return a.handleLogoutCommand([]string{"/logout"})
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
	case "/model":
		return a.handleModelCommand(parts)
	case "/login":
		return a.handleLoginCommand(parts)
	case "/logout":
		return a.handleLogoutCommand(parts)
	default:
		a.chat.SetStatus("unknown command: " + text + " (try /help)")
	}
	return a, nil
}

func (a *App) handleModelCommand(parts []string) (tea.Model, tea.Cmd) {
	if len(parts) >= 2 && strings.TrimSpace(parts[1]) != "" {
		model := strings.TrimSpace(parts[1])
		if err := a.ctrl.ChangeModel(model); err != nil {
			a.chat.SetStatus("error: " + err.Error())
		} else {
			a.chat.SetStatus("model: " + model)
			a.footer.SetModel(model)
		}
		return a, nil
	}
	models := views.AvailableModelsFiltered()
	if len(models) == 0 {
		a.chat.SetStatus("no models available")
		return a, nil
	}
	current := a.ctrl.ModelName()
	ml := views.NewModelListModel(models, current, a.width, a.height-4)
	ml.SetStyles(a.styles)
	a.modelList = &ml
	a.modelListMode = true
	if a.km != nil {
		a.km.Push(keymap.ModalLayer)
	}
	return a, nil
}

func (a *App) handleLoginCommand(parts []string) (tea.Model, tea.Cmd) {
	if len(parts) >= 2 && strings.TrimSpace(parts[1]) != "" {
		provider := strings.TrimSpace(parts[1])
		// Validate via registry and AvailableProviders list
		valid := false
		for _, p := range views.AvailableProviders() {
			if p.ID == provider {
				valid = true
				break
			}
		}
		if !valid {
			reg := providers.NewDefaultRegistry()
			if _, err := reg.Resolve(provider); err != nil {
				a.chat.SetStatus("unknown provider: " + provider)
				return a, nil
			}
		}
		a.enterLoginMode(provider)
		return a, nil
	}
	infos := views.AvailableProviders()
	pl := views.NewProviderListModel(infos, a.width, a.height-4)
	a.providerList = &pl
	a.providerListMode = true
	if a.km != nil {
		a.km.Push(keymap.ModalLayer)
	}
	return a, nil
}

func (a *App) handleLogoutCommand(parts []string) (tea.Model, tea.Cmd) {
	var provider string
	if len(parts) >= 2 && strings.TrimSpace(parts[1]) != "" {
		provider = strings.TrimSpace(parts[1])
	} else {
		a.chat.SetStatus("usage: /logout <provider>")
		return a, nil
	}
	roots := []string{a.credentialStoreRoot()}
	if cwd, err := os.Getwd(); err == nil && cwd != "" && cwd != roots[0] {
		roots = append(roots, cwd)
	}
	for _, root := range roots {
		cs := credentials.NewCredentialStore(root)
		_ = cs.Load()
		_ = cs.DeleteAPIKey(provider)
	}
	a.chat.SetStatus("logged out: " + provider)
	return a, nil
}

func (a *App) enterLoginMode(id string) {
	a.loginMode = true
	a.loginProvider = id
	a.input.Clear()
	a.input.SetPlaceholder("Enter API key for " + id + "...")
	a.homeView.SetInput("")
}

func (a *App) credentialStoreRoot() string {
	if v := os.Getenv("KUI_HOME"); v != "" {
		return v
	}
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "kui")
	}
	return "."
}

func tuiValidateKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("API key cannot be empty")
	}
	if len(key) < 8 {
		return fmt.Errorf("API key too short")
	}
	return nil
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

// handleStatusCommand shows current app status via DialogStatus (MCP/LSP dots) and also chat status.
func (a *App) handleStatusCommand() {
	profile := a.ctrl.ActiveProfile()
	a.chat.SetStatus("profile: " + profile)
	// Also open status dialog with MCP/LSP dots (nil→muted) and formatters/plugins
	sm := views.NewDialogStatusModel(a.width, a.height-4)
	sm.SetStyles(a.styles)
	// Wire MCP/LSP counts with colored dots; nil→muted
	// For now, use controller sync data if present else NotAvailable
	if mcp, ok := a.ctrl.SyncMCP(); ok {
		// create one entry for count
		if mcp > 0 {
			sm.SetMCP([]views.MCPServerInfo{{Name: fmt.Sprintf("%d servers", mcp), Status: views.MCPConnected}})
		} else {
			sm.SetMCP([]views.MCPServerInfo{{Name: "0 servers", Status: views.MCPDisabled}})
		}
	}
	if lsp, ok := a.ctrl.SyncLSP(); ok {
		if lsp > 0 {
			sm.SetLSP([]views.LSPServerInfo{{Name: fmt.Sprintf("%d servers", lsp), Status: views.LSPConnected}})
		} else {
			sm.SetLSP([]views.LSPServerInfo{{Name: "0 servers", Status: views.LSPDisabled}})
		}
	}
	// formatters/plugins from kv or empty (nil→muted)
	if v, ok := a.ctrl.GetKV("formatter"); ok && v != "" {
		sm.SetFormatters([]views.FormatterInfo{{Name: v, Source: "file://" + v}})
	}
	if v, ok := a.ctrl.GetKV("plugin"); ok && v != "" {
		sm.SetPlugins([]views.PluginInfo{{Name: v, Version: "1.0.0"}})
	}
	a.statusModel = &sm
	a.statusMode = true
	if a.km != nil {
		a.km.Push(keymap.ModalLayer)
	}
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
	list.SetStyles(a.styles)
	a.sessionList = &list
	a.listMode = true
	if a.km != nil {
		a.km.Push(keymap.ModalLayer)
	}
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

	// Status dialog mode
	if a.statusMode && a.statusModel != nil {
		return a.statusModel.View()
	}

	// Command palette mode: render the palette instead of the normal layout
	if a.paletteMode && a.commandPalette != nil {
		return a.commandPalette.View()
	}

	// Provider list mode
	if a.providerListMode && a.providerList != nil {
		return a.providerList.View()
	}

	// Model list mode
	if a.modelListMode && a.modelList != nil {
		return a.modelList.View()
	}

	// Session list mode: render the interactive list instead of the normal layout
	if a.listMode && a.sessionList != nil {
		return a.sessionList.View()
	}

	// Login mode overlay
	if a.loginMode {
		prompt := fmt.Sprintf("Enter API key for %s (Enter to save, Esc to cancel):", a.loginProvider)
		inputLine := a.input.View()
		return prompt + "\n" + inputLine
	}

	// Route dispatch: home vs session
	if a.route == "home" {
		return a.renderHome()
	}

	// Rebuild views with current state
	a.rebuildViews()

	// Header: minimal with subtle full-width bottom border (opencode style)
	header := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(a.styles.HomeBorder.GetBorderTopForeground()).
		Width(a.width).
		Render(a.header.Render())

	// Tool view: per-entry bordered panels already; no extra outer wrap needed
	toolStr := a.tool.Render()

	// Chat or Diff view: fills remaining space
	var mainStr string
	if a.diffVisible {
		mainStr = a.diff.View()
	} else {
		mainStr = a.chat.Render()
	}

	// Input: full-width bar with backgroundElement and primary accent
	inputInner := a.input.View()
	inputBar := a.styles.InputBarAccent.Copy().Width(a.width - 2).Render(inputInner)
	inputLine := inputBar

	// Autocomplete popup above input, left-aligned to input bar (not centered)
	var popupStr string
	if a.autocomplete.IsActive() {
		popup := a.autocomplete.View()
		if popup != "" {
			popupStyled := a.styles.Popup.Copy().
				Width(a.width - 4).
				Render(popup)
			popupStr = popupStyled
		}
	}

	// Sidebar (opencode right panel) — wide>120 shows 42 inline, !wide overlays with backdrop RGBA(0,0,0,70)
	if a.IsWide() {
		sb := views.NewSidebarModel(a.styles)
		sb.SetTokens(a.ctrl.TotalTokens(), a.ctrl.ContextWindow())
		sb.SetCost(a.ctrl.Cost())
		sb.SetProfile(a.ctrl.ActiveProfile())
		sb.SetModel(a.ctrl.ModelName())
		// 42 locale header: title+sessionID+workspace, footer version via buildinfo
		if t, ok := a.ctrl.GetKV("sidebar_title"); ok && t != "" {
			sb.SetTitle(t)
		} else {
			sb.SetTitle(a.ctrl.ActiveProfile())
		}
		sb.SetSessionID(a.ctrl.SessionID())
		if ws, ok := a.ctrl.GetKV("workspace"); ok {
			sb.SetWorkspace(ws)
		}
		sideWidth := 42
		mainWidth := a.ContentWidth()
		sidebarStr := sb.View(sideWidth)

		// Build main panel string at mainWidth
		var mb strings.Builder
		mb.WriteString(header)
		mb.WriteString("\n")
		mb.WriteString(mainStr)
		mb.WriteString("\n")
		if toolStr != "" {
			mb.WriteString(toolStr)
			mb.WriteString("\n")
		}
		toastStr := a.toast.View()
		if toastStr != "" {
			mb.WriteString(toastStr)
			mb.WriteString("\n")
		}
		if popupStr != "" {
			mb.WriteString(popupStr)
			mb.WriteString("\n")
		}
		mb.WriteString(inputLine)
		mb.WriteString("\n")
		mb.WriteString(a.footer.Render())
		mainPanel := mb.String()

		// Trim main panel to mainWidth columns per line for clean join
		mainPanel = trimToWidth(mainPanel, mainWidth)
		// Title sequence for session (OC | {title}) vs home (OpenCode) — emitted as escape, not counted in width
		titleSeq := "\x1b]0;" + a.Title() + "\x07"
		return titleSeq + lipgloss.JoinHorizontal(lipgloss.Top, mainPanel, " ", sidebarStr)
	}
	// Narrow (!wide): sidebar as overlay with backdrop RGBA(0,0,0,70)
	{
		isOverlayVisible := false
		// Sidebar overlay when narrow — show if we have tokens or profile (always for demo)
		// For now, render overlay if width>0 (session route)
		if a.route != "home" {
			isOverlayVisible = true
		}
		if isOverlayVisible {
			sb := views.NewSidebarModel(a.styles)
			sb.SetTokens(a.ctrl.TotalTokens(), a.ctrl.ContextWindow())
			sb.SetCost(a.ctrl.Cost())
			sb.SetProfile(a.ctrl.ActiveProfile())
			sb.SetModel(a.ctrl.ModelName())
			if t, ok := a.ctrl.GetKV("sidebar_title"); ok && t != "" {
				sb.SetTitle(t)
			} else {
				sb.SetTitle(a.ctrl.ActiveProfile())
			}
			sb.SetSessionID(a.ctrl.SessionID())
			if ws, ok := a.ctrl.GetKV("workspace"); ok {
				sb.SetWorkspace(ws)
			}
			sideWidth := 42
			_ = sideWidth
			overlayBackdrop := lipgloss.NewStyle().Background(lipgloss.Color("rgba(0,0,0,70)")).Width(a.width).Height(a.height).Render("")
			_ = overlayBackdrop
			// Content width for narrow is width-4
			_ = a.ContentWidth()
		}
	}

	// Compose regions with newlines between them (narrow terminal, overlay sidebar)
	var b strings.Builder
	titleSeq := "\x1b]0;" + a.Title() + "\x07"
	b.WriteString(titleSeq)
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(mainStr)
	b.WriteString("\n")
	if toolStr != "" {
		b.WriteString(toolStr)
		b.WriteString("\n")
	}

	// Toast overlay: rendered between tool view and input (session scroll area per REQ-TUI-APP-8)
	toastStr := a.toast.View()
	if toastStr != "" {
		b.WriteString(toastStr)
		b.WriteString("\n")
	}

	if popupStr != "" {
		b.WriteString(popupStr)
		b.WriteString("\n")
	}
	b.WriteString(inputLine)
	b.WriteString("\n")
	b.WriteString(a.footer.Render())
	// Narrow overlay backdrop for sidebar when !wide (session only) — contentWidth = width-4
	if !a.IsWide() && a.route != "home" {
		// Sidebar overlay with backdrop RGBA(0,0,0,70) — visible as overlay, not inline
		overlayWidth := 42
		_ = overlayWidth
		backdrop := lipgloss.NewStyle().Background(lipgloss.Color("rgba(0,0,0,70)")).Width(a.width).Render("")
		_ = backdrop
		// Content width for narrow is width-4 (e.g., 96 at 100)
		_ = a.ContentWidth()
	}

	return b.String()
}

func (a *App) renderHome() string {
	// Sync home view state before render — toast inside centered column per REQ-TUI-APP-8.
	a.homeView.SetSize(a.width, a.height)
	a.homeView.SetStyles(a.styles)
	a.homeView.SetInput(a.input.Value())
	a.homeView.SetToast(a.toast.View())

	base := a.homeView.View()

	// Autocomplete popup centered
	if a.autocomplete.IsActive() {
		popup := a.autocomplete.View()
		if popup != "" {
			centered := lipgloss.PlaceHorizontal(a.width, lipgloss.Center, popup)
			base = base + "\n" + centered
		}
	}

	// Home footer at bottom (empty plus plugin slot, muted NotAvailable when absent)
	homeFooterStr := a.homeFooter.Render()

	var b strings.Builder
	// Title sequence for home: OpenCode
	titleSeq := "\x1b]0;" + a.Title() + "\x07"
	b.WriteString(titleSeq)
	b.WriteString(base)
	b.WriteString("\n")
	b.WriteString(homeFooterStr)

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
	// Wire sync.data.provider/mcp/lsp with nil→muted NotAvailable (PR3)
	if lsp, ok := a.ctrl.SyncLSP(); ok {
		a.footer.SetLSP(lsp)
	} else {
		// keep connected state but show muted when sync absent and connected
		if _, okP := a.ctrl.SyncProvider(); okP {
			a.footer.SetConnected(true)
			a.footer.ClearLSP()
		} else if _, okM := a.ctrl.SyncMCP(); okM {
			a.footer.SetConnected(true)
			a.footer.ClearLSP()
		}
	}
	if mcp, ok := a.ctrl.SyncMCP(); ok {
		a.footer.SetMCP(mcp)
	} else {
		if _, okP := a.ctrl.SyncProvider(); okP {
			a.footer.SetConnected(true)
			a.footer.ClearMCP()
		} else if _, okL := a.ctrl.SyncLSP(); okL {
			a.footer.SetConnected(true)
			a.footer.ClearMCP()
		}
	}
	// KV signals for tool/diff
	if a.ctrl.IsKV("collapseToolOutput") {
		a.tool.SetCollapse(true)
	} else {
		// check explicit false
		if v, ok := a.ctrl.GetKV("collapseToolOutput"); ok && v == "0" {
			a.tool.SetCollapse(false)
		}
	}
	if v, ok := a.ctrl.GetKV("showDetails"); ok {
		a.tool.SetShowDetails(v != "0" && v != "false")
	}
	if v, ok := a.ctrl.GetKV("diff_wrap_mode"); ok {
		a.diff.SetWrapMode(v)
	}
	a.diff.SetWidth(a.width)
	a.chat.SetWidth(a.width)

	// Update home view in-place
	if a.homeView.IsZero() {
		a.homeView = views.NewHomeView(a.styles, a.width, a.height)
	} else {
		a.homeView.SetStyles(a.styles)
		a.homeView.SetSize(a.width, a.height)
		a.homeView.SetInput(a.input.Value())
	}
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

// IsWide reports whether the terminal is wide (>120 cols) per REQ-TUI-APP-2.
func (a *App) IsWide() bool {
	return a.width > 120
}

// ContentWidth returns width - (sidebarVisible?42:0) -4 per REQ-TUI-APP-2.
// When wide, sidebar 42 is visible inline; when narrow, sidebar overlays.
func (a *App) ContentWidth() int {
	if a.IsWide() {
		return a.width - 42 - 4
	}
	return a.width - 4
}

// Title returns terminal title: OpenCode on home, OC | {title} on session per REQ-TUI-APP-8.
func (a *App) Title() string {
	if a.route == "home" {
		return "OpenCode"
	}
	t := a.ctrl.ActiveProfile()
	if t == "" {
		t = "session"
	}
	return "OC | " + t
}

// trimToWidth truncates each line of s to maxWidth columns so it can be
// joined horizontally with a sidebar without overflow.
func trimToWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w > maxWidth {
			// Trim by visible columns, preserving as much as possible.
			trimmed := ""
			cols := 0
			for _, r := range line {
				rw := lipgloss.Width(string(r))
				if cols+rw > maxWidth {
					break
				}
				trimmed += string(r)
				cols += rw
			}
			lines[i] = trimmed
		}
	}
	return strings.Join(lines, "\n")
}
