# Proposal: Plugin System

## Intent

kui has 90% of plugin infrastructure (Extension/ExtensionAPI interfaces, dynamic extensions via JSON-RPC 2.0, MCP servers, hook registry, permissions). What's missing is the user-facing layer: no CLI to install/manage plugins, no unified manifest, no permission model for untrusted plugins, and command registration is stubbed. This change completes the plugin story so users can discover, install, and run third-party plugins with confidence.

## Scope

### In Scope
- Unified plugin manifest (`kui-plugin.yaml`) extending existing `extension.yaml`
- Plugin CLI commands: `kui plugin install`, `kui plugin list`, `kui plugin remove`
- Plugin permissions model (allow/ask/deny with glob patterns) for dynamic extensions
- Wire `RegisterCommand` to TUI command palette dispatch
- Plugin directory conventions: `~/.config/kui/plugins/` (global), `.kui/plugins/` (project)

### Out of Scope
- Plugin registry/marketplace (remote discovery) — deferred to `plugin-registry`
- WASM plugins — premature, not proven in Go TUI context
- Go `plugin` package — insecure, not cross-platform
- Plugin versioning/dependency resolution — future refinement
- Hot-reload of plugins mid-session

## Capabilities

### New Capabilities
- `plugin-cli`: CLI commands for plugin install/list/remove with manifest validation
- `plugin-registry`: Filesystem-based plugin discovery from `~/.config/kui/plugins/` and `.kui/plugins/`

### Modified Capabilities
- `extension-system`: Add permission model requirement — dynamic extensions MUST declare permissions in manifest; system MUST enforce allow/ask/deny before execution
- `dynamic-extension-lifecycle`: Add permission check before subprocess spawn; manifest format upgrade from `extension.yaml` to `kui-plugin.yaml`

## Approach

Enhance the proven subprocess model (JSON-RPC 2.0) — no new transport. Create `kui-plugin.yaml` as the unified manifest format superseding `extension.yaml`. Build CLI via existing cobra command structure. Permission enforcement wraps the existing spawn path. Command dispatch wires `RegisterCommand` output to the TUI command palette.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/extensions/dynamic/` | Modified | Manifest parsing upgrade, permission enforcement |
| `internal/core/extension.go` | Modified | Command registration wiring |
| `cmd/kui/plugin.go` (new) | New | CLI commands for plugin management |
| `internal/plugin/` (new) | New | Plugin discovery, manifest parsing, permission model |
| `internal/tui/` | Modified | Command palette dispatch integration |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Manifest format breaks existing extensions | Medium | Backward-compatible: support both `extension.yaml` and `kui-plugin.yaml` |
| Permission model too restrictive | Low | Start with warn-only mode, enforce after real-world feedback |
| CLI adds binary size | Low | Minimal deps; cobra already in use |

## Rollback Plan

Revert to pre-plugin-system state: remove `internal/plugin/`, `cmd/kui/plugin.go`, restore `extension.yaml` parsing. Dynamic extensions continue working via existing `extension.yaml` path. No data loss — plugins are filesystem-based.

## Dependencies

- Existing `internal/extensions/dynamic/` (JSON-RPC client, subprocess management)
- Existing `cmd/kui/` (cobra command structure)

## Success Criteria

- [ ] `kui plugin install ./my-plugin` installs a plugin from a local directory
- [ ] `kui plugin list` shows all installed plugins with status
- [ ] `kui plugin remove my-plugin` removes a plugin and its artifacts
- [ ] Dynamic extensions declare permissions in `kui-plugin.yaml`
- [ ] System enforces permissions before subprocess spawn
- [ ] `RegisterCommand` commands appear in TUI command palette
- [ ] All existing tests pass (`go test ./...`)
