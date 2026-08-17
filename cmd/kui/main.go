// Command kui runs the agent loop once for a prompt given as command-line
// arguments, and manages profiles through the profile subcommands
// (REQ-CLI-1, REQ-CLI-3). Exit codes follow D13: 0 success, 1 runtime
// failure, 2 usage error. The final answer goes to stdout; errors and usage
// go to stderr (REQ-CLI-2).
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/biggs-100/kui/internal/adapters/profile"
	"github.com/biggs-100/kui/internal/adapters/providers/openai"
	"github.com/biggs-100/kui/internal/adapters/skills"
	"github.com/biggs-100/kui/internal/adapters/store"
	"github.com/biggs-100/kui/internal/adapters/tools"
	"github.com/biggs-100/kui/internal/agent"
	"github.com/biggs-100/kui/internal/core"

	// Blank import triggers init() self-registration of example extensions (D6).
	_ "github.com/biggs-100/kui/internal/extensions/example"

	"github.com/biggs-100/kui/internal/mcp"
	"github.com/biggs-100/kui/internal/tui"
)

// maxIterations bounds the provider calls per run so a misbehaving provider
// cannot loop forever (D7).
const maxIterations = 10

// defaultModel is the terminal fallback of the REQ-CLI-4 resolution chain when
// no saved model, no profile.yaml model, and no OPENAI_MODEL are present. It
// mirrors the provider's own default so the CLI never sends an empty model.
const defaultModel = "gpt-4o-mini"

const usage = `kui [--] PROMPT...

Runs the agent loop once and prints the final answer to stdout.

Subcommands:
  kui tui                          start the interactive TUI
  kui profile list                 list profiles, marking the active one
  kui profile switch <name> [-- PROMPT...]
                                   activate <name> for the session; with --,
                                   also run a session that switches to it
                                   mid-run with the profile-context marker
  kui profile model <name> <model> set and persist a per-profile model

Use "kui -- PROMPT..." to run a prompt that starts with "profile" or a dash.

Environment:
  OPENAI_API_KEY   required; API key for the chat-completions endpoint
  OPENAI_BASE_URL  optional; defaults to https://api.openai.com/v1
  OPENAI_MODEL     optional; defaults to gpt-4o-mini
  KUI_HOME         optional; config directory override (defaults to the
                   platform user config directory)
`

const profileUsage = `kui profile SUBCOMMAND

profile subcommands:
  list                 list profiles, marking the active one
  switch <name> [-- PROMPT...]
                       activate <name> for the session; with --, also run a
                       session that switches to it mid-run
  model <name> <model> set and persist a per-profile model
`

func main() {
	os.Exit(run(os.Args[1:]))
}

// run parses the command line and returns the process exit code (D13).
func run(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "kui: determine working directory: %v\n", err)
		return 1
	}

	// The profile subcommand group handles its own usage.
	if args[0] == "profile" {
		return runProfile(root, args[1:])
	}

	// The tui subcommand starts the interactive TUI (REQ-CLI-5).
	if args[0] == "tui" {
		return runTUI(root, Options{})
	}

	// Parse CLI flags into Options and remaining positional args (the prompt).
	opts, remaining, err := parseFlags(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kui: %v\n", err)
		return 2
	}
	if len(remaining) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}

	return runPrompt(root, opts, remaining)
}

// configRoot returns the directory holding the global kui configuration:
// KUI_HOME when set, else the platform user config directory (D18). The .kui
// state store, the global profile.yaml layer, the global skills, and the
// profiles/ tree all live under it.
func configRoot() string {
	if home := os.Getenv("KUI_HOME"); home != "" {
		return home
	}
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "kui")
	}
	return "."
}

// runProfile dispatches the profile management subcommands (REQ-CLI-3,
// REQ-PCLI-1..3).
func runProfile(root string, args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, profileUsage)
		return 2
	}

	cfgRoot := configRoot()
	st := store.New(cfgRoot)
	loader := profile.NewLoader(filepath.Join(cfgRoot, "profiles"), root, cfgRoot)

	switch args[0] {
	case "list":
		return profileList(st, loader)
	case "switch":
		return profileSwitch(st, loader, args[1:], root)
	case "model":
		return profileModel(st, loader, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "kui: unknown profile subcommand %q\n", args[0])
		fmt.Fprint(os.Stderr, profileUsage)
		return 2
	}
}

