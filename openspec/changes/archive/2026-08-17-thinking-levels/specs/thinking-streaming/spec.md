# Delta for thinking-streaming (provider-openai-streaming)

## ADDED Requirements

### Requirement: REQ-THINK-13 — SSE Reasoning Content Parsing

The SSE parser MUST extract `choices[].delta.reasoning_content` from streaming chunks. When this field is present and non-empty, it MUST be captured as the reasoning delta.

#### Scenario: Reasoning content in delta

- GIVEN an SSE chunk with `choices[0].delta.reasoning_content: "Let me think..."`
- WHEN the parser processes the chunk
- THEN the reasoning content is captured as a string

#### Scenario: Empty reasoning content

- GIVEN an SSE chunk with `choices[0].delta.reasoning_content: ""`
- WHEN the parser processes the chunk
- THEN no reasoning delta is emitted

### Requirement: REQ-THINK-14 — StreamChunk ReasoningDelta

The `StreamChunk` struct MUST include a `ReasoningDelta string` field. SSE reasoning content chunks MUST be emitted as `StreamChunk` values with `ReasoningDelta` populated and `TextDelta` empty.

#### Scenario: Reasoning delta emitted

- GIVEN an SSE chunk with reasoning content
- WHEN the chunk is parsed
- THEN a `StreamChunk` with `ReasoningDelta` set is emitted on the channel

#### Scenario: Reasoning and text in same chunk

- GIVEN an SSE chunk with both `reasoning_content` and `content`
- WHEN the chunk is parsed
- THEN two `StreamChunk` values are emitted: one with `ReasoningDelta` and one with `TextDelta`

### Requirement: REQ-THINK-15 — TUI Reasoning Style

The TUI MUST render reasoning tokens in a visually distinct style from normal text output. Reasoning tokens SHOULD use faint or italic styling to differentiate them from model responses.

#### Scenario: Reasoning displayed distinctly

- GIVEN a StreamChunk with ReasoningDelta set
- WHEN the TUI renders it
- THEN the reasoning text appears in a different style (faint/italic) than normal output

#### Scenario: Reasoning interleaved with text

- GIVEN alternating reasoning and text StreamChunks
- WHEN the TUI renders them
- THEN reasoning appears in distinct style and text appears in normal style
