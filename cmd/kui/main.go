// Command kui runs the agent loop once for a prompt given as command-line
// arguments, and manages profiles through the profile subcommands
// (REQ-CLI-1, REQ-CLI-3). Exit codes follow D13: 0 success, 1 runtime
// failure, 2 usage error. The final answer goes to stdout; errors and usage
// go to stderr (REQ-CLI-2).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/biggs-100/kui/internal/adapters/extensions"
	"github.com/biggs-100/kui/internal/adapters/permissions"
	"github.com/biggs-100/kui/internal/adapters/profile"
	"github.com/biggs-100/kui/internal/adapters/providers"
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
  kui setup [--provider <name>]   configure API key for a provider
  kui tui [--resume <id>]         start the interactive TUI
  kui session list                list saved sessions
  kui session resume <id>         start TUI with a restored session
  kui profile list                 list profiles, marking the active one
  kui profile switch <name> [-- PROMPT...]
                                   activate <name> for the session; with --,
                                   also run a session that switches to it
                                   mid-run with the profile-context marker
  kui profile model <name> <model> set and persist a per-profile model

Flags:
  --provider, -p <provider>        select provider: openai (default), opencode
  --model, -m <model>              override the resolved model (highest priority)
  --tools <list>                   comma-separated tool names to include
  --exclude-tools <list>           comma-separated tool names to exclude
  --no-tools                       disable all tools
  --no-extensions, -ne             skip extension loading
  --no-skills, -ns                 skip skill index building
  --no-session                     (no-op, reserved for future use)
  --resume <id>                    restore session when starting TUI
  --verbose                        enable debug output to stderr
  --mode <text|json>               output format (default: text)
  --approve, -a                    bypass all permission checks
  --print                          write the answer to stdout regardless of mode
  --thinking <off|low|medium|high>  reasoning effort level (default: off)

Use "kui -- PROMPT..." to run a prompt that starts with "profile" or a dash.

Environment:
  OPENAI_API_KEY   required; API key for the OpenAI chat-completions endpoint
  OPENAI_BASE_URL  optional; defaults to https://api.openai.com/v1
  OPENAI_MODEL     optional; defaults to gpt-4o-mini
  OPENCODE_API_KEY required for --provider opencode; API key for OpenCode
  OPENCODE_BASE_URL optional; defaults to https://opencode.ai/zen/go/v1
  KUI_PROVIDER     optional; default provider when --provider not specified
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
  thinking <name> <level>
                       set and persist a per-profile thinking level
                       (off, low, medium, high)
`

const sessionUsage = `kui session SUBCOMMAND

session subcommands:
  list                 list saved sessions with metadata
  resume <id>          start TUI with a restored session
