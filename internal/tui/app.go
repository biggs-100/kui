package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/biggs-100/kui/internal/adapters/git"
	"github.com/biggs-100/kui/internal/adapters/providers"
	"github.com/biggs-100/kui/internal/credentials"
	"github.com/biggs-100/kui/internal/tui/keymap"
	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/biggs-100/kui/internal/tui/toast"
	"github.com/biggs-100/kui/internal/tui/ui"
	"github.com/biggs-100/kui/internal/tui/views"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
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

	// scrollVP scrolls the conversation (chat or diff) inside its budgeted
	// slot so long content never pushes the pinned input/footer around.
	scrollVP viewport.Model

	// vpContentHeight tracks the previous rendered content height so the
	// viewport can stick to the bottom when new content arrives while it
	// was already pinned there.
	vpContentHeight int

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

	// keymap stack base→modal
	km *keymap.Keymap

	// lastEsc tracks the previous Esc press for the double-Esc interrupt.
	lastEsc time.Time

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

// Init returns the initial command: the footer tick drives the welcome
// status cycle. The controller's event pump is started externally (by Run).
func (a *App) Init() tea.Cmd {
	return scheduleFooterTick()
}

// footerTickMsg advances the footer welcome cycle periodically.
type footerTickMsg struct{}

func scheduleFooterTick() tea.Cmd {
	return tea.Tick(10*time.Second, func(time.Time) tea.Msg { return footerTickMsg{} })
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

	case shellDoneMsg:
		a.chat.AppendShell(msg.script, msg.output, msg.err)
		a.chat.SetStatus("shell: " + msg.script)
		return a, nil

	case compactDoneMsg:
		a.chat.LoadHistory(a.ctrl.Messages())
		a.vpContentHeight = 0
		if msg.err != nil {
			a.chat.SetStatus("compact failed: " + msg.err.Error())
		} else {
			a.chat.SetStatus("conversation compacted")
		}
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

	case bgChangedMsg:
		// Background subagent state changed; the sidebar re-reads the
		// snapshot on the next View render — nothing else to do here.
		return a, nil

	case footerTickMsg:
		a.footer.Tick()
		return a, scheduleFooterTick()

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
		// base→modal Esc stack: modals handle their own Esc earlier.
		// On the base layer, double-Esc within 5s interrupts the running
		// turn (upstream session.interrupt behavior).
		now := time.Now()
		if !a.lastEsc.IsZero() && now.Sub(a.lastEsc) <= 5*time.Second {
			a.lastEsc = time.Time{}
			if a.ctrl.Interrupt() {
				a.chat.SetStatus("interrupted")
			}
			return a, nil
		}
		a.lastEsc = now
		if a.ctrl.IsRunning() {
			a.chat.SetStatus("esc again to interrupt")
		}
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

	case tea.KeyCtrlD:
		a.diffVisible = !a.diffVisible
		if a.diffVisible {
			// Real working-tree diffs from the git adapter — the panel never
			// opens empty on purpose.
			wd, err := os.Getwd()
			if err == nil {
				diffs, derr := git.DiffCommand(wd)
				if derr != nil {
					a.chat.SetStatus("diff: " + derr.Error())
				} else {
					a.diff.SetDiffs(diffs)
				}
			}
		}
		return a, nil
	}

	// No bare-letter interceptions here on purpose: with the input focused,
	// every rune belongs to the prompt. (The old empty-input d/g/K hooks made
	// prompts starting with those letters impossible to type.)

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
				if strings.HasPrefix(submitted, "!") {
					return a.submitShell(submitted)
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
		// Shell mode: "!cmd" executes LOCALLY — it never reaches the LLM.
		if strings.HasPrefix(submitted, "!") {
			return a.submitShell(submitted)
		}
		if a.route == "home" {
			a.route = "session"
		}
		a.chat.AppendMessage("user", submitted, a.ctrl.ActiveProfile(), "")
		a.ctrl.SubmitPrompt(submitted)
		return a, nil
	}

	// --- Scroll keys: page the conversation viewport (chat/diff slot) ---
	if msg.Type == tea.KeyPgUp || msg.Type == tea.KeyPgDown {
		var cmd tea.Cmd
		a.scrollVP, cmd = a.scrollVP.Update(msg)
		return a, cmd
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
		a.chat.Clear()
		a.vpContentHeight = 0 // viewport re-sticks from empty content
		a.chat.SetStatus("conversation view cleared")
		return a, nil
	case "/new":
		a.handleNewCommand()
		return a, nil
	case "/compact":
		return a, startCompact(a)
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
		return a, a.handleThemeCommand(parts)
	case "/status":
		a.handleStatusCommand()
	case "/clear":
		a.chat.Clear()
		a.vpContentHeight = 0
		a.chat.SetStatus("conversation view cleared")
	case "/new":
		a.handleNewCommand()
	case "/compact":
		return a, startCompact(a)
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

// handleThemeCommand switches the active theme and schedules its toast dismissal.
func (a *App) handleThemeCommand(parts []string) tea.Cmd {
	if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
		a.chat.SetStatus("usage: /theme <name|next|prev>")
		return nil
	}
	sub := strings.TrimSpace(parts[1])

	switch sub {
	case "next", "prev":
		return a.cycleTheme(sub == "next")
	default:
		return a.switchTheme(sub)
	}
}

