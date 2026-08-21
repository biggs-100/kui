# Exploration: tui-opencode-full-parity — Total UX/UI Parity with OpenCode

## Goal
Achieve **total** visual, layout, interaction, and edge-case parity with OpenCode's real TUI (`packages/tui/src`, `packages/opencode` reference). Previous changes `tui-redesign` (centered home, route system, OpenCode gray theme, sidebar breakpoint) and `tui-parity` (removed fabricated `319k`, `mimo-*`, version literals, wired live `/models` discovery) fixed the largest deceptions. User now requires every remaining delta closed — no approximations.

## Current State (after tui-redesign + tui-parity)

### Architecture today
```
App (route: home | session)
├── home: HomeView { LogoModel, HomePromptModel } + HomeFooterModel + toast
├── session: HeaderModel | ChatModel | ToolModel | DiffModel | FooterModel | InputModel (textarea) + toast + autocomplete
├── overlays: CommandPaletteModel | SessionListModel | ModelListModel | ProviderListModel (bubbles/list wrappers)
└── Controller { profiles, model resolver, streaming, session store, tokens/cost }
Theme: Theme struct (~25 fields, BG/FG/Border/Primary/etc) + Styles (lipgloss) — defaults to OpenCodeTheme (#1a1a1a/#333)
```

### What already matches
- Route split home vs session, vertical centering via `lipgloss.PlaceHorizontal`, bordered home prompt (~70 cols), minimal gray palette.
- All hardcoded fakes removed: footer shows `— tokens` or real `TotalTokens()/Cost()`; sidebar shows `0 tokens 0% $0.00` truthfully; `AvailableModels()` no longer contains `mimo-*`; no scattered hex literals except vetted theme tokens (with 3 residual `#2a2a2a`/`#252525`/`#e0af68` that must move into Theme).
- Live model discovery (`liveModelsForProvider` per provider, 5-min TTL, `/models` endpoints), `/login` fuzzy, `@` file completions, fuzzy palette.
- Sidebar only renders real `tokens/cost/profile/model`, input bar uses `#2a2a2a`+`#569cd6` accent to mimic OpenCode.

### Source-of-truth snapshot (kui)
- `internal/tui/app.go` (1083 LOC) — View() builds 3-region vertical stack, sidebar only when `width>=110` (30 cols), trimToWidth for JoinHorizontal, toast between tool and input.
- `views/home.go:60-102` — manual topPad calc, logo + prompt centered, footer reserve 2.
- `views/logo.go` — single 6-line `██╗` block, single `LogoAccent` color.
- `views/home_prompt.go:58-74` — `width-20` capped 70, rounded border with `HomeBorder`.
- `views/header.go` — tabs `fmt.Sprintf(" %s ", p)` joined with `" "`, active `Bold+TabActive`, inactive `Faint+TextMuted`.
- `views/footer.go:51-95` — `dir • tokens · $cost • model • ctrl+p commands` all `HomeMuted`.
- `views/chat.go:124-179` — `"you"/"assistant"` labels, `(profile/model)` faint, markdown via regex replace, status `● ` in `HomeMuted`.
- `views/sidebar.go:52-118` — sections `Context` + `Session` only, header `LogoAccent`, `sep` rule, `Sidebar.Width(width)`.
- `views/tool.go:56-94` — per-entry `Panel` rounded `#252525/#333`, `○ pending`.
- `views/diff.go:70-131` — `CHANGED FILES` + `▶` cursor + `+N/-N` + hunk render inside `Panel`.
- `markdown/renderer.go` — regex heading/bold/italic/code/blockquote/list, code block `Background #252525`, Thought `#e0af68`.
- `theme/opencode.go` — 25-field palette; `theme.go` DefaultTheme `kui-default` `#1a1b26`; `styles.go` all lipgloss styles with residual literals `#2a2a2a`, `#569cd6`, `#252525`, `#e0af68`.
- `controller.go` — real `TotalTokens/ContextWindow/Cost/ModelName` via `TrackUsage`.

