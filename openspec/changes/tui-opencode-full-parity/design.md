# Design: TUI OpenCode Full Parity

## Technical Approach

Extend Bubble Tea+lipgloss (no OpenTUI). `Theme` source (40+ fields+Tint+JSON). `ui/border`+`dialog` give `┃╹▀▶`. 4 PRs ≤400 lines; dump goldens `View()→txt`; `parity_test.go` guards hex.
Spec→PR: theme→1, home→2, app+chat+tool→3, dialog+keymap→4.

| PR | Slice | Contains | Golden |
|----|-------|----------|--------|
| 1 | Foundations | `theme/*`+`ui/border+dialog`+`util/locale`+keymap scaffold | theme parse |
| 2 | Home | `logo/home/prompt/header/footer` flex 75/70% | home 80/120/160 |
| 3 | Session | `chat(┃╹) tool collapse diff sidebar42 footer tick` | session/diff |
| 4 | Overlays | `DialogSelect palette/model/status keymap base/modal toast/title` | palette/model |

## Architecture Decisions

| Decision | Alt | Trade | Choice |
|---|---|---|---|
| Theme | extend vs new | new dupes | Extend 15 fields keep `ParseBytes` |
| Tint | cached vs func | hides math | `Tint(bg,fg,0.25)` testable |
| Borders | Normal vs const | drifts `│` | `SplitBorder{┃╹}+▀` goldens 80/120/160 |
| Dialogs | sep vs generic | dupe fuzzy | `DialogSelect[T]` weighted `title*2+cat` |
| Home | topPad vs flex | flickers | Flex `max(1,(h-c)/2)+h4` |
| Markdown | chroma vs token | ignores tokens | Token+chroma `markdown*` |
| Keymap | scattered vs registry | misses | `base/modal+leader` |
| Verify | png vs dump | needs viewer | Dump `-update` |

## Data Flow

```
tea.Msg → App.Update → keymap[base/modal] → Input/Dialog → registry → Controller
       → App.View → Styles → border
         ├ Home: Logo Tint → Prompt 75/70% pool ! → flex → toast
         ├ Session: w>120?42:overlay70 cw=w-42? -4 Header Chat(┃╹) Tool Diff(▶) Sidebar Input Footer(tick•⊙△)
         └ Dialog RGBA150 60/88/116 Place Center bgMenu
```

`w>120` else overlay. `WindowSizeMsg→rebuildViews`.

## File Changes

| File | Act | What |
|------|-----|------|
| `theme/theme.go` | M | +Panel/Element/Menu,Selected,Markdown8,SyntaxOp,Diff*Bg,Hl,LineNum,Opacity |
| `theme/opencode.go` | M | hexes from `assets/opencode.json` |
| `theme/styles.go` | M | `#2a2a2a→Element #569cd6→Primary #252525→Element #e0af68→Warning` |
| `theme/tint.go` | N | `Tint,SelectedForeground,GetSyntaxRules,BuildChromaStyle` |
| `ui/border.go` | N | `EmptyBorder,SplitBorder(┃╹),▀` |
| `ui/dialog.go` | N | `Dialog 60/88/116 RGBA150 Place Center h/4 modal Esc` |
| `ui/dialog_select.go` | N | `Item DialogSelect[T] fuzzy76 bgMenu` |
| `util/locale.go` | N | `FormatNumber/Money,TodayTime,FormatDuration` |
| `util/layout.go` | N | `ContentWidth,IsWide>120,FlexSpacer,TruncateMiddle` |
| `util/collapse.go` | N | `CollapseOutput` |
| `keymap/keymap.go` | N | `Layer,Binding,FormatKeyBindings` |
| `views/logo.go` | M | `█▀▀█` pairs `Tint(0.25)` |
| `views/home.go` | M | Flex 75/70% |
| `views/home_prompt.go` | M | `w*70/100 cap75 SplitBorder+▀ maxH max(6,h/3) ! extmarks` |
| `views/header.go` | M | Gap `TabActiveBG` hide home |
| `views/footer+session_footer.go` | M/N | Home empty; Session `•⊙△` vs `welcome` tick nil→omit |
| `views/sidebar.go` | M | 42 locale `tokens% $` `title/ID/share` ver if `ReadBuildInfo` |
| `views/chat.go` | M | `Part{Kind Queued Hover} ┃╹` per part |
| `markdown/renderer.go` | M | `markdown*` + `HighlightCode` |
| `views/tool.go` | M | `showDetails Collapse diffBg` |
| `views/diff.go` | M | `▶ +N/-N lineNumBg word/none` |
| `views/*list,command_palette.go` | M | `Dialog+Select 76 Esc filter→close nano disabled` |
| `views/dialog_status/theme/workspace.go` | N | dots `success/error` picker muted |
| `app.go` | M | `>120`42 overlay header input popup toast/title |
| `controller.go` | M | `SyncData nil→omit` kv |
| `autocomplete/commands/input.go` | M | slash `/model /theme` `!` `●File` `session.*` |
| `testdata/*.txt` | N | goldens 80/120/160 |
| `views/parity_test.go` | M | guard `#[0-9a-f]{6}` + `319k/mimo` 40+ round-trip |

## Interfaces / Contracts

```go
type Theme struct{
 Background,BackgroundPanel,BackgroundElement,BackgroundMenu string
 SelectedListItemText,Border,BorderActive,BorderSubtle string
 DiffAddedBg,DiffRemovedBg,DiffContextBg,DiffLineNumberBg,DiffHighlight string
 MarkdownText,MarkdownHeading,MarkdownLink,MarkdownCode,MarkdownBlockQuote string
 SyntaxComment,SyntaxKeyword,SyntaxFunction,SyntaxString,SyntaxType,SyntaxOperator string
 ThinkingOpacity float64
}
func Tint(bg,fg string,a float64) string
var SplitBorder=lipgloss.Border{Left:"┃",Bottom:"╹"}
type Dialog struct{Size int} //60/88/116
type Item struct{Title,Category,Detail,Value string;Disabled bool}
type DialogSelect[T any] struct{} //title*2+cat truncate76
type Part struct{Kind PartKind;Queued,Hover bool}
```

## Testing Strategy

| Layer | Check | How |
|-------|-------|-----|
| Unit | Tint/Parse/Truncate/locale | table `t.Run` |
| Unit | 40+ hex guard Discover `t.TempDir` | `parity_test.go` |
| Unit | Dialog fuzzy grouping Esc nano | `Update`+`View()` |
| Golden | Home 75/70% logo `┃╹` tool diff dialog 80/120/160 | dump `-update` |
| Integ | wide>120 overlay header footer | `teatest` `Short()` |
| Contract | slash `!` leader spinner | table `!`/`●` |

`go test ./internal/tui -run TestTint` → `go test ./...` → `-update && git diff`.

## Threat Matrix

N/A — `!` muted display only; no `exec/VCS/PR` boundary. Future shell→ reload matrix.

## Risks

| Risk | Mitigate→Task |
|------|---------------|
| Fabrication `mimo` | grep nil→omit →PR1 guard |
| Char drift `┃╹` | consts+goldens →PR1 `Left=="┃"` |
| Keymap gaps | table two-step Esc →PR4 |
| Jank | fences TTL debounce →PR3 bench |

## Migration / Rollout

No migration. Branch `feat/tui-opencode-full-parity` PR1→4 `revert` safe `test/vet/gofmt`.

## Open Questions

- [ ] `ReadBuildInfo` else omit ver
- [ ] `sync.data` nil→muted ok; follow-up `workspace-permission`
- [ ] Verify `opencode.json` hexes pre-PR1
