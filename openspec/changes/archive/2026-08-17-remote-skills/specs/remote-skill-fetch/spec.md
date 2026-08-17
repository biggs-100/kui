# remote-skill-fetch Specification

## Purpose

HTTP client and disk cache for fetching skills from remote registries. Implements the OpenCode-compatible registry protocol (`index.json` + per-skill files) with atomic cache swap and version tracking.

## Requirements

### Requirement: REQ-RS-1 — Registry Serves index.json

The registry MUST serve an `index.json` manifest at its base URL containing an array of skill objects. Each object MUST include `name`, `description`, `triggers`, and `version` fields.

#### Scenario: Valid index fetch

- GIVEN a registry at "https://example.com/skills/"
- WHEN the client fetches index.json
- THEN a JSON object with a "skills" array is returned
- AND each entry contains name, description, triggers, version

#### Scenario: Malformed index

- GIVEN a registry returning invalid JSON
- WHEN the client fetches index.json
- THEN the client returns a parse error
- AND the registry is skipped with a warning log

### Requirement: REQ-RS-2 — Registry Serves Skill Files

The registry MUST serve individual skill files (SKILL.md, skill.yaml) at `/{skillName}/{filename}` relative to the base URL.

#### Scenario: Fetch SKILL.md

- GIVEN a skill "go-testing" in the registry
- WHEN the client requests /go-testing/SKILL.md
- THEN the file content is returned

#### Scenario: Missing skill file

- GIVEN a skill name with no files on the server
- WHEN the client requests the file
- THEN a 404 is returned and the entry is skipped with a warning

### Requirement: REQ-RS-3 — OpenCode Protocol Compatibility

The registry protocol MUST be compatible with OpenCode's `index.json` format. Skills may provide metadata via `skill.yaml` (preferred) or YAML frontmatter in `SKILL.md` (fallback).

#### Scenario: skill.yaml preferred

- GIVEN a skill with both skill.yaml and SKILL.md frontmatter
- WHEN metadata is parsed
- THEN skill.yaml values take precedence

#### Scenario: Frontmatter fallback

- GIVEN a skill with only SKILL.md frontmatter (no skill.yaml)
- WHEN metadata is parsed
- THEN frontmatter values are used

### Requirement: REQ-RS-4 — Registry Is Optional

The system MUST function fully without any configured registry. Local skills (global, project, profile layers) MUST work independently of remote skill availability.

#### Scenario: No registry configured

- GIVEN a profile with only local skill names
- WHEN the index is built
- THEN all local skills are discovered normally
- AND no HTTP requests are made

#### Scenario: Registry configured but unreachable

- GIVEN a profile with a registry URL that times out
- WHEN the index is built
- THEN a warning is logged
- AND local skills are still available

### Requirement: REQ-RS-5 — Client Fetches index.json

The HTTP client MUST fetch `index.json` from a configured registry URL using `net/http`. The client MUST set a reasonable timeout (default 10s) per request.

#### Scenario: Successful fetch

- GIVEN a reachable registry URL
- WHEN index.json is fetched
- THEN the parsed skill list is returned

#### Scenario: DNS resolution failure

- GIVEN a registry URL with unresolvable hostname
- WHEN index.json is fetched
- THEN a descriptive error is returned within the timeout window

### Requirement: REQ-RS-6 — Client Downloads Skill Files

The HTTP client MUST download individual skill files from the registry. Files MUST be written to a temporary directory before atomic promotion.

#### Scenario: Download complete skill

- GIVEN a skill with SKILL.md and skill.yaml
- WHEN the skill is downloaded
- BOTH files are present in the temp directory

#### Scenario: Partial download failure

- GIVEN a skill where SKILL.md succeeds but skill.yaml fails
- WHEN the download completes
- THEN the skill is skipped (incomplete) and a warning is logged

### Requirement: REQ-RS-7 — Graceful Network Error Handling

The client MUST handle network errors (timeout, DNS failure, connection refused, TLS errors) gracefully. Errors MUST be logged as warnings and MUST NOT crash the index build.

#### Scenario: Timeout

- GIVEN a registry that never responds
- WHEN the 10s timeout expires
- THEN a timeout warning is logged and the registry is skipped

#### Scenario: Connection refused

- GIVEN a registry URL pointing to a closed port
- WHEN the client connects
- THEN a connection-refused warning is logged and the registry is skipped

