# Exploration: Dynamic Extensions

## Current State

kui's extension system is **compiled-in only**. Extensions register via Go `init()` self-registration into a package-level slice in `internal/adapters/extensions/registry.go`. At startup, `LoadAll(api)` initializes every registered extension in registration order; `ShutdownAll()` tears them down in reverse.

**What exists today:**
- `core.Extension` interface: `Name()`, `Init(ExtensionAPI) error`, `Shutdown() error`
- `core.ExtensionAPI` interface: `RegisterTool`, `RegisterHook`, `RegisterCommand`
- `core.HookRegistry`: event→handler map with registration-order execution and error short-circuit
- `core.HookContext`: mutable context for message mutation, tool blocking
- `adapters/extensions/registry.go`: global `Register()`, `LoadAll()`, `ShutdownAll()` — package-level slices
- `internal/runtime/runtime.go`: `Build`/`Reload`/`Close` lifecycle — already calls `LoadAll` during build and `ShutdownAll` during teardown
- 3 lifecycle hooks wired into `core.Agent.Run`: `before_provider_request`, `before_tool_execution`, `after_tool_execution`
- Example extension: `internal/extensions/example/example.go` — registers a tool + hook via `init()`

**What the reload change already provides:**
- `/reload` in TUI tears down extensions, rebuilds the full registry, and re-runs `LoadAll`
- This means compiled-in extensions already reinitialize on `/reload` — but the *set* of extensions is fixed at compile time

**What "dynamic" means here — the gap:**
The archived `2026-08-17-extensions` proposal explicitly scoped out:
> "Runtime/dynamic extension loading (WASM, plugin DLLs), Hot-reload or extension hot-swap"

Dynamic extensions would mean: **discover and load new extensions from the filesystem at runtime, without recompiling the binary**. This is parity with how Pi handles extensions.

## Affected Areas

| File | Impact | Why |
|------|--------|-----|
| `internal/adapters/extensions/registry.go` | **Core change** | Currently holds `global []core.Extension` — a compile-time slice. Needs runtime discovery + loading. |
| `internal/core/extension.go` | **Possibly extended** | `Extension` interface may need a manifest/metadata method for runtime validation (version, dependencies, capabilities). |
| `internal/runtime/runtime.go` | **Modified** | `Build` and `Reload` need to discover + load dynamic extensions alongside compiled-in ones. |
| `cmd/kui/main.go` | **Modified** | Extension discovery path needs configuration (where to look for extensions). |
| `internal/runtime/extapi.go` | **Possibly extended** | `extAPI` may need unregistration support for dynamic unload. |
| `openspec/specs/extension-system/spec.md` | **Modified** | New requirements for runtime discovery, loading, unloading, manifest format. |
| `openspec/specs/extension-discovery/spec.md` | **Modified** | New requirements for filesystem discovery, validation, dependency resolution. |

## Approaches

### 1. Go `plugin` Package (`plugin.Open` / `plugin.Lookup`)

Load `.so` (Linux) / `.dylib` (macOS) / `.dll` (Windows) files from an extensions directory. Each extension is a Go shared library exporting a registration function.

| Aspect | Detail |
|--------|--------|
| **Discovery** | Scan `~/.config/kui/extensions/` for `*.so`/`*.dylib`/`*.dll` |
| **Loading** | `plugin.Open(path)` → `plugin.Lookup("Register")` → call exported function |
| **Manifest** | Plugin must export a `func Register(api core.ExtensionAPI) error` symbol |
| **Unloading** | **Not possible** — Go plugins cannot be unloaded from memory |
| **Versioning** | Binary compatibility: plugin must be compiled with identical Go version + module versions |
| **Dependencies** | No dependency resolution — all deps must be statically linked into the `.so` |
| **Pros** | Native Go, same process, minimal overhead, simple API |
| **Cons** | Platform-specific, cannot unload, strict version coupling, no cross-language support, `.so`/`.dll` files are large |
| **Effort** | Medium — discovery + loading is straightforward, but version coupling and unload limitation are fundamental |

### 2. Out-of-Process Extensions via JSON-RPC over stdio

Extensions are separate executables (any language) that communicate with kui over stdin/stdout using JSON-RPC 2.0. kui spawns each extension as a subprocess.

| Aspect | Detail |
|--------|--------|
| **Discovery** | Scan `~/.config/kui/extensions/` for executables (or directories with manifest + binary) |
| **Loading** | `exec.Command(path).Start()` → pipe stdin/stdout → JSON-RPC handshake |
| **Manifest** | `extension.yaml` in extension directory: name, version, capabilities, entry point |
| **Unloading** | `process.Signal(SIGTERM)` → wait → `process.Wait()` — clean shutdown |
| **Versioning** | Protocol version in manifest; kui negotiates at handshake |
| **Dependencies** | Extension declares deps in manifest; kui resolves or rejects at load time |
| **Language** | Any language that can do JSON-RPC over stdio (Go, Python, Rust, Node.js, etc.) |
| **Pros** | Full isolation, cross-language, clean unload, crash doesn't bring down kui, matches MCP subprocess model |
| **Cons** | Serialization overhead, subprocess management complexity, debugging harder, no shared memory |
| **Effort** | High — full protocol design, subprocess lifecycle, error handling |

