# Domain model — ubiquitous language, aggregates, invariants, domain events

## Ubiquitous Language (use these exact names)

- **ProcessPath** — the aggregate root. `PathId` (identity, immutable once
  constructed), `MatchPrefix` and `RequiredCapabilities` (revisable while
  Active via `Revise`), `Direct` (immutable — a structural fact about
  routing shape, reserved for a future multi-hop topology, never an
  operator-tunable parameter), `Status` (Active/Deactivated). This
  aggregate is a like-for-like migration of the retired YAML file's
  schema, field-for-field — not a redesign.
- **PathId** — the canonical identity of a process path (e.g. `"PICK"`,
  `"PACK"`, `"REBIN"`, `"SLAM"`). The SAME identity `fulfillment-execution`'s
  `task.Type`, `wes-work-planning`'s `WorkPool.PathId`, and
  `workforce-management`'s `PathPlan.PathId` all reference — this service
  is the one place that identity is DEFINED, not just consumed. Kept as a
  plain string (not an enum) because the whole point of this service is
  that the valid set is operator-configurable, not compiled in.
- **Capability** — a named qualification a station/associate must hold to
  work a process path (e.g. `"pick"`, `"pack"`, `"hazmat"`) — the exact
  same vocabulary `workforce-management`'s `Certification` and
  `fulfillment-execution`'s `Station.Capability` already use. This service
  does not invent a new capability vocabulary; it is the authoritative
  SOURCE for which capabilities a given path requires.
- **MatchPrefix** — the lower-case prefix downstream consumers match a
  caller-supplied id against: `id == matchPrefix` OR `id` starts with
  `matchPrefix + "-"` — **never** a bare substring match without the
  separator (a hypothetical `"picking-station"` must not match `"pick"`).
  Validated lower-case at construction time (`ErrMatchPrefixNotLowercase`)
  — never silently lower-cased for the caller, so persisted data is
  exactly what was validated. This is the fleet's documented pitfall on
  exact-match-vs-prefix-match design: an exact-match default passes
  synthetic fixtures (`"PICK"` in, `"PICK"` out) while rejecting every
  real caller-supplied value (`"pick-zone-a"`) in production.
- **Direct** — a structural fact about the path's routing shape (reserved
  for a future multi-hop topology). Immutable once set at `Define` time —
  never revisable via `Revise`.
- **Status (ACTIVE / DEACTIVATED)** — the activation lifecycle. There is
  no "draft" state — a path is live the instant it is defined; this
  service has no review workflow. Deactivation is one-way and idempotent
  (`Deactivate` on an already-deactivated path is a no-op success, not an
  error) — matches the fleet's "duplicate/redelivered command is a no-op"
  convention (see `fulfillment-execution`'s `WorkPool.Complete`).
  `ErrPathDeactivated` blocks any mutation on a deactivated path: it is a
  closed historical record, the same posture `labor-performance`'s
  `LaborStandard.Close` takes.

## Aggregates & invariants (enforce in domain, unit-tested)

- **ProcessPath.Define**: `matchPrefix` must be non-empty
  (`ErrEmptyMatchPrefix`) and lower-case (`ErrMatchPrefixNotLowercase`);
  `requiredCapabilities` must be non-empty (`ErrNoRequiredCapabilities`). A
  malformed path definition must never be constructible — no "fix it
  later" state.
- **ProcessPath.Revise**: only permitted while Active
  (`ErrPathDeactivated` otherwise). Returns `changed bool` — a no-op
  revision (identical `matchPrefix` and `requiredCapabilities`) returns
  `false` and raises nothing, so consumers never have to diff two
  identical payloads to notice nothing changed. `pathId` and `direct` are
  never revisable.
- **ProcessPath.Deactivate**: idempotent — deactivating an
  already-deactivated path is a no-op success and does NOT republish
  `ProcessPathDeactivated`.
- **A path's identity is permanent once created.** `DefinePath` rejects
  (409) re-defining an id that already exists, active or deactivated — a
  caller wanting to re-use an id after deactivation must be told
  explicitly, never silently overwritten.

## Domain events (past tense, on `warehouse.process-path-management.events`)

`ProcessPathCreated`, `ProcessPathUpdated`, `ProcessPathDeactivated` —
defined in `internal/domain/shared/events.go`. One shared topic (not one
per event type), matching the fleet's convention; every message is keyed
by `path_id` so a consumer replaying the topic sees one path's events in
publish order. `ProcessPathDeactivated`'s payload carries only `path_id` —
`match_prefix`/`direct`/`required_capabilities` are omitted (not
empty-arrayed) since a deactivation carries no definition data.
