package orchestration

// Engine wires all orchestration components together.
type Engine struct {
	registry   *AgentRegistry
	spawner    *Spawner
	tool       *Tool
	rules      *Rules
	gatekeeper *Gatekeeper
}

// NewEngine creates an engine that wires all components with default configuration.
func NewEngine(registryPaths []string) *Engine {
	registry := NewAgentRegistry(registryPaths)
	spawner := NewSpawner(registry)
	tool := NewTool(spawner)

	return &Engine{
		registry:   registry,
		spawner:    spawner,
		tool:       tool,
		rules:      DefaultRules(),
		gatekeeper: NewGatekeeper(1),
	}
}

// Tool returns the orchestrator tool.
func (e *Engine) Tool() *Tool {
	return e.tool
}

// Rules returns the delegation rules.
func (e *Engine) Rules() *Rules {
	return e.rules
}

// Gatekeeper returns the gatekeeper.
func (e *Engine) Gatekeeper() *Gatekeeper {
	return e.gatekeeper
}
