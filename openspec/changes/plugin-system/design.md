# Design: Plugin System

## Technical Approach

Extend the proven subprocess model (JSON-RPC 2.0 over stdio) with a unified manifest format (`kui-plugin.yaml`), filesystem-based discovery from `~/.config/kui/plugins/` and `.kui/plugins/`, a permission model (warn-only initially), CLI commands via existing cobra-like dispatch, and command registration wiring to the TUI palette. The `internal/plugin/` package owns discovery, manifest, registry, installer, and permission enforcement. The existing `internal/extensions/dynamic/` package remains the subprocess runtime — plugins are loaded as dynamic extensions.

## Architecture Decisions

| Decision | Choice | Alternatives | Rationale |
|----------|--------|-------------|-----------|
| Manifest format | `kui-plugin.yaml` (YAML) | TOML, JSON | YAML matches existing `extension.yaml` and `profile.yaml` conventions |
| Plugin transport | JSON-RPC 2.0 subprocess | Go `plugin` pkg, WASM | Subprocess proven in `internal/extensions/dynamic/`; `plugin` pkg insecure/cross-platform broken; WASM premature |
| Permission storage | `~/.config/kui/permissions.yaml` | SQLite, JSON | Human-readable/editable per REQ-PERM-6; YAML consistent with project |
| Discovery priority | Project-local overrides global | Global-only, merge | Matches existing `extensions.yaml` merge behavior in `config.go` |
| Command dispatch | Wire `RegisterCommand` to `tui.CommandRegistry` | Separate plugin command system | Reuses existing palette infrastructure; minimal new code |
| Package location | `internal/plugin/` (new) | Inside `extensions/dynamic/` | Clean separation: plugin management vs subprocess runtime |

## Data Flow

```
User ──→ cmd/kui/plugin.go ──→ internal/plugin/registry.go
                                      │
                    ┌─────────────────┼─────────────────┐
                    ▼                 ▼                  ▼
             manifest.go        installer.go       discovery.go
             (parse/validate)   (copy dir)         (scan dirs)
                    │                 │                  │
                    └─────────────────┼─────────────────┘
                                      ▼
                           internal/plugin/permissions.go
                           (check/grant permissions)
                                      ▼
                           extensions/dynamic/manager.go
                           (spawn subprocess, register tools/commands)
                                      ▼
                           tui/commands.go
                           (palette dispatch)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/plugin/manifest.go` | Create | PluginManifest struct, parse, validate, semver check |
| `internal/plugin/registry.go` | Create | PluginRegistry: Install, Remove, List, Get |
| `internal/plugin/installer.go` | Create | Copy plugin dir, validate manifest, checksum |
| `internal/plugin/discovery.go` | Create | Scan global/project dirs, load manifests |
| `internal/plugin/permissions.go` | Create | PermissionManager: check, grant, persist |
| `internal/plugin/errors.go` | Create | Typed errors: ManifestErr, PermissionErr, NotFoundErr |
| `cmd/kui/plugin.go` | Create | CLI: list, install, remove, info subcommands |
| `cmd/kui/main.go` | Modify | Add `plugin` subcommand dispatch |
| `cmd/kui/extapi.go` | Modify | Wire `RegisterCommand` to collect commands |
| `internal/extensions/dynamic/extension.go` | Modify | Add permission check before spawn |
| `internal/extensions/dynamic/manager.go` | Modify | Integrate PluginManifest loading |
| `internal/tui/commands.go` | Modify | Add `RegisterPluginCommand` method |

## Interfaces / Contracts

```go
// PluginManifest — kui-plugin.yaml representation
type PluginManifest struct {
    Name            string   `yaml:"name"`
    Version         string   `yaml:"version"`
    Type            PluginType `yaml:"type"`
    EntryPoint      string   `yaml:"entry_point"`
    Description     string   `yaml:"description,omitempty"`
    Capabilities    []string `yaml:"capabilities,omitempty"`
    Permissions     []string `yaml:"permissions,omitempty"`
    ProtocolVersion string   `yaml:"protocol_version,omitempty"`
}

type PluginType string // tool | hook | command | theme | skill

// PluginRegistry — filesystem CRUD
type PluginRegistry interface {
    Install(src string, scope Scope) (*PluginManifest, error)
    Remove(name string) error
    List(scope Scope) ([]PluginInfo, error)
    Get(name string) (*PluginInfo, error)
}

// PermissionManager — enforcement
type PermissionManager interface {
    Check(plugin string, perms []string) (allGranted bool, denied []string)
    Grant(plugin, permission string) error
    Deny(plugin, permission string) error
    Load() error  // re-reads permissions.yaml
}
```

## Sequence Diagrams

**Plugin Install:**
```
User → CLI → installer.Validate() → manifest.Parse() → copy dir → registry.Record()
```

**Plugin Load:**
```
discovery.Scan(global) + discovery.Scan(project)
  → manifest.Parse() [kui-plugin.yaml or fallback extension.yaml]
  → permissions.Check()
  → dynamic.Manager.LoadAll() → DynamicExtension.Init()
    → subprocess spawn → JSON-RPC handshake → ListTools → RegisterTool
    → RegisterCommand → tui.CommandRegistry
```

**Command Dispatch:**
```
User → Ctrl+P → CommandPalette → Select plugin cmd
  → extAPI.ExecuteCommand(name) → JSON-RPC call to subprocess
  → response displayed in TUI
```

**Permission Enforcement:**
```
Before spawn → permissions.Check(plugin, requiredPerms)
  ├─ all granted → spawn
  ├─ denied + enforce mode → block, log error
  └─ denied + warn-only → log warning, spawn
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Manifest parse/validate, permission check, semver | Table-driven tests with golden files |
| Unit | Registry install/remove/list | Mock filesystem (t.TempDir) |
| Unit | Discovery scan + override | Temp dirs with global/project structure |
| Integration | Install → discover → load → command dispatch | End-to-end with mock subprocess |
| Integration | Permission warn-only vs enforce | Two test runs with mode toggle |

## Threat Matrix

| Boundary | Applicability | Design Response | RED Tests |
|----------|--------------|-----------------|-----------|
| Subprocess spawn (plugin entry point) | **Applicable** | Validate entry_point exists + permission check before `exec.Command`; no shell interpretation | Test: invalid path → spawn fails; test: missing permission → blocked |
| Filesystem paths (install target) | **Applicable** | Sanitize plugin name; reject `..` in paths; fixed install dirs only | Test: `../escape` name rejected; test: absolute path rejected |
| Git repository selection | N/A | No VCS integration | — |
| Commit state | N/A | No VCS integration | — |
| Push/PR commands | N/A | No VCS integration | — |

## Migration / Rollout

1. **Phase 1**: Ship `kui-plugin.yaml` support alongside `extension.yaml`. Deprecation warning on legacy format.
2. **Phase 2**: Permission warn-only mode — log warnings but never block.
3. **Phase 3**: After real-world feedback, default to enforce mode.
4. **Existing extensions**: Continue working via `extension.yaml` → `PluginManifest` conversion with type "tool" and empty permissions.

## Open Questions

- [ ] Should command execution go through the existing `DynamicExtension` client, or add a `CallCommand` JSON-RPC method? (Affects protocol version bump.)
- [ ] Plugin update mechanism: `kui plugin update <name>` from URL, or leave to manual reinstall?
- [ ] Should `kui plugin install` support `--force` to skip confirmation on overwrite?
