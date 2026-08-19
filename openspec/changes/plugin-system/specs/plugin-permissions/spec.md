# Plugin Permissions Specification

## Purpose

Permission model for plugin isolation. Plugins declare required permissions in their manifest; the system enforces these before subprocess spawn. Initially runs in warn-only mode, transitioning to enforce mode after real-world feedback.

## Requirements

### Requirement: REQ-PERM-1 — Permission Declarations

Plugins MUST declare permissions in `kui-plugin.yaml` under a `permissions` list. Each permission is a string using a namespace:action format (e.g., `tools:read`, `hooks:on_turn_start`, `commands:register`, `filesystem:read`, `network:outbound`).

#### Scenario: Plugin declares permissions

- GIVEN a manifest with permissions ["tools:read", "filesystem:read"]
- WHEN the plugin is loaded
- THEN the declared permissions are recorded in the plugin metadata

#### Scenario: No permissions declared

- GIVEN a manifest without a permissions field
- WHEN the plugin is loaded
- THEN permissions default to empty (no special permissions required)

### Requirement: REQ-PERM-2 — Permission Enforcement on Spawn

Before spawning a plugin subprocess, the system MUST check that all declared permissions are granted. If any permission is denied, the system MUST NOT spawn the subprocess and MUST return an error.

#### Scenario: All permissions granted

- GIVEN a plugin requiring permissions ["tools:read"] and the user has granted "tools:read"
- WHEN the plugin subprocess is spawned
- THEN the spawn proceeds normally

#### Scenario: Permission denied

- GIVEN a plugin requiring permission "network:outbound" and the user has denied it
- WHEN the plugin subprocess is spawned
- THEN the spawn is blocked
- AND an error message indicates which permission was denied

### Requirement: REQ-PERM-3 — Warn-Only Mode (Initial)

The system MUST default to warn-only mode: undeclared permissions are logged as warnings but do not block execution. This mode MUST be configurable via `kui` config or environment variable. Warn-only mode MUST transition to enforce mode after the initial release cycle.

#### Scenario: Warn-only mode logs undeclared permission

- GIVEN a plugin using filesystem access without declaring "filesystem:read"
- WHEN warn-only mode is active
- THEN a warning is logged
- AND the plugin subprocess is still spawned

#### Scenario: Enforce mode blocks undeclared permission

- GIVEN a plugin using filesystem access without declaring "filesystem:read"
- WHEN enforce mode is active
- THEN the spawn is blocked
- AND an error indicates undeclared permission

### Requirement: REQ-PERM-4 — Permission Granularity

Permissions MUST be granular enough to control: tool registration (`tools:read`, `tools:write`), hook registration (`hooks:{event_name}`), command registration (`commands:register`), filesystem access (`filesystem:read`, `filesystem:write`), network access (`network:outbound`).

#### Scenario: Tool registration permission

- GIVEN a plugin with permission "tools:read" but not "tools:write"
- WHEN the plugin calls RegisterTool
- THEN the registration succeeds (tools:read covers tool access)

#### Scenario: Network permission

- GIVEN a plugin without permission "network:outbound"
- WHEN the plugin attempts an outbound network request
- THEN the request is blocked in enforce mode

### Requirement: REQ-PERM-5 — User Consent Flow

On first load of a plugin with undeclared permissions, the system MUST present the user with a consent prompt listing each permission and allowing allow/deny per permission. User decisions MUST be persisted for subsequent loads.

#### Scenario: First-time plugin consent

- GIVEN a new plugin with permissions ["tools:read", "network:outbound"]
- WHEN the plugin is loaded for the first time
- THEN a consent prompt is displayed listing both permissions
- AND the user can allow or deny each permission individually

#### Scenario: Persisted consent

- GIVEN a user who previously granted "tools:read" to plugin "my-plugin"
- WHEN the plugin is loaded again
- THEN the consent prompt is not shown
- AND the previously granted permission is applied automatically

### Requirement: REQ-PERM-6 — Permission Config

Permission grants and denials MUST be stored in a config file at `~/.config/kui/permissions.yaml`. This file MUST be human-readable and editable. The system MUST re-read this file on each plugin load.

#### Scenario: Config file does not exist

- GIVEN no permissions.yaml exists
- WHEN a plugin requiring permissions is loaded
- THEN the consent flow is triggered for all permissions

#### Scenario: Manual config edit

- GIVEN a user manually edits permissions.yaml to grant "network:outbound" to "my-plugin"
- WHEN the plugin is loaded
- THEN the manually granted permission is applied
