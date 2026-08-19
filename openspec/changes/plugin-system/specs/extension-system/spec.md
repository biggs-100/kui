# Delta for extension-system

## ADDED Requirements

### Requirement: REQ-EXT-7 — PluginManifest Integration

The system MUST support `PluginManifest` as the primary manifest format for extensions. Extensions loaded via `kui-plugin.yaml` MUST be treated as first-class plugins with full capability and permission metadata. The existing `Extension` interface remains unchanged.

#### Scenario: Extension loaded via kui-plugin.yaml

- GIVEN a directory with a valid `kui-plugin.yaml`
- WHEN the extension system discovers it
- THEN a PluginManifest is parsed and used for lifecycle management
- AND the extension is registered as a dynamic extension

#### Scenario: Extension loaded via extension.yaml (legacy)

- GIVEN a directory with only `extension.yaml`
- WHEN the extension system discovers it
- THEN a PluginManifest is constructed from legacy fields
- AND a deprecation warning is logged

### Requirement: REQ-EXT-8 — Capability Declarations

Each extension MAY declare capabilities in its manifest. The system MUST expose capabilities for querying but MUST NOT use them for access control. Capabilities are informational metadata.

#### Scenario: Query extension capabilities

- GIVEN an extension with capabilities ["tools:read", "hooks:on_turn_start"]
- WHEN capabilities are queried
- THEN the declared capabilities are returned

#### Scenario: Extension without capabilities

- GIVEN an extension manifest without a capabilities field
- WHEN capabilities are queried
- THEN an empty capability list is returned

### Requirement: REQ-EXT-9 — Permission Model Integration

The extension system MUST integrate with the permission model. Before spawning a dynamic extension subprocess, the system MUST verify that all declared permissions are granted. Denied permissions MUST block the spawn.

#### Scenario: Permission check before spawn

- GIVEN a dynamic extension requiring "network:outbound" permission
- WHEN the extension is about to be spawned
- THEN the permission system checks if "network:outbound" is granted
- AND the spawn proceeds only if permission is granted

#### Scenario: Permission denied blocks extension

- GIVEN a dynamic extension requiring "tools:write" permission
- WHEN the permission is denied
- THEN the extension subprocess is NOT spawned
- AND an error is logged with the extension name and denied permission
