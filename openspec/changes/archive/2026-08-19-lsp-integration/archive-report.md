# Archive Report: lsp-integration

**Date**: 2026-08-19
**Status**: Complete
**Mode**: openspec

## Summary

LSP (Language Server Protocol) integration for kui. Provides JSON-RPC 2.0 transport over stdio, diagnostic caching, LSP tools (hover, definition, references), runtime composition, file sync hooks, and TUI inline diagnostics.

## Artifacts

- proposal.md ✅
- design.md ✅
- specs/ ✅ (6 domains: agent-tools, lsp-client, lsp-diagnostics, lsp-integration, lsp-tools, tui-chat)
- tasks.md ✅ (40/40 tasks complete)

## Specs Synced

| Domain | Action | Details |
|--------|--------|---------|
| agent-tools | Updated | Added LSP tools to REQ-TOOLS-4 registration surface |
| lsp-client | Created | New domain — JSON-RPC 2.0 transport, handshake, lifecycle |
| lsp-diagnostics | Created | New domain — diagnostic cache, severity, file queries |
| lsp-integration | Created | New domain — runtime composition, lazy startup, TUI display |
| lsp-tools | Created | New domain — lsp_diagnostics, lsp_hover, lsp_definition, lsp_references |
| tui-chat | Updated | Added inline diagnostic annotations to REQ-TUI-CHAT-2 |

## Stale Checkbox Reconciliation

None required — all tasks were already marked complete.

## Key Files

- `internal/lsp/` — LSP client, cache, manager, tools
- `internal/runtime/runtime.go` — LSP wired in Build/Reload/Close
- `internal/tui/views/chat.go` — inline diagnostic rendering
