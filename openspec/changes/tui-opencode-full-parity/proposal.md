# Proposal: TUI OpenCode Full Parity

## Intent

Achieve total TUI parity vs `packages/tui/src` — visual, layout, interaction, edge cases. Closes 40+ residuals after `tui-redesign`/`tui-parity` (center home, route, live models): theme 15 fields, logo tint, prompt extmarks/shell, sidebar 30@110, footer dots, chat `┃/╹`, dialog modal, grouped palette, markdown tokens, keymap, literals.

## Scope

### In Scope
- Theme 40+ (`backgroundPanel/Element/Menu`, `markdown*/syntax*/diff*`, `thinkingOpacity`) + `tint` + JSON loader; remove `#2a2a2a/#252525/#e0af68`.
- Home: logo tint shadow, flex-spacer centering, prompt 75/70%, `backgroundElement`+`SplitBorder`.
- Session: sidebar 42@120, footer welcome tick→`• LSP ⊙ MCP △`+`/status`, chat per-part `┃/╹`+hover+locale, tool collapse, diff tree.
- Overlays: `ui/dialog` (150 overlay, 60/88/116, modal stack) + `DialogSelect` (fuzzysort, groups, `backgroundMenu`); palette/model/status dialogs.
- Markdown tokens + keymap `base/modal`+leader.

### Out of Scope
- Workspace/permission/editor — muted `NotAvailable` (follow-up).
- Plugin slots — omit; never fake `mimo/319k/context7`.
- Perf 60fps — out of scope beyond TTL.

## Capabilities

### New Capabilities
- `tui-theme-system`: full Theme + tint + JSON.
- `tui-dialog-overlay`: overlay + grouped select.

### Modified Capabilities
- `tui-app`: breakpoints, overlay sidebar, toast/title.
- `tui-home`: logo, spacers, prompt.
- `tui-chat`: per-part, border, markdown.
- `tui-tool-view`: collapse, diff highlight.

## Approach

Auto-chain Feature Branch Chain, 4 PRs ≤400 lines, text-dump goldens (no PNG).

| PR | Slice | Contains | Golden |
|----|-------|----------|--------|
| 1 | Foundations | `theme/*` + `ui/border+dialog` + `util/locale` | theme |
| 2 | Home | `logo|home|home_prompt|header|footer` | home 80/120/160 |
| 3 | Session | `chat|tool|diff|sidebar|footer` | session/diff |
| 4 | Overlays | `DialogSelect` palette/model/status + `keymap` + toast | palette/model |

Chain `1→2→3→4` on tracker `tui-opencode-full-parity` draft.

## Affected Areas

| Area | Impact | Change |
|------|--------|--------|
| `internal/tui/theme/*` | Modified | 40 fields, `tint`, JSON |
| `internal/tui/ui/*` | New | `border`, `dialog`, `dialog-select` |
| `internal/tui/views/*` | Modified | home/sidebar/footer/chat/tool/diff |
| `internal/tui/{app,controller,autocomplete,commands}.go` | Modified | width/keymap/slash args |
| `internal/tui/{markdown,testdata}/*` | Modified/New | tokens + goldens |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Fabrication | High | `parity_test.go` bans fakes; nil→omit |
| Char drift `┃` vs `│` | Med | Custom `Border`; goldens 80/120/160 |
| Keymap/markdown gap | Med | Full binding table; fenced highlight only |

## Rollback Plan

Per-PR `git revert` isolated. Tracker never merges until children land. No migrations.

## Dependencies

- `packages/tui/src/theme/assets/opencode.json`.
- `sync.data.*` — degrade to muted if absent.
- `sahilm/fuzzy` weighted filter.

## Success Criteria

- [ ] No hex outside `theme`; Theme 40+ fields
- [ ] Home 80/120/160 ±1 col vs OpenCode
- [ ] Footer tick→`• ⊙` or muted; chat `┃/╹` per-part
- [ ] Dialog grouped, Esc filter-then-close
- [ ] `go test`/`go vet`/`gofmt` clean; goldens committed
