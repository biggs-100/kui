# tool-filtering Specification

## Purpose

CLI flags that restrict the tool set available to the agent loop. Allowlist (`--tools`), denylist (`--exclude-tools`), and full-disable (`--no-tools`) control which tools the loop may invoke.

## Requirements

### Requirement: REQ-CLI-14 — --tools Allowlist

The CLI MUST accept `--tools, -t <list>` as a comma-separated list of tool names. When specified, only the named tools MUST be available to the loop. All other tools from `tools.Default()` MUST be excluded.

#### Scenario: Single tool allowlisted

- GIVEN `--tools read_file` and tools including `read_file`, `write_file`, `bash`
- WHEN the tool filter is applied
- THEN only `read_file` is available to the loop

#### Scenario: Multiple tools allowlisted

- GIVEN `--tools read_file,write_file`
- WHEN the tool filter is applied
- THEN only `read_file` and `write_file` are available

#### Scenario: Short flag -t

- GIVEN args `["-t", "bash"]`
- WHEN the CLI parses flags and applies the filter
- THEN only `bash` is available

### Requirement: REQ-CLI-15 — --exclude-tools Denylist

The CLI MUST accept `--exclude-tools, -xt <list>` as a comma-separated list of tool names. Named tools MUST be removed from the default tool set. Unnamed tools remain available.

#### Scenario: Single tool excluded

- GIVEN `--exclude-tools bash` and tools including `read_file`, `bash`
- WHEN the tool filter is applied
- THEN `bash` is not available and `read_file` remains

#### Scenario: Multiple tools excluded

- GIVEN `--exclude-tools bash,write_file`
- WHEN the tool filter is applied
- THEN `bash` and `write_file` are unavailable

### Requirement: REQ-CLI-16 — --no-tools Full Disable

The CLI MUST accept `--no-tools, -nt` as a boolean flag. When set, the tool registry MUST be empty — no tools are available to the loop.

#### Scenario: --no-tools flag

- GIVEN `--no-tools`
- WHEN the tool filter is applied
- THEN the tool registry is empty

#### Scenario: Short flag -nt

- GIVEN args `["-nt"]`
- WHEN the CLI parses flags and applies the filter
- THEN the tool registry is empty

### Requirement: REQ-CLI-17 — Exclude Wins Over Include

When both `--tools` and `--exclude-tools` are specified, the exclude list MUST take precedence. A tool appearing in both lists MUST NOT be available.

#### Scenario: Tool in both lists

- GIVEN `--tools read_file,bash --exclude-tools bash`
- WHEN the tool filter is applied
- THEN `read_file` is available and `bash` is NOT

#### Scenario: Exclude superset of include

- GIVEN `--tools read_file --exclude-tools read_file,write_file`
- WHEN the tool filter is applied
- THEN no tools are available

### Requirement: REQ-CLI-18 — Filtering at CLI Layer

Tool filtering MUST be applied at the CLI layer after `runtime.Build()` returns the full tool set. The filter MUST NOT modify the runtime or tool registration internals.

#### Scenario: Filter applied post-build

- GIVEN a runtime with tools built via `runtime.Build()`
- WHEN `--tools read_file` is specified
- THEN the filtered tool set is passed to the loop
- AND `runtime.Build()` was called with the full default set
