# Tasks: LSP Integration

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

PR 1: Protocol + Cache + Client core (~400 lines). PR 2: Client methods + Manager + Tools (~400 lines). PR 3: Runtime + File sync + TUI (~300 lines).

## Phase 1: LSP Protocol Foundation

- [x] 1.1 Create `internal/lsp/protocol.go` — LSP types. **Test**: JSON round-trip.
- [x] 1.2 Create `internal/lsp/errors.go` — error types. **Test**: error strings.

## Phase 2: LSP Client Core

- [x] 2.1 Create `internal/lsp/client.go` — Client struct, Content-Length framing. **Test**: framing round-trip via io.Pipe.
- [x] 2.2 Implement sendRequest/sendNotification with JSON-RPC 2.0. **Test**: ID matching.
- [x] 2.3 Implement Initialize handshake. **Test**: mock server handshake.
- [x] 2.4 Implement notification dispatch for publishDiagnostics. **Test**: routes correctly.

## Phase 3: LSP Client Methods

- [ ] 3.1 Implement DidOpen, DidClose with version tracking. **Test**: version increments.
- [ ] 3.2 Implement Hover. **Test**: mock returns content.
- [ ] 3.3 Implement Definition. **Test**: mock returns location.
- [ ] 3.4 Implement References. **Test**: mock returns list.

## Phase 4: Diagnostic Cache

- [x] 4.1 Create `internal/lsp/cache.go` — DiagnosticCache, sync.RWMutex, Update. **Test**: -race clean.
- [x] 4.2 Implement ByFile and Summary. **Test**: counts severities.

## Phase 5: LSP Manager

- [ ] 5.1 Create `internal/lsp/manager.go` — LSPManager, ServerState, Config. **Test**: state transitions.
- [ ] 5.2 Implement Start, Stop, Restart. **Test**: lifecycle transitions.
- [ ] 5.3 Implement ensureRunning + auto-restart. **Test**: lazy start, crash restart.

## Phase 6: LSP Tools

- [ ] 6.1 Create `internal/lsp/tool.go` — lsp_diagnostics. **Test**: returns cache data.
- [ ] 6.2 Implement lsp_hover. **Test**: delegates to client.
- [ ] 6.3 Implement lsp_definition. **Test**: delegates to client.
- [ ] 6.4 Implement lsp_references. **Test**: delegates to client.
- [ ] 6.5 Implement Tools() with lsp_ prefix. **Test**: names correct.

## Phase 7: Runtime Integration

- [ ] 7.1 Add LSP field to Runtime struct. **Test**: compiles.
- [ ] 7.2 Wire LSPManager in Build(). **Test**: gopls missing degrades gracefully.
- [ ] 7.3 Wire restart in Reload(). **Test**: clears diagnostics.
- [ ] 7.4 Wire shutdown in Close(). **Test**: stops server.

## Phase 8: File Sync Hooks

- [ ] 8.1 Define FileSyncer interface. **Test**: mock satisfies it.
- [ ] 8.2 Hook didOpen in read_file. **Test**: triggers didOpen.
- [ ] 8.3 Hook didOpen/didChange in write_file. **Test**: triggers didChange.

## Phase 9: TUI Integration

- [ ] 9.1 Add SetLSPStatus to FooterModel. **Test**: compiles.
- [ ] 9.2 Update footer Render(). **Test**: includes "LSP:".
- [ ] 9.3 Add inline diagnostics to ChatModel. **Test**: annotation below line.

## Phase 10: Testing

- [x] 10.1 Create mock_server_test.go — Content-Length framed mock. **Test**: handshake works.
- [x] 10.2 Create client_test.go — framing, crash, cancellation. **Test**: -race pass.
- [x] 10.3 Create cache_test.go — update, query, concurrency. **Test**: -race clean.
- [ ] 10.4 Create tool_test.go — wrappers with mock. **Test**: error paths.
- [ ] 10.5 Create manager_test.go — state machine. **Test**: table-driven.
- [ ] 10.6 Add go-lsp/lspclient to go.mod. **Test**: go build passes.
