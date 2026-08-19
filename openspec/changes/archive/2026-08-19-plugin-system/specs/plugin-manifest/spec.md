# Plugin Manifest Specification

## Purpose

Unified manifest format (`kui-plugin.yaml`) for declaring plugin metadata, type, entry point, capabilities, and permissions. Supersedes the legacy `extension.yaml` format while maintaining backward compatibility.

## Requirements

### Requirement: REQ-PMAN-1 — Manifest Format

Plugins MUST declare metadata in a `kui-plugin.yaml` file with the following required fields: `name` (string), `version` (string, semver format), `type` (enum), `entry_point` (string). Optional fields: `description`, `capabilities`, `permissions`, `protocol_version`.

#### Scenario: Valid manifest with required fields

- GIVEN a `kui-plugin.yaml` with name, version, type, and entry_point
- WHEN the manifest is parsed
- THEN a valid PluginManifest is returned

#### Scenario: Missing required field

- GIVEN a `kui-plugin.yaml` missing the `version` field
- WHEN the manifest is parsed
- THEN a validation error is returned indicating the missing field

#### Scenario: Invalid version format

- GIVEN a `kui-plugin.yaml` with version "not-a-version"
- WHEN the manifest is parsed
- THEN a validation error indicates invalid semver format

### Requirement: REQ-PMAN-2 — Plugin Types

The manifest `type` field MUST be one of: `tool`, `hook`, `command`, `theme`, `skill`. Each type determines how the plugin is loaded and what registration methods it may use.

#### Scenario: Tool type plugin

- GIVEN a manifest with type "tool"
- WHEN the plugin is loaded
- THEN it is expected to register tools via RegisterTool

#### Scenario: Invalid type rejected

- GIVEN a manifest with type "unknown"
- WHEN the manifest is validated
- THEN a validation error is returned

### Requirement: REQ-PMAN-3 — Capability Declarations

The manifest MAY declare capabilities under a `capabilities` list. Each capability is a string identifying what the plugin provides (e.g., "tools:read", "hooks:on_turn_start", "commands:custom-cmd"). Capabilities MUST be used for discovery and filtering, not enforcement.

#### Scenario: Plugin declares capabilities

- GIVEN a manifest with capabilities ["tools:read", "hooks:on_turn_start"]
- WHEN the plugin is registered
- THEN its capabilities are available for querying

#### Scenario: No capabilities declared

- GIVEN a manifest without a capabilities field
- WHEN the plugin is loaded
- THEN capabilities default to empty list

### Requirement: REQ-PMAN-4 — Backward Compatibility with extension.yaml

The system MUST support reading legacy `extension.yaml` files as a fallback when no `kui-plugin.yaml` is present. Legacy manifests MUST be treated as type "tool" with no capabilities or permissions. A deprecation warning MUST be logged.

#### Scenario: Legacy extension.yaml loaded

- GIVEN a directory with `extension.yaml` but no `kui-plugin.yaml`
- WHEN the plugin is discovered
- THEN the extension.yaml is parsed as a PluginManifest with type "tool"
- AND a deprecation warning is logged

#### Scenario: kui-plugin.yaml takes precedence

- GIVEN a directory with both `kui-plugin.yaml` and `extension.yaml`
- WHEN the plugin is discovered
- THEN only `kui-plugin.yaml` is used

### Requirement: REQ-PMAN-5 — Manifest Validation

The system MUST validate manifests on install and on load. Validation MUST check: required fields present, version is valid semver, type is a known enum value, entry_point exists relative to manifest directory.

#### Scenario: Validation on install

- GIVEN a plugin directory being installed
- WHEN `kui plugin install` reads the manifest
- THEN validation runs and rejects invalid manifests before copying

#### Scenario: Validation on load

- GIVEN an installed plugin with a corrupted manifest
- WHEN the plugin system discovers it
- THEN a validation error is logged
- AND the plugin is skipped (not loaded)

#### Scenario: Entry point file does not exist

- GIVEN a manifest with entry_point "bin/my-plugin" but the file is missing
- WHEN the manifest is validated
- THEN a validation error indicates entry_point not found
