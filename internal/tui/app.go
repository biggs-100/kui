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

	"github.com/biggs-100/kui/internal/adapters/providers"
	"github.com/biggs-100/kui/internal/core"
	"github.com/biggs-100/kui/internal/credentials"
	"github.com/biggs-100/kui/internal/tui/keymap"
	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/biggs-100/kui/internal/tui/ui"
	"github.com/biggs-100/kui/internal/tui/views"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// App is the root Bubble Tea model: a pi-style single column — minimal
// header + transcript viewport + status container + bordered editor + 2-line
// dim footer (REQ-TUI-APP-1/2/6). It delegates profile switching and prompt
// submission to the Controller, and translates controller events into tea.Msg
// values for the Update cycle.
//
// App never runs UI work on the agent loop's goroutine — all UI updates flow
// through tea.Cmd (D3 channel+Cmd handoff).
type App struct {
	ctrl   *Controller
	chat   views.ChatModel
	tool   views.ToolModel
	footer views.FooterModel
	styles *theme.Styles

	width        int
	height       int
	input        InputModel
	autocomplete AutocompleteModel
	quitting     bool

	// scrollVP scrolls the transcript inside its budgeted slot so long
	// content never pushes the pinned editor/footer around.
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

	// Registry holds all command metadata and dispatches commands.
	registry *CommandRegistry

	// currentTheme tracks the active theme name for cycling.
	currentTheme string

	// keymap stack base→modal
	km *keymap.Keymap

	// lastEsc tracks the previous Esc press for the double-Esc interrupt.
	lastEsc time.Time

	// In-app mouse selection (upstream copy-on-select): drag highlights
	// cells over the last rendered frame; releasing copies the text.
	selActive bool
	selStartX int
	selStartY int
	selEndX   int
	selEndY   int
	lastRows  []string           // visible-text snapshot of the last painted frame
	copySink  func(string) error // injectable clipboard writer (tests)

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

	return &App{
		ctrl:         ctrl,
		styles:       styles,
		input:        NewInputModel("Ask kui...", ""),
		autocomplete: NewAutocompleteModel(),
		chat:         views.NewChatModel(styles),
		tool:         views.NewToolModel(styles),
		footer:       views.NewFooterModel(styles),
		registry:     NewCommandRegistry(),
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
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.rebuildViews()
		return a, nil

	case tea.KeyMsg:
		if a.selActive {
			// Any keypress dismisses a held selection (copied already on
			// release, like the upstream copy-on-select flow).
			a.selActive = false
		}
		return a.handleKey(msg)

	case tea.MouseMsg:
		return a.handleMouse(msg)

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

	case copyDoneMsg:
		if msg.err != nil {
			a.chat.SetStatus("copy failed: " + msg.err.Error())
		} else {
			a.selActive = false // selection served; drop the highlight
			a.chat.SetStatus("copied to clipboard")
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
		// Upstream behavior: Ctrl+C clears a NON-EMPTY input first (so a
		// copy-shortcut reflex never kills the session); it exits only when
		// the input is already empty.
		if strings.TrimSpace(a.input.Value()) != "" {
			a.input.SetValue("")
			a.autocomplete.Deactivate()
			a.chat.SetStatus("input cleared · ctrl+c again to exit")
			return a, nil
		}
		_ = a.ctrl.SaveSession()
		a.quitting = true
		return a, tea.Quit
	}

	// No bare-letter interceptions here on purpose: with the input focused,
	// every rune belongs to the prompt.

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
				if strings.HasPrefix(submitted, "!") {
					return a.submitShell(submitted)
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

	// --- Enter: submit or command (home vs session) ---
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
		// Shell mode: "!cmd" executes LOCALLY — it never reaches the LLM.
		if strings.HasPrefix(submitted, "!") {
			return a.submitShell(submitted)
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
	case "/copy":
		return a.handleCopyCommand()
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
	case "/copy":
		return a.handleCopyCommand()
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
	pl.SetStyles(a.styles)
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

// switchTheme loads a theme by name, updates all views, and reports the
// switch on the transient status line (toasts removed, REQ-TUI-DLG-5).
func (a *App) switchTheme(name string) tea.Cmd {
	t := theme.Load(name)
	a.styles = theme.NewStyles(t)
	a.currentTheme = name
	a.chat.SetStatus("theme: " + name)
	a.rebuildViews()
	return nil
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
	a.chat.SetStatus("")
}

// compactDoneMsg carries the outcome of a manual /compact call.
type compactDoneMsg struct{ err error }

// copyDoneMsg carries the outcome of a /copy clipboard write.
type copyDoneMsg struct{ err error }

// handleCopyCommand copies the LAST assistant answer to the system
// clipboard (upstream messages.copy parity).
func (a *App) handleCopyCommand() (tea.Model, tea.Cmd) {
	msgs := a.ctrl.Messages()
	last := ""
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == core.RoleAssistant || msgs[i].Role == "assistant" {
			last = msgs[i].Content
			break
		}
	}
	if strings.TrimSpace(last) == "" {
		a.chat.SetStatus("nothing to copy yet")
		return a, nil
	}
	text := last
	return a, func() tea.Msg { return copyDoneMsg{err: copyToClipboard(text)} }
}

// copyToClipboard writes text to the system clipboard via the platform tool.
func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/C", "clip")
	case "darwin":
		cmd = exec.Command("pbcopy")
	default:
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy")
		} else if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else {
			return fmt.Errorf("no clipboard tool found (install wl-copy or xclip)")
		}
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

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

// renderScrollbar draws the 1-column conversation scrollbar: an
// element-fill track with a border-colored thumb, positioned by scroll
// fraction (upstream scrollbox styling). Empty string when unusable.
func (a *App) renderScrollbar(height, offset, contentH int) string {
	if height < 1 || contentH <= 0 {
		return ""
	}
	trackBg := a.styles.Theme.BackgroundElement
	if trackBg == "" {
		return ""
	}
	track := lipgloss.NewStyle().Background(lipgloss.Color(trackBg)).Render(" ")
	thumb := track
	if fg := a.styles.Theme.Border; fg != "" {
		thumb = lipgloss.NewStyle().
			Foreground(lipgloss.Color(fg)).
			Background(lipgloss.Color(trackBg)).
			Render("▌")
	}

	maxOffset := contentH - height
	thumbLen := max(1, height*height/contentH)
	pos := 0
	if maxOffset > 0 {
		pos = int(float64(offset) / float64(maxOffset) * float64(height-thumbLen))
	}
	if pos < 0 {
		pos = 0
	}
	if pos > height-thumbLen {
		pos = height - thumbLen
	}

	rows := make([]string, height)
	for j := range rows {
		rows[j] = track
		if j >= pos && j < pos+thumbLen {
			rows[j] = thumb
		}
	}
	return strings.Join(rows, "\n")
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
// spinnerFrames animates the transient status spinner in Accent.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// View renders the pi-style single column: minimal header + transcript
// viewport + status container + bordered editor + 2-line dim footer
// (REQ-TUI-APP-1/2/6/8). There is no sidebar, canvas, route switch, or
// toast. Resize reflows without crash; narrow terminals only shrink
// transcript/editor.
func (a *App) View() string {
	if a.quitting {
		return ""
	}

	if a.width == 0 || a.height == 0 {
		return "loading..."
	}

	// Modal overlay path (REQ-TUI-DLG-1/3): selectors render as CENTERED
	// modal dialogs over the dimmed frame — never as fullscreen takeovers.
	// The paletteMode/listMode/loginMode flags still own the keys (plumbing
	// unchanged); only the rendering is a dialog box + backdrop.
	if overlay := a.activeOverlay(); overlay != "" {
		titleSeq := "\x1b]0;" + a.Title() + "\a"
		return titleSeq + overlay
	}

	// Rebuild views with current state
	a.rebuildViews()

	headerStr := a.headerLine()
	statusStr := a.statusBlock()
	editorStr := a.editorBox()
	a.footer.SetWidth(a.width)
	footerStr := a.footer.Render()

	transcript := a.transcript()

	// --- Height budget (REQ-TUI-APP-2): every terminal row belongs to ---
	// --- exactly one slot so the frame fills a.height and the editor  ---
	// --- plus footer stay pinned at the bottom edge.                   ---
	editorH := lipgloss.Height(editorStr)
	chatH := a.height - 1 - 2 - editorH - 2
	if chatH < 1 {
		chatH = 1
	}
	// The transcript slot is a real viewport: long content scrolls inside
	// its budget instead of pushing pinned regions around. When the viewport
	// was already at the bottom (or this is the first frame) it sticks to
	// the bottom as content grows; a scrolled-up position is preserved.
	wasAtBottom := a.vpContentHeight == 0 ||
		a.scrollVP.YOffset+a.scrollVP.Height >= a.vpContentHeight
	contentLines := lipgloss.Height(transcript)
	showScroll := contentLines > chatH && chatH > 1
	vpW := a.width
	if showScroll {
		vpW = a.width - 2
	}
	a.scrollVP.Width = vpW
	a.scrollVP.Height = chatH
	a.scrollVP.SetContent(transcript)
	a.vpContentHeight = contentLines
	if wasAtBottom {
		a.scrollVP.GotoBottom()
	}
	transcript = a.scrollVP.View()
	if showScroll {
		sb := a.renderScrollbar(chatH, a.scrollVP.YOffset, contentLines)
		if sb != "" {
			// Side-by-side join: the scrollbar is a COLUMN, never stacked.
			transcript = lipgloss.JoinHorizontal(lipgloss.Top, transcript, sb)
		}
	}

	frame := headerStr + "\n" + transcript + "\n" + statusStr + "\n" + editorStr + "\n" + footerStr

	// Autocomplete popup floats OVER the transcript just above the editor
	// instead of owning a budget slot.
	if a.autocomplete.IsActive() {
		if popup := a.autocomplete.View(); popup != "" {
			popupStyled := trimToWidth(a.styles.Popup.Copy().Width(a.width-4).Render(popup), a.width)
			popupH := lipgloss.Height(popupStyled)
			rows := strings.Split(frame, "\n")
			editorStart := len(rows) - 2 - editorH
			frame = strings.Join(pasteBlockLeft(rows, strings.Split(popupStyled, "\n"), editorStart, a.width), "\n")
			_ = popupH
		}
	}

	titleSeq := "\x1b]0;" + a.Title() + "\x07"
	return titleSeq + a.finalizeFrame(frame)
}

// activeOverlay returns the centered dialog frame for whichever modal mode
// is open (status/palette/provider/model/session/login), or "" on the base
// layer. All selectors share this one overlay path (REQ-TUI-DLG-1/3).
func (a *App) activeOverlay() string {
	switch {
	case a.statusMode && a.statusModel != nil:
		return a.statusModel.View()
	case a.paletteMode && a.commandPalette != nil:
		return a.commandPalette.View()
	case a.providerListMode && a.providerList != nil:
		return a.providerList.View()
	case a.modelListMode && a.modelList != nil:
		return a.modelList.View()
	case a.listMode && a.sessionList != nil:
		return a.sessionList.View()
	case a.loginMode:
		return a.loginOverlay()
	default:
		return ""
	}
}

// loginOverlay renders the API-key prompt as a CENTERED modal dialog estilo
// pi (REQ-TUI-DLG-3): prompt + full-width ─ separator + input over the dim
// backdrop. Typing still flows to a.input via handleKey (Enter saves, Esc
// cancels) — rendering only, no logic change.
func (a *App) loginOverlay() string {
	prompt := fmt.Sprintf("Enter API key for %s (Enter to save, Esc to cancel):", a.loginProvider)
	size := ui.NarrowSize(60, a.width)
	sep := ui.Rule(size - 2)
	if a.styles != nil && a.styles.Theme != nil && a.styles.Theme.BorderSubtle != "" {
		sep = lipgloss.NewStyle().Foreground(lipgloss.Color(a.styles.Theme.BorderSubtle)).Render(sep)
	}
	body := prompt + "\n" + sep + "\n" + strings.TrimRight(a.input.View(), " ")
	return ui.NewDialog(size, body).View(a.width, a.height)
}

// headerLine renders the minimal dim header: `kui | {session}` plus
// collapsed key hints (REQ-TUI-APP-1/2). Truncates, never fabricates.
func (a *App) headerLine() string {
	title := a.Title()
	hints := "^P palette · TAB profile · /help"
	line := title
	if lipgloss.Width(line+"   "+hints) <= a.width {
		line += "   " + hints
	}
	if a.width > 0 && lipgloss.Width(line) > a.width {
		out := ""
		for _, r := range line {
			if lipgloss.Width(out+string(r)) > a.width {
				break
			}
			out += string(r)
		}
		line = out
	}
	return a.styles.HomeMuted.Render(line)
}

// statusBlock renders the transient status container above the editor:
// an Accent spinner + muted text while busy or when a transient message is
// set; IdleStatus reserves 2 empty lines so the layout never jumps
// (REQ-TUI-APP-6, REQ-TUI-DLG-5).
func (a *App) statusBlock() string {
	text := strings.TrimSpace(a.chat.Status())
	busy := a.ctrl.IsRunning()
	if !busy && text == "" {
		return "\n"
	}
	if text == "" {
		text = "thinking…"
	}
	frame := spinnerFrames[time.Now().UnixMilli()/80%int64(len(spinnerFrames))]
	spinner := lipgloss.NewStyle().Foreground(lipgloss.Color(a.styles.Theme.Accent)).Render(frame)
	first := spinner + " " + a.styles.HomeMuted.Render(text)
	return first + "\n"
}

// editorBox renders the bordered editor at a fixed height with a dynamic
// theme border: BorderActive (cyan) focused, BorderSubtle blurred while a
// modal overlay owns the keys (REQ-TUI-CHAT-1). Width accounts for the
// rounded border (2 cols) so no line exceeds the terminal width
// (REQ-TUI-APP-2 narrow-fit; REQ-TUI-APP-10 golden width lock).
func (a *App) editorBox() string {
	inner := strings.TrimRight(a.input.View(), " ")
	border := a.styles.Theme.BorderActive
	if a.modalOverlayOpen() {
		border = a.styles.Theme.BorderSubtle
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(border)).
		Background(lipgloss.Color(a.styles.Theme.BackgroundElement)).
		Padding(0, 1).
		Width(max(0, a.width-2)).
		Render(inner)
}

// transcript joins chat + tool blocks for the viewport slot. Empty renders
// "" — the frame shows only editor + footer (REQ-TUI-CHAT-7).
func (a *App) transcript() string {
	chatStr := trimToWidth(a.chat.Render(), a.width)
	toolStr := trimToWidth(a.tool.Render(), a.width)
	if chatStr == "" {
		return toolStr
	}
	if toolStr == "" {
		return chatStr
	}
	return chatStr + "\n" + toolStr
}

// gitBranch returns the current git branch for the footer, or "" when it
// cannot be determined (omitted, never fabricated).
func gitBranch() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = wd
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" || branch == "HEAD" {
		return ""
	}
	return branch
}

// finalizeFrame applies the live selection highlight,
// caching each row's plain visible text for mouse-selection extraction.
func (a *App) finalizeFrame(frame string) string {
	painted := strings.Split(fitFrame(frame, a.height), "\n")
	a.lastRows = make([]string, len(painted))
	for i, r := range painted {
		a.lastRows[i] = stripVisibleANSI(r)
	}
	if a.selActive && !a.modalOverlayOpen() {
		painted = applySelectionHighlight(painted, a.selStartX, a.selStartY, a.selEndX, a.selEndY)
	}
	return strings.Join(painted, "\n")
}

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

// rebuildViews synchronizes the view models with the controller state:
// footer identity/stats from live data only (omit unknowns), tool collapse
// signals, transcript widths, and textarea field colors.
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

	// Footer L1: cwd + (branch) + session — all live, omitted when unknown.
	if ws, ok := a.ctrl.GetKV("workspace"); ok && ws != "" {
		a.footer.SetDir(ws)
	} else if wd, err := os.Getwd(); err == nil {
		a.footer.SetDir(shortenHome(wd))
	}
	a.footer.SetBranch(gitBranch())
	if sid := a.ctrl.SessionID(); sid != "" {
		a.footer.SetSession(sid)
	} else {
		a.footer.SetSession("")
	}

	// Footer L2: stats left, (provider) model + thinking right.
	a.footer.SetModel(a.ctrl.ModelName())
	if provider, ok := a.ctrl.SyncProvider(); ok {
		a.footer.SetProvider(provider)
	} else {
		a.footer.SetProvider("")
	}
	if a.ctrl.IsRunning() {
		a.footer.SetThinking("thinking")
	} else if th, ok := a.ctrl.GetKV("thinking"); ok && th != "" {
		a.footer.SetThinking(th)
	} else {
		a.footer.SetThinking("")
	}
	a.footer.SetTokens(a.ctrl.TotalTokens(), a.ctrl.ContextWindow())
	if cost := a.ctrl.Cost(); cost > 0 {
		a.footer.SetCost(cost)
	}
	a.footer.SetProfiles(profiles, active)
	a.footer.SetWidth(a.width)

	// Prompt field colors: override bubbles' ANSI-black textarea defaults so
	// the cursor line and placeholder blend with the element fill instead of
	// painting dark boxes through it.
	if t := a.styles.Theme; t != nil {
		fieldBg := lipgloss.NewStyle().Background(lipgloss.Color(t.BackgroundElement))
		ph := lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.TextMuted)).
			Background(lipgloss.Color(t.BackgroundElement))
		a.input.SetFieldColors(fieldBg, ph)
	}
	// KV signals for tool output
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
	// Transcript regions render at full terminal width (no sidebar).
	a.chat.SetWidth(a.width)
	a.tool.SetWidth(a.width)
}

func (a *App) chatView() *views.ChatModel {
	return &a.chat
}

// Title returns the terminal title: always `kui | {profile}` on the
// transcript layout (REQ-TUI-APP-8). There is no home route anymore.
func (a *App) Title() string {
	t := a.ctrl.ActiveProfile()
	if t == "" {
		t = "session"
	}
	return "kui | " + t
}

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
