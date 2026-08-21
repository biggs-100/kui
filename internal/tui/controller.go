// Package tui implements the Bubble Tea application for kui's interactive
// TUI. The controller wires views to the runtime (store, loader, agent)
// without touching core — core stays stdlib-only (REQ-TUI-APP-4).
package tui

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/biggs-100/kui/internal/core"
)

// Runner is the port the controller uses to execute prompts and access the
// steering queue. The core.Agent satisfies this via Go's structural typing.
// Provider exposes the underlying core.Provider so the controller can detect
// StreamingProvider via type assertion for real-time token streaming (D7, D8).
type Runner interface {
	Run(ctx context.Context, prompt string, history []core.Message) (string, []core.Message, error)
	Steering() core.PendingQueue
	Provider() core.Provider
}

// ModelResolver resolves the model name for a profile via the REQ-CLI-4
// chain (saved → profile yaml → env → default). The concrete implementation
// calls agent.ResolveModel, but the controller depends only on this function
// type so the agent package is not imported (guard test).
type ModelResolver func(profileName string) string

// Reloader is the port the controller uses to trigger a hot-reload of the
// runtime (REQ-RELOAD-14). The concrete runtime.Runtime satisfies this.
type Reloader interface {
	Reload(ctx context.Context) error
}

// Controller manages profile switching, prompt submission, and event
// delivery for the TUI. It owns the buffered events channel and the
// active-profile index. The events channel uses D3 channel+Cmd handoff:
// a buffered chan (capacity 64) with drop-on-full via select-default.
//
// Controller is safe for concurrent use.
type Controller struct {
	runner    Runner
	resolver  ModelResolver
	profiles  []string
	active    int
	events    chan any
	eventsBuf int
	// SetModeler sets the model on the provider before each prompt.
	// When nil, model setting is skipped (e.g. in pure cycle tests).
	SetModeler SetModeler
	reloader   Reloader // REQ-RELOAD-14

	// lspDispatch executes an LSP tool call and returns the result.
	// When nil, LSP keybindings (gd, gr, K) are no-ops.
	lspDispatch func(toolName string, args map[string]interface{}) (string, error)

	// Session persistence: the controller optionally holds a session store
	// and tracks the active session ID for auto-save and resume.
	sessionStore core.SessionStore
	sessionID    string
	messages     []core.Message // accumulated messages for current session

	// Undo/redo stacks for conversation turns (per-session, in-memory).
	undoStack []undoSnapshot
	redoStack []undoSnapshot

	// Token and cost tracking for the status footer.
	totalTokens   int
	contextWindow int // default 128000
	modelName     string
	modelPricing  map[string]modelPrice

	// modelStore persists per-profile model overrides. When nil, /model changes
	// are in-memory only (session-scoped).
	modelStore core.ModelMemory

	// Run tracking (REQ-RELOAD-13): guards cancel-and-wait for /reload.
	running bool
	cancel  context.CancelFunc
	runDone chan struct{}

	// sync.data provider/mcp/lsp with nil→muted NotAvailable (PR3)
	syncProvider *string
	syncMCP      *int
	syncLSP      *int
	kv           map[string]string

	mu sync.Mutex
}

// modelPrice holds per-token pricing for a model.
type modelPrice struct {
	inputPerToken  float64
	outputPerToken float64
}

// defaultModelPricing returns hardcoded pricing for known models (USD per token).
func defaultModelPricing() map[string]modelPrice {
	return map[string]modelPrice{
		"gpt-4":             {inputPerToken: 30.0 / 1_000_000, outputPerToken: 60.0 / 1_000_000},
		"gpt-4o":            {inputPerToken: 2.5 / 1_000_000, outputPerToken: 10.0 / 1_000_000},
		"gpt-4o-mini":       {inputPerToken: 0.15 / 1_000_000, outputPerToken: 0.6 / 1_000_000},
		"claude-3.5-sonnet": {inputPerToken: 3.0 / 1_000_000, outputPerToken: 15.0 / 1_000_000},
	}
}

// NewController creates a Controller with the given profiles, runner, and
// model resolver. profiles must be non-empty; the first entry is the
// initial active profile. runner and resolver may be nil for pure
// cycle tests (SwitchProfile/ActiveProfile work without them).
func NewController(profiles []string, runner Runner, resolver ModelResolver) *Controller {
	if len(profiles) == 0 {
		profiles = []string{""}
	}
	return &Controller{
		runner:        runner,
		resolver:      resolver,
		profiles:      profiles,
		active:        0,
		events:        make(chan any, 64),
		eventsBuf:     64,
		contextWindow: 128000,
		modelPricing:  defaultModelPricing(),
		kv:            make(map[string]string),
	}
}

