package agent

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/biggs-100/kui/internal/adapters/skills"
	"github.com/biggs-100/kui/internal/core"
)

// fakeProvider implements core.Provider, returning a scripted response per
// call and recording every message sequence it received.
type fakeProvider struct {
	responses [][]core.Message
	calls     [][]core.Message
}

func (f *fakeProvider) Chat(_ context.Context, messages []core.Message, _ []core.Tool) ([]core.Message, error) {
	f.calls = append(f.calls, append([]core.Message(nil), messages...))
	if len(f.responses) == 0 {
		return []core.Message{{Role: core.RoleAssistant, Content: "done"}}, nil
	}
	response := f.responses[0]
	f.responses = f.responses[1:]
	return response, nil
}

// writeSkillDir writes a skill.yaml (and optional SKILL.md body) under a
// layer root so a skills index can discover it.
func writeSkillDir(t *testing.T, root, name, trigger, body string) {
	t.Helper()
	dir := filepath.Join(root, "skills", name)
	writeFile(t, filepath.Join(dir, "skill.yaml"),
		"name: "+name+"\ndescription: description for "+name+"\ntriggers:\n  - "+trigger+"\n")
	if body != "" {
		writeFile(t, filepath.Join(dir, "SKILL.md"), body)
	}
}

// newTestSkills builds a skills index over the given per-skill bodies.
func newTestSkills(t *testing.T, skillsByTrigger map[string]string) *skills.Index {
	t.Helper()
	root := t.TempDir()
	for name, body := range skillsByTrigger {
		trigger := strings.Split(name, "-")[0]
		writeSkillDir(t, root, name, trigger, body)
	}
	index, err := skills.NewIndex(root, t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatalf("NewIndex failed: %v", err)
	}
	return index
}

func TestSystemMessagesIndexOnly(t *testing.T) {
	// REQ-SKILL-3, index-only prompt: the system messages list skill names,
	// descriptions and triggers but NEVER contain any body text.
	index := newTestSkills(t, map[string]string{
		"go-testing": "# go-testing\nsecret instructions for go tests\n",
		"k8s":        "# k8s\nsecret kubectl recipes\n",
	})
	agent := NewAgent(newTestManager(t, "name: coder\n", "", "bash"), index, nil, 4)

	messages := agent.SystemMessages()
	if len(messages) != 1 {
		t.Fatalf("SystemMessages() has %d messages, want exactly 1", len(messages))
	}
	if messages[0].Role != core.RoleSystem {
		t.Errorf("SystemMessages()[0].Role = %q, want %q", messages[0].Role, core.RoleSystem)
	}
	content := messages[0].Content
	for _, want := range []string{"go-testing", "k8s", "description for go-testing", "go-test"} {
		if !strings.Contains(content, want) {
			t.Errorf("SystemMessages content missing %q: %q", want, content)
		}
	}
	if strings.Contains(content, "secret instructions") || strings.Contains(content, "secret kubectl") {
		t.Errorf("SystemMessages content contains skill body text: %q", content)
	}
}

func TestSystemMessagesNoSkills(t *testing.T) {
	// An agent with no indexed skills seeds no system messages.
	index := newTestSkills(t, map[string]string{})
	agent := NewAgent(newTestManager(t, "name: coder\n", "", "bash"), index, nil, 4)
	if got := agent.SystemMessages(); len(got) != 0 {
		t.Errorf("SystemMessages() = %v, want none for an empty skills index", got)
	}
}

func TestLoadSkillOnInvocation(t *testing.T) {
	// REQ-SKILL-3, body loads on invocation: LoadSkill returns the full body
	// of the named skill at the moment it is requested.
	index := newTestSkills(t, map[string]string{
		"go-testing": "# go-testing\n\nRun and debug Go tests.\n",
	})
	agent := NewAgent(newTestManager(t, "name: coder\n", "", "bash"), index, nil, 4)

	body, err := agent.LoadSkill("go-testing")
	if err != nil {
		t.Fatalf("LoadSkill returned error: %v", err)
	}
	if want := "# go-testing\n\nRun and debug Go tests.\n"; body != want {
		t.Errorf("LoadSkill() = %q, want %q", body, want)
	}
}

func TestLoadSkillUnknown(t *testing.T) {
	// Loading a skill that is not indexed returns a typed error naming it.
	index := newTestSkills(t, map[string]string{"go-testing": "body\n"})
	agent := NewAgent(newTestManager(t, "name: coder\n", "", "bash"), index, nil, 4)

	_, err := agent.LoadSkill("nope")
	var loadErr *core.SkillLoadError
	if !errors.As(err, &loadErr) {
		t.Fatalf("LoadSkill error = %v, want *core.SkillLoadError", err)
	}
	if loadErr.Name != "nope" {
		t.Errorf("SkillLoadError.Name = %q, want %q", loadErr.Name, "nope")
	}
}

