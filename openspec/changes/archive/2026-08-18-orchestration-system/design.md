# Design: Orchestration System

## Architecture Overview

The orchestration system builds on kui's existing hexagonal architecture. It adds orchestration-level abstractions on top of existing ports without modifying the core domain.

```
┌─────────────────────────────────────────────────────────┐
│  User Profile (YAML)                                   │
│  ├── agents: {explore, worker, reviewer, ...}          │
│  ├── orchestration: {delegation, gatekeeper, dedup}    │
│  └── models: {balanced, deep-reasoning, fast}          │
└────────────────────────┬────────────────────────────────┘
                         │ loads
┌────────────────────────▼────────────────────────────────┐
│  Orchestration Engine                                   │
│  ├── AgentRegistry (load from profile)                  │
│  ├── OrchestratorTool (fan-out/fan-in, chain, route)   │
│  ├── InProcessSpawner (wraps core.Agent)                │
│  ├── ResultAggregator (structured results)              │
│  ├── DelegationRouter (rules → agent selection)         │
│  ├── Gatekeeper (auto mode validation)                  │
│  └── LaunchDedup (session-scoped)                       │
└────────────────────────┬────────────────────────────────┘
                         │ uses
┌────────────────────────▼────────────────────────────────┐
│  Existing Ports (unchanged)                             │
│  ├── core.Agent (agent loop)                            │
│  ├── core.Tool (tool interface)                         │
│  ├── core.Provider (LLM provider)                       │
│  └── core.HookRegistry (lifecycle hooks)                │
└─────────────────────────────────────────────────────────┘
```

## Component Design

### 1. Enhanced Agent Definition Schema

**Location**: `internal/orchestration/agentdef.go`

Extend existing `subagent.AgentDef` with orchestration metadata:

```go
// AgentDef defines an agent's capabilities and orchestration metadata.
type AgentDef struct {
    // Identity
    Name        string `yaml:"name"`
    Description string `yaml:"description"`

    // Tools
    ToolsInclude []string `yaml:"tools.include"`  // allowed tools (empty = all)
    ToolsExclude []string `yaml:"tools.exclude"`  // excluded tools

    // Model
    Model    string `yaml:"model"`    // model override (balanced/deep-reasoning/fast)
    Thinking string `yaml:"thinking"` // thinking level (off/low/medium/high)
    Provider string `yaml:"provider"` // provider override (openai/anthropic)

    // Orchestration
    MaxIterations int      `yaml:"max_iterations"` // max tool calls
    Permissions   []Rule   `yaml:"permissions"`     // file access rules

    // Prompt
    SystemPrompt string `yaml:"-"` // body from Markdown
}
```

**Resolution chain**: `.kui/agents/<name>.md` → `~/.config/kui/agents/<name>.md` → built-in agents

### 2. In-Process Agent Spawner

**Location**: `internal/orchestration/spawner.go`

```go
// Spawner creates isolated agent instances in-process.
type Spawner struct {
    provider core.Provider
    registry *core.ToolRegistry
    profiles *agent.Manager
}

// SpawnRequest defines what to spawn.
type SpawnRequest struct {
    AgentName string          // agent definition name
    Task      string          // task description
    Context   string          // additional context (optional)
    Model     string          // model override (optional)
    Thinking  string          // thinking override (optional)
}

// SpawnResult contains the agent's output.
type SpawnResult struct {
    Output     string           // final text output
    Messages   []core.Message   // conversation history
    Tokens     TokenUsage       // token accounting
    Duration   time.Duration    // execution time
    Error      error            // if failed
}

// Spawn creates an isolated agent and runs it.
func (s *Spawner) Spawn(ctx context.Context, req SpawnRequest) (*SpawnResult, error) {
    // 1. Load agent definition
    // 2. Filter tools based on agent's include/exclude lists
    // 3. Create isolated core.Agent with filtered tools
    // 4. Run agent with task
    // 5. Return structured result
}
```

**Key design decisions**:
- Each spawned agent gets its own `core.Agent` instance (no shared state)
- Tool filtering happens at spawn time (not runtime)
- Agent inherits parent's provider but can override model/thinking
- Panic recovery wraps each spawn

### 3. Orchestrator Tool

**Location**: `internal/orchestration/tool.go`

```go
// Tool is the orchestration tool exposed to the parent agent.
type Tool struct {
    spawner    *Spawner
    aggregator *ResultAggregator
    dedup      *LaunchDedup
}

// Schema returns JSON schema for the tool.
func (t *Tool) Schema() string {
    return `{
        "name": "orchestrate",
        "description": "Spawn specialized agents for focused tasks",
        "parameters": {
            "operation": "fan-out|fan-in|chain|spawn",
            "agents": [{"name": "explore", "task": "..."}],
            "aggregate": "merge|summary|select"
        }
    }`
}

// Execute runs the orchestration operation.
func (t *Tool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
    // Parse operation type
    // Dedup check
    // Spawn agents (parallel for fan-out, sequential for chain)
    // Aggregate results
    // Return structured output
}
```

**Operations**:

| Operation | Behavior | Use Case |
|-----------|----------|----------|
| `spawn` | Single agent, wait for result | Focused task |
| `fan-out` | Parallel agents, wait all | Multi-perspective analysis |
| `fan-in` | Aggregate results from fan-out | Result synthesis |
| `chain` | Sequential agents, output feeds next | Pipeline work |

