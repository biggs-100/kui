package subagent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// AgentDef represents a markdown-based agent definition.
type AgentDef struct {
	// Name is the agent identifier (matches filename without .md).
	Name string `yaml:"name"`
	// Description is a human-readable description of what the agent does.
	Description string `yaml:"description"`
	// Tools is the list of tools the agent can use.
	Tools []string `yaml:"tools"`
	// Model overrides the default model for this agent (optional).
	Model string `yaml:"model,omitempty"`
	// Thinking overrides the default thinking level (optional).
	Thinking string `yaml:"thinking,omitempty"`
	// Prompt is the system prompt (body of the markdown file, after frontmatter).
	Prompt string `yaml:"-"`
	// Path is the file path where this definition was loaded from.
	Path string `yaml:"-"`
}

// ParseAgentDef parses a markdown agent definition file.
// Format: YAML frontmatter (between ---) followed by markdown body.
func ParseAgentDef(content string, path string) (*AgentDef, error) {
	// Extract frontmatter.
	if !strings.HasPrefix(content, "---") {
		return nil, fmt.Errorf("no frontmatter found in %s", path)
	}

	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid frontmatter format in %s", path)
	}

	var def AgentDef
	if err := yaml.Unmarshal([]byte(parts[1]), &def); err != nil {
		return nil, fmt.Errorf("invalid YAML frontmatter in %s: %w", path, err)
	}

	if def.Name == "" {
		return nil, fmt.Errorf("agent definition missing name in %s", path)
	}

	def.Prompt = strings.TrimSpace(parts[2])
	def.Path = path

	return &def, nil
}

// LoadAgentDef loads an agent definition from a file.
func LoadAgentDef(path string) (*AgentDef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read agent definition: %w", err)
	}
	return ParseAgentDef(string(data), path)
}

// LoadAgentDefsFromDir loads all agent definitions from a directory.
// Files must have .md extension and valid frontmatter.
func LoadAgentDefsFromDir(dir string) (map[string]*AgentDef, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read agents dir: %w", err)
	}

	defs := make(map[string]*AgentDef)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		def, err := LoadAgentDef(path)
		if err != nil {
			continue // skip invalid definitions
		}
		defs[def.Name] = def
	}

	return defs, nil
}

// ResolveAgent searches for an agent definition in:
// 1. Project dir: .kui/agents/{name}.md
// 2. Global dir: ~/.config/kui/agents/{name}.md
// 3. Built-in agents (embedded)
func ResolveAgent(name, projectDir, globalDir string) (*AgentDef, error) {
	// Try project dir.
	projectPath := filepath.Join(projectDir, ".kui", "agents", name+".md")
	if def, err := LoadAgentDef(projectPath); err == nil {
		return def, nil
	}

	// Try global dir.
	globalPath := filepath.Join(globalDir, "agents", name+".md")
	if def, err := LoadAgentDef(globalPath); err == nil {
		return def, nil
	}

	// Try built-in agents.
	if def := GetBuiltinAgent(name); def != nil {
		return def, nil
	}

	return nil, fmt.Errorf("agent %q not found", name)
}

// ListAgents returns all available agent names from project, global, and built-in.
func ListAgents(projectDir, globalDir string) []string {
	seen := make(map[string]bool)
	var names []string

	// Project agents.
	projectDir = filepath.Join(projectDir, ".kui", "agents")
	if entries, err := os.ReadDir(projectDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				name := strings.TrimSuffix(e.Name(), ".md")
				if !seen[name] {
					seen[name] = true
					names = append(names, name)
				}
			}
		}
	}

	// Global agents.
	globalDir = filepath.Join(globalDir, "agents")
	if entries, err := os.ReadDir(globalDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				name := strings.TrimSuffix(e.Name(), ".md")
				if !seen[name] {
					seen[name] = true
					names = append(names, name)
				}
			}
		}
	}

	// Built-in agents.
	for name := range builtinAgents {
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}

	return names
}

// builtinAgents contains the built-in agent definitions.
var builtinAgents = map[string]*AgentDef{
	"explore": {
		Name:        "explore",
		Description: "Read-only exploration and analysis of codebases",
		Tools:       []string{"read", "grep", "glob"},
		Prompt: `You are a read-only exploration agent. Your job is to investigate the codebase and return a structured analysis.

## Rules
- Read real code, never guess
- Do NOT modify any files
- Return a structured analysis with: Current State, Affected Areas, Recommendations, Risks

## Output Format
Return your findings in a clear, structured format with sections.`,
	},
	"worker": {
		Name:        "worker",
		Description: "Implementation worker for writing code",
		Tools:       []string{"read", "write", "edit", "grep", "glob", "bash"},
		Prompt: `You are an implementation worker. Your job is to write code according to the given task.

## Rules
- Follow existing code patterns and conventions
- Write tests when appropriate
- Run tests to verify your changes
- Return a summary of what you changed

## Output Format
Return: files changed, what was done, test results.`,
	},
	"reviewer": {
		Name:        "reviewer",
		Description: "Code review and analysis",
		Tools:       []string{"read", "grep", "glob"},
		Prompt: `You are a code reviewer. Your job is to analyze code and provide feedback.

## Rules
- Focus on correctness, security, and maintainability
- Provide specific, actionable feedback
- Reference file paths and line numbers
- Categorize findings by severity

## Output Format
Return: findings with severity (CRITICAL/WARNING/SUGGESTION), file references, and recommendations.`,
	},
}

// GetBuiltinAgent returns a built-in agent definition by name.
func GetBuiltinAgent(name string) *AgentDef {
	def, ok := builtinAgents[name]
	if !ok {
		return nil
	}
	// Return a copy to prevent mutation.
	copy := *def
	return &copy
}