### Requirement: REQ-RS-8 — Context Cancellation

All HTTP requests MUST respect context cancellation. When the parent context is cancelled, in-flight requests MUST abort promptly.

#### Scenario: Cancelled context

- GIVEN a fetch in progress
- WHEN the context is cancelled
- THEN the HTTP request aborts and the function returns the context error

### Requirement: REQ-RS-9 — Cache Directory Structure

Fetched skills MUST be cached at `{configRoot}/skills/cache/{sha256hex}/`. The SHA256 hex digest is derived from `(registryURL, skillName, version)`.

#### Scenario: Cache path derivation

- GIVEN registry "https://r.com/skills/", skill "go-testing", version "1.0"
- WHEN the cache path is computed
- THEN it is `{configRoot}/skills/cache/{sha256("https://r.com/skills/go-testing/1.0")}/`

#### Scenario: Different versions produce different paths

- GIVEN the same skill at version "1.0" and "2.0"
- WHEN cache paths are computed
- THEN the two paths are different

### Requirement: REQ-RS-10 — Version Tracking

Each cached skill directory MUST contain a `.kui-version` file holding the cached version string. This enables cache-hit checks without re-downloading.

#### Scenario: Version file present

- GIVEN a cached skill at version "1.0"
- WHEN .kui-version is read
- THEN it contains "1.0"

#### Scenario: Version mismatch triggers re-download

- GIVEN cached version "1.0" and registry version "2.0"
- WHEN the cache is checked
- THEN a re-download is triggered

### Requirement: REQ-RS-11 — Atomic Cache Swap

Cache writes MUST use atomic swap: download to a staging temp directory, then rename to the final cache path on success. This prevents partial/corrupt cache entries.

#### Scenario: Successful atomic write

- GIVEN a new skill download
- WHEN files are written to staging
- AND staging is renamed to the final cache path
- THEN the cache entry is complete and consistent

#### Scenario: Failed download leaves no residue

- GIVEN a download that fails midway
- WHEN the staging directory is cleaned up
- THEN no partial entry exists at the final cache path

### Requirement: REQ-RS-12 — Cache Hit Skips Download

When the cached version matches the registry version, the client MUST skip the HTTP download and use the cached files directly.

#### Scenario: Cache hit

- GIVEN cached version "1.0" matches registry version "1.0"
- WHEN the skill is loaded
- THEN no HTTP request is made
- AND the cached SKILL.md is used

#### Scenario: Cache miss

- GIVEN cached version "1.0" but registry version "2.0"
- WHEN the skill is loaded
- THEN a fresh download occurs
- AND the cache is updated

### Requirement: REQ-RS-17 — NewIndex Accepts Registry URLs

The `NewIndex` function MUST accept a list of registry URLs as a parameter. URLs are fetched concurrently at index build time.

#### Scenario: Multiple registries

- GIVEN two registry URLs in the config
- WHEN NewIndex is called
- THEN both registries are fetched concurrently
- AND skills from both are merged into the index

#### Scenario: Empty URL list

- GIVEN no registry URLs
- WHEN NewIndex is called
- Then only local skills are indexed

### Requirement: REQ-RS-18 — Registry Failures Log Warnings

Registry fetch failures MUST log warnings and MUST NOT fail the entire index build. Local skills MUST remain available regardless of registry state.

#### Scenario: One registry fails

- GIVEN two registries, one unreachable
- WHEN the index is built
- THEN a warning is logged for the failed registry
- AND skills from the successful registry are included
- AND all local skills are included

### Requirement: REQ-RS-19 — Remote Skills Appear in List

Remote skills MUST appear in `skills.List()` alongside local skills. Callers MUST NOT need to distinguish remote from local skills in the listing.

#### Scenario: Mixed listing

- GIVEN local skills "a", "b" and remote skills "c", "d"
- WHEN skills.List() is called
- THEN all four skills are returned

### Requirement: REQ-RS-20 — Remote Skills Support Frontmatter

Remote skills fetched from registries MUST support the same `SKILL.md` frontmatter format as local skills. Metadata parsing MUST be consistent across local and remote skills.

#### Scenario: Frontmatter parsed from remote SKILL.md

- GIVEN a remote skill with YAML frontmatter in SKILL.md
- WHEN the skill metadata is extracted
- THEN name, description, and triggers match the frontmatter values
