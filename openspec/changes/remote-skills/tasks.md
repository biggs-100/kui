# Tasks: Remote Skill Fetching from Registries

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 450–550 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (Registry Client + Cache) → PR 2 (Config + Index Integration) → PR 3 (CLI/TUI Wiring) |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Registry client + disk cache + frontmatter parser | PR 1 | `go test ./internal/adapters/skills/... -run "TestRegistry\|TestCache\|TestFrontmatter" -v` | httptest server serving fake index.json + files; temp dirs for cache | `registry.go`, `cache.go`, `frontmatter.go` + tests — revert removes all remote code, no impact on local skills |
| 2 | Config URL classification + 4-layer index integration | PR 2 | `go test ./internal/adapters/skills/... -run "TestClassify\|TestRemoteLayer\|TestCollision" -v` | Mock registry returning remote skills; filesystem layers for collision tests | `index.go` changes + `classifySkillsPaths()` — revert restores 3-layer scan |
| 3 | CLI/TUI wiring + failure isolation | PR 3 | `go test ./cmd/kui/... -v 2>&1 \| head -20 && go vet ./...` | Manual: `skills: ["https://..."]` in profile config, verify warnings + local fallback | `cmd/kui/main.go` changes — revert removes registry URL plumbing |

## Slice A: Registry Client + Cache (PR 1)

### Phase 1: Foundation — Registry Client

- [x] 1.1 RED: Create `internal/adapters/skills/registry_test.go` — test `FetchIndex` returns parsed `RegistryIndex` from httptest server serving valid `index.json` (REQ-RS-1, REQ-RS-5)
- [x] 1.2 GREEN: Create `internal/adapters/skills/registry.go` — `RegistryClient` struct, `FetchIndex(ctx)` using `net/http`, parse JSON into `RegistryIndex` (REQ-RS-1, REQ-RS-5)
- [x] 1.3 RED: Add test for `FetchIndex` with malformed JSON returns parse error (REQ-RS-1)
- [x] 1.4 GREEN: Handle JSON decode error in `FetchIndex`
- [x] 1.5 RED: Add test for `FetchFile` returns content from httptest server (REQ-RS-2)
- [x] 1.6 GREEN: Implement `FetchFile(ctx, skillName, filename)` — `GET {base}/{skillName}/{filename}` (REQ-RS-2)
- [x] 1.7 RED: Add test for `FetchFile` 404 returns error (REQ-RS-2)
- [x] 1.8 RED: Add test for context cancellation aborts request (REQ-RS-8)
- [x] 1.9 GREEN: Add `http.Client` with 10s timeout; propagate context to `http.NewRequestWithContext` (REQ-RS-7, REQ-RS-8)

### Phase 2: Foundation — Disk Cache

- [x] 2.1 RED: Create `internal/adapters/skills/cache_test.go` — test `Dir()` returns correct SHA256 path from `(baseURL, skillName, version)` (REQ-RS-9)
- [x] 2.2 GREEN: Create `internal/adapters/skills/cache.go` — `Cache` struct with `Dir()`, SHA256 hex derivation (REQ-RS-9)
- [x] 2.3 RED: Add test for different versions produce different cache paths (REQ-RS-9)
- [x] 2.4 RED: Add test for `IsCached` returns true when `.kui-version` matches (REQ-RS-10, REQ-RS-12)
- [x] 2.5 GREEN: Implement `IsCached(baseURL, skillName, version)` — read `.kui-version`, compare (REQ-RS-10)
- [x] 2.6 RED: Add test for `Store` writes files atomically — temp→rename (REQ-RS-11)
- [x] 2.7 GREEN: Implement `Store(baseURL, skillName, version, files)` — create temp dir, write files, write `.kui-version`, rename to final path (REQ-RS-11)
- [x] 2.8 RED: Add test that failed `Store` leaves no partial entry at final path (REQ-RS-11)

### Phase 3: Foundation — Frontmatter Parser

