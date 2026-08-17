# Proposal: Remote Skill Fetching from Registries

## Intent

kui skills are local-only — discovered from `global → project → profile` filesystem layers. There's no way to share skills across teams or machines without manually copying files. Adding remote skill fetching from registries enables a shared skill ecosystem: teams publish skills to a URL, users add them via config, kui fetches and caches them locally. This achieves parity with OpenCode's remote skill support and follows the same registry protocol (`index.json` + files).

## Scope

### In Scope
- HTTP client for registry fetching (`net/http` stdlib)
- Registry protocol: `index.json` manifest + per-skill `skill.yaml` / `SKILL.md` files
- Disk cache: `{configRoot}/skills/cache/{sha256hex}/`, version tracking, atomic swap
- Config: `skills: []string` accepts URLs alongside local names (reutiliza field existente)
- Integration: remote skills layer between global and project in the Index scan order
- Failure isolation: registry failures log warnings, local skills always work
- Dual metadata: `skill.yaml` (local) and YAML frontmatter in `SKILL.md` (remote)

### Out of Scope
- Registry server implementation (users provide their own or use existing hosting)
- Authentication / API keys for private registries (deferred)
- Automatic skill updates / polling (manual fetch on startup only)
- Skill dependency resolution (flat list, no DAG)
- Web UI for browsing registries

## Capabilities

### New Capabilities
- `remote-skill-fetch`: HTTP client, registry protocol parsing, cache management, atomic swap

### Modified Capabilities
- `profile-skills`: Index scan order changes from `global → project → profile` to `global → remote → project → profile`; `skills: []string` field now accepts URLs; remote skill metadata parsing supports frontmatter in SKILL.md

## Approach

Hexagonal: core defines `RegistryClient` port (FetchIndex, FetchSkill). Adapters implement HTTP transport and disk cache.

Registry protocol (OpenCode-compatible):
- `index.json`: `{ "skills": [{ "name": "...", "description": "...", "triggers": [...], "version": "..." }] }`
- Per-skill: `skill.yaml` (preferred) or YAML frontmatter in `SKILL.md` (fallback)
- Files served from same base URL as `index.json`

Cache strategy:
- SHA256 of `(registryURL, skillName, version)` → cache dir path
- Atomic swap: download to temp dir, rename on success
- On startup: fetch index, compare versions, update only changed skills
- Cache is warm after first fetch; subsequent startups skip HTTP if versions match

Config integration:
- `skills: ["go-testing", "https://example.com/skills/index.json"]` — strings are local names or registry URLs
- URLs are fetched at startup, names resolve from local layers as before
- Profile `skills` field continues to work unchanged — just extended with URL entries

Index layer insertion:
- Remote skills scan between global and project: `global → remote → project → profile`
- Remote skills never shadow profile skills (profile is always nearest)
- Same collision logic: nearer layer wins, shadowed layers recorded

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/skills/registry/` | New | HTTP client, registry protocol, index.json parsing |
| `internal/skills/cache/` | New | Disk cache, SHA256 paths, atomic swap |
| `internal/adapters/skills/index.go` | Modified | New `scanRemote()` layer, URL detection in config |
| `internal/adapters/profile/loader.go` | Modified | `Skills []string` now accepts URLs (no schema change) |
| `cmd/kui/main.go` | Modified | Initialize registry client, pass to index builder |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Registry unreachable at startup | Med | Log warning, skip remote layer, local skills unaffected |
| Malformed index.json or skill files | Low | Validate schema, skip invalid entries with warnings |
| Cache corruption | Low | Atomic swap (write temp → rename); verify on load |
| Skill name collision remote vs local | Med | Nearer layer wins (project/profile always beat remote) |
| Large registry index slows startup | Low | Cache-first: skip HTTP if cached version matches |

## Rollback Plan

Remove `internal/skills/registry/` and `internal/skills/cache/`. Revert `index.go` to 3-layer scan only. Remove URL entries from config (or keep — they'll be ignored). No data migration — cache directory is inert. kui reverts to local-only skills.

## Dependencies

- None (pure Go, stdlib `net/http`, `crypto/sha256`)

## Success Criteria

- [ ] `skills: ["https://example.com/skills/index.json"]` fetches and caches remote skills
- [ ] Remote skills appear in index between global and project layers
- [ ] Cache hit skips HTTP on subsequent startups
- [ ] Registry failure logs warning, local skills still work
- [ ] Remote skills respect same collision rules as local layers
- [ ] `TestCoreImportsStdlibOnly` guard test still passes
- [ ] All existing tests pass unmodified