### 4. Result Aggregator

**Location**: `internal/orchestration/aggregator.go`

```go
// ResultAggregator combines results from multiple agents.
type ResultAggregator struct{}

// Aggregate combines multiple spawn results.
func (r *ResultAggregator) Aggregate(results []*SpawnResult, mode string) string {
    switch mode {
    case "merge":
        // Concatenate all outputs with separators
    case "summary":
        // Summarize combined outputs
    case "select":
        // Select best result (by length, completeness, etc.)
    default:
        // Default: merge
    }
}
```

### 5. Delegation Rules

**Location**: `internal/orchestration/rules.go`

```go
// Rules define when to delegate vs execute inline.
type Rules struct {
    ExploreThreshold int  `yaml:"explore_threshold"` // files to read before delegating
    WriteThreshold   int  `yaml:"write_threshold"`   // files to write before delegating
    ContextRule      bool `yaml:"context_rule"`      // delegate reading that prepares a write
}

// ShouldDelegate decides if work should be delegated.
func (r *Rules) ShouldDelegate(action string, fileCount int) bool {
    switch action {
    case "explore":
        return fileCount >= r.ExploreThreshold
    case "write":
        return fileCount >= r.WriteThreshold
    case "context":
        return r.ContextRule
    default:
        return false
    }
}
```

### 6. Gatekeeper

**Location**: `internal/orchestration/gatekeeper.go`

```go
// Gatekeeper validates agent results in auto mode.
type Gatekeeper struct {
    maxRetries int
}

// Validate checks if a result meets quality gates.
func (g *Gatekeeper) Validate(result *SpawnResult) error {
    // 1. Check output is not empty
    // 2. Check no panics/errors
    // 3. Check token budget not exceeded
    // 4. Check result matches expected structure
    return nil
}

// ShouldRetry decides if a failed result should be retried.
func (g *Gatekeeper) ShouldRetry(result *SpawnResult, attempt int) bool {
    return result.Error != nil && attempt < g.maxRetries
}
```

### 7. Launch Dedup

**Location**: `internal/orchestration/dedup.go`

```go
// LaunchDedup prevents duplicate agent spawns.
type LaunchDedup struct {
    mu      sync.Mutex
    seen    map[string]bool // task fingerprint → spawned
}

// IsDuplicate checks if this task was already spawned.
func (d *LaunchDedup) IsDuplicate(agentName, task string) bool {
    d.mu.Lock()
    defer d.mu.Unlock()
    key := agentName + ":" + hash(task)
    if d.seen[key] {
        return true
    }
    d.seen[key] = true
    return false
}

// Reset clears the dedup cache (per session).
func (d *LaunchDedup) Reset() {
    d.mu.Lock()
    defer d.mu.Unlock()
    d.seen = make(map[string]bool)
}
```

## File Structure

```
internal/orchestration/
├── agentdef.go          # Agent definition schema + loading
├── agentdef_test.go
├── spawner.go           # In-process agent spawner
├── spawner_test.go
├── tool.go              # Orchestration tool
├── tool_test.go
├── aggregator.go        # Result aggregation
├── aggregator_test.go
├── rules.go             # Delegation rules
├── rules_test.go
├── gatekeeper.go        # Quality gates
├── gatekeeper_test.go
├── dedup.go             # Launch deduplication
├── dedup_test.go
└── engine.go            # Wires everything together
```

## Integration Points

### With Existing Systems

| Existing Component | Integration |
|-------------------|-------------|
| `core.Agent` | Spawner creates isolated instances |
| `core.ToolRegistry` | Spawner filters tools per agent |
| `agent.Manager` | Engine loads profiles |
| `subagent.AgentDef` | Extended with orchestration metadata |
| `core.HookRegistry` | Engine registers orchestration hooks |
| `mcp.MCPManager` | MCP tools available to spawned agents |

### Profile Integration

```yaml
# Profile YAML additions
orchestration:
  delegation:
    explore_threshold: 4
    write_threshold: 2
    context_rule: true
  gatekeeper:
    enabled: true
    max_retries: 1
  dedup:
    enabled: true
```

## Error Handling

| Error Type | Handling |
|------------|----------|
| Agent spawn panic | Recover, log, return error result |
| Provider timeout | Cancel context, return partial result |
| Tool execution error | Include in SpawnResult.Error |
| Dedup conflict | Skip spawn, return cached result |
| Gatekeeper failure | Retry if within budget, else escalate |

## Testing Strategy

### Unit Tests
- Agent definition parsing and validation
- Tool filtering logic
- Result aggregation modes
- Delegation rule evaluation
- Dedup fingerprinting
- Gatekeeper validation

### Integration Tests
- Spawn isolated agent with filtered tools
- Fan-out with parallel agents
- Chain with sequential agents
- Full orchestration cycle

### Edge Cases
- Empty agent definition
- All tools excluded
- Provider failure mid-spawn
- Concurrent spawn limits
- Recursive spawn prevention

## Migration Path

1. **Phase 1**: Add `internal/orchestration/` package with agent definition extensions
2. **Phase 2**: Add in-process spawner (alternative to process-based)
3. **Phase 3**: Add orchestration tool
4. **Phase 4**: Add delegation rules
5. **Phase 5**: Add gatekeeper and dedup

Each phase is independently testable and deployable. No breaking changes to existing profiles.
