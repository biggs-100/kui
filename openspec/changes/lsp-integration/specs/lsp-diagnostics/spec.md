# lsp-diagnostics Specification

## Purpose

Push-based diagnostic cache that receives `textDocument/publishDiagnostics` notifications from the LSP server and provides query access by file or workspace. Supports severity mapping and TUI rendering.

## Requirements

### Requirement: REQ-DIAG-1 — Push-Based Diagnostic Cache

The system MUST maintain an in-memory cache of diagnostics keyed by file URI. The cache MUST be updated automatically when `publishDiagnostics` notifications arrive from the server. The cache MUST be thread-safe for concurrent reads from TUI and writes from LSP notifications.

#### Scenario: Diagnostics arrive for a file

- GIVEN an open file "main.go"
- WHEN gopls sends `publishDiagnostics` for "main.go" with 2 errors
- THEN the cache stores 2 diagnostics for "main.go"
- AND the diagnostics include severity, range, and message

#### Scenario: Diagnostics cleared for a file

- GIVEN a cache with diagnostics for "main.go"
- WHEN gopls sends `publishDiagnostics` for "main.go" with 0 diagnostics
- THEN the cache entry for "main.go" is cleared

#### Scenario: Concurrent access

- GIVEN a cache being written by LSP notifications
- WHEN the TUI reads diagnostics for display
- THEN no data race occurs (cache is mutex-protected)

### Requirement: REQ-DIAG-2 — Query by File

The system MUST provide a method to retrieve all diagnostics for a given file URI. The result MUST include severity, message, range (start line/col, end line/col), and source.

#### Scenario: Query file with diagnostics

- GIVEN diagnostics cached for "main.go" (1 error, 1 warning)
- WHEN diagnostics are queried for "main.go"
- THEN both diagnostics are returned with severity and range

#### Scenario: Query file with no diagnostics

- GIVEN no diagnostics for "utils.go"
- WHEN diagnostics are queried for "utils.go"
- THEN an empty slice is returned (not an error)

### Requirement: REQ-DIAG-3 — Query by Workspace

The system MUST provide a method to retrieve all diagnostics across all open files. The result MUST include a total count and breakdown by severity.

#### Scenario: Workspace-level query

- GIVEN diagnostics for 3 files: "main.go" (2 errors), "utils.go" (1 warning), "types.go" (0)
- WHEN workspace diagnostics are queried
- THEN the result includes total count of 3
- AND breakdown: 2 errors, 1 warning, 0 info, 0 hint

### Requirement: REQ-DIAG-4 — Severity Mapping

The system MUST map LSP diagnostic severities to four levels: error (1), warning (2), info (3), hint (4). The TUI MUST render each severity with a distinct indicator.

#### Scenario: All severity levels present

- GIVEN diagnostics with severities 1, 2, 3, and 4
- WHEN rendered in TUI
- THEN error shows "●" in red, warning shows "●" in yellow, info shows "●" in cyan, hint shows "●" in gray

### Requirement: REQ-DIAG-5 — TUI Inline Display

The chat view MUST render diagnostic annotations inline below affected lines in diff/file display. Diagnostics MUST NOT interrupt the main content flow. The footer MUST show a summary count of current diagnostics.

#### Scenario: Inline diagnostic rendering

- GIVEN a file with an error at line 15
- WHEN the file is displayed in TUI
- THEN an error annotation appears below line 15
- AND the annotation includes the severity indicator and message

#### Scenario: Footer diagnostic count

- GIVEN 3 errors and 2 warnings across open files
- WHEN the TUI footer renders
- THEN it displays "3 errors, 2 warnings"
