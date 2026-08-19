# Design: LSP Integration

## Technical Approach

Mirror the MCP client architecture (Client + Manager + Tool wrappers) with two key differences: (1) Content-Length framing over stdio (not newline-delimited), and (2) push-based diagnostic notifications from server. The LSP subsystem lives in `internal/lsp/` as a standalone package. One gopls instance per workspace. Lazy startup — first LSP tool call triggers background initialization.

## Architecture Decisions

| Decision | Options | Tradeoff | Choice |
|----------|---------|----------|--------|
| Transport framing | Content-Length (LSP spec) vs newline-delimited | LSP spec requires Content-Length; gopls will reject newline-delimited | Content-Length framing |
| Server lifecycle | Eager (Build time) vs lazy (first call) | Eager adds 1-3s startup; lazy keeps TUI responsive but first tool call blocks | Lazy startup |
| Library | go-lsp/lspclient vs hand-rolled protocol | go-lsp/lspclient handles framing + JSON-RPC; hand-rolled is more control but more code | go-lsp/lspclient (verify availability first) |
| Diagnostic cache | Push-based (server → cache) vs pull-based (poll) | Push is natural for LSP; pull adds latency and waste | Push-based with mutex |
| Tool registration | Static (Build time) vs dynamic (after server ready) | Static means tools appear but fail if server not ready; dynamic means tools appear later | Static registration with lazy server init — tools return "not ready" until server starts |
| File sync | Hook into existing tools vs separate file watcher | Hook is simpler, no new goroutine; file watcher is complex and has OS portability issues | Hook into read_file/write_file |

## Data Flow

```
                    ┌─────────────────────────────────────────────┐
                    │              Runtime.Build                  │
                    │  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
                    │  │ Registry │  │LSPManager│  │   TUI    │  │
                    │  └────┬─────┘  └────┬─────┘  └────┬─────┘  │
                    └───────┼─────────────┼─────────────┼────────┘
                            │             │             │
           register LSP     │   lazy init │  diagnostics│
           tools at Build   │   on first  │  push via   │
                            │   tool call │  channel    │
                            ▼             ▼             ▼
                    ┌──────────┐   ┌──────────┐   ┌──────────┐
                    │ lsp_*    │   │ LSPClient│   │Diagnostic│
                    │ Tool impl│──▶│ (stdio)  │──▶│  Cache   │
                    └──────────┘   └────┬─────┘   └──────────┘
                                        │
                                        │ subprocess
                                        ▼
                                   ┌──────────┐
                                   │  gopls   │
                                   │ (stdio)  │
                                   └──────────┘
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/lsp/client.go` | Create | JSON-RPC 2.0 client with Content-Length framing, handshake, file sync |
| `internal/lsp/protocol.go` | Create | LSP protocol types (Position, Range, Diagnostic, Hover, Location, etc.) |
| `internal/lsp/cache.go` | Create | Thread-safe diagnostic cache, push-based, per-file |
| `internal/lsp/manager.go` | Create | Server lifecycle, lazy startup, tool registration |
| `internal/lsp/tool.go` | Create | Four `core.Tool` implementations (lsp_diagnostics, lsp_hover, lsp_definition, lsp_references) |
| `internal/lsp/errors.go` | Create | LSP-specific error types (LSPConnectionError, LSPToolError, ServerNotReadyError) |
| `internal/lsp/client_test.go` | Create | Unit tests for transport, handshake, framing |
| `internal/lsp/cache_test.go` | Create | Unit tests for diagnostic cache |
| `internal/lsp/tool_test.go` | Create | Unit tests for tool wrappers |
| `internal/lsp/mock_server_test.go` | Create | Mock LSP server for integration tests |
| `internal/runtime/runtime.go` | Modify | Wire LSPManager into Build/Reload/Close composition |
| `internal/adapters/tools/read_file.go` | Modify | Add optional LSP file sync hook (didOpen) |
| `internal/adapters/tools/write_file.go` | Modify | Add optional LSP file sync hook (didOpen/didChange) |
| `internal/tui/views/footer.go` | Modify | Add LSP status and diagnostic count display |
| `internal/tui/views/chat.go` | Modify | Add inline diagnostic annotations below affected lines |
| `go.mod` | Modify | Add go-lsp/lspclient dependency |

## Interfaces / Contracts

