package orchestration

import (
	"os"
	"path/filepath"
	"testing"
)

// ─── Task 1.1: RED — Test agent definition parsing ───

func TestAgentDefParse(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantName   string
		wantDesc   string
		wantModel  string
		wantPrompt string
		wantErr    bool
	}{
		{
			name: "valid frontmatter with body",
			content: `---
name: explorer
description: Read-only exploration agent
model: balanced
tools:
  include: [read, grep, glob]
---

You are a read-only explorer. Use tools to gather information.`,
			wantName:   "explorer",
			wantDesc:   "Read-only exploration agent",
			wantModel:  "balanced",
			wantPrompt: "You are a read-only explorer. Use tools to gather information.",
		},
		{
			name: "minimal frontmatter",
			content: `---
name: worker
---

Do work.`,
			wantName:   "worker",
			wantDesc:   "",
			wantModel:  "",
			wantPrompt: "Do work.",
		},
		{
			name:       "empty body",
			content:    "---\nname: test\n---\n",
			wantName:   "test",
			wantPrompt: "",
		},
		{
			name:    "no frontmatter delimiters",
			content: "name: not-parsed\nThis is just text.",
			wantErr: true,
		},
		{
			name:    "only opening delimiter",
			content: "---\nname: broken",
			wantErr: true,
		},
		{
			name:    "empty file",
			content: "",
			wantErr: true,
		},
		{
			name: "all fields populated",
			content: `---
name: reviewer
description: Code review agent
model: deep-reasoning
thinking: high
provider: anthropic
max_iterations: 10
tools:
  include: [read, grep]
  exclude: [write, bash]
---

Review code carefully.`,
			wantName:   "reviewer",
			wantDesc:   "Code review agent",
			wantModel:  "deep-reasoning",
			wantPrompt: "Review code carefully.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "agent.md")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}

			def, err := ParseAgentDef(path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if def.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", def.Name, tt.wantName)
			}
			if def.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", def.Description, tt.wantDesc)
			}
			if def.Model != tt.wantModel {
				t.Errorf("Model = %q, want %q", def.Model, tt.wantModel)
			}
			if def.SystemPrompt != tt.wantPrompt {
				t.Errorf("SystemPrompt = %q, want %q", def.SystemPrompt, tt.wantPrompt)
			}
		})
	}
}

// ─── Task 1.2: RED — Test byte-slice parsing ───

func TestAgentDefParseBytes(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		wantName   string
		wantPrompt string
		wantErr    bool
	}{
		{
			name:       "valid bytes",
			data:       []byte("---\nname: explore\n---\nYou explore."),
			wantName:   "explore",
			wantPrompt: "You explore.",
		},
		{
			name:    "empty bytes",
			data:    []byte{},
			wantErr: true,
		},
		{
			name:    "nil bytes",
			data:    nil,
			wantErr: true,
		},
		{
			name:       "bytes with tools",
			data:       []byte("---\nname: worker\ntools:\n  include: [read, write]\n  exclude: [bash]\n---\nWork hard."),
			wantName:   "worker",
			wantPrompt: "Work hard.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, err := ParseAgentDefBytes(tt.data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if def.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", def.Name, tt.wantName)
			}
			if def.SystemPrompt != tt.wantPrompt {
				t.Errorf("SystemPrompt = %q, want %q", def.SystemPrompt, tt.wantPrompt)
			}
		})
	}
}

// ─── Task 1.3: RED — Test tool include/exclude filtering ───

func TestAgentDefToolFilterInclude(t *testing.T) {
	def := &AgentDef{
		ToolsInclude: []string{"read", "grep"},
	}

	tools := []mockTool{
		{name: "read"},
		{name: "write"},
		{name: "grep"},
		{name: "bash"},
		{name: "glob"},
	}

	result := FilterTools(toolsAsCore(tools), def)
	got := toolNames(result)

	want := []string{"read", "grep"}
	if len(got) != len(want) {
		t.Fatalf("got %d tools, want %d", len(got), len(want))
	}
	for i, name := range want {
		if got[i] != name {
			t.Errorf("tool[%d] = %q, want %q", i, got[i], name)
		}
	}
}

