package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/biggs-100/kui/internal/adapters/profile"
	"github.com/biggs-100/kui/internal/adapters/skills"
	"github.com/biggs-100/kui/internal/adapters/store"
	"github.com/biggs-100/kui/internal/adapters/tools"
	"github.com/biggs-100/kui/internal/agent"
	"github.com/biggs-100/kui/internal/core"
	"github.com/biggs-100/kui/internal/mcp"
)

// maxIterations bounds the provider calls per run so a misbehaving provider
// cannot loop forever (D7).
const maxIterations = 10

// Wiring holds the external dependencies needed to build the TUI runtime.
// Fields mirror the CLI's wiring in runPrompt but are injectable for
// testability (REQ-TUI-APP-1).
type Wiring struct {
	// ProfileRoot is the directory containing named profile subdirectories.
	ProfileRoot string
	// ProjectDir is the project root for layered profile resolution.
	ProjectDir string
	// ConfigRoot is the .kui config directory (KUI_HOME).
	ConfigRoot string
	// Client is a factory that creates the core.Provider. If it returns
	// an error, Run exits with that error before starting the TUI
	// (REQ-TUI-APP-1 startup validation).
	Client func() (core.Provider, error)
	// MaxIter is the loop iteration budget. Zero defaults to 10.
	MaxIter int
}

// SetModeler is the interface for providers that support model switching.
// The openai.Client satisfies this through its SetModel method. The
// controller uses it to apply the REQ-CLI-4 chain before each prompt.
type SetModeler interface {
	SetModel(model string)
}

// agentRunner wraps *agent.Agent to satisfy the Runner interface. The
// agent.Steering() method returns *agent.PendingMessageQueue (concrete),
// but Runner expects core.PendingQueue (interface). This adapter
// bridges the return type.
type agentRunner struct {
	agent *agent.Agent
}

func (r *agentRunner) Run(ctx context.Context, prompt string, history []core.Message) (string, []core.Message, error) {
	return r.agent.Run(ctx, prompt, history)
}

func (r *agentRunner) Steering() core.PendingQueue {
	return r.agent.Steering()
}

func (r *agentRunner) Provider() core.Provider {
	return r.agent.Provider()
}

// modelLoaderAdapter wraps *profile.Loader to satisfy agent.ModelLoader.
// profile.Loader.Resolve returns *profile.Profile, but agent.ModelLoader
// expects *agent.ResolvedProfile.
type modelLoaderAdapter struct {
	loader *profile.Loader
}

func (a *modelLoaderAdapter) Resolve(name string) (*agent.ResolvedProfile, error) {
	p, err := a.loader.Resolve(name)
	if err != nil {
		return nil, err
	}
	return &agent.ResolvedProfile{Model: p.Model}, nil
}

