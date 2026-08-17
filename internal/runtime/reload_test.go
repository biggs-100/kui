package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/biggs-100/kui/internal/core"
)

// writeSkill writes a local skill (skill.yaml + SKILL.md) under cfgRoot.
func writeSkill(t *testing.T, cfgRoot, name string) {
	t.Helper()
	dir := filepath.Join(cfgRoot, "skills", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	meta := "name: " + name + "\ndescription: skill " + name + "\ntriggers:\n  - " + name + "\n"
	if err := os.WriteFile(filepath.Join(dir, "skill.yaml"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# "+name+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeBrokenSkill writes a skill directory whose skill.yaml is malformed so
// the skills index rebuild fails (REQ-RELOAD-4 "skills index error").
func writeBrokenSkill(t *testing.T, cfgRoot, name string) {
	t.Helper()
	dir := filepath.Join(cfgRoot, "skills", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Unclosed flow sequence — guaranteed yaml parse failure.
	if err := os.WriteFile(filepath.Join(dir, "skill.yaml"), []byte("name: "+name+"\ndescription: [unclosed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// ── Reload tests (REQ-RELOAD-3, REQ-RELOAD-4, REQ-RELOAD-17/18) ────────────

func TestReloadPicksUpNewProfile(t *testing.T) {
	cfg := newTestConfig(t)
	writeProfile(t, cfg.ConfigRoot, "mycoder", "read_file")
	setActive(t, cfg.ConfigRoot, "mycoder")
	rt, err := Build(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	// A new profile lands on disk after startup.
	writeProfile(t, cfg.ConfigRoot, "newcoder", "read_file,write_file")

	res := rt.Reload(context.Background())
	if res.Err != nil {
		t.Fatalf("Reload() error: %v", res.Err)
	}
	found := false
	for _, p := range res.Profiles {
		if p == "newcoder" {
			found = true
		}
	}
	if !found {
		t.Errorf("Reload() Profiles = %v, want newcoder to appear (REQ-RELOAD-3)", res.Profiles)
	}
	if len(rt.Profiles) != 2 {
		t.Errorf("rt.Profiles = %v, want both profiles after reload", rt.Profiles)
	}
}

func TestReloadPicksUpNewSkill(t *testing.T) {
	cfg := newTestConfig(t)
	writeProfile(t, cfg.ConfigRoot, "mycoder", "read_file")
	setActive(t, cfg.ConfigRoot, "mycoder")
	writeSkill(t, cfg.ConfigRoot, "alpha")
	rt, err := Build(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}
	if _, ok := rt.Skills.Get("alpha"); !ok {
		t.Fatal("precondition: alpha skill not indexed at build time")
	}

	// A new skill lands on disk after startup.
	writeSkill(t, cfg.ConfigRoot, "beta")

	res := rt.Reload(context.Background())
	if res.Err != nil {
		t.Fatalf("Reload() error: %v", res.Err)
	}
	if _, ok := rt.Skills.Get("beta"); !ok {
		t.Error("Reload() did not index the new skill (REQ-RELOAD-3)")
	}
	if _, ok := rt.Skills.Get("alpha"); !ok {
		t.Error("Reload() dropped an existing skill")
	}
	if res.Skills != 2 {
		t.Errorf("ReloadResult.Skills = %d, want 2", res.Skills)
	}
}

func TestReloadRebuildsProvider(t *testing.T) {
	cfg := newTestConfig(t)
	writeProfile(t, cfg.ConfigRoot, "mycoder", "read_file")
	setActive(t, cfg.ConfigRoot, "mycoder")
	rt, err := Build(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}
	oldProvider := rt.Provider

	// The client factory now yields a fresh provider instance (re-reads env).
	newProvider := &fakeProvider{name: "recreated"}
	rt.cfg.Client = func() (core.Provider, error) {
		return newProvider, nil
	}

	res := rt.Reload(context.Background())
	if res.Err != nil {
		t.Fatalf("Reload() error: %v", res.Err)
	}
	if rt.Provider == oldProvider {
		t.Error("Reload() did not recreate the provider (REQ-RELOAD-3)")
	}
	if rt.Provider != newProvider {
		t.Error("Reload() provider is not the recreated one")
	}
	if got := rt.Agent.Provider(); got != newProvider {
		t.Error("Reload() did not wire the new provider into the agent")
	}
}

func TestReloadFailureKeepsOldState(t *testing.T) {
	cfg := newTestConfig(t)
	writeProfile(t, cfg.ConfigRoot, "mycoder", "read_file")
	setActive(t, cfg.ConfigRoot, "mycoder")
	writeSkill(t, cfg.ConfigRoot, "alpha")
	rt, err := Build(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}
	oldProvider := rt.Provider
	oldSkills := rt.Skills

	// A broken skill lands on disk so the rebuild fails at the skills index
	// (REQ-RELOAD-4 "e.g. skills index error").
	writeBrokenSkill(t, cfg.ConfigRoot, "broken")

	res := rt.Reload(context.Background())
	if res.Err == nil {
		t.Fatal("Reload() should return an error when the rebuild fails")
	}

	// Old state stays fully active: provider, skills, manager registry, active.
	if rt.Provider != oldProvider {
		t.Error("failed Reload swapped the provider (must keep old state)")
	}
	if rt.Skills != oldSkills {
		t.Error("failed Reload swapped the skills index (must keep old state)")
	}
	if _, ok := rt.Skills.Get("alpha"); !ok {
		t.Error("failed Reload lost an existing skill (old state must remain usable)")
	}
	if got := rt.Manager.Active(); got != "mycoder" {
		t.Errorf("failed Reload changed active profile: got %q, want %q", got, "mycoder")
	}
	if _, ok := rt.Manager.Registry().Get("read_file"); !ok {
		t.Error("failed Reload lost the active registry subset")
	}
}

func TestReloadCallsShutdownAllAndLoadAll(t *testing.T) {
	cfg := newTestConfig(t)
	rt, err := Build(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	testExt.mu.Lock()
	beforeInits := testExt.inits
	beforeShutdowns := testExt.shutdowns
	testExt.mu.Unlock()

	res := rt.Reload(context.Background())
	if res.Err != nil {
		t.Fatalf("Reload() error: %v", res.Err)
	}

	testExt.mu.Lock()
	defer testExt.mu.Unlock()
	if testExt.shutdowns <= beforeShutdowns {
		t.Error("Reload() did not call extensions.ShutdownAll before the rebuild (REQ-RELOAD-18)")
	}
	if testExt.inits <= beforeInits {
		t.Error("Reload() did not re-run extensions.LoadAll with the rebuilt registry (REQ-RELOAD-17)")
	}
	// The extension tool must be present in the post-reload full registry.
	if _, ok := rt.Full.Get(testExtToolName); !ok {
		t.Error("extension tool missing from the registry after reload")
	}
}