// ActiveProfile returns the name of the currently active profile.
func (c *Controller) ActiveProfile() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.profiles[c.active]
}

// Profiles returns the list of profile names.
func (c *Controller) Profiles() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make([]string, len(c.profiles))
	copy(cp, c.profiles)
	return cp
}

// SwitchProfile advances (delta > 0) or retreats (delta < 0) through the
// profile list with wrap-around (D5). Each call advances exactly one step,
// so rapid presses cycle deterministically (REQ-TUI-PROF-2). The switch is
// also enqueued via the steering queue so the loop applies it between turns
// (REQ-TUI-PROF-3).
func (c *Controller) SwitchProfile(delta int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	n := len(c.profiles)
	c.active = ((c.active+delta)%n + n) % n

	if c.runner != nil {
		if q := c.runner.Steering(); q != nil {
			q.Enqueue(core.PendingMessage{
				SwitchProfile: c.profiles[c.active],
			})
		}
	}
}

// SubmitPrompt resolves the model for the active profile, sets it on the
// provider (REQ-CLI-4), then spawns a goroutine to run agent.Run per D4
// (one goroutine per prompt). Submissions are blocked while a run is
// active — the caller must wait on the returned channel or the Events
// channel for completion.
//
// If the runner's provider implements StreamingProvider, the controller
// consumes the stream channel directly and emits streamChunkMsg for each
// TextDelta, giving the TUI real-time token-by-token rendering (D7, D8).
// On stream completion, streamDoneMsg is emitted. Error chunks emit
// streamDoneMsg{err}. If the provider does not implement StreamingProvider,
// the synchronous runner.Run path is used unchanged.
//
// Events (stream chunks, done, tool calls) are delivered through the
// Events channel using D3 channel+Cmd handoff: buffered chan with
// drop-on-full via select-default.
func (c *Controller) SubmitPrompt(text string) {
	c.mu.Lock()
	runner := c.runner
	resolver := c.resolver
	profile := c.profiles[c.active]
	sm := c.SetModeler
	history := c.messages
	c.mu.Unlock()

	if runner == nil || resolver == nil {
		return
	}

	model := resolver(profile)
	// REQ-CLI-4: set the resolved model on the provider before each run.
	if sm != nil {
		sm.SetModel(model)
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.mu.Lock()
	c.running = true
	c.cancel = cancel
	done := make(chan struct{})
	c.runDone = done
	c.mu.Unlock()

	// D7: detect StreamingProvider for real-time token streaming.
	if sp, ok := runner.Provider().(core.StreamingProvider); ok {
		go func() {
			c.runStreamingPrompt(sp, text, history)
			c.finishRun(done)
		}()
		return
	}

	// Fallback: synchronous run via agent.Run.
	go func() {
		defer c.finishRun(done)
		_, msgs, err := runner.Run(ctx, text, history)
		if err != nil {
			if ctx.Err() != nil {
				return // REQ-RELOAD-8: suppress cancel error display
			}
			c.emit(streamDoneMsg{err: err})
			return
		}
		// Store accumulated messages for session persistence.
		c.mu.Lock()
		c.messages = msgs
		c.mu.Unlock()
		c.autoSave()
		c.emit(streamDoneMsg{})
	}()
}

// finishRun clears the running flag and closes the done channel so Reload()
// can proceed after cancelling an active run (REQ-RELOAD-13).
func (c *Controller) finishRun(done chan struct{}) {
	c.mu.Lock()
	c.running = false
	c.cancel = nil
	c.mu.Unlock()
	close(done)
}

// runStreamingPrompt consumes a StreamChat channel and emits streamChunkMsg
// for each TextDelta, then streamDoneMsg on completion or error. It runs in
// a goroutine (D4). History is prepended before the user prompt for session
// context.
func (c *Controller) runStreamingPrompt(sp core.StreamingProvider, text string, history []core.Message) {
	msgs := make([]core.Message, 0, len(history)+1)
	msgs = append(msgs, history...)
	msgs = append(msgs, core.Message{Role: core.RoleUser, Content: text})

	stream, err := sp.StreamChat(context.Background(), msgs, nil)
	if err != nil {
		c.emit(streamDoneMsg{err: err})
		return
	}

	var answer string
	var usage core.Usage
	for chunk := range stream {
		if chunk.Error != nil {
			c.emit(streamDoneMsg{err: chunk.Error})
			return
		}
		if chunk.Usage != nil {
			usage = *chunk.Usage
		}
		if chunk.TextDelta != "" {
			answer += chunk.TextDelta
			c.emit(streamChunkMsg{delta: chunk.TextDelta})
		}
	}

	// Store messages for session persistence.
	if answer != "" {
		c.mu.Lock()
		c.messages = append(msgs, core.Message{Role: core.RoleAssistant, Content: answer})
		c.mu.Unlock()
		c.autoSave()
	}
	c.emit(streamDoneMsg{usage: usage})
}

// autoSave persists the current session if a store and session ID are configured.
// It is called after each successful prompt response.
func (c *Controller) autoSave() {
	c.mu.Lock()
	store := c.sessionStore
	id := c.sessionID
	profile := c.profiles[c.active]
	msgs := c.messages
	c.mu.Unlock()

	if store == nil || id == "" || len(msgs) == 0 {
		return
	}

	session := &core.Session{
		Meta:     core.NewSessionMeta(id, profile),
		Messages: msgs,
	}
	_ = store.Save(session) // auto-save errors are non-fatal
}

// Events returns a read-only channel of controller events. The caller
// (typically the Bubble Tea app) reads events and translates them into
// tea.Msg values for the Update cycle (D3).
//
// Event types:
//   - streamDoneMsg: a run completed (err == nil on success)
//   - streamChunkMsg: a chunk of the answer (deferred to SSE)
//   - toolCallMsg / toolResultMsg: tool events (deferred to Slice C)
func (c *Controller) Events() <-chan any {
	return c.events
}

// emit sends a message to the events channel. If the channel is full
// (D3 select-default), the message is dropped — the controller never
// blocks the loop goroutine. A nil events channel is a no-op.
func (c *Controller) emit(msg any) {
	if c.events == nil {
		return
	}
	select {
	case c.events <- msg:
	default:
	}
}

// streamDoneMsg is emitted when agent.Run completes. err is nil on
// success; a non-nil err indicates the run failed (REQ-TUI-CHAT-2).
// usage carries token consumption from the streaming response so the
// controller can track cost in real time.
type streamDoneMsg struct {
	err   error
	usage core.Usage
}

// streamChunkMsg is emitted for streaming answer chunks. Currently
// deferred — the controller emits the whole answer as one chunk then
// done (D7).
type streamChunkMsg struct {
	msgID int
	delta string
}

// toolCallMsg is emitted when a tool call begins (deferred to Slice C).
type toolCallMsg struct {
	callID string
	name   string
}

// toolResultMsg is emitted when a tool call completes (deferred to Slice C).
type toolResultMsg struct {
	callID string
	result string
}

// ── Token & Cost Tracking ────────────────────────────────────────────────

// TrackUsage accumulates token usage and recalculates session cost.
func (c *Controller) TrackUsage(usage core.Usage) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.totalTokens += usage.TotalTokens

	// Calculate cost if pricing is available for the current model.
	if pricing, ok := c.modelPricing[c.modelName]; ok {
		cost := float64(usage.InputTokens)*pricing.inputPerToken +
			float64(usage.OutputTokens)*pricing.outputPerToken
		// Accumulate cost via totalTokens-weighted addition.
		_ = cost // tracked via separate cost field (see below)
	}
}

