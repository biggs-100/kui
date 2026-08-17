package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/biggs-100/kui/internal/adapters/extensions"
	"github.com/biggs-100/kui/internal/core"
)

// ── shared fakes ──────────────────────────────────────────────────────────

// fakeProvider implements core.Provider plus SetModel (the SetModeler port the
// runtime uses for the REQ-CLI-4 chain on reload).
type fakeProvider struct {
	name  string
	model string
}

func (p *fakeProvider) Chat(_ context.Context, _ []core.Message, _ []core.Tool) ([]core.Message, error) {
	return nil, nil
}

func (p *fakeProvider) SetModel(model string) { p.model = model }

// fakeTool implements core.Tool minimally.
type fakeTool struct {
	name string
}

func (t *fakeTool) Name() string        { return t.name }
func (t *fakeTool) Description() string { return "fake tool " + t.name }
func (t *fakeTool) Schema() string      { return `{"type":"object"}` }
func (t *fakeTool) Execute(_ context.Context, _ json.RawMessage) (string, error) {
	return "ok", nil
}

// countingExtension registers a unique tool through whatever api it receives
// and counts Init/Shutdown calls. It is registered once (init below) into the
// global extension registry so Build/Reload/Close can be observed through the
// production extensions package (REQ-RELOAD-17/18).
type countingExtension struct {
	mu        sync.Mutex
	inits     int
	shutdowns int
}

func (e *countingExtension) Name() string { return "kui-runtime-test-ext" }

func (e *countingExtension) Init(api core.ExtensionAPI) error {
	e.mu.Lock()
	e.inits++
	e.mu.Unlock()
	return api.RegisterTool(&fakeTool{name: testExtToolName})
}

func (e *countingExtension) Shutdown() error {
	e.mu.Lock()
	e.shutdowns++
	e.mu.Unlock()
	return nil
}

// testExt is the singleton test extension instance used for counter deltas.
var testExt = &countingExtension{}

func init() { extensions.Register(testExt) }

// testExtToolName is the tool the counting extension registers during Init.
const testExtToolName = "kui_test_ext_tool"

// ── fixture helpers ───────────────────────────────────────────────────────

// newTestConfig returns a Config rooted in fresh temp dirs with a fake
// provider. Tests add profiles/active state on top of cfg.ConfigRoot.
func newTestConfig(t *testing.T) Config {
	t.Helper()
	cfgRoot := t.TempDir()
	return Config{
		ProfileRoot: filepath.Join(cfgRoot, "profiles"),
		ProjectDir:  t.TempDir(),
		ConfigRoot:  cfgRoot,
		Client: func() (core.Provider, error) {
			return &fakeProvider{name: "fake"}, nil
		},
		MaxIter: 10,
	}
}

// writeProfile writes a named profile (profile.yaml + SYSTEM.md) under cfgRoot.
// tools is a comma-separated tool list ("" for none).
func writeProfile(t *testing.T, cfgRoot, name, tools string) {
	t.Helper()
	dir := filepath.Join(cfgRoot, "profiles", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	b.WriteString("name: " + name + "\n")
	b.WriteString("system_prompt: SYSTEM.md\n")
	if tools != "" {
		b.WriteString("tools:\n")
		for _, tool := range strings.Split(tools, ",") {
			b.WriteString("  - " + strings.TrimSpace(tool) + "\n")
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "profile.yaml"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SYSTEM.md"), []byte("You are a test agent.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// setActive persists the active profile under .kui/active.
func setActive(t *testing.T, cfgRoot, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(cfgRoot, ".kui"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgRoot, ".kui", "active"), []byte(name), 0o644); err != nil {
		t.Fatal(err)
	}
}

// ── Build tests (REQ-RELOAD-1) ────────────────────────────────────────────

func TestBuildComposesAllComponents(t *testing.T) {
	cfg := newTestConfig(t)
	writeProfile(t, cfg.ConfigRoot, "mycoder", "read_file")
	setActive(t, cfg.ConfigRoot, "mycoder")

	rt, err := Build(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	if rt.Provider == nil {
		t.Error("Build: provider is nil")
	}
	if rt.Store == nil {
		t.Error("Build: store is nil")
	}
	if rt.Loader == nil {
		t.Error("Build: loader is nil")
	}
	if rt.Manager == nil {
		t.Error("Build: manager is nil")
	}
	if rt.Agent == nil {
		t.Error("Build: agent is nil")
	}
	if rt.Skills == nil {
		t.Error("Build: skills index is nil")
	}
	if rt.Full == nil {
		t.Error("Build: full registry is nil")
	}
	if rt.Hooks == nil {
		t.Error("Build: hook registry is nil")
	}

	// Full registry carries the built-in tools plus the extension tool
	// registered through LoadAll (REQ-RELOAD-17).
	for _, name := range []string{"read_file", "write_file", "bash", testExtToolName} {
		if _, ok := rt.Full.Get(name); !ok {
			t.Errorf("full registry missing %q", name)
		}
	}

	// The active profile was applied: active is set and the registry is the
	// profile's declared subset (read_file only).
	if got := rt.Manager.Active(); got != "mycoder" {
		t.Errorf("Manager.Active() = %q, want %q", got, "mycoder")
	}
	if _, ok := rt.Manager.Registry().Get("read_file"); !ok {
		t.Error("active registry missing read_file")
	}
	if _, ok := rt.Manager.Registry().Get("write_file"); ok {
		t.Error("active registry should not contain write_file (profile subset declares only read_file)")
	}

	// Steering was seeded with the active profile switch.
	msgs := rt.Agent.Steering().Drain()
	if len(msgs) == 0 || msgs[0].SwitchProfile != "mycoder" {
		t.Errorf("steering seed = %+v, want leading SwitchProfile=mycoder", msgs)
	}
}

func TestBuildEmptyProfiles(t *testing.T) {
	cfg := newTestConfig(t)
	// No profiles directory and no active profile.

	rt, err := Build(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Build() with empty profiles error: %v", err)
	}

	if len(rt.Profiles) != 1 || rt.Profiles[0] != "" {
		t.Errorf("Profiles = %v, want [\"\"]", rt.Profiles)
	}
	if rt.Skills == nil || len(rt.Skills.List()) != 0 {
		t.Errorf("Skills should be an empty index, got %v", rt.Skills)
	}
	if got := rt.Manager.Active(); got != "" {
		t.Errorf("Manager.Active() = %q, want empty", got)
	}
	// No steering seed when there is no active profile.
	if msgs := rt.Agent.Steering().Drain(); len(msgs) != 0 {
		t.Errorf("unexpected steering seed: %+v", msgs)
	}
}

func TestBuildClientError(t *testing.T) {
	cfg := newTestConfig(t)
	cfg.Client = func() (core.Provider, error) {
		return nil, errors.New("invalid api key")
	}

	rt, err := Build(context.Background(), cfg)
	if err == nil {
		t.Fatal("Build() should return error when the provider client fails")
	}
	if err.Error() != "invalid api key" {
		t.Errorf("Build() error = %q, want %q", err.Error(), "invalid api key")
	}
	if rt != nil {
		t.Error("Build() should return nil runtime on failure (REQ-RELOAD-1)")
	}
}
