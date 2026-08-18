# Design: Dynamic Extensions

## Technical Approach

Reuse the MCP subprocess manager pattern to implement runtime extension discovery and loading. Each dynamic extension is an executable with an `extension.yaml` manifest, spawned as a subprocess, connected via JSON-RPC 2.0 over stdio, and registered through the existing `core.Extension` interface. Discovery merges global + project + config-path sources. The `internal/extensions/dynamic/` package owns the manager, protocol client, config loader, and manifest parser. The existing `extensions` registry gains `RegisterDynamic()` for runtime registration alongside compiled-in `init()` registration.

## Architecture Decisions

| Decision | Option | Tradeoff | Decision |
|----------|--------|----------|----------|
| Package location | New `internal/extensions/dynamic/` vs. extend `internal/mcp/` | Separate package keeps MCP protocol clean; extension protocol diverges (hooks/commands in Phase 2) | New package |
| Config format | Separate `extensions.yaml` vs. extend `mcp.yaml` | Separate file mirrors MCP pattern, avoids config collision, cleaner mental model | Separate `extensions.yaml` |
| Extension adapter | Wrap subprocess as `core.Extension` vs. direct `core.Tool` registration | Wrapping as Extension keeps lifecycle consistent (Init/Shutdown), enables future hook/command support | Wrap as Extension |
| Crash strategy | Log+mark unavailable vs. auto-restart | Auto-restart adds complexity; user-triggered `/reload` is simpler and predictable for Phase 1 | Log+mark unavailable |
| Protocol version | Separate version string vs. reuse MCP `2025-03-26` | Extensions may diverge from MCP protocol; independent version allows independent evolution | Separate version `kui-ext/1` |

## Data Flow

```
Discovery Sources               Lifecycle                    Registry
┌─────────────────┐            ┌──────────────┐             ┌──────────┐
│ ~/.config/kui/  │            │  Spawn       │             │ compiled │
│   extensions/   │──┐         │  subprocess  │             │ (init()) │
├─────────────────┤  │         │  ↓           │             └────┬─────┘
│ .kui/extensions/│──┤  scan   │  Initialize  │                  │
├─────────────────┤  ├───────→ │  (handshake) │    Register      │
│ config paths    │──┘         │  ↓           │    Dynamic       │
│ (extensions.yaml)│            │  ListTools   │────────────────→ │ combined │
└─────────────────┘            │  ↓           │                  │ registry │
                               │  Register    │                  │          │
                               │  tools       │                  └──────────┘
                               │  ↓           │
                               │  Active      │
                               └──────┬───────┘
                                      │ crash
                               ┌──────↓───────┐
                               │ Log + mark   │
                               │ unavailable  │
                               │ User /reload │
                               └──────────────┘
```

## Protocol Design

JSON-RPC 2.0 over stdio, lines delimited by `\n`. Reuse MCP's `jsonrpcRequest`/`jsonrpcResponse` framing from `internal/mcp/client.go`.

**Handshake**:
```json
→ {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"kui-ext/1","capabilities":{},"clientInfo":{"name":"kui","version":"0.1.0"}}}
← {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"kui-ext/1","capabilities":{"tools":true}}}
→ {"jsonrpc":"2.0","method":"notifications/initialized"}
```

**Tool discovery**: `extensions/list` returns `{"tools":[{"name":"read","description":"...","inputSchema":{...}}]}`. Extension name prefixes tool names as `{extensionName}_{toolName}`.

**Tool invocation**: `extensions/call` with `{"name":"read","arguments":{...}}`. Returns `{"content":[{"type":"text","text":"..."}]}`.

**Shutdown**: `extensions/shutdown` notification, wait up to 5s, then kill.

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/extensions/dynamic/manager.go` | Create | Extension lifecycle manager: discover, spawn, connect, register, shutdown |
| `internal/extensions/dynamic/client.go` | Create | JSON-RPC 2.0 client over stdio (adapts MCP client pattern) |
| `internal/extensions/dynamic/config.go` | Create | Load `extensions.yaml` from global + project paths, merge, validate |
| `internal/extensions/dynamic/manifest.go` | Create | Parse `extension.yaml` manifests (name, version, protocol_version, entry_point) |
| `internal/extensions/dynamic/extension.go` | Create | `DynamicExtension` adapter: wraps subprocess client as `core.Extension` |
| `internal/extensions/dynamic/tool.go` | Create | `DynamicTool`: wraps subprocess tool calls as `core.Tool` (prefix pattern) |
| `internal/extensions/dynamic/errors.go` | Create | Error types: ManifestError, ProtocolError, SpawnError |
| `internal/adapters/extensions/registry.go` | Modify | Add `RegisterDynamic()` and `dynamicList` slice; `LoadAll` processes both |
| `internal/core/extension.go` | No change | `Extension` and `ExtensionAPI` interfaces unchanged |
| `internal/runtime/runtime.go` | Modify | Add dynamic extension discovery + load step after MCP in `buildComponents` |
| `openspec/changes/dynamic-extensions/design.md` | Create | This document |

## Interfaces / Contracts

```go
// internal/extensions/dynamic/manifest.go
type Manifest struct {
    Name            string `yaml:"name"`
    Version         string `yaml:"version"`
    ProtocolVersion string `yaml:"protocol_version"`
    EntryPoint      string `yaml:"entry_point"`
}

// internal/extensions/dynamic/extension.go
type DynamicExtension struct {
    name   string
    client *Client
    tools  []core.Tool
}

func (d *DynamicExtension) Name() string
func (d *DynamicExtension) Init(api core.ExtensionAPI) error  // spawn, handshake, list tools, register
func (d *DynamicExtension) Shutdown() error                   // send shutdown, wait, kill

// internal/adapters/extensions/registry.go — new additions
func RegisterDynamic(ext core.Extension)  // appends to dynamicList
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Manifest parsing, config merge, name collision resolution | Table-driven tests with temp directories |
| Unit | Client JSON-RPC framing (sendRequest, sendNotification) | Mock stdio pipes, assert serialized bytes |
| Unit | DynamicExtension.Init (handshake + tool registration) | Mock client factory, verify ExtensionAPI calls |
| Unit | Crash handling (subprocess exit → tools marked unavailable) | Kill subprocess mid-test, assert error on Execute |
| Integration | Full lifecycle: discover → spawn → register → call tool → shutdown | Spawn a test binary (Go test binary as subprocess) |
| Integration | Reload cycle: old extensions shut down, new ones loaded | Modify extension dir between Reload calls |
| E2E | Extension in `~/.config/kui/extensions/` loaded at Build time | Test binary with known tool output |

## Threat Matrix

N/A — no routing, shell commands (beyond subprocess spawn which is controlled), VCS/PR automation, executable-file classification, or process-integration boundary beyond the subprocess pattern already established by MCP.

## Migration / Rollout

No data migration required. Extension configs are additive. Compiled-in extensions continue working unchanged. `RegisterDynamic()` is a new function — no existing call sites change.

Phase 1 ships with tools-only. The `ExtensionAPI.RegisterHook()` and `RegisterCommand()` methods exist but are not called by the subprocess protocol until Phase 2.

## Open Questions

- [ ] Should `extensions.yaml` support an `enabled` field per extension (like MCP's `disabled`), or is removal from directory sufficient?
- [ ] Extension sandboxing: trust model for Phase 1 — any filesystem/exec concern beyond MCP precedent?