```go
// internal/lsp/protocol.go — LSP types (subset)
type Position struct {
    Line      int `json:"line"`
    Character int `json:"character"`
}
type Range struct { Start, End Position }
type Diagnostic struct {
    Range    Range  `json:"range"`
    Severity int    `json:"severity"` // 1=error,2=warn,3=info,4=hint
    Source   string `json:"source"`
    Message  string `json:"message"`
}

// internal/lsp/client.go — transport port
type Client struct { /* stdin/stdout pipes, JSON-RPC, Content-Length framing */ }
func NewClient(ctx context.Context, cfg Config) (*Client, error)
func (c *Client) Initialize(ctx context.Context, rootURI string) error
func (c *Client) Shutdown(ctx context.Context) error
func (c *Client) DidOpen(ctx context.Context, uri, langID, content string) error
func (c *Client) DidChange(ctx context.Context, uri, content string, version int) error
func (c *Client) DidClose(ctx context.Context, uri string) error
func (c *Client) Hover(ctx context.Context, uri string, pos Position) (*Hover, error)
func (c *Client) Definition(ctx context.Context, uri string, pos Position) ([]Location, error)
func (c *Client) References(ctx context.Context, uri string, pos Position) ([]Location, error)
func (c *Client) Close() error

// internal/lsp/cache.go — diagnostic storage
type DiagnosticCache struct { /* sync.RWMutex, map[string][]Diagnostic */ }
func (dc *DiagnosticCache) Update(uri string, diags []Diagnostic)
func (dc *DiagnosticCache) ByFile(uri string) []Diagnostic
func (dc *DiagnosticCache) Summary() (errors, warnings, infos, hints int)

// internal/lsp/manager.go — lifecycle + composition
type LSPManager struct { /* client, cache, state, config */ }
type ServerState int // idle, starting, running, stopped, error
func NewLSPManager(cfg Config) *LSPManager
func (m *LSPManager) Start(ctx context.Context) error  // triggers lazy startup
func (m *LSPManager) Stop(ctx context.Context) error
func (m *LSPManager) Restart(ctx context.Context) error
func (m *LSPManager) State() ServerState
func (m *LSPManager) Cache() *DiagnosticCache
func (m *LSPManager) Client() *Client
func (m *LSPManager) Tools() []core.Tool  // returns lsp_* tools

// internal/lsp/config.go
type Config struct {
    ServerPath string        // default: "gopls"
    ServerArgs []string      // default: ["-logfile=/dev/null"]
    RootURI    string        // workspace root
    Timeout    time.Duration // default: 5s
}
```

## Sequence Diagrams

**LSP Handshake (lazy startup on first tool call):**
```
Agent          LSPManager       LSPClient        gopls
  │                │                │               │
  │──lsp_hover()──▶│                │               │
  │                │──Start()──────▶│               │
  │                │                │──exec.Start()─▶
  │                │                │◀──stdin/stdout─│
  │                │                │──initialize()─▶
  │                │                │◀──capabilities─│
  │                │                │──initialized()─▶
  │                │                │               │
  │                │◀──running──────│               │
  │                │──Hover()──────▶│               │
  │                │                │──hover()─────▶
  │◀──result───────│◀──result──────│◀──hover resp──│
```

**Diagnostics push flow:**
```
gopls             LSPClient        DiagnosticCache     TUI Footer
  │                   │                  │                 │
  │──publishDiags()──▶│                  │                 │
  │                   │──Update()───────▶│                 │
  │                   │                  │──Summary()──────▶
  │                   │                  │                 │
  │                   │                  │  (render)       │
```

**Tool call flow (lsp_hover):**
```
Agent──▶LSPManager──▶LSPClient──▶gopls
  │          │           │          │
  │  check state         │          │
  │  ensure running      │          │
  │  ensure file open    │          │
  │          │           │──hover──▶│
  │◀─────────│◀──────────│◀─────────│
```

**Error handling (gopls not found):**
```
Agent──▶LSPManager──▶exec.LookPath("gopls")
  │          │                    │
  │          │◀──── not found ────│
  │◀─error───│                    │
  │  "gopls not found"           │
  │  LSP tools return error      │
```

## Data Structures

