# agent-tools Specification

## Purpose

Built-in tools let the agent read and write files and run shell commands. All tools run synchronously, are constrained to the workspace, and never accept interactive input.

## Requirements

### Requirement: REQ-TOOLS-1 — read_file

The read_file tool MUST return the full text content of an existing file resolved within the workspace root. Paths resolving outside the root MUST be rejected without reading.

#### Scenario: Read existing file

- GIVEN a file "notes.md" inside the workspace root
- WHEN read_file is called with path "notes.md"
- THEN the tool returns the file content

#### Scenario: Missing file

- GIVEN a path that does not exist
- WHEN read_file is called
- THEN the tool returns an error identifying the missing path

#### Scenario: Path escape rejected

- GIVEN a path resolving outside the workspace root, for example "../secret.txt"
- WHEN read_file is called
- THEN the tool returns a path-constraint error
- AND no file outside the root is read

### Requirement: REQ-TOOLS-2 — write_file

The write_file tool MUST create or overwrite the file at a path resolved within the workspace root and MUST report the path written. Paths resolving outside the root MUST be rejected without writing.

#### Scenario: Create new file

- GIVEN a path that does not exist
- WHEN write_file is called with content "hello"
- THEN the file is created with the given content
- AND the tool reports the written path

#### Scenario: Overwrite existing file

- GIVEN an existing file with old content
- WHEN write_file is called with new content
- THEN the file contains the new content

#### Scenario: Path escape rejected

- GIVEN a path resolving outside the workspace root
- WHEN write_file is called
- THEN the tool returns a path-constraint error
- AND no file is written

### Requirement: REQ-TOOLS-3 — bash

The bash tool MUST execute a command synchronously with a mandatory timeout and return stdout, stderr, and the exit code. It MUST NOT accept interactive input. When the command exceeds the timeout, the tool MUST terminate it and return a timeout error.

#### Scenario: Successful command

- GIVEN the command "echo hello"
- WHEN bash executes it
- THEN the tool returns exit code 0 and stdout "hello"

#### Scenario: Command timeout

- GIVEN a command that runs longer than the configured timeout, for example a 10s sleep with a 1s timeout
- WHEN bash executes it
- THEN the tool terminates the command and returns a timeout error

#### Scenario: Non-zero exit

- GIVEN a command that exits with code 1
- WHEN bash executes it
- THEN the tool returns exit code 1 and the captured stderr

### Requirement: REQ-TOOLS-4 — Registration Surface

Each tool MUST expose a stable name, a description, and a JSON parameter schema so the loop can advertise it to the provider. The registration surface MUST also accept MCP-contributed tools from the MCPManager and LSP tools from the LSPManager. MCP tools MUST be registered alongside built-in tools, with names prefixed by server name. LSP tools MUST be registered with the `lsp_` prefix, and MUST only be present when the LSP server is available.
(Previously: Registration surface only included built-in tools and MCP tools. LSP tools were not part of the registry.)

#### Scenario: Enumerate built-in tools

- GIVEN the default tool set
- WHEN the registration surface is queried
- THEN read_file, write_file, and bash are present, each with name, description, and schema

#### Scenario: MCP tools included

- GIVEN an MCPManager with connected server "github" providing tool "create_issue"
- WHEN the registration surface is queried
- THEN "github_create_issue" is present alongside built-in tools

#### Scenario: MCP tool name collision avoided

- GIVEN built-in tool "read_file" and MCP tool "read_file" from server "localfiles"
- WHEN both are registered
- THEN built-in "read_file" remains unchanged
- AND MCP tool is registered as "localfiles_read_file"

#### Scenario: LSP tools registered when available

- GIVEN an LSPManager with gopls running
- WHEN the registration surface is queried
- THEN lsp_diagnostics, lsp_hover, lsp_definition, lsp_references are present

#### Scenario: LSP tools absent when gopls unavailable

- GIVEN an LSPManager with gopls not installed
- WHEN the registration surface is queried
- THEN the four lsp_ tools are NOT present
- AND built-in and MCP tools remain unaffected

### Requirement: REQ-TOOLS-5 — glob

The glob tool MUST return a sorted list of file paths matching a `filepath.Match`-style pattern within the workspace root. It MUST accept `pattern` (string, required) and `path` (string, optional — subdirectory to search; defaults to workspace root). Paths resolving outside the workspace root MUST be rejected without reading.

#### Scenario: Valid recursive pattern

- GIVEN the workspace contains `src/main.go` and `src/utils/helper.go`
- WHEN glob is called with pattern `**/*.go`
- THEN the tool returns both paths sorted alphabetically

#### Scenario: No matches

- GIVEN the workspace contains no `.toml` files
- WHEN glob is called with pattern `**/*.toml`
- THEN the tool returns an empty list

#### Scenario: Path escape rejected

- GIVEN a path argument resolving outside the workspace root, for example `../../secret`
- WHEN glob is called
- THEN the tool returns a path-constraint error
- AND no files outside the root are read

### Requirement: REQ-TOOLS-6 — grep

The grep tool MUST return matching lines from files within the workspace root using Go `regexp` (RE2 syntax). It MUST accept `pattern` (string, required), `path` (string, optional), `include` (string, optional — file glob filter, e.g. `*.go`), and `max_results` (integer, optional, default 100). Paths resolving outside the workspace root MUST be rejected.

#### Scenario: Valid regex match

- GIVEN a file `main.go` containing the line `func main()`
- WHEN grep is called with pattern `func\s+\w+` and include `*.go`
- THEN the tool returns `main.go:1:func main()`

#### Scenario: No matches

