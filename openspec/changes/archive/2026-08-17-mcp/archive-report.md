## Change Archived

**Change**: mcp
**Archived to**: `openspec/changes/archive/2026-08-17-mcp/` (openspec)

### Specs Synced
| Domain | Action | Details |
|--------|--------|---------|
| agent-tools | Updated | REQ-TOOLS-4 modified to accept MCP tools alongside built-in tools |
| mcp-client | Created | New spec for JSON-RPC 2.0 client over stdio |
| mcp-config | Created | New spec for YAML config with global+project merge |
| mcp-manager | Created | New spec for server lifecycle management |
| mcp-tool-bridge | Created | New spec for MCPTool adapter implementing core.Tool |

### Archive Contents
- proposal.md ✅
- specs/ ✅ (5 domain directories)
- design.md ✅
- tasks.md ✅ (20/20 tasks complete)
- verify-report.md ✅

### Source of Truth Updated
The following specs now reflect the new behavior:
- `openspec/specs/agent-tools/spec.md` — REQ-TOOLS-4 updated
- `openspec/specs/mcp-client/spec.md` — new
- `openspec/specs/mcp-config/spec.md` — new
- `openspec/specs/mcp-manager/spec.md` — new
- `openspec/specs/mcp-tool-bridge/spec.md` — new

### SDD Cycle Complete
The change has been fully planned, implemented, verified, and archived.
Ready for the next change.

## Key Learnings

1. On Windows without diff, SHA-256 hash verification provides equivalent byte-identity proof for mechanical archive copies.
2. git mv on Windows moves entire directories including untracked files when the source directory is removed.
3. Merging MODIFIED delta specs requires careful replacement of entire requirement blocks while preserving surrounding requirements.
4. MCP implementation maintained stdlib-only constraint, preserving the TestCoreImportsStdlibOnly guard test.
5. Non-fatal server failures in MCP manager ensure graceful degradation when external tool servers are unavailable.