// profileList enumerates the resolved profiles, marking the active one with a
// leading asterisk. With no profiles it prints an empty list and exits zero
// (REQ-PCLI-1).
func profileList(st *store.Store, loader *profile.Loader) int {
	names, err := loader.Discover()
	if err != nil {
		fmt.Fprintf(os.Stderr, "kui: %v\n", err)
		return 1
	}
	active, err := st.Active()
	if err != nil {
		fmt.Fprintf(os.Stderr, "kui: %v\n", err)
		return 1
	}
	for _, name := range names {
		if name == active {
			_, _ = fmt.Fprintf(os.Stdout, "* %s\n", name)
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "  %s\n", name)
		}
	}
	return 0
}

// profileSwitch activates the named profile for the session (D18). Without a
// prompt it validates the profile and persists .kui/active (session-start
// activation, visible in `profile list`). With "-- PROMPT..." it additionally
// runs a session whose steering switch applies the profile mid-run with the
// profile-context marker, proving the loop wiring end-to-end (REQ-PCLI-2).
func profileSwitch(st *store.Store, loader *profile.Loader, args []string, root string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, profileUsage)
		return 2
	}
	name := args[0]

	// Validate the profile exists and is well-formed before persisting; an
	// unknown profile is an actionable error naming it (REQ-PCLI-2).
	if _, err := loader.Resolve(name); err != nil {
		fmt.Fprintf(os.Stderr, "kui: %v\n", err)
		return 1
	}
	if err := st.SetActive(name); err != nil {
		fmt.Fprintf(os.Stderr, "kui: %v\n", err)
		return 1
	}

	rest := args[1:]
	if len(rest) > 0 && rest[0] == "--" {
		rest = rest[1:]
		if len(rest) == 0 {
			fmt.Fprint(os.Stderr, profileUsage)
			return 2
		}
		return runPrompt(root, Options{}, rest)
	}
	return 0
}

// profileModel sets and persists a per-profile model in .kui/models.json
// (REQ-PCLI-3). An unknown profile is an actionable error naming it; missing
// arguments are a usage error (REQ-CLI-3).
func profileModel(st *store.Store, loader *profile.Loader, args []string) int {
	if len(args) < 2 {
		fmt.Fprint(os.Stderr, profileUsage)
		return 2
	}
	name, model := args[0], args[1]

	if _, err := loader.Resolve(name); err != nil {
		fmt.Fprintf(os.Stderr, "kui: %v\n", err)
		return 1
	}
	if err := st.Set(name, model); err != nil {
		fmt.Fprintf(os.Stderr, "kui: %v\n", err)
		return 1
	}
	return 0
}

