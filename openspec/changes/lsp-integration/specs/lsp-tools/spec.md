# lsp-tools Specification

## Purpose

Four `core.Tool` implementations that expose LSP capabilities to the agent: diagnostics query, hover info, go-to-definition, and find-references. Tools follow the same registration pattern as built-in and MCP tools (REQ-TOOLS-4).

## Requirements

### Requirement: REQ-LSP-TOOL-1 — lsp_diagnostics Tool

The `lsp_diagnostics` tool MUST return all diagnostics for a file specified by `file_path`. Input schema: `{ file_path: string }`. Output: JSON array of `{ severity, message, line, column, source }`.

#### Scenario: Diagnostics for file with errors

- GIVEN a file "main.go" with 2 errors and 1 warning
- WHEN lsp_diagnostics is called with `file_path: "main.go"`
- THEN the result contains 3 diagnostics with severity and message

#### Scenario: File not open

- GIVEN a file not currently open in the LSP
- WHEN lsp_diagnostics is called for that file
- THEN the tool returns an error: "file not open"

### Requirement: REQ-LSP-TOOL-2 — lsp_hover Tool

The `lsp_hover` tool MUST return hover information (type signature, documentation) at a given position. Input schema: `{ file_path: string, line: number, column: number }`. Output: JSON `{ contents: string, range: { start_line, start_col, end_line, end_col } }`.

#### Scenario: Hover over function

- GIVEN "main.go" with function `func Add(a, b int) int`
- WHEN lsp_hover is called at the position of "Add"
- THEN the result contains the function signature and any doc comment

#### Scenario: Hover over empty space

- GIVEN a position with no symbol
- WHEN lsp_hover is called
- THEN the tool returns `null` contents (empty hover)

### Requirement: REQ-LSP-TOOL-3 — lsp_definition Tool

The `lsp_definition` tool MUST return the location of a symbol's definition. Input schema: `{ file_path: string, line: number, column: number }`. Output: JSON `{ file_path: string, line: number, column: number }`.

#### Scenario: Go to function definition

- GIVEN "main.go" calling `Add(1, 2)` from "utils.go"
- WHEN lsp_definition is called at the "Add" reference in "main.go"
- THEN the result points to the definition in "utils.go" at the correct line

#### Scenario: Definition not found

- GIVEN a symbol with no definition (e.g., built-in)
- WHEN lsp_definition is called
- THEN the tool returns "definition not found"

### Requirement: REQ-LSP-TOOL-4 — lsp_references Tool

The `lsp_references` tool MUST return all references to a symbol. Input schema: `{ file_path: string, line: number, column: number }`. Output: JSON array of `{ file_path: string, line: number, column: number, is_definition: boolean }`.

#### Scenario: Find all references

- GIVEN function `Add` defined in "utils.go" and called in "main.go" and "test.go"
- WHEN lsp_references is called at the definition of `Add`
- THEN the result contains 3 entries (definition + 2 call sites)

#### Scenario: No references

- GIVEN a private function used only at its definition
- WHEN lsp_references is called
- THEN the result contains only the definition site

### Requirement: REQ-LSP-TOOL-5 — Tool Registration

All four LSP tools MUST be registered in the tool registry alongside built-in and MCP tools. LSP tools MUST have the `lsp_` prefix. Registration MUST follow REQ-TOOLS-4 patterns. Tools MUST NOT be registered if the LSP server is not available.

#### Scenario: LSP tools registered when gopls available

- GIVEN gopls is installed and LSP server can start
- WHEN the tool registry is queried
- THEN lsp_diagnostics, lsp_hover, lsp_definition, lsp_references are present

#### Scenario: LSP tools absent when gopls missing

- GIVEN gopls is not installed
- WHEN the tool registry is queried
-THEN the four lsp_ tools are NOT present

### Requirement: REQ-LSP-TOOL-6 — Error Responses

All LSP tools MUST return structured error responses when the operation fails. Errors MUST include a human-readable message and a machine-readable error code. The agent MUST NOT crash on LSP tool errors.

#### Scenario: Server not running

- GIVEN the LSP server has not started
- WHEN any lsp_ tool is called
- THEN the tool returns `{ error: "LSP server not running", code: "server_not_running" }`

#### Scenario: Symbol not found

- GIVEN a position with no symbol at that location
- WHEN lsp_hover or lsp_definition is called
- THEN the tool returns `{ error: "no symbol at position", code: "no_symbol" }`
