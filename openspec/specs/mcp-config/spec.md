# mcp-config Specification

## Purpose

Configuration for MCP servers that kui should connect to. Supports global and project-level config files with merge semantics.

## Requirements

### Requirement: REQ-MCP-1 — Config File Locations

The system MUST load MCP configuration from two locations: global `~/.config/kui/mcp.yaml` and project-level `.kui/mcp.yaml`. Both files are optional. If neither exists, MCP is disabled.

#### Scenario: Global config only

- GIVEN a global `~/.config/kui/mcp.yaml` with one server
- AND no project-level `.kui/mcp.yaml`
- WHEN config is loaded
- THEN the global server is present in the merged config

#### Scenario: Project overrides global

- GIVEN a global config with server "github" command "gh"
- AND a project config with server "github" command "gh-project"
- WHEN config is loaded
- THEN the merged config has server "github" with command "gh-project"

#### Scenario: No config files

- GIVEN neither global nor project config exists
- WHEN config is loaded
- THEN MCP is disabled (empty server list)

### Requirement: REQ-MCP-2 — Config Format

The config file MUST be YAML with a top-level `servers` map. Each entry key is the server name. Each entry MUST have `type: local` for stdio-based servers.

#### Scenario: Valid local server entry

- GIVEN a config file with `servers: { myserver: { type: local, command: ["echo"] } }`
- WHEN config is parsed
- THEN the server "myserver" is present with type "local"

#### Scenario: Unknown type rejected

- GIVEN a config file with `type: remote`
- WHEN config is parsed
- THEN an error is returned for the unknown type

### Requirement: REQ-MCP-3 — Server Config Fields

Each server config MUST support: `command` (string array, required), `cwd` (string, optional), `environment` (map, optional), `disabled` (bool, optional default false), `timeout` (duration, optional default 30s).

#### Scenario: Minimal config

- GIVEN a server with only `command: ["node", "server.js"]`
- WHEN config is parsed
- THEN defaults apply: cwd is working directory, no extra env, enabled, 30s timeout

#### Scenario: Full config

- GIVEN a server with command, cwd, environment, disabled, timeout specified
- WHEN config is parsed
- THEN all fields are preserved exactly as specified

### Requirement: REQ-MCP-4 — Config Merge Semantics

When both global and project configs define the same server name, the project config MUST win entirely (not field-by-field merge). Servers only in one file are included as-is.

#### Scenario: Server only in global

- GIVEN global config has server "a" and project has no "a"
- WHEN merged
- THEN server "a" from global is present

#### Scenario: Server only in project

- GIVEN project config has server "b" and global has no "b"
- WHEN merged
- THEN server "b" from project is present

#### Scenario: Server in both — project wins

- GIVEN global server "c" with command ["x"] and project server "c" with command ["y"]
- WHEN merged
- THEN server "c" has command ["y"] (project's entire entry)
