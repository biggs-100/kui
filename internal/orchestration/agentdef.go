package orchestration

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// AgentDef defines an agent's capabilities and orchestration metadata.
type AgentDef struct {
	Name          string   `yaml:"name"`
	Description   string   `yaml:"description"`
	ToolsInclude  []string `yaml:"tools.include"`
	ToolsExclude  []string `yaml:"tools.exclude"`
	Model         string   `yaml:"model"`
	Thinking      string   `yaml:"thinking"`
	Provider      string   `yaml:"provider"`
	MaxIterations int      `yaml:"max_iterations"`
	SystemPrompt  string   `yaml:"-"` // body from Markdown
}

// agentTool is the interface FilterTools operates on.
type agentTool interface {
	Name() string
}

// frontmatter is the YAML structure parsed between --- delimiters.
type frontmatter struct {
	Name          string   `yaml:"name"`
	Description   string   `yaml:"description"`
	ToolsInclude  []string `yaml:"tools.include"`
	ToolsExclude  []string `yaml:"tools.exclude"`
	Model         string   `yaml:"model"`
	Thinking      string   `yaml:"thinking"`
	Provider      string   `yaml:"provider"`
	MaxIterations int      `yaml:"max_iterations"`
}

// ParseAgentDef reads an agent definition from a file path.
func ParseAgentDef(path string) (*AgentDef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading agent def: %w", err)
	}
	return ParseAgentDefBytes(data)
}

// ParseAgentDefBytes parses an agent definition from a byte slice.
// Expects YAML frontmatter between --- delimiters, followed by optional markdown body.
func ParseAgentDefBytes(data []byte) (*AgentDef, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty agent definition")
	}

	// Find frontmatter delimiters
	lines := bytes.Split(data, []byte("\n"))
	if len(lines) < 2 || string(lines[0]) != "---" {
		return nil, fmt.Errorf("missing opening frontmatter delimiter '---'")
	}

	// Find closing ---
	closingIdx := -1
	for i := 1; i < len(lines); i++ {
		if string(lines[i]) == "---" {
			closingIdx = i
			break
		}
	}
	if closingIdx == -1 {
		return nil, fmt.Errorf("missing closing frontmatter delimiter '---'")
	}

	// Parse YAML frontmatter
	var fm frontmatter
	fmBytes := bytes.Join(lines[1:closingIdx], []byte("\n"))
	if err := yaml.Unmarshal(fmBytes, &fm); err != nil {
		return nil, fmt.Errorf("parsing frontmatter: %w", err)
	}

	// Extract markdown body (everything after closing ---)
	var body string
	if closingIdx+1 < len(lines) {
		bodyLines := lines[closingIdx+1:]
		body = strings.TrimSpace(string(bytes.Join(bodyLines, []byte("\n"))))
	}

	return &AgentDef{
		Name:          fm.Name,
		Description:   fm.Description,
		ToolsInclude:  fm.ToolsInclude,
		ToolsExclude:  fm.ToolsExclude,
		Model:         fm.Model,
		Thinking:      fm.Thinking,
		Provider:      fm.Provider,
		MaxIterations: fm.MaxIterations,
		SystemPrompt:  body,
	}, nil
}

// FilterTools applies include/exclude rules from the agent definition.
// If ToolsInclude is non-empty, only those tools pass through.
// Then ToolsExclude removes any listed tools.
func FilterTools(tools []agentTool, def *AgentDef) []agentTool {
	if def == nil {
		return tools
	}

	includeSet := toSet(def.ToolsInclude)
	excludeSet := toSet(def.ToolsExclude)

	result := make([]agentTool, 0, len(tools))
	for _, tool := range tools {
		name := tool.Name()
		// If include list exists, tool must be in it
		if len(includeSet) > 0 && !includeSet[name] {
			continue
		}
		// If exclude list exists, tool must not be in it
		if excludeSet[name] {
			continue
		}
		result = append(result, tool)
	}
	return result
}

func toSet(items []string) map[string]bool {
	if len(items) == 0 {
		return nil
	}
	set := make(map[string]bool, len(items))
	for _, item := range items {
		set[item] = true
	}
	return set
}

// ─── Agent Registry ───

// AgentRegistry discovers and loads agent definitions from filesystem paths
// with fallback to built-in agents.
type AgentRegistry struct {
	paths  []string
	agents map[string]*AgentDef
}

// NewAgentRegistry creates a registry that searches the given paths in order.
func NewAgentRegistry(paths []string) *AgentRegistry {
	return &AgentRegistry{
		paths:  paths,
		agents: make(map[string]*AgentDef),
	}
}

// Load finds an agent by name: search paths in order, then built-ins.
// First match wins.
func (r *AgentRegistry) Load(name string) (*AgentDef, error) {
	// Search filesystem paths in order
	for _, dir := range r.paths {
		path := dir + "/" + name + ".md"
		def, err := ParseAgentDef(path)
		if err == nil && def.Name == name {
			return def, nil
		}
	}

	// Fallback to built-in agents
	builtins := GetBuiltinAgents()
	if def, ok := builtins[name]; ok {
		return def, nil
	}

	return nil, fmt.Errorf("agent %q not found", name)
}

// List returns all discovered agents: filesystem agents first, then built-ins.
func (r *AgentRegistry) List() []*AgentDef {
	var result []*AgentDef
	seen := make(map[string]bool)

	// Collect from filesystem paths
	for _, dir := range r.paths {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			path := dir + "/" + entry.Name()
			def, err := ParseAgentDef(path)
			if err != nil || def.Name == "" {
				continue
			}
			if !seen[def.Name] {
				result = append(result, def)
				seen[def.Name] = true
			}
		}
	}

	// Add built-ins
	builtins := GetBuiltinAgents()
	for _, def := range builtins {
		if !seen[def.Name] {
			result = append(result, def)
			seen[def.Name] = true
		}
	}

	return result
}

// ─── Built-in Agents ───

// GetBuiltinAgents returns the hardcoded built-in agent definitions.
func GetBuiltinAgents() map[string]*AgentDef {
	return map[string]*AgentDef{
		"explore": {
			Name:         "explore",
			Description:  "Read-only exploration agent",
			ToolsInclude: []string{"read", "grep", "glob"},
			SystemPrompt: "You are a read-only exploration agent. Use tools to gather information without modifying any files.",
		},
		"worker": {
			Name:         "worker",
			Description:  "Implementation agent with full tool access",
			ToolsInclude: []string{"read", "write", "edit", "bash"},
			SystemPrompt: "You are an implementation agent. Use tools to write, edit, and execute code.",
		},
		"reviewer": {
			Name:         "reviewer",
			Description:  "Code review agent with read-only access",
			ToolsInclude: []string{"read", "grep"},
			SystemPrompt: "You are a code review agent. Analyze code for quality, security, and correctness.",
		},
	}
}