- GIVEN the workspace contains Go files with no `TODO` comments
- WHEN grep is called with pattern `TODO` and include `*.go`
- THEN the tool returns an empty list

#### Scenario: Binary file skipped

- GIVEN the workspace contains `image.png`
- WHEN grep is called with pattern `.*`
- THEN the binary file is excluded from results

#### Scenario: Max results cap

- GIVEN 200 lines across files match the pattern
- WHEN grep is called with default max_results
- THEN the tool returns at most 100 results

#### Scenario: Path escape rejected

- GIVEN a path resolving outside the workspace root
- WHEN grep is called
- THEN the tool returns a path-constraint error
- AND no files outside the root are read

### Requirement: REQ-TOOLS-7 — web_fetch

The web_fetch tool MUST perform an HTTP GET and return the response body as text. It MUST accept `url` (string, required) and `format` (string, optional — `text` or `markdown`, default `text`). A mandatory 30-second timeout MUST apply to the entire request. The tool MUST NOT execute JavaScript or render dynamic content.

#### Scenario: Valid URL

- GIVEN a URL returning HTML with a `<title>` tag
- WHEN web_fetch is called
- THEN the tool returns the page text content

#### Scenario: Timeout

- GIVEN a URL that takes longer than 30 seconds to respond
- WHEN web_fetch is called
- THEN the tool returns a timeout error after 30 seconds
- AND no partial content is returned

#### Scenario: Invalid URL

- GIVEN a URL with an invalid scheme, for example `ftp://example.com`
- WHEN web_fetch is called
- THEN the tool returns a URL validation error

#### Scenario: Network error

- GIVEN a URL for an unreachable host
- WHEN web_fetch is called
- THEN the tool returns a network error describing the failure

### Requirement: REQ-TOOLS-8 — edit_file

The edit_file tool MUST replace exactly one text block in an existing file resolved within the workspace root, matched byte-exact (whitespace included) over content normalized to LF, and MUST report a unified diff of the change. Paths resolving outside the root MUST be rejected before any I/O. It MUST accept `path` (string, required), `old_text` (string, required, non-empty), `new_text` (string, required), and `occurrence` (integer, optional, 1-based; when absent the match MUST be unique).

#### Scenario: Exact replace with unified diff

- GIVEN a file "notes.md" containing "hello world\n"
- WHEN edit_file is called with old_text "world" and new_text "kui"
- THEN the file contains "hello kui\n"
- AND the result starts with a success line and contains a `diff --git` unified block

#### Scenario: No match returns guidance

- GIVEN a file "notes.md" containing "hello\n"
- WHEN edit_file is called with old_text "bye"
- THEN the tool returns an error stating 0 matches
- AND the error suggests reading the file with `read_file` and copying the exact text
- AND the file is left unmodified

#### Scenario: Ambiguous multi-match without occurrence

- GIVEN a file containing "foo" twice
- WHEN edit_file is called with old_text "foo" and no occurrence
- THEN the tool returns an ambiguity error reporting the match count
- AND the error asks for more context or an explicit occurrence
- AND the file is left unmodified

#### Scenario: Occurrence selects Nth match

- GIVEN a file containing "foo one\nfoo two\n"
- WHEN edit_file is called with old_text "foo" and occurrence 2
- THEN only the second occurrence is replaced

#### Scenario: Occurrence out of range

- GIVEN a file containing "foo" once
- WHEN edit_file is called with old_text "foo" and occurrence 3
- THEN the tool returns a range error
- AND the file is left unmodified

#### Scenario: Deletion via empty new_text

- GIVEN a file containing "line1\nremove me\nline3\n"
- WHEN edit_file is called with old_text "remove me\n" and new_text ""
- THEN the file contains "line1\nline3\n"
- AND the result diff shows the removal

#### Scenario: No-op rejected

- GIVEN any existing file
- WHEN edit_file is called with old_text equal to new_text
- THEN the tool returns a no-op error
- AND the file is left unmodified

#### Scenario: Missing file

- GIVEN a path that does not exist
- WHEN edit_file is called
- THEN the tool returns an error identifying the missing path

#### Scenario: Empty old_text rejected

- GIVEN any existing file
- WHEN edit_file is called with empty old_text
- THEN the tool returns a validation error
- AND the file is left unmodified

#### Scenario: Path escape rejected

- GIVEN a path resolving outside the workspace root, for example "../secret.txt"
- WHEN edit_file is called
- THEN the tool returns a path-constraint error
- AND no file outside the root is read or written

#### Scenario: Binary file rejected

- GIVEN a file with a NUL byte in its first 512 bytes
- WHEN edit_file is called
- THEN the tool returns a binary-file error
- AND the file is left unmodified

#### Scenario: CRLF preserved

- GIVEN a file with CRLF line endings and old_text expressed with LF
- WHEN edit_file is called
- THEN the replacement matches and the file keeps CRLF endings

#### Scenario: BOM preserved

- GIVEN a UTF-8 file with a BOM prefix
- WHEN edit_file is called
- THEN the replacement matches without the caller including the BOM
- AND the written file keeps the BOM

#### Scenario: Atomic write via tmp+rename

- GIVEN an existing file inside the workspace root
- WHEN edit_file succeeds
- THEN the new content is written via a temp file in the same directory plus rename
- AND no temp file remains after the operation
- AND the file mode is normalized to 0644 (the original mode is not preserved)

#### Scenario: LSP DidChange notified

- GIVEN an edit_file tool built with a non-nil syncer
- WHEN edit_file succeeds
- THEN the syncer receives DidChange with the file URI and the new content