// cycleTheme moves to the next or previous theme in the list.
func (a *App) cycleTheme(forward bool) tea.Cmd {
	names := theme.ThemeNames()
	if len(names) == 0 {
		a.chat.SetStatus("no themes found")
		return nil
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

	return a.switchTheme(names[idx])
}

// switchTheme loads a theme by name, updates all views, and schedules the toast dismissal.
func (a *App) switchTheme(name string) tea.Cmd {
	t := theme.Load(name)
	a.styles = theme.NewStyles(t)
	a.currentTheme = name
	cmd := a.toast.Notify("theme: "+name, toast.LevelSuccess, 3*time.Second)
	a.rebuildViews()
	return cmd
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
	// The view must reflect the restored history — otherwise the panel keeps
	// showing the turn that was just reverted.
	a.chat.LoadHistory(a.ctrl.Messages())
	a.vpContentHeight = 0
	a.chat.SetStatus("undid last turn")
}

// handleRedoCommand redoes the last undone message pair.
func (a *App) handleRedoCommand() {
	if !a.ctrl.Redo() {
		a.chat.SetStatus("nothing to redo")
		return
	}
	a.chat.LoadHistory(a.ctrl.Messages())
	a.vpContentHeight = 0
	a.chat.SetStatus("redid last turn")
}

// handleNewCommand starts a fresh session: clears the conversation view and
// rotates the controller to a new persisted session ID (upstream /new).
func (a *App) handleNewCommand() {
	a.ctrl.StartNewSession()
	a.chat.Clear()
	a.vpContentHeight = 0
	a.route = "home"
	a.chat.SetStatus("")
}

// compactDoneMsg carries the outcome of a manual /compact call.
type compactDoneMsg struct{ err error }

// startCompact runs compaction off the UI goroutine — the LLM summarization
// can take seconds.
func startCompact(a *App) tea.Cmd {
	return func() tea.Msg { return compactDoneMsg{err: a.ctrl.CompactNow()} }
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

// shellDoneMsg carries the result of a real local "!"-shell execution.
type shellDoneMsg struct {
	script string
	output string
	err    error
}

// submitShell routes a "!command" submission to real local execution.
func (a *App) submitShell(submitted string) (tea.Model, tea.Cmd) {
	script := strings.TrimSpace(strings.TrimPrefix(submitted, "!"))
	if script == "" {
		a.chat.SetStatus("usage: !<command>")
		return a, nil
	}
	if a.route == "home" {
		a.route = "session"
	}
	return a, a.execShell(script)
}

// execShell runs the script through the platform shell with a 30s cap and
// returns the combined output as a message.
func (a *App) execShell(script string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.CommandContext(ctx, "cmd", "/C", script)
		} else {
			cmd = exec.CommandContext(ctx, "sh", "-c", script)
		}
		out, err := cmd.CombinedOutput()
		return shellDoneMsg{script: script, output: string(out), err: err}
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

	// Explicit region widths: in wide mode every main-column region renders
	// at ContentWidth so no post-hoc truncation is needed; narrow mode keeps
	// full-width regions and overlays the rail on top. There is no header —
	// the upstream design keeps session identity inside the rail.
	mainWidth := a.width
	if a.IsWide() {
		mainWidth = a.ContentWidth()
	}

	// Tool view: per-entry bordered panels already; no extra outer wrap needed
	toolStr := trimToWidth(a.tool.Render(), mainWidth)

	// Chat or Diff view: fills its budgeted slot (see height budget below)
	var mainStr string
	if a.diffVisible {
		mainStr = trimToWidth(a.diff.View(), mainWidth)
	} else {
		mainStr = a.chat.Render()
	}

	// Input area: OpenCode-style raised field — a left ┃ bar tinted toward
	// the primary accent over an element-fill panel, a meta row (profile ·
	// model) inside the same fill, and a half-block fade-out row beneath so
	// the field reads as fading into the background instead of ending.
	// Trailing bare spaces are trimmed from the textarea render: bubbles
	// pads to its own internal width with UNSTYLED spaces, which would punch
	// a transparent hole through the element fill.
	inputInner := strings.TrimRight(a.input.View(), " ")
	barColor := a.styles.Theme.Border
	if barColor != "" && a.styles.Theme.Primary != "" {
		barColor = theme.Tint(a.styles.Theme.Border, a.styles.Theme.Primary, 0.55)
	}
	fieldStyle := lipgloss.NewStyle().
		Border(ui.SplitBorder).
		BorderForeground(lipgloss.Color(barColor)).
		BorderBottom(false).
		Background(lipgloss.Color(a.styles.Theme.BackgroundElement)).
		Padding(1, 2, 0, 2).
		Width(mainWidth - 4)
	field := fieldStyle.Render(inputInner)

	var metaLine string
	if profile := a.ctrl.ActiveProfile(); profile != "" {
		name := lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.Color(a.styles.Theme.Primary)).Render(profile)
		model := a.ctrl.ModelName()
		dot := lipgloss.NewStyle().
			Foreground(lipgloss.Color(a.styles.Theme.TextMuted)).Render(" · ")
		var modelName string
		if model != "" {
			modelName = lipgloss.NewStyle().
				Foreground(lipgloss.Color(a.styles.Theme.Text)).Render(model)
		}
		metaLine = name + dot + modelName
	}
	metaRow := ""
	if metaLine != "" {
		metaRow = lipgloss.NewStyle().
			Background(lipgloss.Color(a.styles.Theme.BackgroundElement)).
			Padding(0, 2).
			Width(mainWidth - 4).
			Render(metaLine)
	}
	fade := lipgloss.NewStyle().Foreground(lipgloss.Color(barColor)).Render("╹") +
		lipgloss.NewStyle().Foreground(lipgloss.Color(a.styles.Theme.BackgroundElement)).
			Render(strings.Repeat("▀", max(0, mainWidth-2)))

	inputLine := field
	if metaRow != "" {
		inputLine += "\n" + metaRow
	}
	inputLine += "\n" + fade

	// Autocomplete popup: floats OVER the conversation just above the
	// prompt box instead of occupying its own budget slot.
	var popupStr string
	if a.autocomplete.IsActive() {
		popup := a.autocomplete.View()
		if popup != "" {
			popupStyled := a.styles.Popup.Copy().
				Width(mainWidth - 4).
				Render(popup)
			popupStr = trimToWidth(popupStyled, mainWidth)
		}
	}

	// Toast: floating notification chip pasted over the conversation
	// (bottom-right) instead of occupying its own budget slot.
	toastStr := ""
	if raw := a.toast.View(); raw != "" {
		toastStr = trimToWidth(a.styles.Toast.Render(raw), mainWidth)
	}

	// Footer composes inside its VISIBLE column: wide mode renders it
	// inside the main column (mainWidth); narrow mode keeps it within the
	// strip left of the overlaid rail or the status cluster would hide
	// underneath.
	footerW := mainWidth
	if !a.IsWide() {
		footerW = a.width - 42
	}
	a.footer.SetWidth(footerW)
	footerStr := a.footer.Render()

	// --- Height budget (REQ-TUI-APP-2): assign every terminal row to ---
	// --- exactly one slot so the frame fills a.height and the input ---
	// --- box plus footer stay pinned at the bottom edge.               ---
	inputH := lipgloss.Height(inputLine)
	footerH := lipgloss.Height(footerStr)
	toolH := 0
	if toolStr != "" {
		toolH = lipgloss.Height(toolStr)
	}
	fixed := inputH + footerH + toolH
	chatH := a.height - fixed
	if chatH < 1 {
		chatH = 1
	}
	// The conversation slot is a real viewport: long content scrolls inside
	// its budget instead of pushing pinned regions around. When the viewport
	// was already at the bottom (or this is the first frame) it sticks to
	// the bottom as content grows; a scrolled-up position is preserved.
	wasAtBottom := a.vpContentHeight == 0 ||
		a.scrollVP.YOffset+a.scrollVP.Height >= a.vpContentHeight
	a.scrollVP.Width = mainWidth
	a.scrollVP.Height = chatH
	a.scrollVP.SetContent(mainStr)
	a.vpContentHeight = lipgloss.Height(mainStr)
	if wasAtBottom {
		a.scrollVP.GotoBottom()
	}
	mainStr = a.scrollVP.View()

	buildPanel := func() string {
		var mb strings.Builder
		mb.WriteString(mainStr)
		if toolH > 0 {
			mb.WriteString("\n")
			mb.WriteString(toolStr)
		}
		mb.WriteString("\n")
		mb.WriteString(inputLine)
		mb.WriteString("\n")
		mb.WriteString(footerStr)
		return mb.String()
	}

	// Floating overlay heights (computed after the budget: they never own
	// slots — they paste over the conversation above the prompt box).
	toastH, popupH := 0, 0
	if toastStr != "" {
		toastH = lipgloss.Height(toastStr)
	}
	if popupStr != "" {
		popupH = lipgloss.Height(popupStr)
	}

	frame := buildPanel()

	// Sidebar (opencode right panel) — wide>120 shows 42 inline, !wide overlays with backdrop RGBA(0,0,0,70)
	var titleSeq string
	if a.IsWide() {
		// Sidebar rail stretches to the FULL terminal height so it spans
		// top to bottom with its footer pinned at the bottom edge
		// (REQ-TUI-APP-2).
		sidebarStr := a.newSidebarViewFullHeight(a.height)
		titleSeq = "\x1b]0;" + a.Title() + "\x07"
		frame = lipgloss.JoinHorizontal(lipgloss.Top, frame, " ", sidebarStr)
	} else {
		// Narrow: sidebar overlays the rightmost 42 columns over an
		// RGBA(0,0,0,70) backdrop strip per REQ-TUI-APP-2.
		if a.route != "home" {
			frame = a.applySidebarOverlay(frame)
		}
		titleSeq = "\x1b]0;" + a.Title() + "\x07"
	}

	// Floating overlays are composited as the last pass so they hover over
	// the final frame: popup left-aligned above the prompt box, toast
	// stacked above it (right-aligned to the conversation column in wide
	// mode so it never covers the rail footer).
	if popupH > 0 || toastH > 0 {
		rows := strings.Split(frame, "\n")
		inputStart := len(rows) - footerH - inputH
		toastAlignW := a.width
		if a.IsWide() {
			toastAlignW = mainWidth
		}
		if popupH > 0 {
			rows = pasteBlockLeft(rows, strings.Split(popupStr, "\n"), inputStart, a.width)
		}
		if toastH > 0 {
			rows = pasteBlockRight(rows, strings.Split(toastStr, "\n"), inputStart-popupH, toastAlignW)
		}
		frame = strings.Join(rows, "\n")
	}

	return titleSeq + fitFrame(frame, a.height)
}