### OpenCode truth (packages/tui/src)
- **App shell** (`app.tsx`): SolidJS `@opentui/solid` renderer 60fps, kitty keyboard, mouse, keymap stack (`base`/`modal`), global bindings (`session.list/new/quick_switch`, `command.palette.show`, `model.list`, `agent.list`, `mcp.list`, etc.), route context `home | session | plugin`, startup loading, prompt stash/frecency/history contexts, plugin runtime with `home_logo/home_prompt/home_bottom/sidebar_*` slots.
- **Home** (`routes/home.tsx`): `flexGrow spacer` centering, `height 4` spacer, `Logo` via `logo.ts` left/right pairs with `tint(background, fg, 0.25)` shadow, `Prompt` with `maxWidth 75` or `auto=70%`, `Toast` inside centered column, footer `home_footer` single_winner slot.
- **Session** (`routes/session/index.tsx` + `sidebar.tsx` + `footer.tsx`): `contentWidth = width - (sidebarVisible?42:0) -4`, `wide = width>120`, sidebar `42` cols `backgroundPanel`, scrollbox with sticky bottom + scrollbar + acceleration, messages per-part rendering (User left-border `SplitBorder` `┃` agent-color, hover `backgroundElement`, `QUEUED` badge, files, compaction divider; Assistant via `PART_MAPPING`), permissions/questions overlays, prompt with left `SplitBorder` colored by agent/shell/leader, `backgroundElement`, spinner inside prompt footer, `session.footer` shows `welcome` tick cycle (Get started `/connect` → dot counts) vs `connected` state `• N LSP` + `⊙ N MCP` + `/status`, plus permission triangle.
- **Theme** (`theme/index.ts`, `assets/opencode.json`): 30+ themes, `Theme` has 40+ fields: `primary/secondary/accent/error/warning/success/info/text/textMuted/selectedListItemText/background/backgroundPanel/backgroundElement/backgroundMenu/border/borderActive/borderSubtle/diff* (added/removed/context/hunkHeader/highlight/bg/lineNumber)/markdown* (text/heading/link/linkText/code/blockQuote/emph/strong/hRule/listItem/enum/image...)/syntax* (comment/keyword/function/variable/string/number/type/operator/punctuation)/thinkingOpacity`, plus `generateSystem` (terminal palette fallback), `tint`, `selectedForeground`, `generateSyntax`.
- **Prompt** (`component/prompt/index.tsx`): `TextareaRenderable` with `extmarks` (file/agent/paste virtual text `● [File]/[Image]/[Pasted ~N lines]`), modes `normal/shell` (`!` at offset 0), random placeholder, `maxHeight = max(6, height/3)` or config, traits, draft stash, editor context chips, agent/model/variant footer with `tint + fadeColor + createFadeIn`, right slot, `SplitBorder` + `EmptyBorder` decorative bottom `▀`, money `Intl.NumberFormat`, `formatDuration`, etc.
- **Dialog system** (`ui/dialog.tsx` + `dialog-select.tsx`): overlay `RGBA(0,0,0,150)`, sizes `60/88/116`, modal mode stack pushes `"modal"`, esc/ctrl+c close with selection guard, top padding `height/4`; `DialogSelect` does `fuzzysort` weighted `title*2+category`, grouping by `category`, scrollbox with `scrollAcceleration`, filter `InputRenderable` focused, actions as keybound commands (favorite, connect provider), footer hints (left/right), emptyView, preserveSelection with double rAF re-scroll, `isDeepEqual` value identity, categoryView, details truncation `truncateMiddle(76)`, highlight vs muted foreground via `selectedForeground`.
- **Command palette** (`component/command-palette.tsx`): queries `getCommandEntries(visibility:reachable, namespace:palette)` minus hidden + `COMMAND_PALETTE_COMMAND`, formats bindings via `formatKeyBindings` (leader token, aliases pgup→pgup etc), groups suggested on top when no filter.
- **Model dialog** (`dialog-model.tsx`): favorites/recent/provider sections, `disabled` for `opencode/*-nano`, `Free` footer for `cost.input==0`, sorting by `free→releaseDate desc→title`, fuzzy across title+category, `flat:true`, current model dot `●`.
- **Status dialog** (`dialog-status.tsx`): MCP Servers count + per-server `•` colored `connected=success/failed=error/disabled=textMuted/needs_auth=warning` + error string detail; LSP Servers similarly; Formatters; Plugins parsed `file://` or `name@version`.

