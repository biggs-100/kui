package agent

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/biggs-100/kui/internal/adapters/profile"
	"github.com/biggs-100/kui/internal/core"
)

// fakeTool is a minimal core.Tool for building a registry in tests.
type fakeTool struct {
	name string
}

func (t *fakeTool) Name() string        { return t.name }
func (t *fakeTool) Description() string { return "fake tool: " + t.name }
func (t *fakeTool) Schema() string      { return `{"type":"object"}` }
func (t *fakeTool) Execute(context.Context, json.RawMessage) (string, error) {
	return "ok", nil
}

// writeFile creates path (and its parents) with the given content.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

// newTestManager builds a manager over a temp profile tree. It writes
// profile.yaml for "coder" (with an optional SYSTEM.md body) in the profile
// root and registers a full registry carrying toolNames.
func newTestManager(t *testing.T, profileYAML, systemBody string, toolNames ...string) *Manager {
	t.Helper()
	root := t.TempDir()
	profileRoot := filepath.Join(root, "profiles")
	projectDir := filepath.Join(root, "project")
	globalDir := filepath.Join(root, "global")
	profileDir := filepath.Join(profileRoot, "coder")
	writeFile(t, filepath.Join(profileDir, "profile.yaml"), profileYAML)
	if systemBody != "" {
		writeFile(t, filepath.Join(profileDir, "SYSTEM.md"), systemBody)
	}
	full := core.NewRegistry()
	for _, name := range toolNames {
		if err := full.Register(&fakeTool{name: name}); err != nil {
			t.Fatalf("Register(%s): %v", name, err)
		}
	}
	loader := profile.NewLoader(profileRoot, projectDir, globalDir)
	return NewManager(loader, full)
}

