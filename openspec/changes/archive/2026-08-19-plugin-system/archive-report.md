# Archive Report: plugin-system

**Date**: 2026-08-19
**Status**: Complete
**Mode**: openspec

## Summary

Plugin system for kui. Provides manifest parsing (kui-plugin.yaml), plugin discovery (global + project-local), registry/installer, permissions (enforce + warn-only modes), command dispatch, and CLI subcommands (list, install, remove, info).

## Artifacts

- proposal.md ✅
- design.md ✅
- specs/ ✅ (6 domains: dynamic-extension-lifecycle, extension-system, plugin-cli, plugin-command-dispatch, plugin-manifest, plugin-permissions)
- tasks.md ✅ (43/43 tasks complete)

## Specs Synced

| Domain | Action | Details |
|--------|--------|---------|
| dynamic-extension-lifecycle | Updated | Modified Subprocess Spawning (permission/manifest validation); Added REQ-DEL-MAN, REQ-DEL-PERM, REQ-DEL-DISCOVER |
| extension-system | Updated | Added REQ-EXT-7 (PluginManifest Integration), REQ-EXT-8 (Capability Declarations), REQ-EXT-9 (Permission Model Integration) |
| plugin-cli | Created | New domain — kui plugin list/install/remove/info subcommands |
| plugin-command-dispatch | Created | New domain — RegisterPluginCommand, ExecutePluginCommand, lifecycle cleanup |
| plugin-manifest | Created | New domain — kui-plugin.yaml format, semver, entry_point, capabilities |
| plugin-permissions | Created | New domain — PermissionManager, enforce/warn-only, permissions.yaml persistence |

## Stale Checkbox Reconciliation

None required — all tasks were already marked complete.

## Key Files

- `internal/plugin/` — manifest, discovery, registry, installer, permissions
- `internal/tui/commands.go` — plugin command registration
- `cmd/kui/plugin.go` — CLI subcommands