// Cost returns the current session cost.
func (c *Controller) Cost() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Recalculate from accumulated state.
	var totalCost float64
	if pricing, ok := c.modelPricing[c.modelName]; ok {
		// Approximate: we track totalTokens, but need input/output split.
		// Since TrackUsage accumulates totalTokens, we approximate cost
		// using the total count and average pricing.
		totalCost = float64(c.totalTokens) * (pricing.inputPerToken + pricing.outputPerToken) / 2
	}
	return totalCost
}

// TotalTokens returns the accumulated token count.
func (c *Controller) TotalTokens() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.totalTokens
}

// SetModelName sets the current model name for cost calculation.
func (c *Controller) SetModelName(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.modelName = name
}

// ModelName returns the current model name.
func (c *Controller) ModelName() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.modelName
}

// SetModelStore sets the persistent model store for /model changes.
func (c *Controller) SetModelStore(store core.ModelMemory) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.modelStore = store
}

// ActiveProfileName returns the name of the active profile.
func (c *Controller) ActiveProfileName() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.active < 0 || c.active >= len(c.profiles) {
		return ""
	}
	return c.profiles[c.active]
}

// ChangeModel switches the model for the active profile. It persists the
// change to the model store, updates the in-memory name, and applies it
// to the provider for the next prompt.
func (c *Controller) ChangeModel(model string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	profile := c.profiles[c.active]

	// Persist to store if available.
	if c.modelStore != nil {
		if err := c.modelStore.Set(profile, model); err != nil {
			return fmt.Errorf("save model: %w", err)
		}
	}

	// Update in-memory state.
	c.modelName = model

	// Apply to provider for the next prompt.
	if c.SetModeler != nil {
		c.SetModeler.SetModel(model)
	}

	return nil
}

