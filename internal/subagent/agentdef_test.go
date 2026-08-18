package subagent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseAgentDefValid(t *testing.T) {
	content := `---
name: test-agent
description: A test agent
tools:
  - read
  - bash
model: gpt-4o
thinking: medium
---

You are a test agent.

## Rules
- Be helpful
- Follow instructions`

	def, err := ParseAgentDef(content, "test.md")
	if err != nil {
		t.Fatalf("ParseAgentDef() error = %v", err)
	}
	if def.Name != "test-agent" {
		t.Errorf("Name = %q, want %q", def.Name, "test-agent")
	}
	if def.Description != "A test agent" {
		t.Errorf("Description = %q, want %q", def.Description, "A test agent")
	}
	if len(def.Tools) != 2 {
		t.Errorf("Tools len = %d, want 2", len(def.Tools))
	}
	if def.Model != "gpt-4o" {
		t.Errorf("Model = %q, want %q", def.Model, "gpt-4o")
	}
	if def.Thinking != "medium" {
		t.Errorf("Thinking = %q, want %q", def.Thinking, "medium")
	}
	if def.Prompt == "" {
		t.Error("Prompt is empty")
	}
}

func TestParseAgentDefNoFrontmatter(t *testing.T) {
	_, err := ParseAgentDef("just markdown", "test.md")
	if err == nil {
		t.Error("expected error for no frontmatter")
	}
}

func TestParseAgentDefMissingName(t *testing.T) {
	content := `---
description: No name
tools:
  - read
---

Body`
	_, err := ParseAgentDef(content, "test.md")
	if err == nil {
		t.Error("expected error for missing name")
	}
}

func TestLoadAgentDef(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "agent.md")
	os.WriteFile(path, []byte(`---
name: loaded-agent
description: Loaded from file
tools:
  - read
---

Body text`), 0644)

	def, err := LoadAgentDef(path)
	if err != nil {
		t.Fatalf("LoadAgentDef() error = %v", err)
	}
	if def.Name != "loaded-agent" {
		t.Errorf("Name = %q, want %q", def.Name, "loaded-agent")
	}
	if def.Path != path {
		t.Errorf("Path = %q, want %q", def.Path, path)
	}
}

func TestLoadAgentDefsFromDir(t *testing.T) {
	tmp := t.TempDir()

	// Valid agent.
	os.WriteFile(filepath.Join(tmp, "agent1.md"), []byte(`---
name: agent1
description: First agent
tools:
  - read
---
Body`), 0644)

	// Another valid agent.
	os.WriteFile(filepath.Join(tmp, "agent2.md"), []byte(`---
name: agent2
description: Second agent
tools:
  - bash
---
Body`), 0644)

	// Invalid file (no frontmatter).
	os.WriteFile(filepath.Join(tmp, "invalid.md"), []byte(`just markdown`), 0644)

	// Non-markdown file.
	os.WriteFile(filepath.Join(tmp, "skip.txt"), []byte(`text`), 0644)

	defs, err := LoadAgentDefsFromDir(tmp)
	if err != nil {
		t.Fatalf("LoadAgentDefsFromDir() error = %v", err)
	}
	if len(defs) != 2 {
		t.Errorf("got %d agents, want 2", len(defs))
	}
	if _, ok := defs["agent1"]; !ok {
		t.Error("agent1 not found")
	}
	if _, ok := defs["agent2"]; !ok {
		t.Error("agent2 not found")
	}
}

func TestResolveAgentProject(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "project")
	globalDir := filepath.Join(tmp, "global")
	agentsDir := filepath.Join(projectDir, ".kui", "agents")
	os.MkdirAll(agentsDir, 0755)

	os.WriteFile(filepath.Join(agentsDir, "my-agent.md"), []byte(`---
name: my-agent
description: Project agent
tools:
  - read
---
Body`), 0644)

	def, err := ResolveAgent("my-agent", projectDir, globalDir)
	if err != nil {
		t.Fatalf("ResolveAgent() error = %v", err)
	}
	if def.Name != "my-agent" {
		t.Errorf("Name = %q, want %q", def.Name, "my-agent")
	}
}

func TestResolveAgentBuiltin(t *testing.T) {
	tmp := t.TempDir()

	def, err := ResolveAgent("explore", tmp, tmp)
	if err != nil {
		t.Fatalf("ResolveAgent() error = %v", err)
	}
	if def.Name != "explore" {
		t.Errorf("Name = %q, want %q", def.Name, "explore")
	}
}

func TestResolveAgentNotFound(t *testing.T) {
	tmp := t.TempDir()
	_, err := ResolveAgent("nonexistent", tmp, tmp)
	if err == nil {
		t.Error("expected error for nonexistent agent")
	}
}

func TestListAgents(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "project")
	globalDir := filepath.Join(tmp, "global")

	// Create project agent.
	os.MkdirAll(filepath.Join(projectDir, ".kui", "agents"), 0755)
	os.WriteFile(filepath.Join(projectDir, ".kui", "agents", "proj-agent.md"), []byte(`---
name: proj-agent
description: Project agent
tools:
  - read
---
Body`), 0644)

	names := ListAgents(projectDir, globalDir)

	// Should include project agent and built-in agents.
	found := false
	for _, n := range names {
		if n == "proj-agent" {
			found = true
			break
		}
	}
	if !found {
		t.Error("proj-agent not found in ListAgents")
	}

	// Should include built-in agents.
	foundBuiltin := false
	for _, n := range names {
		if n == "explore" {
			foundBuiltin = true
			break
		}
	}
	if !foundBuiltin {
		t.Error("explore (builtin) not found in ListAgents")
	}
}

func TestGetBuiltinAgent(t *testing.T) {
	def := GetBuiltinAgent("explore")
	if def == nil {
		t.Fatal("GetBuiltinAgent(explore) returned nil")
	}
	if def.Name != "explore" {
		t.Errorf("Name = %q, want %q", def.Name, "explore")
	}

	// Should return a copy.
	def.Name = "mutated"
	orig := GetBuiltinAgent("explore")
	if orig.Name != "explore" {
		t.Error("GetBuiltinAgent returned mutable reference")
	}
}
