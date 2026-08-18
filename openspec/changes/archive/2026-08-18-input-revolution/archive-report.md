# Archive: input-revolution

## Summary

Replaced kui's raw string input with a full-featured text editor using `charmbracelet/bubbles/textarea`. Added input history (JSONL persistence, up/down navigation), slash-command autocomplete with fuzzy matching, and clipboard paste support. All 25 tasks completed, 79/79 tests passing, delivered via 2 stacked PRs.

## Artifacts

- Proposal: `openspec/changes/input-revolution/proposal.md`
- Design: `openspec/changes/input-revolution/design.md`
- Tasks: `openspec/changes/input-revolution/tasks.md`

## Implementation

### Files Created
- `internal/tui/input.go` — `InputModel` wrapping `textarea.Model` with history + autocomplete
- `internal/tui/history.go` — `InputHistory` JSONL persistence, dedup, 50-entry limit
- `internal/tui/autocomplete.go` — `AutocompleteModel` fuzzy slash-command matching
- `internal/tui/input_test.go` — Unit tests for InputModel, history, paste
- `internal/tui/history_test.go` — Unit tests for history load/save/dedup/trim
- `internal/tui/autocomplete_test.go` — Unit tests for filter, navigation, dismiss

### Files Modified
- `internal/tui/app.go` — Replaced `input string` with `InputModel`, rewrote key handling
- `internal/tui/app_test.go` — Updated tests for new input model
- `go.mod` / `go.sum` — Added `charmbracelet/bubbles` dependency

### Tests
- 79/79 passing

## Verification

- Requirements: 25/25 tasks complete
- Tests: 79/79 passing
- Coverage: 66.2% (internal/tui), 93.8% (views)
- Warnings: None

## PRs

1. InputModel + History (PR 1)
2. Autocomplete + App integration (PR 2)
