---
title: Aggregates and Invariants
sidebar_label: Aggregates and Invariants
description: The ProcessPath aggregate and its three invariants — non-empty lower-case matchPrefix, non-empty requiredCapabilities, deactivation is terminal/idempotent.
---

# Aggregates and Invariants

## Verdict: Generic Subdomain

Process Path Management is a **Generic Subdomain**, the same classification
as `facility-layout`. A process-path catalogue is a well-understood,
industry-common concern — it is not a competitive differentiator the way
`fulfillment-execution`'s Pick/Pack/SLAM task lifecycle or
`inventory-storage`'s chaotic-storage inventory truth are. What makes it
worth extracting into its own bounded context is exactly the same argument
`facility-layout` was built on: **no single existing context owns it, and
several need it identically** — `fulfillment-execution`,
`wes-work-planning`, and `workforce-management` all reference the same
`PathId` identity and the same capability requirements, but none of them
should be the single source of truth for what a path IS.

| Context | Classification | Reasoning |
| --- | --- | --- |
| `fulfillment-execution` | Core | The Pick/Pack/SLAM task lifecycle; throughput and accuracy at scale. |
| `wes-work-planning` | Core | The conductor — waveless release and flow balance. |
| `workforce-management` | Supporting | Labor & workforce allocation. |
| `labor-performance` | Supporting | Actual-vs-standard performance scoring. |
| `facility-layout` | Generic | Physical warehouse structure, extracted once rather than duplicated. |
| **`process-path-management`** | **Generic** | **The process-path catalogue — extracted once rather than duplicated across three consumers.** |

## The ProcessPath aggregate

The single aggregate root in this domain. Fields: `PathId` (immutable
identity), `MatchPrefix` (revisable), `Direct` (immutable), `RequiredCapabilities`
(revisable), `Status` (Active/Deactivated), `CreatedAt`/`UpdatedAt`.

### Invariant 1 — non-empty, lower-case matchPrefix

`MatchPrefix` must be non-empty and lower-case. Enforced identically at
`Define` and at `Revise` time (`validate` is a single shared function, not
duplicated logic that could drift). Rejected with a typed sentinel
(`ErrEmptyMatchPrefix` / `ErrMatchPrefixNotLowercase`) rather than silently
coerced — persisted data is exactly what was validated, never a
silently-lower-cased value the caller didn't actually send.

```go
if matchPrefix == "" {
    return ErrEmptyMatchPrefix
}
if matchPrefix != strings.ToLower(matchPrefix) {
    return ErrMatchPrefixNotLowercase
}
```

### Invariant 2 — non-empty requiredCapabilities

`RequiredCapabilities` must contain at least one capability. A process path
with zero required capabilities is not a meaningful business fact — every
real path (PICK, PACK, SLAM, REBIN) requires at least the capability named
after itself. Enforced by the same shared `validate` function as Invariant
1, at both `Define` and `Revise` time.

### Invariant 3 — deactivation is terminal and idempotent

Once a `ProcessPath` transitions to `Deactivated`, it is a closed historical
record:

- **Terminal.** `Revise` on a Deactivated path returns `ErrPathDeactivated`
  — there is no path back to Active. A deactivated path's identity is
  permanent (see `ErrPathAlreadyExists`'s own doc comment: re-using a
  deactivated id is refused, never silently reopened).
- **Idempotent.** Calling `Deactivate` on an already-deactivated path is a
  no-op success, not an error — matching this fleet's established
  "duplicate/redelivered command is a no-op, not an error" convention (see
  `fulfillment-execution`'s `WorkPool.Complete`). This means a retried
  operator action or a redelivered command never surfaces a spurious error,
  and — enforced one layer up, in the `DeactivatePath` use case — never
  republishes `ProcessPathDeactivated` a second time either.

```go
func (p *ProcessPath) Deactivate(now time.Time) {
    if p.status == StatusDeactivated {
        return
    }
    p.status = StatusDeactivated
    p.updatedAt = now
}
```

## Why deactivation carries no position on in-flight work

This service takes no position on work already assigned to a path in a
downstream context at the moment it is deactivated — that is each
consumer's own operational concern, the same "read model, not a command"
posture `workforce-management`'s ADR-0002 established for
`PathUnderstaffed`. `ProcessPathDeactivated` tells consumers to stop
accepting NEW work against this path; it is not a command to cancel
anything already in flight.

## Revision is a real no-op, not a spurious event

`Revise` returns a `changed` boolean. When the caller's request is
byte-for-byte identical to the path's current `MatchPrefix` and
`RequiredCapabilities`, `changed` is `false` and `RevisePath`'s use case
does not republish `ProcessPathUpdated` — consumers never have to diff two
identical payloads to notice nothing changed.