// newSidebarModel builds the sidebar model from live controller state (shared
// by wide inline layout and narrow overlay).
func (a *App) newSidebarModel() views.SidebarModel {
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
	if ws, ok := a.ctrl.GetKV("workspace"); ok && ws != "" {
		sb.SetWorkspace(ws)
	} else if wd, err := os.Getwd(); err == nil {
		// Workspace is always real: fall back to the process working
		// directory (home prefix shortened to ~). Never fabricate beyond it.
		sb.SetWorkspace(shortenHome(wd))
	}
	// Subagents: real background task state only. Section renders only when
	// a source is attached AND at least one task exists (nil→omit, PR3 rule).
	if snap, has := a.ctrl.SubagentSnapshot(); has {
		var tasks []views.SubTask
		for _, t := range snap.Running {
			tasks = append(tasks, views.SubTask{
				Title: t.Title, Running: true, At: t.StartedAt.Format("15:04"),
			})
		}
		// Most recent finished first, capped for display.
		for i := len(snap.Finished) - 1; i >= 0 && len(tasks) < maxSidebarSubTasks; i-- {
			f := snap.Finished[i]
			tasks = append(tasks, views.SubTask{
				Title: f.Title, Err: f.Failed, At: f.FinishedAt.Format("15:04"),
			})
		}
		done, failed := 0, 0
		for _, f := range snap.Finished {
			if f.Failed {
				failed++
			} else {
				done++
			}
		}
		if len(snap.Running)+len(snap.Finished) > 0 {
			sb.SetSubagents(len(snap.Running), done, failed, tasks)
		}
	}
	// MCP: real per-server connection states; empty → section omitted.
	if servers := a.ctrl.MCPServers(); len(servers) > 0 {
		rows := make([]views.MCPServerState, len(servers))
		for i, s := range servers {
			rows[i] = views.MCPServerState{Name: s.Name, Connected: s.Connected}
		}
		sb.SetMCPServers(rows)
	}
	return sb
}

