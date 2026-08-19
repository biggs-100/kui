# Delta for agent-tools

## ADDED Requirements

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