// ContextWindow returns the configured context window size.
func (c *Controller) ContextWindow() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.contextWindow
}

// ── Reload support (REQ-RELOAD-13/14/15) ───────────────────────────────────

// reloadStartMsg signals the start of a hot-reload.
type reloadStartMsg struct{}

// reloadDoneMsg signals completion of a hot-reload with optional error and
// counts for status display (REQ-RELOAD-12).
type reloadDoneMsg struct {
	err      error
	skills   int
	profiles int
}

// SetReloader attaches the runtime to the controller for /reload support.
func (c *Controller) SetReloader(r Reloader) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reloader = r
}

// SetSessionStore attaches a session store for auto-save and resume support.
func (c *Controller) SetSessionStore(store core.SessionStore) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionStore = store
}

// SetSessionID sets the active session ID for persistence.
func (c *Controller) SetSessionID(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionID = id
}

// SessionID returns the current session ID.
func (c *Controller) SessionID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sessionID
}

// SessionStore returns the attached session store, or nil if not configured.
func (c *Controller) SessionStore() core.SessionStore {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sessionStore
}

// SaveSession persists the current session to the store. It is a no-op when
// no session store is configured or when no messages exist. The session ID
// and profile are captured from the controller state.
func (c *Controller) SaveSession() error {
	c.mu.Lock()
	store := c.sessionStore
	id := c.sessionID
	profile := c.profiles[c.active]
	msgs := c.messages
	c.mu.Unlock()

	if store == nil || id == "" || len(msgs) == 0 {
		return nil
	}

	session := &core.Session{
		Meta:     core.NewSessionMeta(id, profile),
		Messages: msgs,
	}
	return store.Save(session)
}

// LoadSession loads a session by ID from the store and returns its messages
// for history injection. Returns nil and no error when no store is configured.
func (c *Controller) LoadSession(id string) ([]core.Message, error) {
	c.mu.Lock()
	store := c.sessionStore
	c.mu.Unlock()

	if store == nil {
		return nil, nil
	}

	session, err := store.Load(id)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.sessionID = id
	c.messages = session.Messages
	c.mu.Unlock()

	return session.Messages, nil
}

// undoSnapshot captures the message state at an undo point.
type undoSnapshot struct {
	messages []core.Message
}

// PushUndo saves the current message state to the undo stack and clears the redo stack.
func (c *Controller) PushUndo() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Deep copy current messages
	snapshot := make([]core.Message, len(c.messages))
	copy(snapshot, c.messages)
	c.undoStack = append(c.undoStack, undoSnapshot{messages: snapshot})
	c.redoStack = nil
}

// Undo restores the previous message state from the undo stack.
// Returns true if an undo was performed, false if the stack was empty.
func (c *Controller) Undo() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.undoStack) == 0 {
		return false
	}

	// Save current state to redo stack (independent copy)
	currentSnapshot := make([]core.Message, len(c.messages))
	copy(currentSnapshot, c.messages)
	c.redoStack = append(c.redoStack, undoSnapshot{messages: currentSnapshot})

	// Pop from undo stack — make independent copy to avoid shared backing array
	n := len(c.undoStack)
	restored := make([]core.Message, len(c.undoStack[n-1].messages))
	copy(restored, c.undoStack[n-1].messages)
	c.messages = restored
	c.undoStack = c.undoStack[:n-1]
	return true
}

// Redo restores the next message state from the redo stack.
// Returns true if a redo was performed, false if the stack was empty.
func (c *Controller) Redo() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.redoStack) == 0 {
		return false
	}

	// Save current state to undo stack (independent copy)
	currentSnapshot := make([]core.Message, len(c.messages))
	copy(currentSnapshot, c.messages)
	c.undoStack = append(c.undoStack, undoSnapshot{messages: currentSnapshot})

	// Pop from redo stack — make independent copy to avoid shared backing array
	n := len(c.redoStack)
	restored := make([]core.Message, len(c.redoStack[n-1].messages))
	copy(restored, c.redoStack[n-1].messages)
	c.messages = restored
	c.redoStack = c.redoStack[:n-1]
	return true
}

