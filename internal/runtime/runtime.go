// Package runtime owns kui's runtime composition (D1). Build assembles a
// complete runtime snapshot — provider, profiles, skills, MCP, tool registry,
// hooks, and steering — from a Config; Reload re-reads configurable state and
// swaps only on a clean build; Close tears MCP and extensions down. It is the
// single composition path shared by runPrompt and tui.Run (REQ-RELOAD-2).
package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/biggs-100/kui/internal/adapters/extensions"
	"github.com/biggs-100/kui/internal/adapters/profile"
	"github.com/biggs-100/kui/internal/adapters/skills"
	"github.com/biggs-100/kui/internal/adapters/store"
	"github.com/biggs-100/kui/internal/adapters/tools"
	"github.com/biggs-100/kui/internal/agent"
	"github.com/biggs-100/kui/internal/core"
	"github.com/biggs-100/kui/internal/extensions/dynamic"
	"github.com/biggs-100/kui/internal/lsp"
	"github.com/biggs-100/kui/internal/mcp"
)

// defaultMaxIter bounds the provider calls per run so a misbehaving provider
// cannot loop forever. Zero Config.MaxIter selects it.
const defaultMaxIter = 10

// Config carries the external dependencies Build needs. Client is a factory
// that recreates the core.Provider (re-reading env) on every build, which is
// what lets Reload pick up provider configuration changes (D3).
type Config struct {
	// ProfileRoot is the directory containing named profile subdirectories.
	ProfileRoot string
	// ProjectDir is the project root for layered profile resolution.
	ProjectDir string
	// ConfigRoot is the .kui config directory (KUI_HOME).
	ConfigRoot string
	// Client creates the core.Provider. A returned error fails the build.
	Client func() (core.Provider, error)
	// MaxIter is the loop iteration budget. Zero defaults to 10.
	MaxIter int
	// ProviderName is the resolved provider name for logging and capability checks.
	ProviderName string
}

// Runtime is a fully assembled runtime snapshot (D1). Reload re-uses the same
// Runtime instance, swapping its components in place so the TUI can keep its
// controller/agent references stable across reloads.
type Runtime struct {
	Store    *store.Store
	Loader   *profile.Loader
	Agent    *agent.Agent
	Provider core.Provider
	Manager  *agent.Manager
	Skills   *skills.Index
	Full     *core.Registry
	MCP      *mcp.MCPManager
	Hooks    *core.HookRegistry
	Dynamic  *dynamic.Manager
	LSP      *lsp.LspManager
	Profiles []string

	cfg Config // retained so Reload can re-run the build from disk

	mu     sync.Mutex
	closed bool
}

// setModeler is the port for providers that support model switching
// (openai.Client satisfies it). Reload re-applies the REQ-CLI-4 chain to the
// recreated provider through it.
type setModeler interface {
	SetModel(model string)
}

// modelLoaderAdapter bridges profile.Loader.Resolve (*profile.Profile) onto
// agent.ModelLoader (*agent.ResolvedProfile) for the REQ-CLI-4 resolver chain.
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

