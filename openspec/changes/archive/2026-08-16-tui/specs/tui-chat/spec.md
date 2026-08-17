# tui-chat Specification

## Purpose

The chat view shows the conversation: user prompts, streaming assistant answers, and explicit `{profile, model}` context on every prompt.

## Requirements

### Requirement: REQ-TUI-CHAT-1 — Prompt Submission

The user MUST be able to type a prompt in the input and submit it with Enter. The submitted prompt MUST be sent to the agent loop carrying the session's active `{profile, model}` and MUST appear in the message view.

#### Scenario: Submit a prompt

- GIVEN the TUI running with active profile "coder"
- WHEN the user types a prompt and presses Enter
- THEN the prompt appears in the message view
- AND it is sent carrying `{profile: "coder", model: <resolved>}`

#### Scenario: Empty input ignored

- GIVEN an empty or whitespace-only input
- WHEN the user presses Enter
- THEN no prompt is submitted
- AND the message view is unchanged

### Requirement: REQ-TUI-CHAT-2 — Streaming Answer Rendering

While the provider streams, the chat view MUST render answer content incrementally as events arrive. If the stream fails mid-answer, the view MUST render an error state for that answer and MUST NOT crash the app.

#### Scenario: Incremental render

- GIVEN a provider that streams an answer in chunks
- WHEN chunks arrive
- THEN the answer text grows in the view as each chunk renders

#### Scenario: Stream error mid-answer

- GIVEN a provider whose stream fails partway through an answer
- WHEN the failure occurs
- THEN the partial answer shows an error state
- AND the app keeps running and accepts the next prompt

### Requirement: REQ-TUI-CHAT-3 — Per-Prompt Context Stability

Each prompt MUST capture its own `{profile, model}` at submission time via the resolution chain `store.Get` → resolved → default (REQ-CLI-4). A later profile switch via TAB MUST NOT change the context of already-submitted prompts.

#### Scenario: Resolution chain on submit

- GIVEN profile "coder" with no saved model
- WHEN a prompt is submitted
- THEN its model resolves through the chain and lands on the default

#### Scenario: Prior prompts keep their context

- GIVEN a prompt submitted under profile "coder"
- WHEN the user switches to "writer" with TAB
- THEN the earlier prompt still shows `{profile: "coder"}` with its original model
