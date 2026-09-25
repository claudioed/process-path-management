---
id: 0009-destination-location-role-on-process-path
slug: /adr/0009-destination-location-role-on-process-path
title: 0009. Optional destination LocationRole on a ProcessPath
sidebar_label: 0009. Destination LocationRole
description: ADR 0009 — an optional, immutable, Define-time-only DestinationLocationRole (Drop/WorkCenter/Shipping) on ProcessPath, validated locally and never against a live facility-layout call, preserving this service's zero-inbound-dependency boundary (ADR-0001) while closing facility-layout's Phase B3 follow-on gap.
---

# 0009. Optional destination LocationRole on a ProcessPath

## Status

Accepted

## Context

facility-layout's own ADR-0016 introduced `LocationRole` (Storage, Dock,
WorkCenter, Yard, Drop, Staging, QC, Consolidation, Shipping) as a
first-class attribute of a `LocationSlot` — what a coded location is
*for*. That same ADR's Phase B3 follow-on list named this exact
integration as this service's piece of the work: "process-path-management:
optional destination location role on a path definition (Drop /
WorkCenter / Shipping)."

The forces this ADR has to reconcile:

1. **A process path already has a real, if implicit, notion of "where its
   output goes."** A Pack path's completed work is typically staged at a
   Drop location before consolidation; a QC path's completed work often
   goes back to a WorkCenter for rework or forward to Shipping. Today
   nothing in this service — or anywhere in the fleet — captures that as
   data. It lives only in an operator's head or in warehouse-infra's
   static seed config.
2. **This service has a hard, ADR-0001-established boundary**: zero
   inbound dependency, and it never calls into any other service,
   synchronously or otherwise (see `AGENTS.md`'s "What this service
   deliberately does not own"). Any design that requires a live call to
   facility-layout to validate a `LocationRole` value would violate that
   boundary outright — this is not a matter of style, it is this
   service's entire strategic position as an Open Host Service with zero
   consumer dependencies of its own.
3. **A process path's `Direct` field already establishes the precedent**
   for "a structural, Define-time-only fact about routing shape,
   immutable, never operator-tunable via `Revise`." A destination role is
   the same *kind* of fact — not a rate, not a capability set, not
   something that changes week to week — so it should follow the same
   posture, not invent a new one.
4. **Most process paths have no single destination role at all.** Pick
   paths feed directly into whatever downstream path claims the picked
   unit next; only a handful of path types (Pack, QC, an outbound Rebin)
   have one coherent, nameable destination. A design that forces every
   path to declare a role would be actively wrong for the common case.

## Decision

**We will add an OPTIONAL, immutable `DestinationLocationRole` field to
the `ProcessPath` aggregate, validated locally against a closed value set
mirrored from facility-layout's own `LocationRole` enum (Drop, WorkCenter,
Shipping only — the subset that makes sense as a process path's output
destination), and published on `ProcessPathCreated`/`ProcessPathUpdated`
— but this service NEVER calls facility-layout to validate it.**

1. **New value type**: `shared.DestinationLocationRole`, a closed string
   enum (`Drop`, `WorkCenter`, `Shipping`, plus the zero value
   `DestinationLocationRoleUnset` for "not declared"). `shared.ParseDestinationLocationRole`
   validates a string against this set; an empty string is always valid.
   This is a DELIBERATELY NARROWER set than facility-layout's own
   `LocationRole` enum (which also has Storage, Dock, Yard, Staging, QC,
   Consolidation) — those roles describe REAL PLACES a process path is
   simply never routed toward as a terminal destination in this service's
   model, and admitting them here would let an operator declare a
   nonsensical fact ("this path's output goes to a Storage-role
   location") with no way for this service to ever catch it, since it
   never checks facility-layout live.
2. **Set once, at `Define` time; never revisable, same posture as
   `Direct`.** `ProcessPath.Revise` does not accept it and cannot change
   it — mirrors `Direct`'s own established immutability rationale
   (structural routing-shape fact, not a day-to-day operator parameter).
3. **Declarative routing intent, not a validated fact.** This service
   publishes what an operator DECLARED; it takes no position on whether
   that declaration is currently true in facility-layout's live map. A
   downstream consumer (fulfillment-execution routing a completed task, a
   future console rendering "where does this path's output go") is free
   to cross-check it against a live facility-layout read if it needs to
   — that is each consumer's own concern, the same "read model, not a
   command" posture ADR-0002 (workforce-management) established for a
   different cross-context signal.
4. **Wire contract, both REST and Kafka, omits the field entirely (never
   empty-strings it) when unset** — matching every other optional-field
   convention already established in this fleet (e.g.
   wes-work-planning's `travelDistanceM`, ADR-0017 there).
5. **Additive-only persistence**: a new nullable `destination_location_role`
   TEXT column (with a `CHECK` constraint mirroring the enum) on the
   existing `process_paths` table, migration `0004`. NULL means "not
   declared" — mapped explicitly, not conflated with an empty string, so
   a future consumer reading the column directly sees the same
   present/absent distinction the API and events already enforce.

## Consequences

### Easier

- **A process path can now say where its output goes**, closing a real
  data gap and setting up the next B3 follow-on that depends on it:
  fulfillment-execution (or a future console) routing/rendering
  completed work by declared destination.
- **No new integration surface, no new failure mode.** This service's
  zero-dependency boundary (ADR-0001) is fully preserved — there is no
  new outbound port, no new `*_MODE=http|permissive` toggle, nothing that
  can degrade or fail open/closed, because there is nothing to call.
- **Symmetric with the existing `Direct` field's design** — a developer
  who understands why `Direct` is immutable and Define-time-only
  immediately understands `DestinationLocationRole`'s identical posture.

### Harder

- **The declared value can silently drift out of sync with reality.**
  If facility-layout's zone map is re-organized and the Drop location a
  Pack path pointed to is reclassified, nothing in this service notices
  or flags it — this is a deliberate trade for preserving the
  zero-dependency boundary, but it means `DestinationLocationRole` is
  advisory metadata, not a live guarantee, and every consumer must treat
  it that way.
- **The recognized value set (Drop/WorkCenter/Shipping) is a local,
  hand-maintained copy of a subset of facility-layout's own
  `LocationRole` enum**, not derived from it programmatically. If
  facility-layout's ADR-0016 enum ever grows a new role that also makes
  sense as a process-path destination, someone has to notice and update
  `shared.ParseDestinationLocationRole` here by hand — there is no
  compile-time or contract-test link between the two enums.
- **A caller who typos or picks a real-but-wrong facility-layout role
  (e.g. `Storage`) gets a clean 422 at Define time**, which is the
  intended guard-rail — but it also means this service's validation is
  necessarily stricter than "any string facility-layout would also
  recognize," a real (if narrow) constraint on expressiveness in exchange
  for catching the more common mistake.
