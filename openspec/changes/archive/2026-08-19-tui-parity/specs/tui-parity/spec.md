# Spec: TUI Parity — truthful, OpenCode-aligned rendering

Capability: `tui-parity`
Status: proposed

## Requirements

### REQ-TUI-PARITY-1: No fabricated version strings
The footer and sidebar SHALL NOT render any hardcoded application version, semver,
or build string. kui has no version source; OpenCode renders a dynamic
`InstallationVersion` which kui does not possess.

- **Given** the TUI renders the footer
- **When** no version source is available
- **Then** it MUST NOT contain `OpenCode 1.18.18`, `1.2.1`, or any literal semver

- **Given** the sidebar initializes
- **When** no version source is available
- **Then** it MUST NOT store or render a `version` field

### REQ-TUI-PARITY-2: Token/cost reflect real controller values
Token count and cost SHALL be rendered only from `controller.TotalTokens()` and
`controller.Cost()`, which are accumulated via `TrackUsage` on each `streamDoneMsg`.

- **Given** a session with no completed stream
- **When** the footer/sidebar renders
- **Then** it SHALL show `—` (or `0 tokens 0% $0.00`), never a fabricated nonzero value

- **Given** a stream completed with 1234 input + 567 output tokens and known pricing
- **When** the footer/sidebar renders
- **Then** it SHALL show the real `TotalTokens()` value and real `Cost()`

### REQ-TUI-PARITY-3: No status dots without a data source
MCP and LSP status indicators SHALL NOT be rendered unless kui tracks a real state
for them. Currently `SetMCPStatus`/`SetLSPStatus` have no callers.

- **Given** kui tracks no MCP/LSP server state
- **When** the sidebar/footer renders
- **Then** it MUST NOT show `context7 • engram`, `○ disconnected`, `● Connected`,
  or any MCP/LSP dot, unless a real status is supplied

- **Given** a future change wires real MCP/LSP status
- **When** `SetMCPStatus`/`SetLSPStatus` are called with real data
- **Then** the indicator MAY render that real state

### REQ-TUI-PARITY-4: Model catalog is live, not fabricated
`/model` SHALL list only models discovered from configured providers' live
endpoints. No fabricated model IDs SHALL appear.

- **Given** `AvailableModels()` static catalog
- **When** it is rendered
- **Then** it MUST NOT contain `mimo-v2-free`, `mimo-v2.5`, or any non-existent ID

- **Given** no provider is configured
- **When** `/model` opens
- **Then** it SHALL show an empty/help state, not fabricated models

- **Given** a provider is configured and reachable
- **When** `/model` opens
- **Then** it SHALL list models from `liveModelsForProvider`

### REQ-TUI-PARITY-5: No scattered hex literals in views
Color values in `internal/tui/views/*.go` SHALL come from `theme.Styles` /
`theme.Theme`, not inline hex strings.

- **Given** a view renders a colored element
- **When** the color is chosen
- **Then** it SHALL use a `theme` token (e.g. `theme.Background`, `theme.Accent`),
  never a literal like `#1a1a1a`, `#569cd6`, `#252525`, `#4ec9b0`

### REQ-TUI-PARITY-6: Sidebar subagent/version block removed
The sidebar SHALL NOT render a fabricated "Subagents" block with run/done counters.

- **Given** the sidebar renders
- **When** no real subagent run data exists
- **Then** it MUST NOT show `v1.2.1 • 0 run • 0 done • Σ 0`

## Non-Goals
- Adding real LSP/MCP status tracking (future change).
- Adding a kui version string (future change).
- Changing layout dimensions or palette hue.

## Acceptance
All scenarios above pass via `go test ./...`; grep for the forbidden literals
returns zero matches in `internal/tui/views/`.
