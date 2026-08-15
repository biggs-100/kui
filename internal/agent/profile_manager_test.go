package agent

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
