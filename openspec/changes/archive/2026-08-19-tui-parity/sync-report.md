# Sync Report: TUI Parity

Change: `tui-parity`
Status: SYNCED (artifacts reflect verified state)

## Artifacts
- `proposal.md` — intent, fake evidence table, scope, success criteria.
- `specs/tui-parity/spec.md` — 6 requirements (Given/When/Then, RFC 2119).
- `design.md` — data-source authority table; per-component removal plan.
- `tasks.md` — 8 tasks (impl + TDD), all complete.
- `verify-report.md` — RED→GREEN evidence, test updates, spec conformance.

## Canonical spec alignment
The `tui-parity` capability spec is net-new (no prior canonical `specs/tui-parity`).
No existing canonical spec was modified. The change introduces the requirement that
every rendered TUI field is either controller-backed or omitted — this is the new
baseline for future TUI work.

## Final state at sync
- All 6 requirements verified passing via `go test ./...`.
- Forbidden literals absent from `internal/tui/views/*.go`.
- Dead code removed; theme tokens used for all colors.
- Working tree modified, NOT committed (commit pending explicit authorization).
- SDD attempt ledger settle blocked on maintainer reset (bookkeeping only).

## Out of scope (deferred, not in this change)
- Real LSP/MCP status tracking (future change wires `SetMCPStatus`/`SetLSPStatus`).
- kui version string (future change, if desired).
- Layout dimension tweaks (already correct from `tui-redesign`).
