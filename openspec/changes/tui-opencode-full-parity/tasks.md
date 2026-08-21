# Tasks: TUI OpenCode Full Parity

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Lines | 1200 |
| Risk | High |
| Chained | Yes |
| Split | PR1→PR2→PR3→PR4 |
| Delivery | auto-chain |
| Chain | feature-branch-chain |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | PR | Test | Harness | Rollback |
|------|------|----|------|---------|----------|
| 1 | Theme 40+ | PR1 | `go test theme` | `go vet` | `theme/*` |
| 2 | Dialog | PR1 | `go test ui` | `go vet` | `ui/*` |
| 3 | Home | PR2 | `go test HomeGolden` | `cat home*.txt` | `views/home*` |
| 4 | Session | PR3 | `go test Chat` | `cat chat*.txt` | `views/chat*` |
| 5 | Overlays | PR4 | `go test Dialog` | `cat dialog*.txt` | `ui/dialog_select` |

Chain tracker `feat/tui-opencode-full-parity` draft PR1→tracker PR2→PR1 PR3→PR2 PR4→PR3 `📍` clean diff.

## Phase 1: Foundations PR1

- [x] 1.1 `internal/tui/theme/theme.go` 40+ [REQ-TUI-THEME-1] S
- [x] 1.2 `internal/tui/theme/tint.go` Tint 0.25 [REQ-TUI-THEME-2] 1.1-S
- [x] 1.3 `internal/tui/theme/loader.go` Parse Discover [REQ-TUI-THEME-3] 1.1-S
- [x] 1.4 `internal/tui/theme/styles.go` hex→tokens [REQ-TUI-THEME-4] 1.1-S
- [x] 1.5 `internal/tui/views/parity_test.go` guard [REQ-TUI-THEME-4] 1.4-S
- [x] 1.6 `internal/tui/ui/border.go` Split ┃╹ ▀ [REQ-TUI-APP-8] S
- [x] 1.7 `internal/tui/ui/dialog.go` 60/88/116 modal [REQ-TUI-DLG-1] 1.6-S
- [x] 1.8 `internal/tui/util/locale.go` FormatNumber [REQ-TUI-APP-9] S
- [x] 1.9 `internal/tui/keymap/keymap.go` base/modal [REQ-TUI-APP-10] S
- [x] 1.10 verify `go test ./internal/tui/theme,ui` `go vet` `gofmt`

## Phase 2: Home PR2

- [x] 2.1 `internal/tui/views/logo.go` █▀▀█ Tint [REQ-TUI-HOME-2] 1.2-S
- [x] 2.2 `internal/tui/views/home.go` flex 75/70% [REQ-TUI-HOME-1] 1.8-S
- [x] 2.3 `internal/tui/views/home_prompt.go` Split▀ pool ! [REQ-TUI-HOME-3] 2.2-M
- [x] 2.4 `internal/tui/views/session_footer.go` tick •⊙ [REQ-TUI-HOME-4] 1.8-S
- [x] 2.5 `internal/tui/views/header.go` gap TabActiveBG S
- [x] 2.6 `internal/tui/app.go` wide>120 overlay title [REQ-TUI-APP-2] S
- [x] 2.7 goldens `testdata/home_*.txt` 80/120/160 S
- [x] 2.8 verify `go test -run TestHome` `stat`≤400

## Phase 3: Session PR3

- [x] 3.1 `internal/tui/views/chat.go` Part ┃╹ QUEUED [REQ-TUI-CHAT-2] 1.6-M
- [x] 3.2 `internal/tui/markdown/renderer.go` tokens chroma [REQ-TUI-CHAT-4] 1.2-S
- [x] 3.3 `internal/tui/views/tool.go` Collapse highlight [REQ-TUI-TOOL-1] 1.8-S
- [x] 3.4 `internal/tui/views/diff.go` ▶ +N/-N word/none [REQ-TUI-TOOL-3] 1.6-S
- [x] 3.5 `internal/tui/views/sidebar.go` 42 locale [REQ-TUI-APP-2] 1.8-S
- [x] 3.6 `internal/tui/controller.go` nil→omit kv 2.4-S
- [x] 3.7 goldens `testdata/chat_*.txt` `diff_*.txt` S
- [x] 3.8 verify `go test Chat|Tool|Diff` `go vet`

## Phase 4: Overlays PR4

- [x] 4.1 `internal/tui/ui/dialog_select.go` title*2 truncate76 [REQ-TUI-DLG-2] 1.7-M
- [x] 4.2 `internal/tui/views/command_palette.go` suggested hidden [REQ-TUI-DLG-3] 4.1-S
- [x] 4.3 `internal/tui/views/model_list.go` nano disabled ● [REQ-TUI-DLG-3] 4.1-S
- [x] 4.4 `internal/tui/views/dialog_status.go` • success/error [REQ-TUI-DLG-3] 4.1-S
- [x] 4.5 `internal/tui/views/session_list.go` 76 Esc [REQ-TUI-DLG-4] 4.1-S
- [x] 4.6 `internal/tui/autocomplete.go` /model ! ●File 1.9-M
- [x] 4.7 `internal/tui/app.go` base→modal Esc 4.2-S
- [x] 4.8 goldens `testdata/dialog_*.txt` 120 S
- [x] 4.9 verify `go test ./...` `stat`≤400

## Phase 5: Guard

- [x] 5.1 per-PR `stat`<400 test vet fmt — PR1 9814c03 2790+182=2972, PR2 1a58964 1039+372=1411, PR3 cc2df54 1596+195=1791, PR4 9cd066c 2059+294=2353 — all `size:exception` High risk forecast 1200, mitigation feature-branch-chain; `go test -run TestParity` 5 PASS, `go vet ./internal/tui/...` clean, `gofmt -l` clean (0 listings)
- [x] 5.2 final `go test -short` parity pass — `go test ./internal/tui/... -count=1 -short` 8 pkgs PASS (tui 4.97s views 1.54s theme 0.96s ui 0.98s util 0.94s markdown 1.29s keymap 0.84s toast 0.86s), parity_test bans hex `#2a2a2a/#252525/#569cd6/#e0af68` + `#[0-9a-f]{6}` outside theme + mimo/319k nil→omit, goldens 13 txt present (home 80/120/160 chat 80/120/160 diff 80/120/160 dialog 4×120), `go build -o kui.exe ./cmd/kui` 18059264 bytes PASS