// maxSidebarSubTasks caps visible subagent rows in the sidebar.
const maxSidebarSubTasks = 6

// shortenHome replaces the user home prefix with ~ for compact display.
// Paths outside home are returned unchanged.
func shortenHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+string(os.PathSeparator)) {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}

// newSidebarViewFullHeight builds the 42-col sidebar stretched to exactly
// height rows: sections on top, footer (workspace path above version line)
// pinned at the bottom. Shared by the wide inline layout and the narrow
// overlay (REQ-TUI-APP-2).
func (a *App) newSidebarViewFullHeight(height int) string {
	return a.newSidebarModel().ViewFullHeight(42, height)
}

// applySidebarOverlay composes body with the sidebar drawn over the rightmost
// 42 visible columns, keeping total visible width == a.width. The strip behind
// the sidebar uses the RGBA(0,0,0,70) backdrop per REQ-TUI-APP-2. The sidebar
// rail stretches to the body's exact line count so its footer stays pinned at
// the bottom; when the rail's own content is taller than the body, the body
// is padded so the rail (including its footer) is never truncated.
func (a *App) applySidebarOverlay(body string) string {
	const sidebarWidth = 42
	bodyLines := strings.Split(body, "\n")
	// The rail stretches to the FULL terminal height (not the body's line
	// count) so its footer stays pinned at the bottom edge even when the
	// base frame is short; the max() below pads the shorter side.
	overlay := trimToWidth(a.newSidebarViewFullHeight(a.height), sidebarWidth)
	baseMax := a.width - sidebarWidth
	backdropPad := lipgloss.NewStyle().Background(lipgloss.Color("rgba(0,0,0,70)"))
	overlayLines := strings.Split(overlay, "\n")

	n := len(bodyLines)
	if len(overlayLines) > n {
		n = len(overlayLines)
	}
	out := make([]string, n)
	for i := 0; i < n; i++ {
		left := ""
		if i < len(bodyLines) {
			left = trimToWidth(bodyLines[i], baseMax)
		}
		if gap := baseMax - lipgloss.Width(left); gap > 0 {
			left += strings.Repeat(" ", gap)
		}
		right := ""
		if i < len(overlayLines) {
			right = overlayLines[i]
		}
		if gap := sidebarWidth - lipgloss.Width(right); gap > 0 {
			right += backdropPad.Render(strings.Repeat(" ", gap))
		}
		out[i] = left + right
	}
	return strings.Join(out, "\n")
}

