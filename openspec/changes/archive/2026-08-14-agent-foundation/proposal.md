# Proposal: Agent Foundation (kui)

## Intent

kui is an empty Go repository (initial commit 065febc, no go.mod). This change bootstraps the agent runtime foundation: Go module, hexagonal layout with architecture guard test, working agent loop (prompt → tool call → tool result → answer), one provider adapter, and a minimal CLI proving the loop end-to-end. Profiles, TUI, and MCP support land in later changes on this verified core.

## Scope

### In Scope
- `go.mod`/`go.sum` for `github.com/biggs-100/kui` (Go 1.26)
- Hexagonal layout: `internal/core` (stdlib-only domain), adapters, `cmd/kui`
- Architecture guard test (core imports stdlib only)
- Agent loop with tool dispatch and bounded iterations
- Tools: `read_file`, `write_file`, `bash` (with timeout)
- OpenAI-compatible provider; base URL + key via env vars; clear error when key missing
- ADRs: hexagonal layout, env-based credentials, OpenAI-compatible-only

### Out of Scope
- Profiles, hot switching, TUI (Bubble Tea), MCP servers, plugins, skills (next changes)
- Non-compatible providers; streaming; session persistence/history

## Capabilities

> Contract with sdd-spec. No existing specs (greenfield).

### New Capabilities
- `agent-loop`: core loop (prompt → tool → result → answer), tool contract, termination rules
- `agent-tools`: built-in `read_file`, `write_file`, `bash` with timeout and path constraints
- `provider-openai-compatible`: chat-completions adapter, env credentials, base-URL override, error surface
- `agent-cli`: minimal CLI running the loop

### Modified Capabilities
None — greenfield, no existing specs.

## Approach

- TDD bootstrap per config (`tdd: true`, runner `go test ./...`)
- `internal/core`: loop + Provider/Tool ports, zero third-party deps, enforced by guard test (ai-project-foundation pattern)
- Provider adapter via `net/http`, httptest fixtures, lenient JSON parsing
- `cmd/kui`: thin CLI wiring core + default provider/tools
- Layered verification: unit (loop/tools) → integration (httptest) → CLI smoke via `go run`

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `go.mod`, `go.sum` | New | Module bootstrap |
| `internal/core/` | New | Domain: loop, ports, guard test |
| `internal/adapters/providers/openai/` | New | OpenAI-compatible client |
| `internal/adapters/tools/` | New | read_file, write_file, bash |
| `cmd/kui/` | New | Minimal CLI |
| `openspec/specs/` | New | Specs from this change |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Provider response variance across compatible servers | Med | httptest fixtures, lenient parsing, documented base-URL config |
| Shell tool security (escaping, runaway processes) | Med | Mandatory timeout, no interactive input, file-scoped ops |
| Loop overbuild (history, streaming) | Med | Strict scope: single-session loop, no persistence |
| Toolchain drift (Go 1.26.1, golangci-lint 2.12.2) | Low | Pin versions in CI config; verify via build/lint |

## Rollback Plan

Drop the branch pre-merge; post-merge `git revert` the change commit(s). Never merge delta specs into `openspec/specs/`. Greenfield — no migrations or data.

## Dependencies

- Go 1.26 toolchain; golangci-lint 2.12.2 (both detected)
- Runtime network access to a chat-completions endpoint
- Core stays stdlib-only; no third-party runtime deps

## Success Criteria

- [ ] `go test ./...` green, including architecture guard test
- [ ] `go build ./...` and `golangci-lint run ./...` clean
- [ ] Live run `kui "list files in ."` completes the loop via the bash tool
- [ ] Missing `OPENAI_API_KEY` yields an actionable error
- [ ] ADRs committed for hexagonal layout and env-based credentials