func TestApplySwitchResolvesProfileAndReturnsMessages(t *testing.T) {
	// REQ-PROFILE-3 + REQ-LOOP-6: a successful switch returns the new system
	// prompt and a profile-context marker naming the profile, and records the
	// active profile and its resolved model (D17; SetModel lands in PR 5).
	manager := newTestManager(t, `
name: coder
model: gpt-4o
system_prompt: SYSTEM.md
tools: [read_file]
`, "You are the coder profile.\n", "bash", "read_file")

	messages, err := manager.ApplySwitch(context.Background(), "coder")
	if err != nil {
		t.Fatalf("ApplySwitch returned error: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("ApplySwitch returned %d messages, want 2 (system prompt + marker)", len(messages))
	}
	if messages[0].Role != core.RoleSystem || messages[0].Content != "You are the coder profile.\n" {
		t.Errorf("messages[0] = %+v, want the SYSTEM.md body as a system message", messages[0])
	}
	if messages[1].Role != core.RoleSystem || !strings.Contains(messages[1].Content, "coder") {
		t.Errorf("messages[1] = %+v, want a system marker naming coder", messages[1])
	}
	if manager.Active() != "coder" {
		t.Errorf("Active() = %q, want %q", manager.Active(), "coder")
	}
	if manager.Model() != "gpt-4o" {
		t.Errorf("Model() = %q, want %q", manager.Model(), "gpt-4o")
	}
}

func TestApplySwitchUnknownProfile(t *testing.T) {
	// REQ-PROFILE-3: switching to an unknown profile returns a typed error
	// and leaves the active profile unchanged.
	manager := newTestManager(t, "name: coder\n", "body\n", "bash")

	_, err := manager.ApplySwitch(context.Background(), "nope")
	var unknown *core.UnknownProfileError
	if !errors.As(err, &unknown) {
		t.Fatalf("ApplySwitch error = %v, want *core.UnknownProfileError", err)
	}
	if unknown.Name != "nope" {
		t.Errorf("UnknownProfileError.Name = %q, want %q", unknown.Name, "nope")
	}
	if manager.Active() != "" {
		t.Errorf("Active() = %q, want unchanged empty profile", manager.Active())
	}
}

func TestApplySwitchMissingSystemPrompt(t *testing.T) {
	// REQ-PROFILE-1: a profile whose system_prompt file is missing fails
	// activation with a typed error naming the file.
	manager := newTestManager(t, `
name: coder
system_prompt: SYSTEM.md
`, "", "bash")

	_, err := manager.ApplySwitch(context.Background(), "coder")
	var actErr *core.ProfileActivationError
	if !errors.As(err, &actErr) {
		t.Fatalf("ApplySwitch error = %v, want *core.ProfileActivationError", err)
	}
	if !strings.HasSuffix(actErr.File, "SYSTEM.md") {
		t.Errorf("ProfileActivationError.File = %q, want it to name SYSTEM.md", actErr.File)
	}
}

func TestApplySwitchRebuildsRegistryAndRuleset(t *testing.T) {
	// D16: ApplySwitch rebuilds the tool registry subset from the profile's
	// declared tools and the permission evaluator from its rules.
	manager := newTestManager(t, `
name: coder
model: gpt-4o
system_prompt: SYSTEM.md
tools: [read_file]
permissions:
  - pattern: "*"
    action: deny
  - pattern: read_file
    action: allow
`, "body\n", "bash", "read_file")

	_, err := manager.ApplySwitch(context.Background(), "coder")
	if err != nil {
		t.Fatalf("ApplySwitch failed: %v", err)
	}
	var names []string
	for _, tool := range manager.Registry().List() {
		names = append(names, tool.Name())
	}
	if want := []string{"read_file"}; !reflect.DeepEqual(names, want) {
		t.Errorf("Registry() tools = %v, want %v (profile tools only)", names, want)
	}
	rs := manager.Ruleset()
	if rs == nil {
		t.Fatal("Ruleset() = nil, want a rebuilt evaluator")
	}
	if rs.Allow("bash") {
		t.Error("Ruleset().Allow(bash) = true, want false (deny-all with read_file allow)")
	}
	if !rs.Allow("read_file") {
		t.Error("Ruleset().Allow(read_file) = false, want true")
	}
}

func TestApplySwitchSkipsUnknownDeclaredTool(t *testing.T) {
	// A profile declaring a tool that is not registered contributes no tool to
	// the subset; the switch still succeeds (D16).
	manager := newTestManager(t, `
name: coder
system_prompt: SYSTEM.md
tools: [ghost, bash]
`, "body\n", "bash")

	_, err := manager.ApplySwitch(context.Background(), "coder")
	if err != nil {
		t.Fatalf("ApplySwitch failed: %v", err)
	}
	var names []string
	for _, tool := range manager.Registry().List() {
		names = append(names, tool.Name())
	}
	if want := []string{"bash"}; !reflect.DeepEqual(names, want) {
		t.Errorf("Registry() tools = %v, want %v (ghost skipped)", names, want)
	}
}

func TestManagerRegistryDefaultsToFull(t *testing.T) {
	// Before any switch the manager exposes the full registry, so wiring
	// (PR 4) can hand the loop the same set with no special case.
	manager := newTestManager(t, "name: coder\n", "body\n", "bash", "read_file")

	if manager.Registry() == nil {
		t.Fatal("Registry() = nil before any switch, want the full registry")
	}
	if got, ok := manager.Registry().Get("bash"); !ok || got.Name() != "bash" {
		t.Errorf("Registry() before switch = %v, want the full registry with bash", got)
	}
}

// newManagerWithProfile is newTestManager plus the resolved profile directory,
// so tests can mutate the profile on disk (delete it, break its SYSTEM.md)
// between switches and Reload calls.
func newManagerWithProfile(t *testing.T, profileYAML, systemBody string, toolNames ...string) (*Manager, string) {
	t.Helper()
	root := t.TempDir()
	profileRoot := filepath.Join(root, "profiles")
	projectDir := filepath.Join(root, "project")
	globalDir := filepath.Join(root, "global")
	profileDir := filepath.Join(profileRoot, "coder")
	writeFile(t, filepath.Join(profileDir, "profile.yaml"), profileYAML)
	if systemBody != "" {
		writeFile(t, filepath.Join(profileDir, "SYSTEM.md"), systemBody)
	}
	full := core.NewRegistry()
	for _, name := range toolNames {
		if err := full.Register(&fakeTool{name: name}); err != nil {
			t.Fatalf("Register(%s): %v", name, err)
		}
	}
	loader := profile.NewLoader(profileRoot, projectDir, globalDir)
	return NewManager(loader, full), profileDir
}

// registryNames returns the registered tool names of a registry in order.
func registryNames(reg *core.Registry) []string {
	names := make([]string, 0, 1)
	for _, tool := range reg.List() {
		names = append(names, tool.Name())
	}
	return names
}

// newReloadFull builds a fresh registry carrying the given tools, as a reload
// would receive after re-reading disk state.
func newReloadFull(t *testing.T, toolNames ...string) *core.Registry {
	t.Helper()
	full := core.NewRegistry()
	for _, name := range toolNames {
		if err := full.Register(&fakeTool{name: name}); err != nil {
			t.Fatalf("Register(%s): %v", name, err)
		}
	}
	return full
}

func TestManagerConcurrentApplySwitchAndReadRaceFree(t *testing.T) {
	// REQ-RELOAD-9: ApplySwitch and Registry/Ruleset/Active/Model reads run
	// concurrently without a data race (verified by `go test -race`), and
	// state stays consistent after the storm.
	manager := newTestManager(t, `
name: coder
model: gpt-4o
system_prompt: SYSTEM.md
tools: [read_file]
`, "body\n", "bash", "read_file")

	var wg sync.WaitGroup
	errCh := make(chan error, 64)
	for i := 0; i < 32; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			if _, err := manager.ApplySwitch(context.Background(), "coder"); err != nil {
				errCh <- err
			}
		}()
		go func() {
			defer wg.Done()
			_ = manager.Registry()
			_ = manager.Ruleset()
			_ = manager.Active()
			_ = manager.Model()
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Errorf("concurrent ApplySwitch error: %v", err)
	}

	if manager.Active() != "coder" {
		t.Errorf("Active() = %q, want %q", manager.Active(), "coder")
	}
	if manager.Model() != "gpt-4o" {
		t.Errorf("Model() = %q, want %q", manager.Model(), "gpt-4o")
	}
	if want := []string{"read_file"}; !reflect.DeepEqual(registryNames(manager.Registry()), want) {
		t.Errorf("Registry() tools = %v, want %v", registryNames(manager.Registry()), want)
	}
}

