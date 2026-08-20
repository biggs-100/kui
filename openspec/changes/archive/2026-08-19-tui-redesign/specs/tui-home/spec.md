# tui-home Specification

## Purpose

The home screen is the landing view when `kui tui` starts. It displays a centered ASCII logo, a bordered prompt input, and a minimal footer — matching OpenCode's visual style.

## Requirements

### Requirement: REQ-TUI-HOME-1 — Centered Layout

The home screen MUST render three vertically-centered elements: ASCII logo, prompt input, and footer. All elements MUST be horizontally centered within the terminal width.

#### Scenario: Home screen renders centered

- GIVEN a terminal of any size
- WHEN `kui tui` starts
- THEN the logo is horizontally centered
- AND the prompt input is horizontally centered below the logo
- AND the footer is at the bottom of the screen

#### Scenario: Resize maintains centering

- GIVEN the home screen is visible
- WHEN the terminal resizes
- THEN all elements remain horizontally centered
- AND vertical spacing adjusts proportionally

### Requirement: REQ-TUI-HOME-2 — ASCII Logo

The home screen MUST display an ASCII art logo. The default logo MUST be a generic "kui" text in a simple block style. The logo MUST use the primary accent color from the active theme.

#### Scenario: Default logo displays

- GIVEN the home screen renders
- WHEN no custom logo is configured
- THEN a generic "kui" ASCII art is displayed

#### Scenario: Logo uses theme color

- GIVEN the "opencode" theme is active
- WHEN the home screen renders
- THEN the logo uses the theme's primary color

### Requirement: REQ-TUI-HOME-3 — Bordered Prompt

The home screen MUST display a prompt input with a rounded border. The placeholder text MUST show "Ask kui..." or similar. The border MUST use a subtle gray color.

#### Scenario: Prompt renders with border

- GIVEN the home screen is visible
- WHEN the prompt is empty
- THEN a rounded border surrounds the input area
- AND placeholder text is visible inside

#### Scenario: Prompt accepts input

- GIVEN the home screen is visible
- WHEN the user types
- THEN the text appears inside the bordered area
- AND the border remains visible

### Requirement: REQ-TUI-HOME-4 — Minimal Footer

The home screen MUST display a footer at the bottom with: current directory, LSP status (dot indicator), MCP status (dot indicator), and "/status" text. The footer MUST use muted colors.

#### Scenario: Footer shows status

- GIVEN the home screen is visible
- WHEN LSP and MCP are connected
- THEN the footer shows: `directory • LSP • MCP • /status`
- AND the dots are green (connected)

#### Scenario: Footer shows disconnected

- GIVEN the home screen is visible
- WHEN LSP is not connected
- THEN the LSP dot is gray/muted

### Requirement: REQ-TUI-HOME-5 — Prompt Submission

When the user presses Enter on the home screen, the app MUST transition to the session/chat view with the submitted prompt. The transition MUST be instant (no animation in first slice).

#### Scenario: Enter sends prompt

- GIVEN the home screen is visible
- AND the user has typed a prompt
- WHEN the user presses Enter
- THEN the app transitions to the chat view
- AND the prompt is sent to the agent

#### Scenario: Empty prompt does nothing

- GIVEN the home screen is visible
- AND the prompt is empty
- WHEN the user presses Enter
- THEN the app stays on the home screen

### Requirement: REQ-TUI-HOME-6 — Keyboard Shortcuts

The home screen MUST support the same keyboard shortcuts as the chat view: Ctrl+P (command palette), Ctrl+C (quit), Tab (switch profile).

#### Scenario: Ctrl+C quits from home

- GIVEN the home screen is visible
- WHEN the user presses Ctrl+C
- THEN the app exits cleanly

#### Scenario: Ctrl+P opens palette from home

- GIVEN the home screen is visible
- WHEN the user presses Ctrl+P
- THEN the command palette opens
