# Tasks: Dynamic Extensions

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 340–400 (8 new files, 2 modified, ~4 test files) |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | single PR |
| Delivery strategy | auto-chain |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Low

## Phase 1: Foundation — Errors & Types

- [x] 1.1 Create `internal/extensions/dynamic/errors.go`: `ManifestError`, `ProtocolError`, `SpawnError` types
- [x] 1.2 Create `internal/extensions/dynamic/manifest.go`: `Manifest` struct with `LoadManifest(path) (*Manifest, error)`; parse `extension.yaml` (name, version, protocol_version, entry_point)
- [x] 1.3 Create `internal/extensions/dynamic/manifest_test.go`: table-driven — valid YAML, missing fields, malformed YAML, missing file
- [x] 1.4 Create `internal/extensions/dynamic/config.go`: `Config` struct with `LoadConfigs(global, project string) (*Config, error)`; merge global + project + config paths, project wins on collision
- [x] 1.5 Create `internal/extensions/dynamic/config_test.go`: empty dirs, name collision (project wins), config paths extension

## Phase 2: Protocol Client & Adapter

- [ ] 2.1 Create `internal/extensions/dynamic/client.go`: `Client` struct reusing MCP's `jsonrpcRequest`/`jsonrpcResponse` framing; `NewClient(ctx, entryPoint)`, `Initialize(ctx)`, `ListTools(ctx)`, `CallTool(ctx, name, args)`, `Close()`
- [ ] 2.2 Create `internal/extensions/dynamic/client_test.go`: mock stdio pipes, test sendRequest serialization, version mismatch handling, tool list/call framing
- [ ] 2.3 Create `internal/extensions/dynamic/tool.go`: `DynamicTool` struct implementing `core.Tool`; wraps `Client.CallTool`; name = `{extensionName}_{toolName}`; returns error on client failure
- [ ] 2.4 Create `internal/extensions/dynamic/extension.go`: `DynamicExtension` struct implementing `core.Extension`; `Init(api)` spawns client → handshake → list tools → register via `api.Register(tool)`; `Shutdown()` sends shutdown notification, waits 5s, kills
- [ ] 2.5 Create `internal/extensions/dynamic/extension_test.go`: mock client factory → verify Init registers tools; crash scenario → tools return error

## Phase 3: Lifecycle Manager

- [ ] 3.1 Create `internal/extensions/dynamic/manager.go`: `Manager` struct with `NewManager(config)`, `LoadAll(ctx, api)` discovers extensions from config, spawns each as `DynamicExtension`, registers via `RegisterDynamic()`; crash = log + mark unavailable
- [ ] 3.2 Create `internal/extensions/dynamic/manager_test.go`: full lifecycle — discover → spawn → register → shutdown; entry point missing → extension unavailable; crash → other extensions unaffected
- [ ] 3.3 Verify: entry point missing scenario, handshake version mismatch scenario, crash handling scenario, graceful shutdown + timeout kill scenario

## Phase 4: Integration — Registry & Runtime

- [ ] 4.1 Modify `internal/adapters/extensions/registry.go`: add `dynamic []core.Extension` slice, `RegisterDynamic(ext)` (nil panics), `LoadAll` processes both `global` and `dynamic` in order
- [ ] 4.2 Modify `internal/runtime/runtime.go`: add dynamic extension discovery step after MCP in `buildComponents`; load `extensions.yaml` via `dynamic.LoadConfigs`, create `dynamic.Manager`, call `LoadAll`
- [ ] 4.3 Test: mixed extensions coexist (compiled-in + dynamic both active)
- [ ] 4.4 Test: Init failure with mixed extensions (dynamic B fails → compiled-in A.Shutdown called)

## Phase 5: Verification

- [ ] 5.1 Run `go test ./internal/extensions/dynamic/...` — all unit + integration tests pass
- [ ] 5.2 Run `go test ./internal/adapters/extensions/...` — registry tests pass
- [ ] 5.3 Run `go test ./internal/runtime/...` — runtime build/reload tests pass
- [ ] 5.4 Run `go vet ./internal/extensions/dynamic/... ./internal/adapters/extensions/ ./internal/runtime/` — no warnings
- [ ] 5.5 Manual: place test extension in `~/.config/kui/extensions/`, run `Build`, confirm tool appears in registry

## Key Learnings
1. MCP's `jsonrpcRequest`/`jsonrpcResponse` framing and `Client` pattern can be directly reused for the extension protocol, reducing implementation risk.
2. The extension registry's `LoadAll` must process compiled-in (`global`) and dynamic slices sequentially so Init-failure rollback covers both.
3. Separating the dynamic package from `internal/mcp/` keeps the MCP protocol clean while enabling future hook/command divergence in Phase 2.