## Affected Areas

- `internal/tui/app.go` — header bottom border (`NormalBorder` single side, theme Border), input bar (residual `#252525` popup bg, `#2a2a2a`+`#569cd6` literal), autocomplete popup (centered on home vs left-aligned on session, hardcoded `Background #252525`), wide breakpoint 110→120, sidebar width 30→42, main panel trimming, toast placement (OpenCode toast lives inside centered column + session scroll area), route transition still no `NewSession` (`/new`) or workspace/shell mode handling.
- `internal/tui/theme/{theme,styles,opencode}.go` — struct missing ~15 OpenCode fields (`backgroundPanel/Element/Menu`, `selectedListItemText`, `markdown*`, `syntaxOperator/Punctuation`, `diff*Bg/Highlight/LineNumber`, `thinkingOpacity`); styles missing `Panel`→`backgroundPanel` distinction, `DiffHunk` vs `DiffHighlight`, residual literals must become tokens (`InputBar #2a2a2a`, `InputBarAccent #569cd6`, `CodeBlock #252525`, `Thought #e0af68`); needs `tint/selectedForeground/generateSyntax/generateSystem` equivalents and `assets/*.json` import or codegen.
- `internal/tui/views/home.go` + `home_prompt.go` + `logo.go` — logo single block vs OpenCode two-sided shadowed `█▀▀█` with `tint`, prompt maxWidth off-by-5 and placeholder random from `placeholders` not single `Ask kui…`, vertical centering uses topPad loop vs flexGrow spacers (resize flicker), missing `home_bottom` plugin slot.
- `internal/tui/views/header.go` — spacing `" "` vs OpenCode gap, inactive/active styles lack `background` token (`TabActiveBG`) and no agent-color variant; header hidden on home already but OpenCode Home has NO header.
- `internal/tui/views/footer.go` — must split into `SessionFooterModel` mirroring OpenCode `routes/session/footer.tsx` (welcome tick, `• LSP` green/muted, `⊙ MCP` success/error, permission badge `△ N Permission`, `/status` vs `Get started /connect` cycle). Current shows `dir • tokens · $cost • model • ctrl+p`; missing `LSP/MCP` dots (intentionally omitted before but parity says they return as real `sync.data.lsp/mcp` counts — kui needs those backing stores), missing permission indicator, missing welcome animation tick.
- `internal/tui/views/home_footer.go` — currently `dir • LSP ○/● • MCP ○/● • /status` with binary bool; OpenCode home has no footer content except plugin slot (kui's mimic is invention). Needs to match either OpenCode empty home footer or synthesize from real `sync.data.lsp/mcp`.
- `internal/tui/views/chat.go` — labels `you/assistant` should be left-border `SplitBorder` (`┃`/`╹`) with agent color, hover, timestamps `Locale.todayTimeOrDateTime`, queued badge, compaction divider, `stickyScroll`; markdown must use theme `markdown*+syntax*` not regex-only `ToolName/Hint`; inline code uses `markdownCode` bg; need per-part model (`text/reasoning/tool/file/compaction`).
- `internal/tui/views/sidebar.go` — header should be session `title` bold + `sessionID` when `InstallationChannel!="latest"` + workspace label + share URL (if any) + `sidebar_content/sidebar_footer` slots. Current `Context` section should mirror OpenCode plugin `context.tsx`: `Context` title `text` bold, then `tokens.toLocaleString() tokens` + `percent% used` + `$spent` each muted, not single line. Footer should be `• Open Code <version>` with `success` dot (from `InstallationVersion` — kui has no version source, must omit or wire `buildinfo.ReadFile` version).
- `internal/tui/views/tool.go` — needs `collapseToolOutput`, `genericToolOutput` toggle, diff highlight backgrounds, `showDetails/showGenericToolOutput` kv signals, per-tool metadata display, not just `Name → Result` in rounded panel.
- `internal/tui/views/diff.go` — needs file-tree utils, revert diff files, line numbers with `diffLineNumber*Bg`, `EmptyBorder/SplitBorder` border chars, `diffWrapMode` word/none.
- `internal/tui/markdown/renderer.go` — replace ad-hoc `Background #252525` with `styles.CodeBlock` token, `Thought #e0af68` with `theme.warning/syntax*`, add `markdownLink/BlockQuote/ListItem` colors, use `SyntaxStyle.fromTheme(getSyntaxRules)` for fenced blocks, not single `HighlightCode` with `DefaultTheme()`.
- `internal/tui/views/*_list.go` + `command_palette.go` — all four list models need full `Dialog` + `DialogSelect` replacement: categories, details, footer hints, key bindings formatted via `formatKeyBindings`, `hidden/suggested` filtering, `Flat` vs grouped, `preserveSelection`, `backgroundMenu` selection color, overlay centering `Place(width,height,Center,Center)` with `theme.backgroundPanel` not `BGFloat`, sizes `60/88/116`, esc clears filter then closes.
- `internal/tui/{autocomplete,input,commands}.go` + `views/model_list.go` — autocomplete must include slash arg variants (`/model`, `/login`, `/theme next/prev`, `/sessions`, `/resume`, `/rename`, etc.) plus model variant handling (`variant.cycle/list`), Shell `!` mode, `editor` context chip; commands registry must add `session.*` (`share/rename/timeline/fork/compact/unshare/undo/redo/sidebar.toggle/toggle.conceal/...`), `prompt.*`, `terminal.suspend`, etc., and map bindings via `config.keybinds.gather` pattern not hardcoded keys.
- `internal/tui/controller.go` + `run.go` — needs real `sync.data.provider/mcp/lsp/formatter/plugin/session/permission` stores, kv store (`animations_enabled`, `file_context_enabled`, `sidebar`, `timestamps`, `diff_wrap_mode`, `paste_summary_enabled`, etc.), local model/agent stores (favorites/recents, variant), `connected()` derived, `workspace` status, `InstallationVersion` wiring, `editorContext` (filePath/selection), `clipboard` provider.
- New missing components: `views/dialog_status.go`, `views/dialog_theme.go`, `views/dialog_workspace.go`, `ui/border.go` (`EmptyBorder/SplitBorder`), `ui/toast.go` (provider), `context/theme.go` (tint/selectedForeground), `util/format/locale.go`, `util/layout.go`, `util/collapse.go`, keymap system.
- Edge/render gaps: responsive overlay sidebar (when `!wide()` absolute `RGBA(0,0,0,70)` backdrop), spinner color `local.agent.color`, sticky scroll acceleration, terminal title (`OpenCode` vs `OC | {title}`), copy-on-select (right-click), paste handling (`decodePasteBytes`, `[Pasted ~N]`/`[Image N]` virtuals), mouse hover.

## Approaches

### 1) Pixel-perfect rewrite — full component-system port (High effort)
Re-implement OpenCode's TUI component graph in Bubble Tea idiomatically: new `Theme` with all fields + JSON loader for `packages/tui/src/theme/assets/*.json`, `ui/dialog` + `ui/dialog-select` generic (fuzzysort, grouping, actions), `keymap` package (namespaces/modes/leader), prompt with textarea+extmarks, session scrollbox with sticky scroll, per-part message models, sidebar/footer mirroring `sidebar.tsx/footer.tsx`.

- Pros: true parity, every spacing/border/color/typography matches; future OpenCode changes can be cherry-picked; fixes systemic gaps (toast, spinner, locale, layout) in one place.
- Cons: large surface (~30 files, new patterns like `kv`, `sync`, `local`), risk of re-introducing fakes if synthetic defaults fill missing stores; requires discipline not to fabricate (e.g., `InstallationVersion` only if `debug.ReadBuildInfo` present else omit).
- Effort: **High** (~10-14 focused work units / 6-9 days, auto-chain into 3-4 chained PRs). Needs visual dump verification per component (`session_dump.txt` method expanded).

### 2) Incremental patch — close only measurable deltas (Medium effort)
Keep current 4 list wrappers, fix literals→tokens, adjust widths (110→120, 30→42), replace home centering with flex-style spacers, split footer into welcome/connected variants with a 10s tick, expand Theme with 10 critical missing fields (`backgroundPanel/Element`, `markdown*`, `diff*Bg`), refine chat/tool/diff markdown without full part system, wrap existing palette in overlay `RGBA(0,0,0,150)` and add categories via `Command.Category`.

- Pros: lowest regression risk, reuses proven `tui-parity` wiring, ships quickly (2-4 days), still passes `parity_test` no-fabrication guards.
- Cons: leaves deep parity gaps (prompt extmarks/shell/mode, dialog-select grouping+footer actions, locale/tint math, workspace/permission flows, sticky scroll) — user asked "paridad total" will not be met; remaining fakes-risk if LSP/MCP counts stubbed.
- Effort: **Medium**.

### 3) Hybrid — phased minimal→full via migration toggle (Medium-High effort)
Ship Approach 2 as Phase 1 behind build tag or `KUI_EXPERIMENTAL_TUI_PARITY=1` env, then incrementally replace each component with Approach 1 equivalents (`ui/dialog` first, then `Prompt`, then `Session`), with parity goldens (`testdata/*` txt snapshots) updated per slice.

- Pros: visible progress without big-bang review; toggle protects mainline; each slice stays ≤400 lines (chained-PR friendly); visual-dump comparison can run per PR.
- Cons: toggle complexity, double-path rendering to maintain temporarily, slower total timeline.
- Effort: **Medium-High** but review-safe.

## Recommendation

**Approach 1 (full rewrite) sliced via Approach 3's delivery.** Justify: user explicitly said `quiero paridad total — cada aspecto visual, layout, interacción y edge case debe coincidir con la verdad de OpenCode`; incremental alone will be rejected in verification (golden diff will still diverge on sidebar/footer/prompt/markdown/theme coverage). The safe way to land a high-effort change within the 400-line review budget is **auto-chain** (feature branch + 3-4 stacked PRs):

1. **PR1 — Foundations**: `theme` expansion (all OpenCode fields + `tint/selectedForeground` + JSON loader), `ui/border` + `ui/dialog` primitives, `util/locale+format+layout` — purely additive, golden for theme resolution.
2. **PR2 — Shell & Home**: `Logo` shadow tint, `Home` flex spacers, `Prompt` textarea+maxHeight+placeholder pool+right slot, `Header/Footer` welcome tick — proves home pixel match.
3. **PR3 — Session**: `Chat` per-part with left `SplitBorder` + `Tool` collapse + `Diff` tree + `Sidebar` 42-col panel + `Session/Footer` permission/MCP/LSP dots — closes main parity delta.
4. **PR4 — Overlays & Keymap**: `DialogSelect` grouped palette/model/status/theme/workspace, `keymap` modes/leader, clipboard/selection, terminal title, toast/spinner — completes interaction parity and `ctrl+p` truthfulness.

Each PR carries its own `spec` slice (see Tasks) and updates `session_dump.txt` golden via text-dump method (render model → `View()` → write `testdata/*.txt`) so review does not require PNG viewing.

## Risks

- **Fabrication relapse**: any new `sync.data.*` store missing will tempt hardcoding `context7/engram/319k/mimo` again. Mitigation: `parity_test.go` guards expanded to grep for all forbidden literals plus new theme JSON names; `ThemeResolver` must return `nil`/empty when `provider/mcp/lsp` absent rather than synthetic counts.
- **Bubbles API vs OpenTUI divergence**: OpenCode uses `@opentui/core` `BoxRenderable/TextareaRenderable/ScrollBox` not `bubbles/list/textarea`. Lipgloss approximations can drift (border chars `┃` vs `│`, `╹` vs `└`, `▀/▄` shadows). Mitigation: use `lipgloss.Border{...}` custom chars matching `EmptyBorder/SplitBorder`, snapshot goldens per width (`80x24`, `120x30`, `160x40`).
- **Performance**: SolidJS signals + 60fps renderer vs Bubble Tea 30-60fps with `lipgloss.Width` per frame; heavy markdown highlight could jank. Mitigation: keep `HighlightCode` on fenced blocks only, cache `liveModels` TTL, debounce autocomplete filter.
- **Keymap completeness trap**: adding only `Tab/Ctrl+P/d` leaves `session.page.up/down`, `gd/gr/K`, `!` shell, `esc` modal stack uncovered — verification will flag interaction gap. Mitigation: explicit `keymap` spec table mapping OpenCode `appBindingCommands + sessionBindingCommands + promptCommands + dialog.select.*` to kui `tea.KeyMsg` cases.
- **Visual verification blindness** (assistant cannot view PNGs): `session_dump.txt` today is single 14-line dump; must expand to per-component `testdata/` txt renders (home, session empty, chat with message, tool pending/result, diff two-file, command palette, model list, status dialog) and assert in `verify-report.md` with `cat` text diffs.
- **Workspace/permission/editor slots have no kui backing**: full parity would require wiring `agent.NewManager` workspace, permission queue, editor RPC — if omitted the sidebar `WorkspaceLabel`/`PermissionPrompt` will be empty. Mitigation: mark as `NotAvailable` muted text, not fabricated; spec non-goal for this change with explicit follow-up `workspace-permission` delta.

## Unknowns to verify before proposal

- Exact `packages/tui/src/theme/assets/opencode.json` token values vs current `opencode.go` hexes (confirm via `open(ReadFile(opencode.json))` — two values may differ for `borderSubtle` etc.).
- `InstallationVersion` source in kui: `runtime/debug.ReadBuildInfo` Main.Version vs empty string — decide to omit footer `• Open Code <ver>` when empty (no fabrication).
- `sync.data.provider` population path in kui `run.go` (today only `profile.Loader` + `mcp.NewMCPManager`; no provider registry). Model discovery TTL cache currently per-provider HTTP fetch — verify rate limits and need for `sync.data.provider.length==0 → showPopularProviders` fallback (as OpenCode does).
- `config.keybinds.gather` equivalent in kui — currently hardcoded `tea.KeyCtrlP/CtrlC/Tab`; must adopt file `config/keybinds.yaml` or keep hardcoded map with `TuiKeybind.LeaderDefault` constant for fidelity.

## Ready for Proposal

**Yes** — topic is well-scoped (single TUI surface), comparison against truth is evidence-backed, and the phased chain mitigates review budget. Proposal should carry `delivery_strategy: auto-chain` with 4 slices, each ≤400 lines, and a `verify-report.md` checklist that uses text-dump comparison instead of image review.

## References

- kui TUI source: `internal/tui/app.go`, `theme/*.go`, `views/*.go`, `markdown/*`, `controller.go`, `run.go`, `session_dump.txt`
- OpenCode real TUI: `packages/tui/src/{app.tsx,logo.ts,theme/index.ts,keymap.tsx,routes/home.tsx,routes/session/{index,sidebar,footer}.tsx,component/{logo,prompt,command-palette,dialog-*.tsx},ui/{dialog,dialog-select,border}}`
- Prior SDDs: `archive/2026-08-19-tui-redesign/{proposal,design,explore}`, `archive/2026-08-19-tui-parity/{proposal,design,verify-report}`
- Theme assets: `packages/tui/src/theme/assets/opencode.json` (and 29 siblings)

