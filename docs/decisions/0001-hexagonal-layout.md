# ADR-0001: Hexagonal layout with stdlib-only domain core

- Status: accepted
- Date: 2026-08-14

## Context

kui starts as an empty Go repository. The agent loop is the heart of the
product (profiles, tools, and providers will be added on top), and the loop's
termination rules are business-critical. The repository is developed with
heavy AI assistance, so boundaries must be mechanically verifiable rather
than documented only. A flat package layout would let any future change slip
third-party imports or I/O into the domain, making the loop untestable
without network access and hard to reason about.

## Decision

Use a hexagonal layout:

- `internal/core` — the domain: loop (`Agent.Run`), ports (`Provider`,
  `Tool`), message types, tool registry, and typed errors. It imports only
  the standard library and performs no I/O.
- `internal/adapters` — everything that touches the outside world:
  providers and tools.
- `cmd/kui` — composition root: wires provider + tools + loop.

Dependencies point inward: adapters implement core ports, never the reverse.
A guard test (`internal/core/guard_test.go`) inspects the real dependency
graph with `go list -deps` and fails the build on any non-stdlib import.

## Alternatives considered

- **Flat single-package layout**: fastest to start, but no enforceable
  boundary; the loop could silently grow network or filesystem coupling.
- **Layered architecture (handlers/services/domain)**: adds ceremony without
  a mechanical guard; the boundary lives in convention only.
- **Plugin-style core with interface discovery**: over-engineered for the
  first change; registration happens in the composition root instead.

## Consequences

- The loop is testable with in-memory fakes (see `loop_test.go`), with no
  network or filesystem involved.
- The guard test makes the boundary executable: any future non-stdlib import
  into core fails CI.
- Moving a package across layers requires updating the guard and is a
  visible, reviewable event.
- Minor cost: adapters carry a little extra indirection compared to a flat
  layout.

## Verification notes

- Guard test: `go test ./internal/core -run TestCoreImportsStdlibOnly`.
- The guard runs as part of `go test ./...`.
- Reference: `docs/change-checklist.md` discipline — domain rules live in
  exactly one place, adapters never duplicate them.
