# Delta for thinking-provider (provider-openai-streaming)

## MODIFIED Requirements

### Requirement: REQ-OAI-STREAM-4 — JSON Chunk Unmarshalling

Each `data: {...}` line MUST be unmarshalled into the OpenAI streaming response shape. The adapter MUST extract `choices[].delta.content` as `TextDelta` and `choices[].delta.tool_calls` for tool call accumulation. The adapter MUST also extract `choices[].delta.reasoning_content` as `ReasoningDelta` when present.

(Previously: Only content and tool_calls were extracted from delta)

#### Scenario: Text content extracted

- GIVEN an SSE chunk with `choices[0].delta.content: "Hello"`
- WHEN the chunk is parsed
- THEN a `StreamChunk` with `TextDelta: "Hello"` is emitted

#### Scenario: Empty delta ignored

- GIVEN an SSE chunk with empty `delta` fields
- WHEN the chunk is parsed
- THEN no `StreamChunk` is emitted (skipped)

#### Scenario: Reasoning content extracted

- GIVEN an SSE chunk with `choices[0].delta.reasoning_content: "Thinking step by step..."`
- WHEN the chunk is parsed
- THEN a `StreamChunk` with `ReasoningDelta: "Thinking step by step..."` is emitted

#### Scenario: No reasoning_content field

- GIVEN an SSE chunk without `delta.reasoning_content`
- WHEN the chunk is parsed
- THEN `ReasoningDelta` is empty string in the emitted chunk

## ADDED Requirements

### Requirement: REQ-THINK-9 — SetThinking Method

The `openai.Client` MUST expose a `SetThinking(level string)` method that stores the thinking level for use in subsequent requests.

#### Scenario: Set thinking level

- GIVEN an OpenAI client instance
- WHEN `SetThinking("high")` is called
- THEN subsequent requests use "high" as the reasoning effort

#### Scenario: Set thinking to off

- GIVEN an OpenAI client with thinking set to "high"
- WHEN `SetThinking("off")` is called
- THEN subsequent requests omit reasoning_effort

### Requirement: REQ-THINK-10 — Reasoning Effort in Request

When thinking level is not "off" and not empty, the client MUST include `reasoning_effort` in the chat completion request body. The value MUST be one of: "low", "medium", "high".

#### Scenario: Medium thinking sends reasoning_effort

- GIVEN a client with thinking set to "medium"
- WHEN a chat request is built
- THEN the request body contains `"reasoning_effort": "medium"`

#### Scenario: High thinking sends reasoning_effort

- GIVEN a client with thinking set to "high"
- WHEN a chat request is built
- THEN the request body contains `"reasoning_effort": "high"`

### Requirement: REQ-THINK-11 — Omit Reasoning Effort When Off

When thinking level is "off" or empty, the client MUST omit `reasoning_effort` from the request body entirely.

#### Scenario: Thinking off omits field

- GIVEN a client with thinking set to "off"
- WHEN a chat request is built
- THEN the request body does NOT contain `reasoning_effort`

#### Scenario: Empty thinking omits field

- GIVEN a client with thinking set to ""
- WHEN a chat request is built
- THEN the request body does NOT contain `reasoning_effort`

### Requirement: REQ-THINK-12 — Reasoning Effort Serialization

The `ReasoningEffort` field on `chatRequest` MUST be a `*string` with `omitempty` json tag. A nil pointer means the field is omitted from JSON serialization.

#### Scenario: Non-nil pointer serializes

- GIVEN a chatRequest with ReasoningEffort set to pointer to "high"
- WHEN marshalled to JSON
- THEN `"reasoning_effort": "high"` appears in the output

#### Scenario: Nil pointer omitted

- GIVEN a chatRequest with ReasoningEffort as nil pointer
- WHEN marshalled to JSON
- THEN `reasoning_effort` does NOT appear in the output
