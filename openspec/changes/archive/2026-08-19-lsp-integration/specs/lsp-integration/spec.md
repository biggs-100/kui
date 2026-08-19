# lsp-integration Specification

## Purpose

Integration requirements for wiring the LSP subsystem into the kui runtime: composition root, TUI display, keybindings, and lazy startup. Covers how the LSP client, diagnostics, and tools connect to the existing application.

## Requirements

### Requirement: REQ-INT-1 — Runtime Composition

The runtime MUST instantiate an `LSPManager` during `Build` and pass it to tool registration, TUI chat view, and file sync hooks. The manager MUST be available in `Reload` (restart LSP for new workspace) and `Close` (stop LSP server).

#### Scenario: Build wires LSP

- GIVEN a runtime Build call
- WHEN the composition root initializes
- THEN LSPManager is created and registered with the tool registry
- AND the TUI chat view receives a diagnostics reference

#### Scenario: Reload restarts LSP

- GIVEN a running LSP server in workspace A
- WHEN Reload switches to workspace B
- THEN the LSP server restarts with workspace B as rootUri
- AND diagnostics cache is cleared

#### Scenario: Close stops LSP

- GIVEN a running LSP server
- WHEN the runtime shuts down
- THEN the LSP server is stopped gracefully

### Requirement: REQ-INT-2 — Lazy Startup

LSP server startup MUST be lazy — the server is NOT started at Build time. The server MUST start on first LSP tool call or first diagnostic query. Startup MUST be non-blocking for the TUI (background goroutine). The TUI MUST remain responsive during startup.

#### Scenario: First LSP tool call triggers startup

- GIVEN the LSP server not yet started
- WHEN the agent calls `lsp_hover`
- THEN the LSP server starts in the background
- AND the tool call blocks until the handshake completes
- AND the TUI remains responsive

#### Scenario: TUI responsive during startup

- GIVEN the LSP server starting in background
- WHEN the user types in the chat input
- THEN input is accepted normally
- AND the TUI does not freeze

### Requirement: REQ-INT-3 — TUI Footer Display

The TUI footer MUST display the current LSP server status (stopped, starting, running, error) and a diagnostic summary count. The footer MUST update when status or diagnostics change.

#### Scenario: Footer shows LSP status

- GIVEN the LSP server is running
- WHEN the TUI renders
- THEN the footer shows "LSP: running" with diagnostic count

#### Scenario: Footer shows error state

- GIVEN the LSP server crashed
- WHEN the TUI renders
- THEN the footer shows "LSP: error" and no diagnostic count

### Requirement: REQ-INT-4 — Keybindings

The TUI MUST support keybindings for LSP navigation: `gd` for go-to-definition, `gr` for find-references, `K` for hover info. Keybindings MUST operate on the cursor position in the current file view. When no symbol is at the cursor, the keybinding MUST show a "no symbol" message.

#### Scenario: Go to definition

- GIVEN the cursor on a function name in file view
- WHEN the user presses `gd`
- THEN the view navigates to the definition file and line

#### Scenario: Find references

- GIVEN the cursor on a symbol
- WHEN the user presses `gr`
- THEN a reference list is displayed

#### Scenario: Hover info

- GIVEN the cursor on a symbol
- WHEN the user presses `K`
- THEN the type signature and docs appear in a popup or inline

#### Scenario: No symbol at cursor

- GIVEN the cursor on empty space
- WHEN any LSP keybinding is pressed
- THEN a "no symbol at cursor" message is shown

### Requirement: REQ-INT-5 — File Sync Hooks

The `write_file` and `read_file` tools MUST trigger LSP file sync notifications. `write_file` MUST send `didOpen` (if new) or `didChange` (if existing). `read_file` MUST send `didOpen` if the file is not yet tracked. `didClose` MUST fire when a file is evicted from the tracked set.

#### Scenario: write_file triggers didChange

- GIVEN an open tracked file "main.go"
- WHEN write_file updates "main.go" with new content
- THEN `textDocument/didChange` is sent to gopls
- AND the diagnostics cache may update

#### Scenario: read_file triggers didOpen

- GIVEN a file "utils.go" not yet tracked
- WHEN read_file reads "utils.go"
- THEN `textDocument/didOpen` is sent to gopls
- AND the file is now tracked for sync

### Requirement: REQ-INT-6 — Multi-Workspace (Future)

Multi-workspace support (multiple gopls instances for different modules) is OUT OF SCOPE for MVP. The system MUST be designed so that adding multi-workspace later does not require restructuring the LSPManager interface. A single gopls instance per workspace root is sufficient for MVP.

#### Scenario: MVP single workspace

- GIVEN a workspace with one go.mod
- WHEN the LSP server starts
- THEN one gopls instance runs for that workspace root

#### Scenario: Future multi-workspace extension point

- GIVEN the LSPManager interface
- WHEN multi-workspace is implemented later
- THEN the interface supports multiple server instances without breaking changes
