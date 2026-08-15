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

Each tool MUST expose a stable name, a description, and a JSON parameter schema so the loop can advertise it to the provider.

#### Scenario: Enumerate built-in tools

- GIVEN the default tool set
- WHEN the registration surface is queried
- THEN read_file, write_file, and bash are present, each with name, description, and schema
