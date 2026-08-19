# tui-chat Delta for lsp-integration

## MODIFIED Requirements

### Requirement: REQ-TUI-CHAT-2 — Streaming Answer Rendering

While the provider streams, the chat view MUST render answer content incrementally as events arrive. If the stream fails mid-answer, the view MUST render an error state for that answer and MUST NOT crash the app. The chat view MUST render inline diagnostic annotations (errors/warnings) below affected lines when displaying file content or diffs. Diagnostics MUST NOT interrupt the content flow.
(Previously: Streaming answer rendering only handled provider stream events. No diagnostic annotations.)

#### Scenario: Incremental render

- GIVEN a provider that streams an answer in chunks
- WHEN chunks arrive
- THEN the answer text grows in the view as each chunk renders

#### Scenario: Stream error mid-answer

- GIVEN a provider whose stream fails partway through an answer
- WHEN the failure occurs
- THEN the partial answer shows an error state
- AND the app keeps running and accepts the next prompt

#### Scenario: Inline diagnostic rendering

- GIVEN a file display with an error at line 15
- WHEN the file is rendered in the chat view
- THEN an error annotation appears below line 15
- AND the annotation includes severity indicator and message
- AND the content flow is not interrupted

#### Scenario: No diagnostics for clean file

- GIVEN a file with no diagnostics
- WHEN the file is rendered in the chat view
- THEN no diagnostic annotations appear
- AND the display is unchanged from current behavior
