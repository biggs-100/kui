# Plugin CLI Specification

## Purpose

User-facing CLI commands for discovering, installing, removing, and inspecting plugins. Wraps the filesystem-based plugin discovery and manifest validation into a cobra command tree.

## Requirements

### Requirement: REQ-PCLI-1 — Plugin List

`kui plugin list` MUST enumerate all installed plugins from both global (`~/.config/kui/plugins/`) and project (`.kui/plugins/`) directories. Output MUST be a formatted table by default, with `--json` flag for machine-readable output.

#### Scenario: List plugins from global and project dirs

- GIVEN two plugins installed globally and one project-local plugin
- WHEN `kui plugin list` is executed
- THEN a table is printed with name, version, type, and status for each plugin
- AND project-local plugins are visually distinguished from global ones

#### Scenario: List with --json flag

- GIVEN one installed plugin
- WHEN `kui plugin list --json` is executed
- THEN JSON output is printed with plugin metadata array

#### Scenario: No plugins installed

- GIVEN no plugins in global or project directories
- WHEN `kui plugin list` is executed
- THEN a message indicating "No plugins installed" is displayed
- AND exit code is 0

### Requirement: REQ-PCLI-2 — Plugin Install

`kui plugin install <path|url>` MUST install a plugin by reading its manifest, validating it, and copying the plugin directory into the target install location. Local paths MUST be validated for a `kui-plugin.yaml` manifest before installation.

#### Scenario: Install plugin from local directory

- GIVEN a local directory with a valid `kui-plugin.yaml`
- WHEN `kui plugin install ./my-plugin` is executed
- THEN the plugin directory is copied to `~/.config/kui/plugins/{plugin-name}/`
- AND a success message is printed with plugin name and version

#### Scenario: Install plugin from local directory (project scope)

- GIVEN `--project` flag and a valid local plugin directory
- WHEN `kui plugin install --project ./my-plugin` is executed
- THEN the plugin is installed to `.kui/plugins/{plugin-name}/`

#### Scenario: Install from invalid path

- GIVEN a path that does not exist
- WHEN `kui plugin install ./nonexistent` is executed
- THEN an error message is printed indicating path not found
- AND exit code is non-zero

#### Scenario: Install from directory without manifest

- GIVEN a directory without `kui-plugin.yaml`
- WHEN `kui plugin install ./empty-dir` is executed
- THEN an error message indicates missing manifest
- AND exit code is non-zero

#### Scenario: Install plugin with existing name

- GIVEN a plugin named "my-plugin" already installed
- WHEN `kui plugin install ./my-plugin` is executed
- THEN the user is prompted to confirm overwrite
- AND upon confirmation the existing plugin is replaced

### Requirement: REQ-PCLI-3 — Plugin Remove

`kui plugin remove <name>` MUST uninstall a plugin by removing its directory from the install location. The command MUST require confirmation unless `--yes` flag is provided.

#### Scenario: Remove installed plugin with confirmation

- GIVEN an installed plugin named "my-plugin"
- WHEN `kui plugin remove my-plugin` is executed
- THEN a confirmation prompt is displayed
- AND upon confirmation the plugin directory is removed

#### Scenario: Remove with --yes flag

- GIVEN an installed plugin named "my-plugin"
- WHEN `kui plugin remove --yes my-plugin` is executed
- THEN the plugin directory is removed without confirmation

#### Scenario: Remove non-existent plugin

- GIVEN no plugin named "unknown" is installed
- WHEN `kui plugin remove unknown` is executed
- THEN an error message indicates plugin not found
- AND exit code is non-zero

### Requirement: REQ-PCLI-4 — Plugin Info

`kui plugin info <name>` MUST display detailed metadata for an installed plugin: name, version, type, entry point, capabilities, permissions, and install path.

#### Scenario: Show info for installed plugin

- GIVEN an installed plugin named "my-plugin"
- WHEN `kui plugin info my-plugin` is executed
- THEN full plugin metadata is displayed in key-value format

#### Scenario: Info for non-existent plugin

- GIVEN no plugin named "unknown" is installed
- WHEN `kui plugin info unknown` is executed
- THEN an error message indicates plugin not found
- AND exit code is non-zero

### Requirement: REQ-PCLI-5 — Plugin Discovery Directories

The system MUST discover plugins from two directories: global (`~/.config/kui/plugins/`) and project-local (`.kui/plugins/`). Each subdirectory in these paths containing a `kui-plugin.yaml` (or legacy `extension.yaml`) MUST be treated as a plugin. Project-local plugins MUST override global plugins of the same name.

#### Scenario: Both global and project plugins loaded

- GIVEN a global plugin "shared-tool" and a project-local plugin "custom-tool"
- WHEN plugins are discovered
- THEN both plugins are available

#### Scenario: Project-local overrides global

- GIVEN a global plugin "my-tool" v1.0 and a project-local "my-tool" v2.0
- WHEN plugins are discovered
- THEN the project-local v2.0 is used

#### Scenario: Plugin directory missing is non-fatal

- GIVEN the global plugins directory does not exist
- WHEN plugins are discovered
- THEN discovery continues with project plugins only
- AND no error is raised

### Requirement: REQ-PCLI-6 — Output Formatting

The plugin CLI MUST support two output formats: `table` (default) for human-readable output and `json` for machine consumption. The `--output` flag or `--json` shorthand MUST control this.

#### Scenario: Default table output

- GIVEN installed plugins
- WHEN `kui plugin list` is executed
- THEN output is a formatted table with aligned columns

#### Scenario: JSON output

- GIVEN installed plugins
- WHEN `kui plugin list --json` is executed
- THEN output is a valid JSON array of plugin objects
