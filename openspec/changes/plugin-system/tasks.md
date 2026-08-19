# Tasks: Plugin System

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 850–950 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 → PR 2 → PR 3 → PR 4 |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Manifest + Discovery | PR 1 (~200 lines) | `go test ./internal/plugin/...` | N/A — pure library | internal/plugin/manifest.go, discovery.go |
| 2 | Registry + Installer | PR 2 (~220 lines) | `go test ./internal/plugin/...` | N/A — filesystem mock with t.TempDir | internal/plugin/registry.go, installer.go |
| 3 | Permissions + Command Dispatch | PR 3 (~250 lines) | `go test ./internal/plugin/... ./internal/tui/...` | N/A — unit tests only | internal/plugin/permissions.go, internal/tui/commands.go |
| 4 | CLI Commands + Integration | PR 4 (~250 lines) | `go test ./cmd/kui/...` | Manual: `kui plugin list` | cmd/kui/plugin.go |

## Phase 1: Plugin Manifest (internal/plugin/)

- [x] 1.1 RED: Write failing test for `PluginManifest` struct parse — `manifest_test.go`
- [x] 1.2 GREEN: Create `internal/plugin/manifest.go` — PluginManifest struct, Parse(), Validate()
- [x] 1.3 RED: Write failing test for semver validation — invalid version rejected
- [x] 1.4 GREEN: Add semver check to Validate(); support `extension.yaml` fallback with deprecation warning
- [x] 1.5 RED: Write failing test for entry_point existence check — missing file rejected
- [x] 1.6 GREEN: Add entry_point validation to Validate(); verify file exists relative to manifest dir

## Phase 2: Plugin Discovery (internal/plugin/)

- [x] 2.1 RED: Write failing test for Scan() — discovers plugins in global and project dirs
- [x] 2.2 GREEN: Create `internal/plugin/discovery.go` — Scan(), loadManifests(), project-override-global merge
- [x] 2.3 RED: Write failing test for missing directory — Scan() returns empty, no error
- [x] 2.4 GREEN: Handle missing dirs gracefully; log and continue

## Phase 3: Plugin Registry (internal/plugin/)

- [x] 3.1 RED: Write failing test for Install() — copies plugin dir, validates manifest
- [x] 3.2 GREEN: Create `internal/plugin/registry.go` — PluginRegistry interface, filesystemRegistry implementation
- [x] 3.3 RED: Write failing test for Remove() — deletes plugin directory
- [x] 3.4 GREEN: Implement Remove() with confirmation guard
- [x] 3.5 RED: Write failing test for List() — returns all plugins with scope info
- [x] 3.6 GREEN: Implement List() and Get(); merge global + project with project priority

## Phase 4: Plugin Installer (internal/plugin/)

- [x] 4.1 RED: Write failing test for Install() — validates manifest before copy, rejects invalid
- [x] 4.2 GREEN: Create `internal/plugin/installer.go` — copyDir(), validateOnInstall(), checksum
- [x] 4.3 RED: Write failing test for path traversal — `../escape` name rejected
- [x] 4.4 GREEN: Sanitize plugin name; reject `..` and absolute paths in install target

## Phase 5: Plugin Permissions (internal/plugin/)

- [x] 5.1 RED: Write failing test for Check() — all granted returns true, denied returns false
- [x] 5.2 GREEN: Create `internal/plugin/permissions.go` — PermissionManager, Check(), Grant(), Deny(), Load()
- [x] 5.3 RED: Write failing test for warn-only mode — logs warning, allows spawn
- [x] 5.4 GREEN: Implement warn-only vs enforce mode toggle
- [x] 5.5 RED: Write failing test for permissions.yaml — load/save cycle, manual edit detection
- [x] 5.6 GREEN: Implement permission persistence at `~/.config/kui/permissions.yaml`

## Phase 6: Command Dispatch (internal/tui/)

- [x] 6.1 RED: Write failing test for RegisterPluginCommand() — adds command to palette with category
- [x] 6.2 GREEN: Add RegisterPluginCommand() to CommandRegistry; wire to existing handlers map
- [x] 6.3 RED: Write failing test for ExecutePluginCommand() — sends JSON-RPC, displays output
- [x] 6.4 GREEN: Implement plugin command execution via DynamicExtension client
- [x] 6.5 RED: Write failing test for command unregistration — plugin shutdown removes commands
- [x] 6.6 GREEN: Add UnregisterPluginCommands() for plugin lifecycle cleanup

## Phase 7: CLI Commands (cmd/kui/)

- [x] 7.1 RED: Write failing test for `kui plugin list` — table output with name/version/type/status
- [x] 7.2 GREEN: Create `cmd/kui/plugin.go` — list subcommand with --json flag
- [x] 7.3 RED: Write failing test for `kui plugin install` — validates then copies
- [x] 7.4 GREEN: Implement install subcommand with --project flag and overwrite confirmation
- [x] 7.5 RED: Write failing test for `kui plugin remove` — confirmation required, --yes bypasses
- [x] 7.6 GREEN: Implement remove and info subcommands
- [x] 7.7 Wire `plugin` subcommand into `cmd/kui/main.go`

## Phase 8: Testing & Polish

- [x] 8.1 Integration test: install → discover → load → command dispatch flow
- [x] 8.2 Integration test: permission warn-only vs enforce mode toggle
- [x] 8.3 Test helpers: mock plugin directory, temp install paths, test fixtures
- [x] 8.4 Verify all existing tests pass: `go test ./...`

## Key Learnings

1. The existing `internal/extensions/dynamic/manifest.go` provides the `extension.yaml` pattern that must remain backward-compatible.
2. The `CommandRegistry` in `internal/tui/commands.go` already has the structure needed for plugin command registration.
3. Plugin permissions require a consent flow (REQ-PERM-5) which adds complexity to the permissions phase.