func TestAgentDefToolFilterExclude(t *testing.T) {
	def := &AgentDef{
		ToolsExclude: []string{"bash", "write"},
	}

	tools := []mockTool{
		{name: "read"},
		{name: "write"},
		{name: "grep"},
		{name: "bash"},
		{name: "glob"},
	}

	result := FilterTools(toolsAsCore(tools), def)
	got := toolNames(result)

	want := []string{"read", "grep", "glob"}
	if len(got) != len(want) {
		t.Fatalf("got %d tools, want %d: %v", len(got), len(want), got)
	}
	for i, name := range want {
		if got[i] != name {
			t.Errorf("tool[%d] = %q, want %q", i, got[i], name)
		}
	}
}

func TestAgentDefToolFilterCombined(t *testing.T) {
	def := &AgentDef{
		ToolsInclude: []string{"read", "write", "grep", "bash"},
		ToolsExclude: []string{"bash"},
	}

	tools := []mockTool{
		{name: "read"},
		{name: "write"},
		{name: "grep"},
		{name: "bash"},
		{name: "glob"},
	}

	result := FilterTools(toolsAsCore(tools), def)
	got := toolNames(result)

	// Include sets the allowlist, then exclude removes from it
	want := []string{"read", "write", "grep"}
	if len(got) != len(want) {
		t.Fatalf("got %d tools, want %d: %v", len(got), len(want), got)
	}
	for i, name := range want {
		if got[i] != name {
			t.Errorf("tool[%d] = %q, want %q", i, got[i], name)
		}
	}
}

func TestAgentDefToolFilterEmpty(t *testing.T) {
	def := &AgentDef{} // no include or exclude

	tools := []mockTool{
		{name: "read"},
		{name: "write"},
		{name: "grep"},
	}

	result := FilterTools(toolsAsCore(tools), def)
	got := toolNames(result)

	// No filters → all tools pass through
	if len(got) != 3 {
		t.Fatalf("got %d tools, want 3: %v", len(got), got)
	}
}

// ─── Task 1.5: RED — Test agent definition resolution chain ───

