# Tasks: Agent Foundation (kui)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1,500 (greenfield, 20+ new files) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 core → PR 2 tools → PR 3 provider → PR 4 CLI+ADRs |
| Delivery strategy | auto-chain |
| Chain strategy | pending (recommend: stacked-to-main) |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

Commit discipline: work-unit commits — tests with code, one responsibility, reversible (work-unit-commits skill).

### Suggested Work Units

| Unit | Goal | PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|----|---------------------|-----------------|------------------|
| 1 | Module bootstrap + core domain (ports, errors, loop, guard) | PR 1 | `go test ./internal/core/...` | N/A — no runnable process until adapters exist | Revert PR 1 (go.mod + internal/core) |
| 2 | Built-in tools + tests | PR 2 | `go test ./internal/adapters/tools/...` | `go test ./internal/adapters/tools/ -run TestBash -v` — real subprocess kill | Revert PR 2 |
| 3 | OpenAI-compatible provider + httptest | PR 3 | `go test ./internal/adapters/providers/openai/...` | httptest fixtures only; no live key in CI | Revert PR 3 |
| 4 | CLI wiring + ADRs + smoke | PR 4 | `go test ./...` | `go run ./cmd/kui` no args → exit 2 | Revert PR 4 |

## Phase 1: Foundation (module + core domain)

- [x] 1.1 Create `go.mod`/`go.sum`: module `github.com/biggs-100/kui`, go 1.26, no runtime deps
- [x] 1.2 Create `internal/core/provider.go` + `tool.go`: Message/ToolCall types, Provider/Tool ports, ordered Registry (D2–D4)
- [x] 1.3 Create `internal/core/errors.go`: UnknownToolError, IterationLimitError, ToolError (D5–D7)

## Phase 2: Core loop (TDD)

- [x] 2.1 RED `internal/core/loop_test.go`: direct answer, multi-step tool resolution, unknown tool (typed error, zero further calls), budget 3 exhausted, tool failure names tool — in-memory fakes
- [x] 2.2 GREEN `internal/core/loop.go`: Agent.Run, dispatch, termination rules (REQ-LOOP-1..4)
- [x] 2.3 Create `internal/core/guard_test.go`: `go list` stdlib-only import guard (D1)
- [x] 2.4 Commit work unit 1: go.mod + internal/core + tests + guard

## Phase 3: Tools adapter (TDD)

- [x] 3.1 RED `internal/adapters/tools/*_test.go`: read existing/missing/escape rejected, create/overwrite/escape, echo exit 0, non-zero exit, timeout kill (t.TempDir; 1s vs 10s sleep) — REQ-TOOLS-1..3 + bash subprocess boundary
- [x] 3.2 Create `path.go`, `read_file.go`, `write_file.go`: Abs+EvalSymlinks+Rel, reject before IO (D11)
- [x] 3.3 Create `bash.go`: CommandContext+WithTimeout, nil stdin, kill + TimeoutError (D12)
- [x] 3.4 Create `registry.go`: default set read_file/write_file/bash + enumeration test (REQ-TOOLS-4)
- [x] 3.5 Commit work unit 2

## Phase 4: Provider adapter (TDD)

- [x] 4.1 RED `internal/adapters/providers/openai/client_test.go` (httptest): tool-call response, malformed JSON, 401/429/500, Bearer header, key absent/present, custom + default base URL (REQ-PROV-1..4)
- [x] 4.2 Create `client.go`: env creds at construction naming OPENAI_API_KEY, POST {base}/chat/completions, typed errors, key never in errors (D8–D10)
- [x] 4.3 Commit work unit 3

## Phase 5: CLI + wiring

- [x] 5.1 Create `cmd/kui/main.go`: wire provider + tools + loop; exit 0/1/2; answer stdout, errors stderr (D13; REQ-CLI-1..2)
- [x] 5.2 Smoke: `go run ./cmd/kui` — no prompt → usage + exit 2; missing key → actionable error + exit 1
- [x] 5.3 Commit work unit 4

## Phase 6: ADRs + verification

- [x] 6.1 Create ADRs: `docs/decisions/0001-hexagonal-layout.md`, `0002-env-based-credentials.md`, `0003-openai-compatible-only.md`
- [x] 6.2 Full verify: `go test ./...`, `go build ./...`, `golangci-lint run ./...`, `gofmt -l .`
- [x] 6.3 Commit ADRs + finalize docs

## Notes (work unit 4, apply)

- User-approved extension: every chat request now carries an explicit `model`
  field from `OPENAI_MODEL` (default `gpt-4o-mini`), implemented in
  `client.go` with `TestChatSendsDefaultModel` / `TestChatSendsConfiguredModel`.
  REQ-PROV-1 is not violated (it constrains messages + tools, not extra
  fields) — additive, no spec change; flagged in the apply report.
- CLI smoke is automated as subprocess tests (`cmd/kui/main_test.go`,
  TestMain builds the binary): no args → exit 2 + usage on stderr; missing
  key → exit 1 + OPENAI_API_KEY error on stderr; provider 500 → exit 1;
  fake provider → exit 0 + answer on stdout. Manual `go run` smoke confirms
  the same (note: `go run` collapses exit code 2 to 1; the binary itself
  exits 2).