func TestRunWiresSteeringAndReturnsAnswer(t *testing.T) {
	// Run wires the core loop through the wrapper: a queued steering message
	// is injected as a user message before the second provider request, and
	// the final answer is the provider's last response.
	provider := &fakeProvider{responses: [][]core.Message{
		{{Role: core.RoleAssistant, Content: "first"}},
		{{Role: core.RoleAssistant, Content: "final"}},
	}}
	index := newTestSkills(t, map[string]string{})
	agent := NewAgent(newTestManager(t, "name: coder\n", "", "bash"), index, provider, 4)
	agent.Steering().Enqueue(core.PendingMessage{Content: "steering note"})

	answer, _, err := agent.Run(context.Background(), "hello", nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "final" {
		t.Errorf("Run() = %q, want %q", answer, "final")
	}
	if len(provider.calls) != 2 {
		t.Fatalf("provider received %d calls, want 2 (initial turn + steering-injected turn)", len(provider.calls))
	}
	var injected bool
	for _, m := range provider.calls[1] {
		if m.Role == core.RoleUser && m.Content == "steering note" {
			injected = true
		}
	}
	if !injected {
		t.Errorf("second provider call did not receive the queued steering message: %+v", provider.calls[1])
	}
}

func TestRunEmptySteeringReturnsSingleAnswer(t *testing.T) {
	// With no queued messages the wrapper reproduces the plain single-level
	// loop: one provider call and its answer returned.
	provider := &fakeProvider{responses: [][]core.Message{
		{{Role: core.RoleAssistant, Content: "plain"}},
	}}
	index := newTestSkills(t, map[string]string{})
	agent := NewAgent(newTestManager(t, "name: coder\n", "", "bash"), index, provider, 4)

	answer, _, err := agent.Run(context.Background(), "hello", nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "plain" {
		t.Errorf("Run() = %q, want %q", answer, "plain")
	}
	if len(provider.calls) != 1 {
		t.Errorf("provider received %d calls, want 1", len(provider.calls))
	}
}

func TestSetSkillsReplacesIndex(t *testing.T) {
	// REQ-RELOAD-19: SetSkills swaps the skills index so SystemMessages
	// reflects the new index, not the old one.
	oldIndex := newTestSkills(t, map[string]string{"old-skill": "old body\n"})
	newIndex := newTestSkills(t, map[string]string{"new-skill": "new body\n"})
	agent := NewAgent(newTestManager(t, "name: coder\n", "", "bash"), oldIndex, nil, 4)

	agent.SetSkills(newIndex)
	messages := agent.SystemMessages()
	if len(messages) != 1 {
		t.Fatalf("SystemMessages() = %v, want one system message for the new index", messages)
	}
	content := messages[0].Content
	if !strings.Contains(content, "new-skill") {
		t.Errorf("SystemMessages content missing new-skill: %q", content)
	}
	if strings.Contains(content, "old-skill") {
		t.Errorf("SystemMessages content still references old-skill after SetSkills: %q", content)
	}
}

func TestSetProviderReplacesProvider(t *testing.T) {
	// REQ-RELOAD-19: SetProvider swaps the provider so subsequent runs use the
	// new one and Provider() exposes it (StreamingProvider detection).
	first := &fakeProvider{responses: [][]core.Message{
		{{Role: core.RoleAssistant, Content: "first"}},
	}}
	second := &fakeProvider{responses: [][]core.Message{
		{{Role: core.RoleAssistant, Content: "second"}},
	}}
	index := newTestSkills(t, map[string]string{})
	agent := NewAgent(newTestManager(t, "name: coder\n", "", "bash"), index, first, 4)

	agent.SetProvider(second)
	if got := agent.Provider(); got != second {
		t.Fatalf("Provider() = %T, want the new provider", got)
	}
	answer, _, err := agent.Run(context.Background(), "hello", nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "second" {
		t.Errorf("Run() = %q, want %q (from the new provider)", answer, "second")
	}
	if len(first.calls) != 0 {
		t.Errorf("old provider received %d calls, want 0 after SetProvider", len(first.calls))
	}
	if len(second.calls) != 1 {
		t.Errorf("new provider received %d calls, want 1", len(second.calls))
	}
}

func TestSetHooksReplacesRegistry(t *testing.T) {
	// REQ-RELOAD-19/20: SetHooks swaps the hook registry and Agent.Run wires
	// it into the loop, so the new registry's hook fires during a run while
	// the replaced registry's hook does not.
	provider := &fakeProvider{responses: [][]core.Message{
		{{Role: core.RoleAssistant, Content: "done"}},
	}}
	index := newTestSkills(t, map[string]string{})
	agent := NewAgent(newTestManager(t, "name: coder\n", "", "bash"), index, provider, 4)

	oldHooks := core.NewHookRegistry()
	oldFired := false
	_ = oldHooks.Register("before_provider_request", func(core.HookContext) error {
		oldFired = true
		return nil
	})
	agent.SetHooks(oldHooks)

	newHooks := core.NewHookRegistry()
	fired := false
	_ = newHooks.Register("before_provider_request", func(core.HookContext) error {
		fired = true
		return nil
	})
	agent.SetHooks(newHooks)

	if _, _, err := agent.Run(context.Background(), "hello", nil); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !fired {
		t.Error("hook from the new registry did not fire during Run (loop.Hooks not wired)")
	}
	if oldFired {
		t.Error("hook from the replaced registry fired after SetHooks")
	}
}

func TestSetHooksNilIsSafe(t *testing.T) {
	// REQ-RELOAD-20: a nil hook registry keeps the loop's behavior identical —
	// SetHooks(nil) is a safe no-op and no hooks fire.
	provider := &fakeProvider{responses: [][]core.Message{
		{{Role: core.RoleAssistant, Content: "plain"}},
	}}
	index := newTestSkills(t, map[string]string{})
	agent := NewAgent(newTestManager(t, "name: coder\n", "", "bash"), index, provider, 4)

	agent.SetHooks(nil)
	answer, _, err := agent.Run(context.Background(), "hello", nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "plain" {
		t.Errorf("Run() = %q, want %q (nil hooks unchanged behavior)", answer, "plain")
	}
	if len(provider.calls) != 1 {
		t.Errorf("provider received %d calls, want 1", len(provider.calls))
	}
}

// --- Session Persistence (Phase 3: History Integration) ---

func TestRunAcceptsHistory(t *testing.T) {
	// Task 3.1 RED: Verify that []core.Message history is prepended to the
	// provider call. The provider should see history messages before the
	// current user prompt.
	provider := &fakeProvider{responses: [][]core.Message{
		{{Role: core.RoleAssistant, Content: "response"}},
	}}
	index := newTestSkills(t, map[string]string{})
	ag := NewAgent(newTestManager(t, "name: coder\n", "", "bash"), index, provider, 4)

	history := []core.Message{
		{Role: core.RoleUser, Content: "old question"},
		{Role: core.RoleAssistant, Content: "old answer"},
	}

	_, _, err := ag.Run(context.Background(), "new question", history)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	// The provider should have received 1 call with history prepended.
	if len(provider.calls) != 1 {
		t.Fatalf("provider received %d calls, want 1", len(provider.calls))
	}

	call := provider.calls[0]
	// Expected: history[0], history[1], new user prompt
	if len(call) != 3 {
		t.Fatalf("provider received %d messages, want 3 (2 history + 1 prompt)", len(call))
	}
	if call[0].Role != core.RoleUser || call[0].Content != "old question" {
		t.Errorf("call[0] = %+v, want RoleUser 'old question'", call[0])
	}
	if call[1].Role != core.RoleAssistant || call[1].Content != "old answer" {
		t.Errorf("call[1] = %+v, want RoleAssistant 'old answer'", call[1])
	}
	if call[2].Role != core.RoleUser || call[2].Content != "new question" {
		t.Errorf("call[2] = %+v, want RoleUser 'new question'", call[2])
	}
}

func TestRunReturnsFinalMessages(t *testing.T) {
	// Task 3.3 RED: Verify that Run returns the accumulated messages slice
	// including user prompt, assistant response, and tool results.
	provider := &fakeProvider{responses: [][]core.Message{
		{{Role: core.RoleAssistant, Content: "done"}},
	}}
	index := newTestSkills(t, map[string]string{})
	ag := NewAgent(newTestManager(t, "name: coder\n", "", "bash"), index, provider, 4)

	answer, messages, err := ag.Run(context.Background(), "hello", nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "done" {
		t.Errorf("Run() answer = %q, want %q", answer, "done")
	}

	// Messages should include: user prompt + assistant response
	if len(messages) != 2 {
		t.Fatalf("Run() returned %d messages, want 2 (user + assistant)", len(messages))
	}
	if messages[0].Role != core.RoleUser || messages[0].Content != "hello" {
		t.Errorf("messages[0] = %+v, want RoleUser 'hello'", messages[0])
	}
	if messages[1].Role != core.RoleAssistant || messages[1].Content != "done" {
		t.Errorf("messages[1] = %+v, want RoleAssistant 'done'", messages[1])
	}
}

func TestRunEmptyHistoryNoPrepend(t *testing.T) {
	// Task 3.2: nil history should produce identical behavior to old Run.
	provider := &fakeProvider{responses: [][]core.Message{
		{{Role: core.RoleAssistant, Content: "ok"}},
	}}
	index := newTestSkills(t, map[string]string{})
	ag := NewAgent(newTestManager(t, "name: coder\n", "", "bash"), index, provider, 4)

	answer, _, err := ag.Run(context.Background(), "hello", nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "ok" {
		t.Errorf("Run() = %q, want %q", answer, "ok")
	}
	// Provider should see only the user prompt (no history prepended).
	if len(provider.calls) != 1 {
		t.Fatalf("provider received %d calls, want 1", len(provider.calls))
	}
	if len(provider.calls[0]) != 1 {
		t.Errorf("provider received %d messages, want 1 (no history)", len(provider.calls[0]))
	}
}