// Build composes a complete runtime snapshot from cfg (REQ-RELOAD-1). It is
// the single composition path: provider → store/loader → profiles → skills →
// MCP → registry → extension LoadAll → agent → hooks → steering seed.
//
// A composition error returns nil runtime and the error (REQ-RELOAD-1): no
// partially-built runtime escapes.
func Build(ctx context.Context, cfg Config) (*Runtime, error) {
	if cfg.MaxIter == 0 {
		cfg.MaxIter = defaultMaxIter
	}

	// Step 1: provider (startup validation, REQ-TUI-APP-1).
	provider, err := cfg.Client()
	if err != nil {
		return nil, err
	}

	// Step 2: adapters.
	st := store.New(cfg.ConfigRoot)
	loader := profile.NewLoader(cfg.ProfileRoot, cfg.ProjectDir, cfg.ConfigRoot)

	// Step 3: discover profiles.
	names, err := loader.Discover()
	if err != nil {
		return nil, fmt.Errorf("discover profiles: %w", err)
	}
	if len(names) == 0 {
		names = []string{""}
	}

	// Step 4: skills index (uses the active profile's dir and skill URLs).
	skillsIndex, err := buildSkillsIndex(cfg, loader, st)
	if err != nil {
		return nil, err
	}

	// Step 5: registry (builtin tools + MCP tools) + hooks + dynamic extensions + extension LoadAll.
	// WARNING FIX #4: Wire LSP file syncer into file tools before building components.
	// Create LSP manager first so file tools can send DidOpen/DidChange notifications.
	lspMgr := lsp.NewLspManager()
	fileSyncer := lsp.NewLspFileSyncer(lspMgr)
	full, mgr, hooks, dynMgr, err := buildComponentsWithSyncer(ctx, cfg, fileSyncer)
	if err != nil {
		return nil, err
	}

	// Step 5b: register LSP tools in the full registry.
	for _, tool := range lsp.LspTools(lspMgr) {
		_ = full.Register(tool)
	}

	// Step 6: agent runtime.
	manager := agent.NewManager(loader, full)
	ag := agent.NewAgent(manager, skillsIndex, provider, cfg.MaxIter)
	ag.SetHooks(hooks)

	rt := &Runtime{
		Store:    st,
		Loader:   loader,
		Agent:    ag,
		Provider: provider,
		Manager:  manager,
		Skills:   skillsIndex,
		Full:     full,
		MCP:      mgr,
		Hooks:    hooks,
		Dynamic:  dynMgr,
		LSP:      lspMgr,
		Profiles: names,
		cfg:      cfg,
	}

	// Step 7: seed the active profile (D18 parity) — apply the switch up front
	// so the tool subset, permissions, and active profile are in place for the
	// first request, then queue the steering messages the loop drains before
	// the first provider turn.
	activeName, err := st.Active()
	if err != nil {
		_ = extensions.ShutdownAll()
		if dynMgr != nil {
			_ = dynMgr.ShutdownAll()
		}
		if mgr != nil {
			mgr.Shutdown()
		}
		return nil, err
	}
	if activeName != "" {
		if _, err := manager.ApplySwitch(ctx, activeName); err != nil {
			_ = extensions.ShutdownAll()
			if dynMgr != nil {
				_ = dynMgr.ShutdownAll()
			}
			if mgr != nil {
				mgr.Shutdown()
			}
			return nil, fmt.Errorf("activate profile %q: %w", activeName, err)
		}
		seedSteering(ag)
	}

	return rt, nil
}

// buildSkillsIndex rebuilds the skills index from disk for the currently
// active profile (REQ-RELOAD-3). It is shared by Build and Reload.
func buildSkillsIndex(cfg Config, loader *profile.Loader, st *store.Store) (*skills.Index, error) {
	activeName, err := st.Active()
	if err != nil {
		return nil, err
	}
	profileDir := ""
	var skillsURLs []string
	if activeName != "" {
		profileDir = filepath.Join(cfg.ConfigRoot, "profiles", activeName)
		// REQ-RS-13: classify profile skills entries — URLs become remote
		// registries, directory names stay local (REQ-RS-14).
		if resolved, err := loader.Resolve(activeName); err == nil {
			_, skillsURLs = skills.ClassifySkillsPaths(resolved.Skills)
		}
	}
	index, err := skills.NewIndex(cfg.ConfigRoot, cfg.ProjectDir, profileDir, skillsURLs...)
	if err != nil {
		return nil, fmt.Errorf("build skills index: %w", err)
	}
	return index, nil
}

// buildComponents builds a fresh full tool registry (builtin + MCP tools),
// connects the MCP manager (when configured), creates the hook registry,
// loads dynamic extensions from extensions.yaml, and runs extensions.LoadAll
// against the concrete ExtensionAPI so compiled-in extensions become active
// (REQ-RELOAD-16/17). A LoadAll error fails the build; any MCP manager
// created is shut down before returning.
func buildComponents(ctx context.Context, cfg Config) (*core.Registry, *mcp.MCPManager, *core.HookRegistry, *dynamic.Manager, error) {
	return buildComponentsWithSyncer(ctx, cfg, nil)
}