`

// skillsIndex is a type alias for the skills index returned by buildSkillsIndex.
// Using a type alias keeps the function variable testable without importing the
// concrete skills package in test files that only need the type.
type skillsIndex = skills.Index

// loadExtensions is a function variable wrapping extensions.LoadAll so tests
// can verify it is called or skipped based on --no-extensions (REQ-CLI-19).
var loadExtensions = func(api core.ExtensionAPI) error {
	return extensions.LoadAll(api)
}

// buildSkillsIndex is a function variable wrapping skills.NewIndex so tests
// can verify it is called or skipped based on --no-skills (REQ-CLI-20).
var buildSkillsIndex = func(globalDir, projectDir, profileDir string, registryURLs ...string) (*skillsIndex, error) {
	return skills.NewIndex(globalDir, projectDir, profileDir, registryURLs...)
}

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

	// The setup subcommand launches the credential setup wizard.
	if args[0] == "setup" {
		return runSetup(root, args[1:])
	}

	// The profile subcommand group handles its own usage.
	if args[0] == "profile" {
		return runProfile(root, args[1:])
	}

	// The session subcommand group manages session persistence.
	if args[0] == "session" {
		return runSession(root, args[1:])
	}

	// The plugin subcommand group manages plugin lifecycle.
	if args[0] == "plugin" {
		return runPlugin(root, args[1:])
	}

	// The tui subcommand starts the interactive TUI (REQ-CLI-5).
	if args[0] == "tui" {
		// Parse flags first to check for --mode json conflict.
		opts, _, err := parseFlags(args[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "kui: %v\n", err)
			return 2
		}
		// --mode json + tui is rejected (REQ-CLI-23).
		if opts.Mode == "json" {
			fmt.Fprintf(os.Stderr, "kui: --mode json is not supported with the tui subcommand\n")
			return 2
		}
		return runTUI(root, opts)
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

	// Validate --thinking flag early so invalid levels surface as usage errors.
	if opts.Thinking != "" {
		if _, err := resolveThinking(opts.Thinking); err != nil {
			fmt.Fprintf(os.Stderr, "kui: %v\n", err)
			return 2
		}
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
	case "thinking":
		return profileThinking(st, loader, args[1:])
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

// profileThinking sets and persists a per-profile thinking level in the
// profile.yaml file. The level is validated against {off, low, medium, high}
// before writing. An unknown profile is an actionable error naming it; missing
// arguments are a usage error.
func profileThinking(st *store.Store, loader *profile.Loader, args []string) int {
	if len(args) < 2 {
		fmt.Fprint(os.Stderr, profileUsage)
		return 2
	}
	name, level := args[0], args[1]

	// Validate the thinking level before doing any I/O.
	if _, err := resolveThinking(level); err != nil {
		fmt.Fprintf(os.Stderr, "kui: %v\n", err)
		return 2
	}

	if _, err := loader.Resolve(name); err != nil {
		fmt.Fprintf(os.Stderr, "kui: %v\n", err)
		return 1
	}

	profileDir := filepath.Join(configRoot(), "profiles", name)
	profilePath := filepath.Join(profileDir, "profile.yaml")

	// Read existing profile.yaml, update thinking field, write back.
	data, err := os.ReadFile(profilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kui: read profile: %v\n", err)
		return 1
	}

	content := string(data)
	// Check if thinking line already exists and replace it.
	lines := strings.Split(content, "\n")
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "thinking:") {
			lines[i] = "thinking: " + level
			found = true
			break
		}
	}
	if !found {
		// Append thinking field before any permissions block or at end.
		insertIdx := len(lines)
		for i, line := range lines {
			if strings.TrimSpace(line) == "permissions:" {
				insertIdx = i
				break
			}
		}
		// Insert thinking field.
		newLines := make([]string, 0, len(lines)+1)
		newLines = append(newLines, lines[:insertIdx]...)
		newLines = append(newLines, "thinking: "+level)
		newLines = append(newLines, lines[insertIdx:]...)
		lines = newLines
	}

	if err := os.WriteFile(profilePath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "kui: write profile: %v\n", err)
		return 1
	}
	return 0
}

// runSession dispatches the session management subcommands.
func runSession(root string, args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, sessionUsage)
		return 2
	}

	switch args[0] {
	case "list":
		return runSessionList()
	case "resume":
		if len(args) < 2 {
			fmt.Fprint(os.Stderr, sessionUsage)
			return 2
		}
		return runSessionResume(root, args[1])
	default:
		fmt.Fprintf(os.Stderr, "kui: unknown session subcommand %q\n", args[0])
		fmt.Fprint(os.Stderr, sessionUsage)
		return 2
	}
}

// runSessionList lists all saved sessions with metadata.
func runSessionList() int {
	sessionStore := store.NewSessionStore(configRoot())
	metas, err := sessionStore.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "kui: %v\n", err)
		return 1
	}

	if len(metas) == 0 {
		fmt.Fprintln(os.Stdout, "No sessions found.")
		return 0
	}

	fmt.Fprintf(os.Stdout, "%-40s %-12s %s\n", "ID", "PROFILE", "CREATED")
	fmt.Fprintf(os.Stdout, "%-40s %-12s %s\n", "----------------------------------------", "------------", "--------------------------")
	for _, m := range metas {
		fmt.Fprintf(os.Stdout, "%-40s %-12s %s\n", m.ID, m.Profile, m.CreatedAt)
	}
	return 0
}

// runSessionResume loads a session and starts the TUI with its history injected.
func runSessionResume(root, id string) int {
	sessionStore := store.NewSessionStore(configRoot())
	session, err := sessionStore.Load(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kui: %v\n", err)
		return 1
	}

	// Start TUI with the restored session history.
	return runTUIWithHistory(root, session)
}

// runTUIWithHistory starts the TUI with a pre-loaded session history.
func runTUIWithHistory(root string, session *core.Session) int {
	cfgRoot := configRoot()
	st := store.New(cfgRoot)
	loader := profile.NewLoader(filepath.Join(cfgRoot, "profiles"), root, cfgRoot)

	// Resolve provider: flag → profile → env → default "openai" (REQ-SEL-2).
	activeName, _ := st.Active()
	profileProvider := ""
	if resolved, err := loader.Resolve(activeName); err == nil {
		profileProvider = resolved.Provider
	}
	providerName := resolveProvider("", profileProvider)

	wiring := tui.Wiring{
		ProfileRoot: filepath.Join(cfgRoot, "profiles"),
		ProjectDir:  root,
		ConfigRoot:  cfgRoot,
		Client: func() (core.Provider, error) {
			return createProvider(providerName, root)
		},
		MaxIter: maxIterations,
	}

	if err := tui.RunWithHistory(context.Background(), wiring, session); err != nil {
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

	// --verbose: redirect log output to stderr for debug info (REQ-CLI-22).
	if opts.Verbose {
		log.SetOutput(os.Stderr)
		log.Println("kui: verbose mode enabled")
	}

	// --approve: warn about permission bypass (REQ-CLI-26).
	if opts.Approve {
		fmt.Fprintf(os.Stderr, "kui: WARNING: --approve bypasses all permission checks\n")
	}

	cfgRoot := configRoot()
	st := store.New(cfgRoot)
	loader := profile.NewLoader(filepath.Join(cfgRoot, "profiles"), root, cfgRoot)

	// Resolve provider: flag → profile → env → default "openai" (REQ-SEL-2).
	activeName, err := st.Active()
	if err != nil {
		fmt.Fprintf(os.Stderr, "kui: %v\n", err)
		return 1
	}
	profileProvider := ""
	if resolved, err := loader.Resolve(activeName); err == nil {
		profileProvider = resolved.Provider
	}
	providerName := resolveProvider(opts.Provider, profileProvider)

	client, err := createProvider(providerName, root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kui: %v\n", err)
		return 1
	}

	// Thinking degradation: warn if thinking is configured but provider doesn't support it (REQ-THINK-13).
	providers.WarnThinkingDegradation(providerName, client, resolveThinkingLevel(opts.Thinking, loader, activeName), os.Stderr)

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

	// Extension loading (REQ-CLI-19): skip when --no-extensions is set.
	// Extensions register tools and hooks via the ExtensionAPI; skipping
	// LoadAll means only built-in + MCP tools are available.
	if !opts.NoExtensions {
		hooks := core.NewHookRegistry()
		if err := loadExtensions(&extAPI{registry: full, hooks: hooks}); err != nil {
			fmt.Fprintf(os.Stderr, "kui: load extensions: %v\n", err)
			// Extension errors are non-fatal — built-in tools always work.
		}
	}

	// Tool filtering (REQ-CLI-14..18): apply --tools, --exclude-tools,
	// --no-tools after all tools are registered, before agent creation.
	full = filterTools(full, opts.Tools, opts.ExcludeTools, opts.NoTools)

	manager := agent.NewManager(loader, full)

	// --approve: bypass all permission checks (REQ-CLI-26). Set a permissive
	// ruleset that allows every tool call without prompting.
	if opts.Approve {
		manager.SetRuleset(permissions.NewPermissive())
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

	// Skill index (REQ-CLI-20): skip when --no-skills is set. The agent
	// receives a nil skills index — skills are not resolved, not injected
	// into the system prompt, and not available as tool sources.
	var skillsIdx *skillsIndex
	if !opts.NoSkills {
		skillsIdx, err = buildSkillsIndex(cfgRoot, root, profileDir, skillsURLs...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "kui: build skills index: %v\n", err)
			return 1
		}
	}

	ag := agent.NewAgent(manager, skillsIdx, client, maxIterations)

	// --verbose: attach cache stats observer for automatic cache hit reporting.
	if opts.Verbose {
		cacheObs := core.NewCacheStatsObserver(os.Stderr)
		ag.SetObserver(cacheObs)
	}

	// Resolve model for verbose logging and JSON output (REQ-CLI-4, REQ-CLI-11).
	resolvedModel := resolveWithOverride(opts.Model, st, loader, activeName)

	if activeName != "" {
		if sm, ok := client.(interface{ SetModel(string) }); ok {
			sm.SetModel(resolvedModel)
		}
		if opts.Verbose {
			log.Printf("kui: model=%s profile=%s provider=%s\n", resolvedModel, activeName, providerName)
		}

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
		if sm, ok := client.(interface{ SetModel(string) }); ok {
			sm.SetModel(resolveWithOverride(opts.Model, st, loader, ""))
		}
	}

	// Resolve thinking level: flag > profile > "off" (D1-D3).
	thinkingLevel := resolveThinkingLevel(opts.Thinking, loader, activeName)
	if st, ok := client.(interface{ SetThinking(string) }); ok {
		st.SetThinking(thinkingLevel)
	}
	if opts.Verbose {
		log.Printf("kui: thinking=%s\n", thinkingLevel)
	}

	answer, _, err := ag.Run(ctx, strings.Join(args, " "), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kui: %v\n", err)
		return 1
	}

	// --mode json: wrap answer in JSON envelope (REQ-CLI-23).
	if opts.Mode == "json" {
		result := map[string]string{
			"answer":  answer,
			"profile": activeName,
			"model":   resolvedModel,
		}
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "kui: encode json: %v\n", err)
			return 1
		}
		return 0
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

// resolveThinking validates a thinking level string against the allowed values
// {off, low, medium, high}. Empty input returns "off" (the default). Invalid
// input returns an actionable error listing the valid values.
func resolveThinking(level string) (string, error) {
	if level == "" {
		return "off", nil
	}
	switch level {
	case "off", "low", "medium", "high":
		return level, nil
	default:
		return "", fmt.Errorf("invalid thinking level %q: must be one of off, low, medium, high", level)
	}
}

// resolveThinkingLevel applies the layered resolution chain for thinking:
// --thinking flag (highest priority) → profile.yaml thinking → "off" (default).
func resolveThinkingLevel(flagLevel string, loader *profile.Loader, activeName string) string {
	if flagLevel != "" {
		return flagLevel
	}
	if resolved, err := loader.Resolve(activeName); err == nil && resolved.Thinking != "" {
		return resolved.Thinking
	}
	return "off"
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

// resolveProvider applies the layered resolution chain for provider selection:
// --provider flag (highest priority) → profile.yaml provider → KUI_PROVIDER
// env → default "openai" (REQ-SEL-2).
func resolveProvider(flagProvider, profileProvider string) string {
	return providers.ResolveProvider(flagProvider, profileProvider)
}

// createProvider uses the registry to construct a provider from the resolved
// name and environment variables (REQ-SEL-3 fail-fast API key validation).
func createProvider(name, root string) (core.Provider, error) {
	return providers.CreateProvider(providers.NewDefaultRegistry(), name, root)
}

// runTUI starts the interactive TUI (REQ-CLI-5). It validates the provider
// before starting — if startup fails, it prints an actionable error to
// stderr and exits non-zero without rendering the TUI (REQ-TUI-APP-1).
func runTUI(root string, opts Options) int {
	cfgRoot := configRoot()
	st := store.New(cfgRoot)
	loader := profile.NewLoader(filepath.Join(cfgRoot, "profiles"), root, cfgRoot)

	// Resolve provider: flag → profile → env → default "openai" (REQ-SEL-2).
	activeName, _ := st.Active()
	profileProvider := ""
	if resolved, err := loader.Resolve(activeName); err == nil {
		profileProvider = resolved.Provider
	}
	providerName := resolveProvider(opts.Provider, profileProvider)

	wiring := tui.Wiring{
		ProfileRoot: filepath.Join(cfgRoot, "profiles"),
		ProjectDir:  root,
		ConfigRoot:  cfgRoot,
		Client: func() (core.Provider, error) {
			return createProvider(providerName, root)
		},
		MaxIter: maxIterations,
	}

	// --resume: load session history before starting TUI.
	if opts.Resume != "" {
		sessionStore := store.NewSessionStore(cfgRoot)
		session, err := sessionStore.Load(opts.Resume)
		if err != nil {
			fmt.Fprintf(os.Stderr, "kui: resume session: %v\n", err)
			return 1
		}
		wiring.Session = session
	}

	if err := tui.Run(context.Background(), wiring); err != nil {
		fmt.Fprintf(os.Stderr, "kui: %v\n", err)
		return 1
	}
	return 0
}