func TestAgentDefResolution(t *testing.T) {
	// Create project-level agent
	projectDir := t.TempDir()
	agentDir := filepath.Join(projectDir, ".kui", "agents")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	agentContent := "---\nname: custom\nmodel: deep-reasoning\n---\nProject agent."
	if err := os.WriteFile(filepath.Join(agentDir, "custom.md"), []byte(agentContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create global-level agent
	globalDir := t.TempDir()
	globalAgentDir := filepath.Join(globalDir, "agents")
	if err := os.MkdirAll(globalAgentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	globalContent := "---\nname: global-agent\nmodel: fast\n---\nGlobal agent."
	if err := os.WriteFile(filepath.Join(globalAgentDir, "global-agent.md"), []byte(globalContent), 0o644); err != nil {
		t.Fatal(err)
	}

	registry := NewAgentRegistry([]string{
		filepath.Join(projectDir, ".kui", "agents"),
		globalAgentDir,
	})

	t.Run("project agent found", func(t *testing.T) {
		def, err := registry.Load("custom")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if def.Name != "custom" {
			t.Errorf("Name = %q, want %q", def.Name, "custom")
		}
		if def.Model != "deep-reasoning" {
			t.Errorf("Model = %q, want %q", def.Model, "deep-reasoning")
		}
	})

	t.Run("global agent found", func(t *testing.T) {
		def, err := registry.Load("global-agent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if def.Name != "global-agent" {
			t.Errorf("Name = %q, want %q", def.Name, "global-agent")
		}
	})

	t.Run("built-in agent found", func(t *testing.T) {
		def, err := registry.Load("explore")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if def.Name != "explore" {
			t.Errorf("Name = %q, want %q", def.Name, "explore")
		}
	})

	t.Run("project overrides global", func(t *testing.T) {
		// Create a global agent with same name as project
		sharedContent := "---\nname: shared\nmodel: fast\n---\nGlobal version."
		if err := os.WriteFile(filepath.Join(globalAgentDir, "shared.md"), []byte(sharedContent), 0o644); err != nil {
			t.Fatal(err)
		}
		sharedProjectContent := "---\nname: shared\nmodel: deep-reasoning\n---\nProject version."
		if err := os.WriteFile(filepath.Join(agentDir, "shared.md"), []byte(sharedProjectContent), 0o644); err != nil {
			t.Fatal(err)
		}

		def, err := registry.Load("shared")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if def.Model != "deep-reasoning" {
			t.Errorf("Model = %q, want %q (project should override global)", def.Model, "deep-reasoning")
		}
	})
}

func TestAgentRegistryList(t *testing.T) {
	projectDir := t.TempDir()
	agentDir := filepath.Join(projectDir, ".kui", "agents")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Add a project agent
	content := "---\nname: alpha\n---\nAlpha agent."
	if err := os.WriteFile(filepath.Join(agentDir, "alpha.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	registry := NewAgentRegistry([]string{agentDir})
	agents := registry.List()

	// Should include built-in agents + project agents
	names := make(map[string]bool)
	for _, a := range agents {
		names[a.Name] = true
	}

	// Built-in agents must be present
	for _, builtin := range []string{"explore", "worker", "reviewer"} {
		if !names[builtin] {
			t.Errorf("List() missing built-in agent %q", builtin)
		}
	}

	// Project agent must be present
	if !names["alpha"] {
		t.Errorf("List() missing project agent %q", "alpha")
	}
}

// ─── Task 1.7: RED — Test built-in agents ───

func TestBuiltinAgents(t *testing.T) {
	builtins := GetBuiltinAgents()

	// Must have exactly 3 built-in agents
	if len(builtins) != 3 {
		t.Fatalf("got %d built-in agents, want 3", len(builtins))
	}

	// Verify each built-in exists with correct properties
	tests := []struct {
		name        string
		wantInclude []string
	}{
		{name: "explore", wantInclude: []string{"read", "grep", "glob"}},
		{name: "worker", wantInclude: []string{"read", "write", "edit", "bash"}},
		{name: "reviewer", wantInclude: []string{"read", "grep"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, ok := builtins[tt.name]
			if !ok {
				t.Fatalf("built-in agent %q not found", tt.name)
			}
			if def.Name != tt.name {
				t.Errorf("Name = %q, want %q", def.Name, tt.name)
			}
			if len(def.ToolsInclude) != len(tt.wantInclude) {
				t.Fatalf("ToolsInclude has %d items, want %d", len(def.ToolsInclude), len(tt.wantInclude))
			}
			for i, tool := range tt.wantInclude {
				if def.ToolsInclude[i] != tool {
					t.Errorf("ToolsInclude[%d] = %q, want %q", i, def.ToolsInclude[i], tool)
				}
			}
		})
	}
}

// ─── Test helpers ───

type mockTool struct {
	name string
}

func (m mockTool) Name() string        { return m.name }
func (m mockTool) Description() string { return "mock tool: " + m.name }
func (m mockTool) Schema() string      { return "{}" }
func (m mockTool) Execute(_ interface{}, _ interface{}) (string, error) {
	return "", nil
}

// toolsAsCore converts mockTool slice to core.Tool slice using interface conversion.
// We define a local core.Tool-compatible interface to avoid import cycles.
func toolsAsCore(tools []mockTool) []agentTool {
	result := make([]agentTool, len(tools))
	for i, mt := range tools {
		result[i] = mt
	}
	return result
}

func toolNames(tools []agentTool) []string {
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name()
	}
	return names
}