func TestManagerReloadSwapsRegistryAndReappliesActiveProfile(t *testing.T) {
	// REQ-RELOAD-10: Reload swaps the full registry and re-applies the active
	// profile so the profile's subset includes the newly registered tool while
	// the active name and model are preserved. The profile declares new_tool,
	// but the pre-reload full registry does not carry it (REQ-PERM-1 skips
	// unregistered names), so only the reload makes it available.
	manager := newTestManager(t, `
name: coder
model: gpt-4o
system_prompt: SYSTEM.md
tools: [read_file, new_tool]
`, "body\n", "bash", "read_file")
	if _, err := manager.ApplySwitch(context.Background(), "coder"); err != nil {
		t.Fatalf("ApplySwitch failed: %v", err)
	}
	if want := []string{"read_file"}; !reflect.DeepEqual(registryNames(manager.Registry()), want) {
		t.Fatalf("pre-reload Registry() tools = %v, want %v (new_tool not yet registered)", registryNames(manager.Registry()), want)
	}

	// A new full registry registers the profile's missing tool.
	if err := manager.Reload(newReloadFull(t, "bash", "read_file", "new_tool")); err != nil {
		t.Fatalf("Reload returned error: %v", err)
	}

	if manager.Active() != "coder" {
		t.Errorf("Active() = %q, want %q preserved", manager.Active(), "coder")
	}
	if manager.Model() != "gpt-4o" {
		t.Errorf("Model() = %q, want %q preserved", manager.Model(), "gpt-4o")
	}
	if want := []string{"read_file", "new_tool"}; !reflect.DeepEqual(registryNames(manager.Registry()), want) {
		t.Errorf("Registry() tools = %v, want %v (profile subset over the new full registry)", registryNames(manager.Registry()), want)
	}
}

func TestManagerReloadUnknownProfileClearsActive(t *testing.T) {
	// D4 + REQ-RELOAD-10 fallback: when the active profile is deleted on
	// disk, Reload succeeds, clears the active profile, and exposes the new
	// full registry.
	manager, profileDir := newManagerWithProfile(t, `
name: coder
model: gpt-4o
system_prompt: SYSTEM.md
tools: [read_file]
`, "body\n", "bash", "read_file")
	if _, err := manager.ApplySwitch(context.Background(), "coder"); err != nil {
		t.Fatalf("ApplySwitch failed: %v", err)
	}

	// The profile disappears between the switch and the reload.
	if err := os.RemoveAll(profileDir); err != nil {
		t.Fatalf("RemoveAll(profileDir): %v", err)
	}

	if err := manager.Reload(newReloadFull(t, "bash")); err != nil {
		t.Fatalf("Reload with deleted profile = %v, want nil (cleared active, success)", err)
	}
	if manager.Active() != "" {
		t.Errorf("Active() = %q, want empty (cleared)", manager.Active())
	}
	if want := []string{"bash"}; !reflect.DeepEqual(registryNames(manager.Registry()), want) {
		t.Errorf("Registry() tools = %v, want %v (the new full registry)", registryNames(manager.Registry()), want)
	}
}

func TestManagerReloadOtherErrorKeepsOldRegistry(t *testing.T) {
	// REQ-RELOAD-10: when re-applying the active profile fails for a
	// non-UnknownProfileError reason, the old registry stays active and the
	// error is returned.
	manager, profileDir := newManagerWithProfile(t, `
name: coder
model: gpt-4o
system_prompt: SYSTEM.md
tools: [read_file]
`, "body\n", "bash", "read_file")
	if _, err := manager.ApplySwitch(context.Background(), "coder"); err != nil {
		t.Fatalf("ApplySwitch failed: %v", err)
	}

	// Break the active profile's SYSTEM.md so re-apply fails with a
	// ProfileActivationError — an error that is not UnknownProfileError.
	if err := os.Remove(filepath.Join(profileDir, "SYSTEM.md")); err != nil {
		t.Fatalf("Remove(SYSTEM.md): %v", err)
	}

	err := manager.Reload(newReloadFull(t, "bash", "read_file", "new_tool"))
	var actErr *core.ProfileActivationError
	if !errors.As(err, &actErr) {
		t.Fatalf("Reload error = %v, want *core.ProfileActivationError", err)
	}

	// The old registry stays active: still the profile's subset, without the
	// new tool, and the active profile is preserved.
	if want := []string{"read_file"}; !reflect.DeepEqual(registryNames(manager.Registry()), want) {
		t.Errorf("Registry() tools = %v, want %v (old registry kept)", registryNames(manager.Registry()), want)
	}
	if manager.Active() != "coder" {
		t.Errorf("Active() = %q, want %q preserved after failed Reload", manager.Active(), "coder")
	}
}
