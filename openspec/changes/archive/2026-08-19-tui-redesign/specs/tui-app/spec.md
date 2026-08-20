# Delta for tui-app

## MODIFIED Requirements

### Requirement: REQ-TUI-APP-1 — Entrypoint & Lifecycle

`kui tui` MUST start the Bubble Tea program. On first launch, it MUST render the home screen (centered logo + prompt). After prompt submission, it MUST render the session view (chat + tool view). The app MUST quit on `q` or `ctrl+c`. If startup fails, the app MUST NOT render; the CLI MUST exit non-zero with an actionable stderr message.

(Previously: Always renders header, chat, and tool views directly)

#### Scenario: First launch shows home

- GIVEN a valid provider configuration
- WHEN `kui tui` starts
- THEN the home screen renders with centered logo and prompt

#### Scenario: After prompt, shows chat

- GIVEN the home screen is visible
- WHEN the user submits a prompt
- THEN the session view renders with chat and tool views

#### Scenario: Quit on ctrl+c

- GIVEN a running TUI (home or session)
- WHEN the user presses `ctrl+c`
- THEN the app exits cleanly with status zero

### Requirement: REQ-TUI-APP-2 — Layout & Resize

The app MUST support two layouts: home (centered logo + prompt + footer) and session (header + chat + tool view + footer). A window resize MUST reflow the current layout and MUST NOT crash the app.

(Previously: Always renders three regions — header, chat, and tool view)

#### Scenario: Home layout is centered

- GIVEN the home screen is active
- WHEN the app renders
- THEN logo and prompt are horizontally centered

#### Scenario: Session layout is traditional

- GIVEN the session view is active
- WHEN the app renders
- THEN header, chat, tool, and footer regions are visible

#### Scenario: Resize reflows current layout

- GIVEN a running TUI
- WHEN the window resizes
- THEN the current layout (home or session) reflows
- AND the app keeps running

## ADDED Requirements

### Requirement: REQ-TUI-APP-5 — Route System

The app MUST maintain a route state: `home` or `session`. The route MUST switch from `home` to `session` when a prompt is submitted. The route MUST switch from `session` to `home` when the user requests a new session (future: Ctrl+N).

#### Scenario: Initial route is home

- GIVEN `kui tui` starts
- WHEN the app renders for the first time
- THEN the route is `home`

#### Scenario: Prompt submission switches to session

- GIVEN the route is `home`
- WHEN the user submits a prompt
- THEN the route changes to `session`

### Requirement: REQ-TUI-APP-6 — Footer Variants

The app MUST render different footers depending on the route:
- Home: `directory • LSP • MCP • /status` (minimal)
- Session: `tokens (percent%) • $cost • MCP: N connected • LSP: status` (detailed)

(Previously: Single footer with tokens/cost/MCP/LSP)

#### Scenario: Home footer is minimal

- GIVEN the route is `home`
- WHEN the footer renders
- THEN it shows directory, LSP dot, MCP dot, and "/status"

#### Scenario: Session footer is detailed

- GIVEN the route is `session`
- WHEN the footer renders
- THEN it shows tokens, cost, MCP count, and LSP status

### Requirement: REQ-TUI-APP-7 — Theme "opencode"

The app MUST include a new theme named "opencode" with:
- Primary background: dark gray (#1a1a1a)
- Text: light gray (#e0e0e0)
- Muted text: medium gray (#808080)
- Accent: subtle blue (#569cd6)
- Borders: dark gray (#333333)
- Success: green (#4ec9b0)
- Error: red (#f44747)

#### Scenario: opencode theme loads

- GIVEN the user selects the "opencode" theme
- WHEN the app renders
- THEN colors match the palette above