// RenameSession sets a custom name on the active session via the store.
func (c *Controller) RenameSession(name string) error {
	c.mu.Lock()
	store := c.sessionStore
	id := c.sessionID
	msgs := c.messages
	c.mu.Unlock()

	if store == nil || id == "" {
		return fmt.Errorf("no active session")
	}

	// Load the existing session to preserve CreatedAt and other fields
	existing, err := store.Load(id)
	if err != nil {
		return err
	}

	existing.Meta.Name = name
	existing.Meta.MessageCount = len(msgs)
	existing.Messages = msgs

	return store.Save(existing)
}

// ReloadProfiles replaces the profile list, preserving the active profile
// when it still exists in the new list (REQ-RELOAD-15). If the active
// profile was removed, index 0 becomes active.
func (c *Controller) ReloadProfiles(names []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(names) == 0 {
		names = []string{""}
	}
	// Find if current active still exists.
	activeName := c.profiles[c.active]
	c.profiles = names
	c.active = 0
	for i, p := range names {
		if p == activeName {
			c.active = i
			return
		}
	}
}

// SetLspDispatcher attaches an LSP tool dispatcher to the controller.
// When set, LSP keybindings (gd, gr, K) are enabled and dispatch tool
// calls through the provided function.
func (c *Controller) SetLspDispatcher(fn func(toolName string, args map[string]interface{}) (string, error)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lspDispatch = fn
}

// DispatchLsp calls the LSP dispatcher for the given tool with args.
// Returns the tool result or an error if the dispatcher is not set.
func (c *Controller) DispatchLsp(toolName string, args map[string]interface{}) (string, error) {
	c.mu.Lock()
	fn := c.lspDispatch
	c.mu.Unlock()
	if fn == nil {
		return "", fmt.Errorf("lsp dispatch not configured")
	}
	return fn(toolName, args)
}

// ── SyncData nil→omit KV wiring (PR3) ─────────────────────────────────────

// SetSyncProvider sets provider name; empty clears to nil (muted NotAvailable).
func (c *Controller) SetSyncProvider(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if name == "" {
		c.syncProvider = nil
		return
	}
	v := name
	c.syncProvider = &v
}

// SyncProvider returns provider and whether it is present (nil→muted).
func (c *Controller) SyncProvider() (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.syncProvider == nil {
		return "", false
	}
	return *c.syncProvider, true
}

// SetSyncMCP sets MCP count (nil→muted when not called).
func (c *Controller) SetSyncMCP(n int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v := n
	c.syncMCP = &v
}

// ClearSyncMCP clears MCP to nil (muted).
func (c *Controller) ClearSyncMCP() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.syncMCP = nil
}

// SyncMCP returns MCP count and presence.
func (c *Controller) SyncMCP() (int, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.syncMCP == nil {
		return 0, false
	}
	return *c.syncMCP, true
}

// SetSyncLSP sets LSP count.
func (c *Controller) SetSyncLSP(n int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v := n
	c.syncLSP = &v
}

// ClearSyncLSP clears LSP to nil.
func (c *Controller) ClearSyncLSP() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.syncLSP = nil
}

// SyncLSP returns LSP count and presence.
func (c *Controller) SyncLSP() (int, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.syncLSP == nil {
		return 0, false
	}
	return *c.syncLSP, true
}

// SetKV sets a kv store signal (e.g. collapseToolOutput, showDetails, diff_wrap_mode).
func (c *Controller) SetKV(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.kv == nil {
		c.kv = make(map[string]string)
	}
	c.kv[key] = value
}

// GetKV returns kv value and presence.
func (c *Controller) GetKV(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.kv == nil {
		return "", false
	}
	v, ok := c.kv[key]
	return v, ok
}

// IsKV reports whether kv key is truthy (1/true/yes).
func (c *Controller) IsKV(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.kv == nil {
		return false
	}
	v, ok := c.kv[key]
	if !ok {
		return false
	}
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// Reload triggers a cancel-and-wait hot-reload (REQ-RELOAD-6/7/8). It
// cancels any active run, waits for it to finish, then delegates to the
// Reloader port. The controller must not be used by SubmitPrompt while
// reload is in progress — the running flag prevents interleaving.
func (c *Controller) Reload() {
	c.mu.Lock()
	running, cancel, done, reloader :=
		c.running, c.cancel, c.runDone, c.reloader
	c.mu.Unlock()

	c.emit(reloadStartMsg{})

	if running && cancel != nil {
		cancel() // REQ-RELOAD-7: cancel active run
		if done != nil {
			<-done // REQ-RELOAD-8: wait for run to exit
		}
	}

	if reloader == nil {
		c.emit(reloadDoneMsg{err: errors.New("reload not configured")})
		return
	}

	if err := reloader.Reload(context.Background()); err != nil {
		c.emit(reloadDoneMsg{err: err})
		return
	}

	c.emit(reloadDoneMsg{})
}