// buildComponentsWithSyncer builds the full tool registry with optional LSP file
// sync. When syncer is non-nil, read_file and write_file tools are created with
// DidOpen/DidChange notification support (WARNING FIX #4).
func buildComponentsWithSyncer(ctx context.Context, cfg Config, syncer tools.FileSyncer) (*core.Registry, *mcp.MCPManager, *core.HookRegistry, *dynamic.Manager, error) {
	full := core.NewRegistry()
	for _, tool := range tools.DefaultWithSyncer(cfg.ProjectDir, 0, syncer) {
		if err := full.Register(tool); err != nil {
			return nil, nil, nil, nil, fmt.Errorf("register tool: %w", err)
		}
	}

	// MCP integration (REQ-TOOLS-4): load config from global and project
	// paths, connect to enabled servers, and register discovered tools. MCP
	// failures are non-fatal — built-in tools always work.
	var mgr *mcp.MCPManager
	mcpConfig, err := mcp.LoadConfig(
		filepath.Join(cfg.ConfigRoot, "mcp.yaml"),
		filepath.Join(cfg.ProjectDir, ".kui", "mcp.yaml"),
	)
	if err == nil && mcpConfig != nil && len(mcpConfig.Servers) > 0 {
		mgr = mcp.NewMCPManager(mcpConfig)
		if err := mgr.ConnectAll(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "kui: mcp: %v\n", err)
		}
		for _, tool := range mgr.Tools() {
			_ = full.Register(tool)
		}
	}

	// Dynamic extension discovery: load extensions.yaml from global and
	// project config roots, create a manager, and let it discover + load
	// extensions from configured paths. Dynamic extensions register their
	// tools through the same ExtensionAPI used by compiled-in extensions.
	api := &extAPI{registry: full, hooks: core.NewHookRegistry()}
	dynMgr, err := loadDynamicExtensions(ctx, cfg, api)
	if err != nil {
		if mgr != nil {
			mgr.Shutdown()
		}
		return nil, nil, nil, nil, fmt.Errorf("load dynamic extensions: %w", err)
	}

	// Compiled-in extensions via init() self-registration (REQ-DISCOVERY-1).
	if err := extensions.LoadAll(api); err != nil {
		if mgr != nil {
			mgr.Shutdown()
		}
		if dynMgr != nil {
			_ = dynMgr.ShutdownAll()
		}
		return nil, nil, nil, nil, fmt.Errorf("load extensions: %w", err)
	}
	return full, mgr, api.hooks, dynMgr, nil
}

// ReloadResult carries the outcome of a Reload call (REQ-RELOAD-3).
type ReloadResult struct {
	// Err is non-nil when the rebuild failed and old state was kept.
	Err error
	// Profiles lists the profile names discovered after reload.
	Profiles []string
	// Skills is the number of skills in the re-scanned index.
	Skills int
}

