---
title: Ubiquitous Language
sidebar_label: Ubiquitous Language
description: ProcessPath, PathId, Capability, MatchPrefix, Direct, Status — pulled from the domain code's own doc comments.
---

# Ubiquitous Language

The definitions below are pulled directly from
`internal/domain/processpath/process_path.go` and
`internal/domain/shared/shared.go`'s own doc comments — not reinvented for
this page.

## ProcessPath

> `ProcessPath` implements the operator-configurable definition of one
> process path (its canonical identity, the path_id-family match rule
> downstream consumers use, and the capabilities a station/associate must
> hold to work it).

The aggregate root. Identity (`PathId`) is immutable once constructed;
`MatchPrefix` and `RequiredCapabilities` may be revised while Active via
`Revise`; `Direct` is immutable (it describes a structural fact about the
path's routing shape, not an operational parameter operators tune).

This aggregate replaces what was, before this service existed, a static
YAML file
(`warehouse-infra/config/process-paths/sortable-fc.yaml`) loaded once at
boot by three other services. The schema is carried over field-for-field
from that file's own documented schema so this migration is a like-for-like
data model change, not a redesign.

## PathId

> `PathId` is the canonical identity of a process path (e.g. `"PICK"`,
> `"PACK"`, `"REBIN"`, `"SLAM"`). It is the SAME identity
> `fulfillment-execution`'s `task.Type`, `wes-work-planning`'s
> `WorkPool.PathId`, and `workforce-management`'s `PathPlan.PathId` all
> reference — this service is the one place that identity is DEFINED, not
> just consumed.

Kept as a plain string type (not an enum) because the whole point of this
service existing is that the valid set is operator-configurable, not
compiled in.

## Capability

> `Capability` is a named qualification a station/associate must hold to
> work a process path (e.g. `"pick"`, `"pack"`, `"hazmat"`) — the exact
> same vocabulary `workforce-management`'s `Certification` and
> `fulfillment-execution`'s `Station.Capability` already use.

This service does not invent a new capability vocabulary; it is the
authoritative SOURCE for which capabilities a given path requires, so the
existing vocabulary is carried here as a plain string, not redefined.

## MatchPrefix

The lower-case prefix downstream consumers match a caller-supplied id
against: `id == matchPrefix` OR `id` starts with `matchPrefix + "-"` —
never a bare substring match without the separator (a hypothetical
`"picking-station"` must not match `"pick"`). Validated lower-case at
construction time — not lower-cased for the caller — so persisted data is
exactly what was validated, never a silently-transformed value. See this
fleet's own documented pitfall on exact-match-vs-prefix-match design for
why this matters: a published-language lookup that defaults to exact
matching against a bare canonical id passes synthetic test fixtures while
rejecting every real caller-supplied value in production.

## Direct

A structural fact about the path's routing shape (reserved for a future
multi-hop topology, not a day-to-day operational parameter). Immutable once
set at `Define` time — never revisable via `Revise`.

## Status (Active / Deactivated)

> `Status` is the activation lifecycle of a ProcessPath. There is no
> "draft" state — a path is live the instant it is defined, since the
> whole purpose of this service is operators configuring paths that take
> effect immediately, not a review workflow.

Two values: `ACTIVE` and `DEACTIVATED`. Deactivation is a one-way,
idempotent transition — see
[Aggregates & Invariants](/docs/ddd/aggregates-and-invariants) for the full
invariant.
