# Delta for dynamic-extension-lifecycle

## MODIFIED Requirements

### Requirement: Subprocess Spawning

The system MUST spawn extensions as subprocesses via JSON-RPC 2.0 stdio. Before spawning, the system MUST validate the manifest, check permissions, and verify the entry point exists. Spawn failures MUST log and mark extension unavailable. (Previously: spawn without permission or manifest validation)

#### Scenario: Entry point missing

- GIVEN missing entry point
- WHEN spawn attempted
- THEN extension unavailable; others continue

#### Scenario: Permission denied blocks spawn

- GIVEN an extension with undeclared required permission in enforce mode
- WHEN spawn attempted
- THEN extension unavailable with permission error
- AND others continue

#### Scenario: Manifest validation failure

- GIVEN an extension with an invalid manifest (missing required fields)
- WHEN spawn attempted
- THEN extension unavailable with validation error
- AND others continue

## ADDED Requirements

### Requirement: REQ-DEL-MAN — Manifest Validation on Load

The system MUST validate the manifest format (kui-plugin.yaml) before spawning. Validation MUST check required fields, version format, and entry point existence. Invalid manifests MUST prevent spawn.

#### Scenario: Valid manifest allows spawn

- GIVEN an extension with a valid kui-plugin.yaml
- WHEN the extension is loaded
- THEN manifest validation passes
- AND spawn proceeds

#### Scenario: Invalid manifest blocks spawn

- GIVEN an extension with a manifest missing the "version" field
- WHEN the extension is loaded
- THEN manifest validation fails
- AND spawn is blocked
- AND an error is logged

### Requirement: REQ-DEL-PERM — Permission Enforcement

The system MUST check permissions before subprocess spawn. In enforce mode, denied permissions MUST block the spawn. In warn-only mode, missing permissions MUST be logged as warnings but allow spawn.

#### Scenario: Enforce mode blocks unauthorized spawn

- GIVEN an extension requiring "filesystem:write" in enforce mode
- WHEN the permission is not granted
- THEN the subprocess is NOT spawned
- AND an error indicates the denied permission

#### Scenario: Warn-only mode logs and allows

- GIVEN an extension using filesystem access without declaring it in warn-only mode
- WHEN the extension is loaded
- THEN a warning is logged
- AND the subprocess IS spawned

### Requirement: REQ-DEL-DISCOVER — Plugin Directory Discovery

The system MUST discover extensions from both global (`~/.config/kui/plugins/`) and project-local (`.kui/plugins/`) directories. Each subdirectory with a `kui-plugin.yaml` or `extension.yaml` MUST be treated as an extension. Project-local extensions MUST override global ones of the same name.

#### Scenario: Discovery from both directories

- GIVEN a global plugin and a project-local plugin
- WHEN extensions are discovered
- THEN both plugins are found and loaded

#### Scenario: Project overrides global

- GIVEN a global plugin "my-tool" v1.0 and project-local "my-tool" v2.0
- WHEN extensions are discovered
- THEN the project-local v2.0 is used
