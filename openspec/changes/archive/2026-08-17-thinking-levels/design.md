# Design: Thinking Levels (--thinking flag)

## Technical Approach

Add a `--thinking` CLI flag with four kui-native levels (off, low, medium, high) that map to OpenAI `reasoning_effort` values at the provider boundary. The level flows through layered config (global→project→profile) and CLI override, then is wired to the OpenAI client via `SetThinking()`. SSE parsing extracts `reasoning_content` deltas into `StreamChunk.ReasoningDelta` (field already exists in core types but is never populated). TUI rendering of reasoning tokens is explicitly out of scope.

## Architecture Decisions

### D1: Thinking Level Abstraction

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Raw provider values (low/medium/high) | Simple; no mapping needed; leaks provider semantics | Rejected |
| 4 kui-native levels mapped at provider boundary | Extra mapping layer; provider-independent; extensible | **Chosen** |
| Full provider-specific enums per provider | Too much complexity for OpenAI-only scope | Rejected |

**Rationale**: 4 levels (off/low/medium/high) balance simplicity with flexibility. Mapping happens once in `openai/client.go` — `off`/`low` → omit field; `medium`/`high` → pass through. Keeps kui-native semantics clean.

### D2: SetThinking Pattern (Follow SetModel)

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Interface method on Provider | Requires core change; premature abstraction | Rejected |
| State on concrete Client type | Matches SetModel pattern; no interface change | **Chosen** |
| Constructor parameter | Can't override per-run after construction | Rejected |

**Rationale**: The codebase already uses `SetModel(model string)` on `openai.Client` — a stateful setter on the concrete type (line 70-72 of client.go). `SetThinking` follows the identical pattern: store a `thinkingLevel string` field, use it when building `chatRequest`.

### D3: Request Wiring — Pointer with omitempty

| Option | Tradeoff | Decision |
|--------|----------|----------|
| `ReasoningEffort string` always serialized | Empty string sent to API; non-reasoning models get unknown field | Rejected |
| `ReasoningEffort *string` with omitempty | Nil pointer = field omitted from JSON; clean API contract | **Chosen** |
| Conditional marshaling (custom MarshalJSON) | Over-engineered for one field | Rejected |

**Rationale**: `*string` + `omitempty` is the idiomatic Go pattern for optional JSON fields. When thinking is "off" or "", the pointer stays nil → field absent from request body. When non-off, pointer set → field present. Matches OpenAI API: unknown fields silently ignored for non-reasoning models.

### D4: Profile Config — New `thinking` Field

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Separate thinking config file | Extra file to manage; breaks profile cohesion | Rejected |
| `thinking` field in profile.yaml | Natural fit; follows existing model/tools pattern | **Chosen** |
| Environment variable (KUI_THINKING) | Bypasses profile system; inconsistent | Rejected |

**Rationale**: Profile.yaml already has `model`, `tools`, `skills`. Adding `thinking` is structurally identical. The `resolve()` function in loader.go already merges scalar fields nearest-first — `Thinking` follows the same pattern as `Model`.

### D5: SSE Reasoning Parsing

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Parse in streamDelta struct | Minimal change; add one field to existing struct | **Chosen** |
| Separate reasoning parser | Over-engineered; duplicating SSE logic | Rejected |

**Rationale**: The `streamDelta` struct (sse.go:69-72) needs only one new field: `ReasoningContent string`. The `parseSSEChunk` function (sse.go:104) already checks `delta.Content` — add parallel check for `delta.ReasoningContent`. `StreamChunk.ReasoningDelta` already exists in core types (stream.go:7), never populated — this change fills it.

## Data Flow

    CLI --thinking high
         │
         ▼
    Options.Thinking = "high"
         │
         ▼
    resolveThinking() ──→ validates against {off,low,medium,high}
         │
         ▼
    client.SetThinking("high") ──→ Client.thinkingLevel = "high"
         │
         ▼
    chatRequest.ReasoningEffort = &"high"  (pointer set)
         │
         ▼
    JSON: {"reasoning_effort":"high", ...}
         │
         ▼
    SSE: delta.reasoning_content → StreamChunk.ReasoningDelta
         │
         ▼
    TUI: rendered in distinct style (faint/italic)  ← OUT OF SCOPE

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `cmd/kui/flags.go` | Modify | Add `Thinking string` to Options; add `"thinking": true` to stringFlags |
| `internal/adapters/providers/openai/client.go` | Modify | Add `thinkingLevel string` field; add `SetThinking()`; add `ReasoningEffort *string` to chatRequest; wire in Chat() and StreamChat() |
| `internal/adapters/providers/openai/sse.go` | Modify | Add `ReasoningContent string` to streamDelta; extract in parseSSEChunk |
| `internal/adapters/profile/loader.go` | Modify | Add `Thinking string` to Config and Profile; merge in resolve() |
| `cmd/kui/main.go` | Modify | Add `resolveThinking()` function; wire to client.SetThinking(); update usage string |
| `cmd/kui/main.go` | Modify | Add `profileThinking()` subcommand handler; add case in runProfile switch |

## Interfaces / Contracts

```go
// flags.go — Options struct addition
type Options struct {
    // ... existing fields ...
    Thinking string
}

// client.go — SetThinking follows SetModel pattern
func (c *Client) SetThinking(level string) {
    c.thinkingLevel = level
}

// client.go — chatRequest addition
type chatRequest struct {
    // ... existing fields ...
    ReasoningEffort *string `json:"reasoning_effort,omitempty"`
}

// sse.go — streamDelta addition
type streamDelta struct {
    // ... existing fields ...
    ReasoningContent string `json:"reasoning_content"`
}

// profile loader.go — Config addition
type Config struct {
    // ... existing fields ...
    Thinking string `yaml:"thinking"`
}

// profile loader.go — Profile addition
type Profile struct {
    // ... existing fields ...
    Thinking string
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | parseFlags with --thinking (space, equals, invalid, empty) | Table-driven tests in flags_test.go |
| Unit | SetThinking + chatRequest serialization | Test that nil pointer omits field, non-nil includes it |
| Unit | parseSSEChunk with reasoning_content | Test SSE chunk parsing with/without reasoning field |
| Unit | resolveThinking validation | Test valid levels pass, invalid rejected with actionable error |
| Unit | Profile merge for Thinking field | Test layered resolution (global→project→profile) |
| Integration | Full runPrompt with --thinking wired | Verify reasoning_effort appears in request body via httptest |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration required. The `thinking` field in profile.yaml is optional — existing profiles without it resolve to "off" (the default). No data migration, no feature flags, no phased rollout needed.

## Open Questions

- [ ] TUI reasoning rendering is explicitly out of scope — confirm no interim display needed during streaming
- [ ] Whether `resolveThinking()` should live in flags.go or main.go (follows `resolveModel` pattern in main.go)

## Key Learnings

1. StreamChunk.ReasoningDelta already exists in core types but is never populated by any provider.
2. The SetModel pattern on openai.Client provides the exact template for SetThinking.
3. Profile config layered merge in loader.go uses nearest-first scalar resolution — Thinking field follows identical pattern.
4. The chatRequest struct already uses omitempty for optional fields — *string pointer pattern is idiomatic here.
