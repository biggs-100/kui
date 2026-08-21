```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:4a439e83a6d1e36e2ca55cc8aaaa5938271a381a258a8c6d1400ab21b9adf365
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 28/28
scenarios: 63/63
test_command: go test ./... -count=1
test_exit_code: 0
test_output_hash: sha256:0e12680a11f2ff147ce6d909c6ec5b440e1e208c636a8accbe97a1eebde48ab4
build_command: go build ./...
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

# Verification Report — tui-opencode-full-parity (full change, PR1–PR4 + bounded correction)

**Change**: `tui-opencode-full-parity` — full change verification (37/37 tasks)
**Mode**: `openspec` (artifact_store.mode=openspec) | **Branch**: `main` @ `9cd066c1c96f1afadd35ac7ac983db6f57cbc958` + uncommitted bounded correction in worktree
**Date**: 2026-08-21 | **Verifier**: orchestrator-run manual verification (fallback after `sdd_task_result_empty` transport failure of the dedicated sdd-verify sub-agent across three sessions; RDD disabled clone-local by user decision; native attempt ledger ordinal 2, objective `sdd-verify-final`)
**Verdict**: **PASS WITH WARNINGS**

> This report supersedes the stale PR1-only report previously stored at this path. It covers all six capability specs, all four delivered PRs, the final guard tasks, and the bounded narrow-overlay correction executed during this verification.

## Evidence revision derivation

`evidence_revision = SHA256(UTF-8 bytes of the HEAD commit id "9cd066c1c96f1afadd35ac7ac983db6f57cbc958")`. Deterministic and reproducible. The verified candidate tree is HEAD plus the uncommitted correction described in §3 (no source commits were made during verification).

## 1. Runtime Evidence (this session, exit codes real)

| Check | Command | Exit | Result |
|-------|---------|------|--------|
| Full test suite (post-correction) | `go test ./... -count=1` | 0 | 30 packages `ok`, 0 `FAIL`; slowest: cmd/kui ~14.8s, extensions/dynamic ~10.7s |
| Vet | `go vet ./internal/...` | 0 | empty output |
| Build | `go build ./...` | 0 | empty output |
| Golden determinism | `go test ./internal/tui/ -run TestAppGoldenLayout -count=1` ×2 | 0 / 0 | identical pass both runs |
| TUI package suite | `go test ./internal/tui/... -count=1` | 0 | 8/8 packages ok (tui 4.8s, views 1.3s) |

Output hashes: test `sha256:0e12680a11f2ff147ce6d909c6ec5b440e1e208c636a8accbe97a1eebde48ab4`; build output empty (`sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`).

Parity guards (`TestParityFooterNoFakes`, `TestParitySidebarNoFakes`, `TestParityModelCatalogNoFakes`, `TestParityNoHexLiteralsOutsideTheme`, `TestParityStylesUseTokens`): all PASS inside the full suite.

## 2. Spec Compliance Matrix (authoritative counts: 28 requirements / 63 scenarios)

Counts derived with the native heading regexes over the six delta spec files: tui-app 6r/12s, tui-chat 5r/11s, tui-dialog-overlay 4r/11s, tui-home 5r/11s, tui-theme-system 5r/10s, tui-tool-view 3r/8s.

| Capability | Reqs covered | Scenarios covered | Evidence highlights |
|------------|--------------|-------------------|---------------------|
| tui-app | 6/6 | **12/12** | contentWidth math (`TestAppContentWidth`), narrow sidebar overlay with backdrop (`TestAppNarrowSidebarOverlayRenders`, `TestAppNarrowSidebarOverlayWidthBounds`), layout goldens 80/120/160 (`TestAppGoldenLayout` + `testdata/app_*.txt`), keymap base/modal/leader, locale formatting |
| tui-chat | 5/5 | 11/11 | per-part ┃╹ borders via golden, markdown tokens chroma, tool collapse, sidebar 42 locale (`TestSidebarLocale42`, `TestFormatNumber`) |
| tui-dialog-overlay | 4/4 | 11/11 | DialogSelect grouped generic, palette/model/status/session dialogs, Esc stack, backdrop tokens, 4× dialog goldens @120 |
| tui-home | 5/5 | 11/11 | FlexSpacer centering + resize stability, logo Tint, prompt Split▀ 75@160/56@80, shell autocomplete, HomeGolden ×3 widths |
| tui-theme-system | 5/5 | 10/10 | 70-field theme, JSON round-trip, Discover later-dir override, Tint blend, token-only styles |
| tui-tool-view | 3/3 | 8/8 | collapse/highlight, diff ▶+N/-N modes, diff goldens lock two-file rendering |
| **TOTALS** | **28/28** | **63/63** | All requirements and scenarios hold passing runtime covering evidence |

Task completeness: 37/37 tasks checked in `tasks.md`, matching apply-progress TDD tables across PR1 Foundations (`9814c03`), PR2 Home (`1a58964`), PR3 Session (`cc2df54`), PR4 Overlays (`9cd066c`).

## 3. Bounded Correction (executed during verification, STRICT TDD red→green)

Verification surfaced that REQ-TUI-APP-2 "Narrow overlays sidebar" was not just untested but broken: both narrow-mode blocks in `internal/tui/app.go` built the sidebar/backdrop strings and discarded them (`_ =`) — the overlay never reached rendered output. Corrected within the native attempt budget (~233 changed lines < 400):

- **RED**: `TestAppNarrowSidebarOverlayRenders` failed — narrow dump contained no sidebar markers; `TestAppGoldenLayout` failed — no `testdata/app_*.txt` existed.
- **GREEN fix** (`internal/tui/app.go`): removed both dead blocks; extracted shared `newSidebarView()` (used by wide inline path unchanged); added `applySidebarOverlay()` drawing the 42-col sidebar over the rightmost columns with an `rgba(0,0,0,70)` backdrop-padded strip, total visible width == terminal width; title escape moved outside width math.
- **New tests**: `app_overlay_test.go` (overlay renders + width bounds), `app_golden_test.go` (`-update-app-golden` regeneration, byte-exact compare, per-line column ±1 tolerance).
- **Goldens written**: `internal/tui/testdata/app_80.txt`, `app_120.txt`, `app_160.txt`.
- Post-correction full suite green (30 pkgs), vet clean, build clean, gofmt clean on touched files.

## 4. Issues

### CRITICAL
None.

### WARNING
1. **W1 — gofmt not clean repo-wide**: `gofmt -l internal` flags 191 files; empirically ~156 are CRLF-only noise from `core.autocrlf=true`, but 35 files carry real alignment/wrapping diffs (16 production incl. several under internal/tui, 19 tests). Most predate this change's scope. Needs a dedicated normalization pass before any style-gated release.
2. **W2 — budget size:exception (acknowledged)**: per-PR changed lines exceed the 400 budget (PR1 2972, PR2 1411, PR3 1791, PR4 2353) under maintainer-approved `size:exception` slices with auto-chain feature-branch-chain and High forecast 1200. Recorded as accepted exception.
3. **W3 — deferred design utilities**: `util/layout.go` and `util/collapse.go` from design were deferred (PR3 deviation, documented in apply-progress). Functionality lives inline where used.

### SUGGESTION
1. **S1** TOOL-4: spec text names `diff_two_file.txt`; implementation ships `diff_80/120/160.txt`. Align spec wording or rename goldens.
2. **S2** Repo-wide EOL/formatting normalization (fixes W1 permanently).
3. **S3** The narrow-mode sidebar overlay currently always shows on session route ("demo" intent from original code); consider a toggle binding in a follow-up change.

## 5. Fabrication & Drift Checks

| Check | Result |
|-------|--------|
| `mimo` / `319k` / `context7` outside tests | ZERO hits (`git grep` over internal/tui excluding *_test.go) |
| Hex literals outside theme | Guard `TestParityNoHexLiteralsOutsideTheme` PASS; styles tokenized; new overlay code uses theme styles + documented rgba literal |
| Border glyph drift (┃╹▀ vs │└) | Exact-char tests PASS |

## 6. Verdict

**PASS WITH WARNINGS.** Every requirement (28/28) and every scenario (63/63) holds passing runtime covering evidence after the bounded narrow-overlay correction. Suite green and deterministic (30 packages, double-run goldens), vet/build clean, fabrication zero, parity guards green. The three warnings are recorded for follow-up; none breaks spec correctness.

Gate: change may proceed to **archive**. Recommended follow-ups (S1–S3) are post-archive improvements, not merge conditions.

---

*Generated via manual fallback verification (native attempt ordinal 2) after sdd-verify sub-agent transport failure; validated with `gentle-ai sdd-verify-validate --requirements 28 --scenarios 63` prior to persistence.*