### 3. Hybrid: Filesystem Discovery + Compiled-In Loading (Phased)

Phase 1: Filesystem discovery with compiled-in extension registry (discover manifest files, validate, load via existing `init()` path). Phase 2: Out-of-process extensions as a later change.

| Aspect | Detail |
|--------|--------|
| **Phase 1** | Discover `extension.yaml` manifests in `~/.config/kui/extensions/`, validate compatibility, then load via compiled-in `init()` (requires extensions to be imported at build time) |
| **Phase 2** | Add out-of-process extension support for truly runtime-loaded extensions |
| **Pros** | Incremental, validates manifest format early, lower risk, builds on existing infrastructure |
| **Cons** | Phase 1 doesn't actually achieve runtime loading (still compiled-in), Phase 2 is the real work |
| **Effort** | Low (Phase 1) / High (Phase 2) |

### 4. MCP-Style Subprocess Extensions (Aligned with kui's Existing MCP Model)

kui already has an MCP subsystem that manages subprocess servers (`internal/mcp/`). Extensions could be modeled identically: each extension is an MCP-like server spawned as a subprocess, communicating via a defined protocol.

| Aspect | Detail |
|--------|--------|
| **Discovery** | `extensions.yaml` (like `mcp.yaml`) lists extension executables with args/env |
| **Loading** | `exec.Command` → stdio pipe → protocol handshake → capabilities reported |
| **Unloading** | Graceful shutdown via protocol → `process.Wait()` |
| **Reuse** | Shares patterns with `internal/mcp/manager.go` — config loading, subprocess management, tool bridging |
| **Pros** | Consistent with existing MCP architecture, proven subprocess model, kui's team already knows this pattern |
| **Cons** | MCP protocol is specific to tool calling — extensions may need hooks/commands too, not just tools |
| **Effort** | Medium-High — extend MCP pattern to cover hooks/commands, not just tools |

## Recommendation

**Approach 4 (MCP-Style Subprocess Extensions) is the strongest fit** for kui's architecture:

1. **Architectural consistency**: kui already manages MCP subprocess servers with config loading, connection management, and tool bridging. Extensions follow the same pattern — discovery from config, subprocess lifecycle, capability registration.

2. **Existing reload infrastructure**: The `runtime.Reload()` flow already does teardown → rebuild → swap. Adding dynamic extension loading/unloading fits naturally into this cycle.

3. **Isolation**: Subprocess extensions can't crash kui. MCP already proves this pattern works.

4. **Cross-language**: Extensions can be written in any language, matching Pi's ecosystem.

5. **Clean unload**: Unlike Go plugins, subprocess extensions can be terminated cleanly.

**The key difference from MCP**: Extensions need to register hooks and commands, not just tools. The protocol would need to be extended beyond MCP's tool-focused model.

**Phased rollout**:
- **Phase 1**: Manifest-based discovery + validation of extension directories (filesystem scanning, schema validation, compatibility checks)
- **Phase 2**: Subprocess extension loading using an MCP-like protocol extended with hook/command registration
- **Phase 3**: Hot-swap — unload old version, load new version during `/reload`

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Subprocess lifecycle management complexity | High | Reuse MCP manager patterns; test with failing extensions |
| Protocol design scope creep | Med | Start with tools-only, add hooks/commands incrementally |
| Extension version compatibility | Med | Protocol version negotiation at handshake; reject incompatible extensions |
| Performance overhead of subprocess calls | Low-Med | Extensions are loaded once, tool calls are already cross-process in MCP |
| Reload during active extension call | Med | Cancel-and-wait pattern from `runtime.Reload()` applies |
| Security: untrusted extensions | High | Extension signing/verification, sandboxing, permission model |
| Debugging subprocess extensions | Med | Structured logging, extension status in TUI, `--verbose` output |

## Ready for Proposal

**Yes** — the codebase has a clear gap (compiled-in only extensions, no runtime discovery), the existing reload infrastructure provides a natural integration point, and the MCP subprocess model provides a proven pattern to extend. The orchestrator should proceed to `sdd-propose` with a focus on:
1. Extension manifest format (`extension.yaml`)
2. Filesystem discovery in `~/.config/kui/extensions/`
3. Subprocess extension protocol (MCP-like, extended with hooks/commands)
4. Integration with `runtime.Build`/`Reload` lifecycle
5. Extension versioning and compatibility validation