- [x] 3.1 RED: Create `internal/adapters/skills/frontmatter_test.go` — test parsing valid `---` YAML frontmatter from SKILL.md content (REQ-RS-3, REQ-RS-20)
- [x] 3.2 GREEN: Create `internal/adapters/skills/frontmatter.go` — `ParseFrontmatter(data []byte) (*Meta, error)` extracting `---`-delimited YAML header (REQ-RS-3)
- [x] 3.3 RED: Add test for no frontmatter returns empty meta, no error (REQ-RS-3)
- [x] 3.4 RED: Add test for malformed frontmatter YAML returns error

### Phase 4: Slice A Integration Test

- [x] 4.1 RED: Add test for full flow: httptest registry → `FetchIndex` → `FetchFile` → `Cache.Store` → `Cache.IsCached` hit
- [x] 4.2 GREEN: Verify end-to-end registry→cache flow works with mock server

## Slice B: Config + Index Integration (PR 2)

### Phase 5: URL Classification

- [ ] 5.1 RED: Add test in `index_test.go` for `classifySkillsPaths(["go-testing", "https://r.com/skills/index.json"])` → local=["go-testing"], remote=["https://r.com/skills/index.json"] (REQ-RS-13, REQ-RS-14)
- [ ] 5.2 GREEN: Implement `classifySkillsPaths(entries []string) (localNames, registryURLs []string)` — HTTP/HTTPS prefix check (REQ-RS-14)
- [ ] 5.3 RED: Add test for all-URLs and all-names edge cases

### Phase 6: 4-Layer Index

- [ ] 6.1 RED: Add test for `NewIndex` with registry URLs returns remote skills between global and project (REQ-RS-15)
- [ ] 6.2 GREEN: Modify `NewIndex(globalDir, projectDir, profileDir string, registryURLs []string)` signature; add `scanRemote()` layer that fetches registries and adds skills with layer="remote" (REQ-RS-15, REQ-RS-17)
- [ ] 6.3 RED: Add test for remote shadows global but project/profile shadow remote (REQ-SKILL-1, REQ-RS-15)
- [ ] 6.4 RED: Add test for remote skill name prefixed with hostname: `registry.com/go-testing` (REQ-RS-16)
- [ ] 6.5 GREEN: Implement hostname extraction from registry URL and `{"host}/{name"}` prefix in `scanRemote()` (REQ-RS-16)
- [ ] 6.6 RED: Add test for empty registry URLs → only local skills indexed (REQ-RS-17)
- [ ] 6.7 RED: Add test for registry failure logs warning, local skills still present (REQ-RS-4, REQ-RS-18)
- [ ] 6.8 GREEN: Wrap registry fetches in goroutines with `sync.WaitGroup`; catch errors, `log.Printf` warnings, continue (REQ-RS-18)
- [ ] 6.9 RED: Add test for remote skills appear in `List()` alongside local skills (REQ-RS-19)
- [ ] 6.10 RED: Add test for frontmatter metadata used when no skill.yaml for remote skill (REQ-RS-20)
- [ ] 6.11 GREEN: In `scanRemote()`, try `skill.yaml` first, fall back to frontmatter parsing of `SKILL.md` (REQ-RS-3, REQ-RS-20)

### Phase 7: Update Existing Tests

- [ ] 7.1 Update all existing `NewIndex(groot, proot, froot)` calls in `index_test.go` to pass `nil` for new `registryURLs` param — all existing tests must pass unchanged

## Slice C: CLI/TUI Integration (PR 3)

### Phase 8: Wiring

- [ ] 8.1 Modify `cmd/kui/main.go` — classify `profile.Skills` entries via `classifySkillsPaths()`, pass URLs to `skills.NewIndex(...)` (REQ-RS-13)
- [ ] 8.2 Verify `go vet ./...` and `TestCoreImportsStdlibOnly` still pass
- [ ] 8.3 Manual smoke test: add `skills: ["https://httpbin.org/robots.txt"]` to a profile, run `kui tui`, verify warning logged and local skills unaffected

## Key Learnings

1. The existing `NewIndex` 3-layer scan uses `scanLayer` with filesystem walking — the remote layer needs a parallel code path since it fetches over HTTP.
2. `profile.Config.Skills []string` already accepts arbitrary strings — no schema change needed for URL entries.
3. The `Skill` struct's `Layer` field drives collision resolution — remote skills must use layer="remote" to slot between global and project.
