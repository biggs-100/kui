# Delta for agent-tools

## MODIFIED Requirements

### Requirement: REQ-TOOLS-4 — Registration Surface

Each tool MUST expose a stable name, a description, and a JSON parameter schema so the loop can advertise it to the provider. The registration surface MUST also accept MCP-contributed tools from the MCPManager. MCP tools MUST be registered alongside built-in tools, with names prefixed by server name.
(Previously: Registration surface only included built-in tools read_file, write_file, bash.)

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