func (a *App) renderHome() string {
	// Autocomplete popup height is reserved BEFORE sizing so the centered
	// base shrinks instead of pushing the footer off the fixed frame.
	popupStr := ""
	if a.autocomplete.IsActive() {
		popup := a.autocomplete.View()
		if popup != "" {
			popupStr = lipgloss.PlaceHorizontal(a.width, lipgloss.Center, popup)
		}
	}

	// Sync home view state before render — toast inside centered column per REQ-TUI-APP-8.
	// The last terminal row is reserved for the home footer; the popup slot
	// (if any) is reserved above that.
	a.homeView.SetSize(a.width, a.height-1-lipgloss.Height(popupStr))
	a.homeView.SetStyles(a.styles)
	a.homeView.SetInput(a.input.Value())
	a.homeView.SetToast(a.toast.View())

	base := a.homeView.View()

	if popupStr != "" {
		base = base + "\n" + popupStr
	}

	// Home footer at bottom (empty plus plugin slot, muted NotAvailable when absent)
	homeFooterStr := a.homeFooter.Render()

	var b strings.Builder
	// Title sequence for home: kui
	titleSeq := "\x1b]0;" + a.Title() + "\x07"
	b.WriteString(titleSeq)
	b.WriteString(base)
	b.WriteString("\n")
	b.WriteString(homeFooterStr)

	return fitFrame(b.String(), a.height)
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
	// Space-between row: real working directory on the left (workspace KV
	// first, process cwd as honest fallback), full terminal width.
	if ws, ok := a.ctrl.GetKV("workspace"); ok && ws != "" {
		a.footer.SetDir(ws)
	} else if wd, err := os.Getwd(); err == nil {
		a.footer.SetDir(shortenHome(wd))
	}
	a.footer.SetWidth(a.width)
	a.homeFooter.SetWidth(a.width)

	// Prompt field colors: override bubbles' ANSI-black textarea defaults so
	// the cursor line and placeholder blend with the element fill instead of
	// painting dark boxes through it.
	if t := a.styles.Theme; t != nil {
		fieldBg := lipgloss.NewStyle().Background(lipgloss.Color(t.BackgroundElement))
		ph := lipgloss.NewStyle().Foreground(lipgloss.Color(t.TextMuted))
		a.input.SetFieldColors(fieldBg, ph)
	}
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
	// Region widths are explicit: in wide mode the chat/diff column equals
	// the main panel width so nothing is truncated after rendering.
	regionW := a.width
	if a.IsWide() {
		regionW = a.ContentWidth()
	}
	a.diff.SetWidth(regionW)
	a.chat.SetWidth(regionW)

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

// IsWide reports whether the terminal is wide (>120 cols) per REQ-TUI-APP-2.
func (a *App) IsWide() bool {
	return a.width > 120
}

// ContentWidth returns the main-column width: terminal width minus the
// inline rail (42) and one gutter column when wide; width-4 when narrow.
func (a *App) ContentWidth() int {
	if a.IsWide() {
		return a.width - 42 - 1
	}
	return a.width - 4
}

// Title returns terminal title: kui on home, kui | {title} on session per REQ-TUI-APP-8.
func (a *App) Title() string {
	if a.route == "home" {
		return "kui"
	}
	t := a.ctrl.ActiveProfile()
	if t == "" {
		t = "session"
	}
	return "kui | " + t
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

// fitFrame clamps or pads s to exactly height rows so alt-screen frames are
// stable frame to frame and never exceed the terminal height (overflow is
// clipped from the bottom — a degenerate case only on tiny terminals).
func fitFrame(s string, height int) string {
	if height < 1 {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// padToWidth pads s with trailing spaces to exactly w visible columns.
func padToWidth(s string, w int) string {
	if gap := w - lipgloss.Width(s); gap > 0 {
		return s + strings.Repeat(" ", gap)
	}
	return s
}

// pasteBlockLeft replaces the rows [endRow-len(lines), endRow) with lines,
// padding each to width w so frame row widths stay stable. endRow is
// exclusive; rows outside the frame are skipped.
func pasteBlockLeft(rows []string, lines []string, endRow, w int) []string {
	start := endRow - len(lines)
	for i, ln := range lines {
		row := start + i
		if row < 0 || row >= len(rows) {
			continue
		}
		rows[row] = padToWidth(trimToWidth(ln, w), w)
	}
	return rows
}

// pasteBlockRight right-aligns lines into rows [endRow-len(lines), endRow),
// preserving each covered row's left content up to the block column and
// re-padding the tail to w. endRow is exclusive.
func pasteBlockRight(rows []string, lines []string, endRow, w int) []string {
	for i, ln := range lines {
		row := endRow - len(lines) + i
		if row < 0 || row >= len(rows) {
			continue
		}
		lw := lipgloss.Width(ln)
		col := w - lw
		if col < 0 {
			col = 0
		}
		left := trimToWidth(rows[row], col)
		if gap := col - lipgloss.Width(left); gap > 0 {
			left += strings.Repeat(" ", gap)
		}
		rows[row] = padToWidth(left+ln, w)
	}
	return rows
}
