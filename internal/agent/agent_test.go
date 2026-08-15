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

	answer, err := agent.Run(context.Background(), "hello")
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

	answer, err := agent.Run(context.Background(), "hello")
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
