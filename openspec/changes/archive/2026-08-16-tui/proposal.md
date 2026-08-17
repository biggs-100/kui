# Proposal: OpenCode-Style TUI with In-Session Profile Switching

## Intent

kui runs one-shot prompt sessions via CLI: switching profiles means a new invocation, and tool execution is invisible. The product vision needs an interactive primary workflow — an OpenCode-style TUI chat where TAB switches profiles in the same session and tool output is visible.

## Scope

### In Scope
- Bubble Tea TUI in `internal/tui` (views) + controller wiring to runtime.
- Chat view: messages, input, streaming answer.
- Profile tab header; TAB/shift+TAB cycle (opencode `agent_cycle`); active profile per session.
- Per-profile model resolution; explicit `{profile, model}` on every prompt.
- Tool output view (live calls/results).
- `kui tui` entrypoint in `cmd/kui/main.go`.

### Out of Scope
- Interactive permission asks (ask degrades to deny); MCP; persistent sessions; runtime auto-reread of profiles.

## Capabilities

### New Capabilities
- `tui-app`: lifecycle, layout, input, entrypoint.
- `tui-chat`: messages, prompt submission.
- `tui-profile-switcher`: tab header, TAB cycle, per-profile model, session-active.
- `tui-tool-view`: live calls/results.

### Modified Capabilities
- `agent-cli`: add `kui tui` entrypoint.
- `agent-loop`: nil-safe stdlib observer port for tool/turn events.

## Approach

- `internal/tui` owns Bubble Tea + lipgloss; controller connects views to runtime. Core stays stdlib-only (guard test).
- Run `agent.Run` in a goroutine; feed the app via `tea.Cmd`.
- TAB cycles `loader.Discover()` (wraps); switch enqueued via steering queue (REQ-PROFILE-3), applies between turns, history preserved (REQ-LOOP-5/6).
- Model per profile: `store.Get` → resolved → default (REQ-CLI-4); every prompt sends `{profile, model}`.
- Tool view needs events the loop lacks: add optional stdlib observer port; design details.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/kui/main.go` | Modified | `kui tui` dispatch |
| `internal/tui/**` | New | views + controller (Bubble Tea) |
| `internal/core/observer.go` | New | stdlib observer port |
| `go.mod` | Modified | Bubble Tea deps |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Bubble Tea API churn | Med | pin version; logic in controller |
| Observer leaks coupling into core | Med | stdlib-only; nil-safe; guard test |
| Loop blocks while TUI renders | Med | goroutine + `tea.Cmd` |

## Rollback Plan

Remove `kui tui` dispatch; one-shot CLI unchanged. Observer port is optional/nil-safe — removing it reverts core. Fallback: `git revert`.

## Dependencies

- `charmbracelet/bubbletea` (+ `lipgloss`) — first UI deps; confined to `internal/tui`.

## Success Criteria

- [ ] `kui tui` starts; TAB/shift+TAB cycle profiles; switch applies between turns, history intact.
- [ ] Tool calls render during multi-step turns.
- [ ] Every prompt carries `{profile, model}`.
- [ ] `go test ./...` green; core guard test blocks Bubble Tea; teatest covers flows.
