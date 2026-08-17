# Design: MCP Client Support

## Technical Approach

Add a hexagonal MCP client layer: `internal/mcp/` contains config loading, JSON-RPC 2.0 stdio transport, server lifecycle management, and a `MCPTool` adapter bridging MCP tools to `core.Tool`. The manager spawns subprocesses per configured server, discovers tools via `tools/list`, and registers them in `core.Registry` alongside built-in tools. Zero external dependencies — stdlib-only using `os/exec`, `bufio.Scanner`, `encoding/json`.

## Architecture Decisions

### Decision: Config Format

| Option | Tradeoff | Decision |
|--------|----------|----------|
| YAML with `servers` map | Human-readable, consistent with kui's profile.yaml | **Chosen** |
| JSON | Simpler parser, less readable for users | Rejected |
| TOML | Good for config, less ecosystem in Go stdlib | Rejected |

Global (`~/.config/kui/mcp.yaml`) + project (`.kui/mcp.yaml`) layers. Project overrides global entirely (REQ-MCP-4). Each server: `type: local`, `command` (string array), `cwd`, `environment`, `disabled`, `timeout` (default 30s).

### Decision: JSON-RPC 2.0 over Stdio

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Stdio (line-delimited JSON) | Simple, covers 90% of MCP servers | **Chosen** |
| HTTP/SSE | Supports remote servers, more complex | Deferred |
| gRPC | Efficient, heavy dependency | Rejected |

Stdio transport: `os/exec` spawns subprocess, `bufio.Scanner` reads newline-delimited JSON from stdout, writes to stdin. Request/response ID matching via `sync.Mutex` + map. Per-request timeout from config (default 30s).

### Decision: Tool Name Prefixing

| Option | Tradeoff | Decision |
|--------|----------|----------|
| `{server}_{tool}` prefix | Prevents all collisions, clear provenance | **Chosen** |
| Namespace via Registry | Cleaner names, more complex registry | Rejected |
| No prefix | Simple, collisions likely | Rejected |

MCP tool `create_issue` from server `github` becomes `github_create_issue`. Built-in tools keep original names. Registry rejects duplicates via existing `DuplicateToolError`.

### Decision: Manager Lifecycle

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Concurrent ConnectAll | Fast startup for multiple servers | **Chosen** |
| Sequential connect | Simpler error handling | Rejected |
| Lazy connect | Faster first-run, latency on first tool call | Rejected |

`ConnectAll()` starts all enabled servers concurrently. Each server failure is logged and non-fatal (REQ-MCP-15). `Shutdown()` kills all subprocesses idempotently.

## Data Flow

```
mcp.yaml ──→ Config.Load() ──→ []ServerConfig
                                    │
                         MCPManager.ConnectAll()
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
              Client(conn1)   Client(conn2)   Client(conn3)
              (stdio exec)    (stdio exec)    (stdio exec)
                    │               │               │
              tools/list      tools/list      tools/list
                    │               │               │
                    ▼               ▼               ▼
              MCPTool[]        MCPTool[]        MCPTool[]
                    │               │               │
                    └───────┬───────┘───────────────┘
                            ▼
                    core.Registry.Register()
                            │
                            ▼
                    Agent Loop (tool dispatch)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/mcp/config.go` | Create | YAML config loading, global+project merge |
| `internal/mcp/config_test.go` | Create | Config loading tests |
| `internal/mcp/client.go` | Create | JSON-RPC 2.0 over stdio transport |
| `internal/mcp/client_test.go` | Create | Client handshake, tool call, crash tests |
| `internal/mcp/manager.go` | Create | Server lifecycle, tool cache, ConnectAll/Shutdown |
| `internal/mcp/manager_test.go` | Create | Manager lifecycle tests |
| `internal/mcp/tool.go` | Create | MCPTool implements core.Tool |
| `internal/mcp/tool_test.go` | Create | MCPTool adapter tests |
| `internal/mcp/errors.go` | Create | MCPConnectionError, MCPToolError types |
| `internal/adapters/tools/registry.go` | Modify | Accept MCP tools in Default() or new MCP-aware constructor |
| `cmd/kui/main.go` | Modify | Initialize MCP manager, register MCP tools, shutdown on exit |

## Interfaces / Contracts

```go
// internal/mcp/config.go
type ServerConfig struct {
    Type        string            `yaml:"type"`
    Command     []string          `yaml:"command"`
    CWD         string            `yaml:"cwd,omitempty"`
    Environment map[string]string `yaml:"environment,omitempty"`
    Disabled    bool              `yaml:"disabled,omitempty"`
    Timeout     time.Duration     `yaml:"timeout,omitempty"`
}

type Config struct {
    Servers map[string]ServerConfig `yaml:"servers"`
}

// internal/mcp/client.go
type Client struct { /* stdio transport, JSON-RPC framing */ }
func NewClient(command []string, opts ...ClientOption) (*Client, error)
func (c *Client) Initialize(ctx context.Context) error
func (c *Client) ListTools(ctx context.Context) ([]MCPToolDef, error)
func (c *Client) CallTool(ctx context.Context, name string, args json.RawMessage) (string, error)
func (c *Client) Close() error

// internal/mcp/manager.go
type Manager struct { /* server map, tool cache */ }
func NewManager(cfg Config) *Manager
func (m *Manager) ConnectAll(ctx context.Context) error
func (m *Manager) Shutdown()
func (m *Manager) Tools() []core.Tool

// internal/mcp/tool.go
type MCPTool struct { /* name, description, schema, client ref */ }
func (t *MCPTool) Name() string
func (t *MCPTool) Description() string
func (t *MCPTool) Schema() string
func (t *MCPTool) Execute(ctx context.Context, args json.RawMessage) (string, error)
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Config parsing, merge semantics, JSON-RPC framing | Table-driven tests with `go test` |
| Unit | MCPTool adapter (Name, Description, Schema, Execute) | Mock client, verify interface compliance |
| Unit | Manager lifecycle (ConnectAll partial failure, Shutdown idempotent) | Mock subprocess, verify non-fatal behavior |
| Integration | End-to-end with `echo` MCP server (minimal) | Real subprocess, verify handshake+tool call |
| Guard | `TestCoreImportsStdlibOnly` still passes | No external deps in `internal/mcp/` |

## Threat Matrix

N/A — no routing, VCS/PR automation, or executable-file classification boundaries. Subprocess spawning (`os/exec`) is confined to `internal/mcp/client.go` with config-driven commands only — no user-supplied arbitrary execution beyond what `mcp.yaml` declares.

## Migration / Rollout

No data migration required. MCP is stateless runtime config. Rollout: MCP tools are opt-in via `mcp.yaml`. If no config exists, kui behaves identically to current state. Existing tests pass unmodified.

## Open Questions

- [ ] Should `mcp.yaml` support `env_file` for loading environment from `.env` files (common in MCP servers)?
- [ ] Should the manager expose a `Reconnect(serverName)` method for transient failures?
