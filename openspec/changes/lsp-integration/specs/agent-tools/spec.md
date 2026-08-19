# agent-tools Delta for lsp-integration

## MODIFIED Requirements

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
