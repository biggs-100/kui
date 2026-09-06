# tui-home Specification

## Purpose

> RETIRED by `redo-tui-pi-style` (2026-09-06). The home screen capability was DELETED: pi has no home screen and the app always starts in the transcript view. First-run shows the vacant transcript only (editor + footer, no inline changelog — see REQ-TUI-APP-1). This file is kept as a retirement record; it defines NO active requirements.

## Requirements

_No active requirements. All former requirements below were removed._

## REMOVED Requirements

### Requirement: REQ-TUI-HOME-1..7 — Entire Home Capability (Centered Layout, ASCII Logo, Bordered Prompt, Minimal Footer, Prompt Submission, Keyboard Shortcuts, Header Suppression/Shell Mode)

(Reason: pi has no home screen; the app always starts in the transcript view)
(Migration: `Enter` submits from the transcript editor; `Ctrl+P` palette and `Ctrl+C` quit keep working from transcript; logo/home prompt/home footer variants have no replacement)

#### Scenario: No home renders

- GIVEN fresh start with empty history
- WHEN `kui tui` starts
- THEN no logo-centered screen appears; vacant transcript (editor + footer) renders

#### Scenario: Narrow fresh start survives

- GIVEN width 60 with empty history
- WHEN dumped
- THEN transcript + editor + footer render without panic

## Destructive Impact (recorded at archive)

- Deleted: `views/home.go`, `views/home_prompt.go`, `views/home_footer.go`, `views/logo.go`; deleted `home_*` goldens and home route tests.
