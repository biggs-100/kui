# provider-openai-streaming Specification

## Purpose

Implements `StreamingProvider` for the OpenAI-compatible adapter using SSE (Server-Sent Events) parsing with stdlib only.

## ADDED Requirements

### Requirement: REQ-OAI-STREAM-1 — OpenAI StreamingAdapter

The OpenAI adapter MUST implement `StreamingProvider`. `StreamChat` MUST send a POST to `{base_url}/chat/completions` with `"stream": true` in the request body and return a channel of parsed `StreamChunk` values.

#### Scenario: StreamChat returns streaming channel

- GIVEN an OpenAI adapter with valid credentials
- WHEN `StreamChat` is called
- THEN a POST request with `stream: true` is sent
- AND a non-nil channel is returned

#### Scenario: Fall back to Chat on stream unsupported

- GIVEN a server returning 400 for `stream: true`
- WHEN `StreamChat` is invoked
- THEN the adapter returns a clear error indicating streaming is unsupported

### Requirement: REQ-OAI-STREAM-2 — SSE Parsing with bufio.Scanner

The adapter MUST parse SSE responses using `bufio.Scanner` from stdlib. The scanner buffer MUST be at least 256KB to handle large tool call JSON payloads. Lines MUST be parsed by stripping the `data: ` prefix.

#### Scenario: Normal SSE event parsed

- GIVEN an SSE response containing `data: {"choices":[...]}`
- WHEN the scanner reads the line
- THEN the `data: ` prefix is stripped and the JSON is unmarshalled

#### Scenario: Large event fits in buffer

- GIVEN an SSE event with a tool call JSON exceeding 64KB
- WHEN the scanner reads the line
- THEN the event is parsed without buffer overflow

### Requirement: REQ-OAI-STREAM-3 — DONE Sentinel

The adapter MUST detect `data: [DONE]` as the stream completion signal. Upon receiving this sentinel, the adapter MUST send a `StreamChunk{Done: true}` and close the channel.

#### Scenario: [DONE] triggers completion

- GIVEN an SSE stream ending with `data: [DONE]`
- WHEN the scanner reads the sentinel
- THEN a `Done: true` chunk is sent and the channel closes

#### Scenario: No [DONE] before connection drop

- GIVEN an SSE stream where the connection drops without `[DONE]`
- WHEN the scanner reaches EOF
- THEN the adapter sends an error chunk and closes the channel

### Requirement: REQ-OAI-STREAM-4 — JSON Chunk Unmarshalling

Each `data: {...}` line MUST be unmarshalled into the OpenAI streaming response shape. The adapter MUST extract `choices[].delta.content` as `TextDelta` and `choices[].delta.tool_calls` for tool call accumulation.

#### Scenario: Text content extracted

- GIVEN an SSE chunk with `choices[0].delta.content: "Hello"`
- WHEN the chunk is parsed
- THEN a `StreamChunk` with `TextDelta: "Hello"` is emitted

#### Scenario: Empty delta ignored

- GIVEN an SSE chunk with empty `delta` fields
- WHEN the chunk is parsed
- THEN no `StreamChunk` is emitted (skipped)

### Requirement: REQ-OAI-STREAM-5 — Tool Call Accumulation

Tool calls arrive incrementally across SSE chunks. The adapter MUST accumulate tool call index, name, and arguments across chunks, emitting `ToolCallStart`, `ToolCallDelta`, and `ToolCallEnd` events at the appropriate boundaries.

#### Scenario: Tool call accumulated across chunks

- GIVEN SSE chunks with partial tool call arguments arriving sequentially
- WHEN each chunk is parsed
- THEN `ToolCallDelta` events are emitted with accumulated content
- AND `ToolCallEnd` is emitted when the tool call index changes or stream ends

#### Scenario: Tool call start detected

- GIVEN an SSE chunk with `delta.tool_calls[0].function.name` set
- WHEN the chunk is parsed
- THEN a `ToolCallStart` event is emitted with the tool name and ID

### Requirement: REQ-OAI-STREAM-6 — Scanner Buffer Sizing

The `bufio.Scanner` buffer MUST be created with `scanner.Buffer(make([]byte, 0, 256*1024), 256*1024)` to handle events up to 256KB. The buffer size MUST NOT be hardcoded elsewhere.

#### Scenario: Buffer configured at creation

- GIVEN an SSE response with a 200KB tool call event
- WHEN the scanner is created
- THEN the buffer accommodates the full event without error

#### Scenario: Default buffer insufficient for large events

- GIVEN a standard 64KB scanner buffer
- WHEN a 200KB event arrives
- THEN the scanner would fail — proving the 256KB requirement is necessary