**DiagnosticCache** — push-based, per-file, thread-safe:
- `sync.RWMutex` protects the map
- Keyed by document URI (file:///abs/path)
- Updated atomically on `publishDiagnostics` notification
- TUI reads via `ByFile()` (read lock) or `Summary()` (read lock)

**ServerState** — state machine:
- `idle` → `starting` (on Start)
- `starting` → `running` (handshake complete)
- `running` → `stopped` (on Stop/Shutdown)
- `running` → `error` (on crash)
- `error` → `starting` (on auto-restart)

**WorkspaceMapping** — MVP: single workspace, one gopls instance. Future: `map[string]*LSPClient` keyed by rootUri.

## Transport Details

- **Content-Length framing**: Each message is `Content-Length: N\r\n\r\n{json}`. Read the header, parse the length, read exactly N bytes.
- **JSON-RPC 2.0**: `{ "jsonrpc": "2.0", "id": N, "method": "...", "params": {...} }`
- **Stdio pipes**: stdin (write to gopls), stdout (read from gopls). stderr is logged, not parsed.
- **Notification dispatch**: Background goroutine reads stdout, dispatches responses by ID and notifications by method (`publishDiagnostics` → cache update).

## Integration Points

**Runtime composition** (`runtime.go`):
- Build: Create `LSPManager` (lazy, not started). Register LSP tools in `full` registry. Pass manager to TUI for diagnostics reference.
- Reload: Stop old LSP, create new manager with new rootUri. Clear diagnostics cache.
- Close: Stop LSP server gracefully.

**Tool registration** (`tool.go`):
- `lsp_diagnostics`, `lsp_hover`, `lsp_definition`, `lsp_references` registered with `lsp_` prefix
- Registered at Build time even if server not ready (tools return "server not running" until lazy init completes)

**TUI footer** (`footer.go`):
- Add `SetLSPStatus(state string, diagCount string)` method
- Render: `"LSP: running | 3 errors, 2 warnings"`

**TUI inline diagnostics** (`chat.go`):
- ChatModel receives diagnostic annotations per file display
- Below each line with diagnostics, render severity indicator + message

**File sync hooks** (`read_file.go`, `write_file.go`):
- Optional `FileSyncer` interface injected at construction
- `read_file`: if file not tracked, fire `didOpen`
- `write_file`: if tracked, fire `didChange`; if not, fire `didOpen`

## Error Handling Strategy

| Scenario | Behavior |
|----------|----------|
| gopls not installed | `exec.LookPath` returns error → LSP tools return `"gopls not found"` message, tools absent from registry |
| Server crash | State → `error`, next tool call triggers auto-restart |
| Timeout (default 5s) | Configurable per Config; context deadline on Initialize |
| Invalid response | Log and return error to tool caller; server stays running |
| File not open | Auto-send `didOpen` before hover/definition/references |

## Testing Strategy

| Layer | What | Approach |
|-------|------|----------|
| Unit | Content-Length framing, JSON-RPC serialization | Table-driven tests with io.Pipe (same pattern as MCP mock_server_test.go) |
| Unit | DiagnosticCache (update, query, concurrency) | goroutine-spam test with race detector |
| Unit | LSP tool wrappers (input parsing, error paths) | Mock Client, test Execute() |
| Unit | Server state machine transitions | Table-driven state transition tests |
| Integration | Full handshake + tool call | Mock LSP server (pipe-based, same as MCP mock pattern) |
| Integration | File sync hooks (read_file → didOpen) | Mock LSP server + real ReadFile tool |
| Integration | Lazy startup (first tool call) | Mock server with handshake verification |

## Threat Matrix

This design spawns gopls as a subprocess. Applicable rows:

| Boundary | Applicability | Design response | RED tests |
|----------|--------------|-----------------|-----------|
| Documentation-like paths | N/A | No executable file classification in scope | — |
| Git repository selection | N/A | No git operations | — |
| Commit state | N/A | No commit operations | — |
| Push state | N/A | No push operations | — |
| PR commands | N/A | No PR operations | — |
| Subprocess spawn | **Applicable** | `exec.CommandContext` for gopls; command comes from config (server path) or PATH lookup; no user-controlled command injection | Test: gopls not found → graceful error; Test: invalid server path → clear error message |

**Subprocess safety**: Server path comes from `Config.ServerPath` (default "gopls"), not from user input at runtime. Config is loaded from project YAML, not from agent output. `exec.LookPath` validates before spawn.

## Migration / Rollout

No data migration required. LSP state is ephemeral per session. The gopls dependency is optional — when not installed, LSP tools gracefully degrade and are not registered.

## Open Questions

- [ ] **go-lsp/lspclient availability**: Need to verify the library exists and is maintained. If unavailable, hand-roll Content-Length framing + JSON-RPC 2.0 (the MCP client already does this with newline-delimited; swap framing only).
- [ ] **Keybinding conflict resolution**: `gd`, `gr`, `K` may conflict with existing TUI keybindings. Need to audit current keymap.
- [ ] **Diagnostic rendering limits**: When a file has 50+ diagnostics, inline rendering may overflow the viewport. Need a max-annotations-per-file cap (suggest: 10, with "+N more" indicator).
- [ ] **gopls memory footprint**: gopls uses ~100-300MB per instance. For MVP with single workspace, acceptable. Multi-workspace could be a concern.