// runPrompt runs one agent session (REQ-CLI-1). It wires the profile runtime
// at session start: the active profile (from .kui/active) is activated so its
// tool subset, permissions, and model apply from the first request; the
// REQ-CLI-4 resolution chain reconfigures the provider; and the profile's
// SYSTEM.md plus the skills index system messages are seeded through the
// steering queue (PR 4 note). The CLI keeps one agent and one history.
func runPrompt(root string, opts Options, args []string) int {
	ctx := context.Background()

	client, err := openai.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "kui: %v\n", err)
		return 1
	}

	cfgRoot := configRoot()
	st := store.New(cfgRoot)
	loader := profile.NewLoader(filepath.Join(cfgRoot, "profiles"), root, cfgRoot)

	full := core.NewRegistry()
	for _, tool := range tools.Default(root, 0) {
		if err := full.Register(tool); err != nil {
			fmt.Fprintf(os.Stderr, "kui: register tool: %v\n", err)
			return 1
		}
	}

	// MCP integration (REQ-TOOLS-4): load config from global and project
	// paths, connect to enabled servers, and register discovered tools.
	// MCP failures are non-fatal — built-in tools always work.
	mcpConfig, err := mcp.LoadConfig(
		filepath.Join(cfgRoot, "mcp.yaml"),
		filepath.Join(root, ".kui", "mcp.yaml"),
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

	// Tool filtering (REQ-CLI-14..18): apply --tools, --exclude-tools,
	// --no-tools after all tools are registered, before agent creation.
	full = filterTools(full, opts.Tools, opts.ExcludeTools, opts.NoTools)

	manager := agent.NewManager(loader, full)

	activeName, err := st.Active()
	if err != nil {
		fmt.Fprintf(os.Stderr, "kui: %v\n", err)
		return 1
	}

	profileDir := ""
	var skillsURLs []string
	if activeName != "" {
		profileDir = filepath.Join(cfgRoot, "profiles", activeName)
		// REQ-RS-13: classify profile skills entries — URLs become remote
		// registries, directory names stay local (REQ-RS-14).
		if resolved, err := loader.Resolve(activeName); err == nil {
			_, skillsURLs = skills.ClassifySkillsPaths(resolved.Skills)
		}
	}
	skillsIndex, err := skills.NewIndex(cfgRoot, root, profileDir, skillsURLs...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kui: build skills index: %v\n", err)
		return 1
	}

	ag := agent.NewAgent(manager, skillsIndex, client, maxIterations)

	if activeName != "" {
		// REQ-CLI-4: resolve the per-profile model chain and reconfigure the
		// provider before the session starts. The --model flag takes highest
		// priority via resolveWithOverride (REQ-CLI-11).
		client.SetModel(resolveWithOverride(opts.Model, st, loader, activeName))

		// D18 session-start activation: apply the switch up front so the tool
		// subset, permissions, and active profile are in place for the first
		// request...
		if _, err := manager.ApplySwitch(ctx, activeName); err != nil {
			fmt.Fprintf(os.Stderr, "kui: %v\n", err)
			return 1
		}
		// ...and seed the system context through the steering queue so the
		// loop drains the profile switch (appending the SYSTEM.md and the
		// profile-context marker) and the skills listing before the second
		// provider request (PR 4 note, D18).
		ag.Steering().Enqueue(core.PendingMessage{SwitchProfile: activeName})
		if sys := ag.SystemMessages(); len(sys) > 0 {
			ag.Steering().Enqueue(core.PendingMessage{Content: sys[0].Content})
		}
	}

	// When no profile is active, the --model flag still applies (REQ-CLI-11).
	if activeName == "" {
		client.SetModel(resolveWithOverride(opts.Model, st, loader, ""))
	}

	answer, err := ag.Run(ctx, strings.Join(args, " "))
	if err != nil {
		fmt.Fprintf(os.Stderr, "kui: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintln(os.Stdout, answer)
	return 0
}

// resolveWithOverride applies the REQ-CLI-4 resolution chain with the
// --model override as highest priority. When override is non-empty, it is
// returned immediately (REQ-CLI-11). Otherwise the standard chain applies:
// saved model → profile.yaml model → OPENAI_MODEL → default (REQ-CLI-4).
func resolveWithOverride(override string, st *store.Store, loader *profile.Loader, name string) string {
	if override != "" {
		return override
	}
	return resolveModel(st, loader, name)
}

// resolveModel applies the REQ-CLI-4 resolution chain: the profile's saved
// model (ModelMemory, .kui/models.json), then the layered profile.yaml model
// (profile → project → global, merged by the loader), then OPENAI_MODEL, then
// the built-in default.
func resolveModel(st *store.Store, loader *profile.Loader, name string) string {
	if model, ok := st.Get(name); ok {
		return model
	}
	if resolved, err := loader.Resolve(name); err == nil && resolved.Model != "" {
		return resolved.Model
	}
	if model := os.Getenv("OPENAI_MODEL"); model != "" {
		return model
	}
	return defaultModel
}

// runTUI starts the interactive TUI (REQ-CLI-5). It validates the provider
// before starting — if startup fails, it prints an actionable error to
// stderr and exits non-zero without rendering the TUI (REQ-TUI-APP-1).
func runTUI(root string, opts Options) int {
	cfgRoot := configRoot()
	wiring := tui.Wiring{
		ProfileRoot: filepath.Join(cfgRoot, "profiles"),
		ProjectDir:  root,
		ConfigRoot:  cfgRoot,
		Client: func() (core.Provider, error) {
			return openai.NewClient()
		},
		MaxIter: maxIterations,
	}

	if err := tui.Run(context.Background(), wiring); err != nil {
		fmt.Fprintf(os.Stderr, "kui: %v\n", err)
		return 1
	}
	return 0
}
