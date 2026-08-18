# Proposal: Dynamic Extensions

## Intent

kui's extension system is compiled-in only — extensions register via Go `init()` and the set is fixed at compile time. Users cannot add extensions without rebuilding the binary. This change adds runtime extension discovery and loading via MCP-style subprocess extensions, giving users the ability to extend kui with tools, hooks, and commands without recompilation.

## Scope

### In Scope
- Extension manifest format (`extension.yaml`) with name, version, capabilities, entry point
- Filesystem discovery from `~/.config/kui/extensions/` (global) and `.kui/extensions/` (project-level, project wins on collision)
- Subprocess extension protocol: MCP-like JSON-RPC 2.0 over stdio, extended with hook/command registration
- Extension lifecycle: discover → handshake → register tools → active → graceful shutdown
- Integration with `runtime.Build`/`Reload` — dynamic extensions load/unload alongside compiled-in ones
- Crash handling: mark tools unavailable, user `/reload`s to recover
- Protocol version negotiation at handshake; reject incompatible extensions

### Out of Scope
- CLI commands for extension management (`kui extension install` etc.)
- Permission/sandboxing model (trust the installer in Phase 1)
- Auto-restart on crash
- Hook and command registration via subprocess protocol (Phase 2)
- Extension signing or marketplace/registry

## Capabilities

### New Capabilities
- `dynamic-extension-protocol`: Subprocess JSON-RPC protocol for extension handshake, tool registration, and lifecycle management
- `dynamic-extension-discovery`: Filesystem scanning of extension directories, manifest parsing, and compatibility validation
- `dynamic-extension-config`: Global + project-level extension configuration (mirrors MCP config pattern)

### Modified Capabilities
- `extension-system`: Add runtime extension registration/unregistration alongside compiled-in `init()` registration; `ExtensionAPI` gains unregister support for dynamic unload
- `extension-discovery`: Extend startup discovery to include filesystem-scanned extensions alongside compiled-in registry

## Approach

Reuse the MCP subprocess manager pattern from `internal/mcp/`. Each extension is an executable in a directory with an `extension.yaml` manifest. kui spawns extensions as subprocesses, connects via JSON-RPC 2.0 over stdio, performs a handshake with protocol version, then calls `extensions/list` to discover tools. Extensions register tools using the same `core.Tool` interface. The `runtime.Build`/`Reload` lifecycle is extended to discover + load dynamic extensions after compiled-in ones.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/adapters/extensions/registry.go` | Modified | Add dynamic extension registration/unregistration |
| `internal/runtime/runtime.go` | Modified | Integrate dynamic extension load/unload into Build/Reload |
| `internal/mcp/manager.go` | Reference | Patterns reused for subprocess lifecycle |
| `openspec/specs/extension-system/spec.md` | Modified | New requirements for runtime registration |
| `openspec/specs/extension-discovery/spec.md` | Modified | New requirements for filesystem discovery |
| New: `internal/extensions/dynamic/` | New | Extension manager, protocol client, config loader |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Subprocess lifecycle complexity | High | Reuse MCP manager patterns; test with failing extensions |
| Protocol design scope creep | Med | Tools-only in Phase 1, hooks/commands deferred |
| Extension version incompatibility | Med | Protocol version negotiation; reject at handshake |
| Reload during active extension call | Med | Cancel-and-wait from `runtime.Reload()` |

## Rollback Plan

Revert the dynamic extension loading path. Compiled-in extensions continue working via `init()` registration as before. Remove `internal/extensions/dynamic/` package and revert changes to `registry.go` and `runtime.go`. No data migration needed — extension configs are additive.

## Dependencies

- Existing MCP subprocess manager patterns (`internal/mcp/`)
- `runtime.Reload()` lifecycle (already handles teardown/rebuild)

## Success Criteria

- [ ] Extensions in `~/.config/kui/extensions/` are discovered and loaded at startup
- [ ] Project-level `.kui/extensions/` overrides global on name collision
- [ ] Extension subprocess crash does not bring down kui
- [ ] `/reload` loads newly added extensions and unloads removed ones
- [ ] Compiled-in extensions continue working alongside dynamic ones
- [ ] Protocol version mismatch is rejected with clear error
