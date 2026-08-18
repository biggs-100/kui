# Dynamic Extension Discovery

## Purpose

Runtime discovery of extensions from filesystem paths and config files.

## Requirements

### Requirement: Discovery Sources

The system MUST scan global (`~/.config/kui/extensions/`) and project-level (`.kui/extensions/`) directories. Project extensions MUST override global on name collision.

#### Scenario: Name collision resolved by project

- GIVEN global `foo` and project `foo`
- WHEN discovery completes
- THEN project `foo` wins

#### Scenario: Empty directories

- GIVEN no extensions in any source
- WHEN discovery completes
- THEN no filesystem extensions registered

### Requirement: Manifest Format

Each extension MUST have `extension.yaml` with `name`, `version`, `protocol_version`, `entry_point`. Malformed manifests MUST be skipped with warning.

#### Scenario: Missing manifest skipped

- GIVEN directory without `extension.yaml`
- WHEN discovery runs
- THEN skipped silently

### Requirement: Config-Based Sources

Global and project config MAY add directories via `extensions.paths`. Project config wins on conflict.

#### Scenario: Config adds directory

- GIVEN config with `extensions.paths: ["/custom"]`
- WHEN discovery runs
- THEN `/custom` is also scanned