// Run builds the TUI runtime from the given wiring and starts the Bubble Tea
// program. It returns an error if startup validation fails (e.g. invalid
// provider configuration) — in that case no TUI renders (REQ-TUI-APP-1).
//
// Run blocks until the user quits the TUI or the context is cancelled.
func Run(ctx context.Context, w Wiring) error {
	if w.MaxIter == 0 {
		w.MaxIter = maxIterations
	}

	// Step 1: Validate provider (startup validation, REQ-TUI-APP-1).
	provider, err := w.Client()
	if err != nil {
		return err
	}

	// Step 2: Build adapters.
	st := store.New(w.ConfigRoot)
	loader := profile.NewLoader(filepath.Join(w.ConfigRoot, "profiles"), w.ProjectDir, w.ConfigRoot)

	full := core.NewRegistry()
	for _, tool := range tools.Default(w.ProjectDir, 0) {
		if err := full.Register(tool); err != nil {
			return fmt.Errorf("register tool: %w", err)
		}
	}

	// MCP integration (REQ-TOOLS-4): load config from global and project
	// paths, connect to enabled servers, and register discovered tools.
	// MCP failures are non-fatal — built-in tools always work.
	mcpConfig, err := mcp.LoadConfig(
		filepath.Join(w.ConfigRoot, "mcp.yaml"),
		filepath.Join(w.ProjectDir, ".kui", "mcp.yaml"),
	)
	if err == nil && mcpConfig != nil && len(mcpConfig.Servers) > 0 {
		mgr := mcp.NewMCPManager(mcpConfig)
		defer mgr.Shutdown()
		if err := mgr.ConnectAll(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "kui: mcp: %v\n", err)
		}
		for _, tool := range mgr.Tools() {
			_ = full.Register(tool)
		}
	}

	// Step 3: Build agent runtime.
	manager := agent.NewManager(loader, full)

	profileDir := ""
	var skillsURLs []string
	if activeName, err := st.Active(); err == nil && activeName != "" {
		profileDir = filepath.Join(w.ConfigRoot, "profiles", activeName)
		// REQ-RS-13: classify profile skills entries — URLs become remote
		// registries, directory names stay local (REQ-RS-14).
		if resolved, err := loader.Resolve(activeName); err == nil {
			_, skillsURLs = skills.ClassifySkillsPaths(resolved.Skills)
		}
	}
	skillsIndex, err := skills.NewIndex(w.ConfigRoot, w.ProjectDir, profileDir, skillsURLs...)
	if err != nil {
		return fmt.Errorf("build skills index: %w", err)
	}

	ag := agent.NewAgent(manager, skillsIndex, provider, w.MaxIter)

	// Step 4: Discover profiles for the controller.
	names, err := loader.Discover()
	if err != nil {
		return fmt.Errorf("discover profiles: %w", err)
	}
	if len(names) == 0 {
		names = []string{""}
	}

	// Step 5: Build the model resolver (REQ-CLI-4 chain).
	envModel := os.Getenv("OPENAI_MODEL")
	mla := &modelLoaderAdapter{loader: loader}
	resolver := func(profileName string) string {
		return agent.ResolveModel(st, mla, profileName, envModel)
	}

	// Step 6: Build controller and set up model setter.
	ar := &agentRunner{agent: ag}
	ctrl := NewController(names, ar, resolver)
	if sm, ok := provider.(SetModeler); ok {
		ctrl.SetModeler = sm
	}

	// Step 6b: Wire session store for persistence.
	sessionStore := store.NewSessionStore(w.ConfigRoot)
	ctrl.SetSessionStore(sessionStore)
	if active, err := st.Active(); err == nil && active != "" {
		ctrl.SetSessionID(store.GenerateSessionID(active))
	}

	// Step 7: Build the Bubble Tea app.
	app := NewApp(ctrl)

	// Step 8: Start the controller event pump goroutine. It reads from
	// the controller's Events channel and sends events to the Bubble Tea
	// program via tea.Cmd (D3 channel+Cmd handoff, REQ-TUI-APP-3).
	pgm := tea.NewProgram(app)
	go pumpEvents(ctrl, pgm)

	// Step 9: Seed the active profile if one is saved (D18).
	if activeName, err := st.Active(); err == nil && activeName != "" {
		// Apply the switch up front so the tool subset, permissions,
		// and active profile are in place for the first request.
		if _, err := ag.Manager().ApplySwitch(ctx, activeName); err != nil {
			return fmt.Errorf("activate profile %q: %w", activeName, err)
		}
		ag.Steering().Enqueue(core.PendingMessage{SwitchProfile: activeName})
		if sys := ag.SystemMessages(); len(sys) > 0 {
			ag.Steering().Enqueue(core.PendingMessage{Content: sys[0].Content})
		}
	}

	// Step 10: Run the Bubble Tea program (blocks until quit).
	_, err = pgm.Run()
	return err
}

// pumpEvents reads events from the controller and sends them to the Bubble
// Tea program. It runs until the controller's events channel is closed.
func pumpEvents(ctrl *Controller, pgm *tea.Program) {
	for ev := range ctrl.Events() {
		pgm.Send(ev)
	}
}