// Reload re-reads all configurable state and swaps components in place. It
// follows the pi session.reload ordering (D3): teardown → rebuild → swap. If
// any step fails, the old components stay active (build-new-then-swap, D2).
func (r *Runtime) Reload(ctx context.Context) ReloadResult {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return ReloadResult{Err: errors.New("runtime closed")}
	}

	// Teardown (D3 steps 1-3).
	_ = extensions.ShutdownAll()
	if r.Dynamic != nil {
		_ = r.Dynamic.ShutdownAll()
	}
	if r.MCP != nil {
		r.MCP.Shutdown()
	}
	if r.LSP != nil {
		_ = r.LSP.StopAll() // WARNING FIX #5: stop old LSP servers before rebuilding
		r.LSP.Cache().ClearAll()
	}

	// Save old state for rollback.
	oldProvider := r.Provider
	oldActive := r.Manager.Active()

	// D3 step 4: recreate provider (re-reads env).
	provider, err := r.cfg.Client()
	if err != nil {
		r.Provider = oldProvider
		return ReloadResult{Err: fmt.Errorf("recreate provider: %w", err)}
	}

	// D3 step 5: re-discover profiles.
	names, err := r.Loader.Discover()
	if err != nil {
		r.Provider = oldProvider
		return ReloadResult{Err: fmt.Errorf("discover profiles: %w", err)}
	}
	if len(names) == 0 {
		names = []string{""}
	}

	// D3 step 6: re-scan skills.
	skillsIndex, err := buildSkillsIndex(r.cfg, r.Loader, r.Store)
	if err != nil {
		r.Provider = oldProvider
		return ReloadResult{Err: fmt.Errorf("rebuild skills: %w", err)}
	}

	// D3 step 7: rebuild full registry (builtin + MCP + extensions).
	full, mgr, hooks, dynMgr, err := buildComponents(ctx, r.cfg)
	if err != nil {
		r.Provider = oldProvider
		return ReloadResult{Err: fmt.Errorf("rebuild components: %w", err)}
	}

	// Step 7b: re-register LSP tools in the rebuilt registry (CRITICAL FIX #1).
	// buildComponents() creates a fresh registry without LSP tools — we must
	// re-register them so lsp_* tools remain available after Reload.
	for _, tool := range lsp.LspTools(r.LSP) {
		_ = full.Register(tool)
	}

	// Swap (D2): apply new state atomically.
	r.Provider = provider
	r.Profiles = names
	r.Skills = skillsIndex
	r.Full = full
	r.MCP = mgr
	r.Hooks = hooks
	r.Dynamic = dynMgr
	r.Agent.SetProvider(provider)
	r.Agent.SetSkills(skillsIndex)
	r.Agent.SetHooks(hooks)

	// D3 step 8: re-apply active profile via Manager.Reload.
	if oldActive != "" {
		_ = r.Manager.Reload(full)
	}

	// D3 step 9: re-seed steering.
	seedSteering(r.Agent)

	return ReloadResult{
		Profiles: r.Profiles,
		Skills:   len(r.Skills.List()),
	}
}

// Close tears down MCP, dynamic extensions, and compiled-in extensions
// (REQ-RELOAD-5). It is idempotent.
func (r *Runtime) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}
	r.closed = true

	if r.MCP != nil {
		r.MCP.Shutdown()
	}
	if r.LSP != nil {
		_ = r.LSP.StopAll()
	}
	if r.Dynamic != nil {
		_ = r.Dynamic.ShutdownAll()
	}
	return extensions.ShutdownAll()
}

// seedSteering queues the active-profile switch and the skills system message
// for the loop to drain before the next provider turn (D18, PR 4 note).
func seedSteering(ag *agent.Agent) {
	if active := ag.Manager().Active(); active == "" {
		return
	}
	ag.Steering().Enqueue(core.PendingMessage{SwitchProfile: ag.Manager().Active()})
	if sys := ag.SystemMessages(); len(sys) > 0 {
		ag.Steering().Enqueue(core.PendingMessage{Content: sys[0].Content})
	}
}

// loadDynamicExtensions reads extensions.yaml from global and project config
// roots, creates a dynamic.Manager, and calls LoadAll to discover and load
// extensions from configured paths. Returns the manager for lifecycle
// management (shutdown/reload). When no extensions.yaml exists or no paths
// are configured, returns a nil manager and no error.
func loadDynamicExtensions(ctx context.Context, cfg Config, api core.ExtensionAPI) (*dynamic.Manager, error) {
	globalPath := filepath.Join(cfg.ConfigRoot, "extensions.yaml")
	projectPath := filepath.Join(cfg.ProjectDir, ".kui", "extensions.yaml")

	dynCfg, err := dynamic.LoadConfigs(globalPath, projectPath)
	if err != nil {
		return nil, fmt.Errorf("load dynamic extensions config: %w", err)
	}

	// No paths configured — nothing to do.
	if len(dynCfg.Paths) == 0 {
		return nil, nil
	}

	mgr := dynamic.NewManager(dynCfg)
	if err := mgr.LoadAll(ctx, api); err != nil {
		return nil, fmt.Errorf("dynamic manager load: %w", err)
	}
	return mgr, nil
}
