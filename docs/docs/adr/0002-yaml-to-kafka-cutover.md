---
id: 0002-yaml-to-kafka-cutover
slug: /adr/0002-yaml-to-kafka-cutover
title: 0002. Cutting the fleet over from the static YAML catalogue to this service's events
sidebar_label: 0002. The YAML cutover
description: ADR 0002 — the executed cutover of fulfillment-execution, wes-work-planning and workforce-management from warehouse-infra's static process-paths YAML to this service's Kafka topic, its rollback, and the store/topic divergence trap it exposed.
---

# 0002. Cutting the fleet over from the static YAML catalogue to this service's events

## Status

Accepted, and **executed** — the three consumers run from this service's
Kafka topic in the `warehouse` kind cluster as of 2026-09-06.

ADR 0001 decided this service should exist and that propagation would be
event-driven. This ADR records how the switchover was actually performed,
what it cost, and how to reverse it — the operational half ADR 0001
deliberately left open.

## Context

`warehouse-infra/config/process-paths/sortable-fc.yaml` was a **Published
Language in file form**: fulfillment-execution, wes-work-planning and
workforce-management each read that same file once at boot, and each
treated a missing or malformed file as a fatal startup error. That gave a
strong guarantee — no service ever served traffic against an incomplete
catalogue — and one severe limitation: **changing a path required
redeploying three services.**

The three consumers already shipped a `PATH_CATALOGUE_SOURCE=file|kafka`
switch (defaulting to `file`) and warehouse-infra already had a
`deploy_process_path_kafka_source` Terraform flag wiring both sides
together. What did not exist was the migration of the file's *content*
into this service, so flipping the flag would have pointed three consumers
at a catalogue that did not contain the paths the building actually runs.

## Decision

Cut over in a fixed order, with the catalogue content migrated first:

1. **Seed this service from the YAML** using
   `warehouse-infra/scripts/seed-process-paths.py` — an idempotent
   reconciler that creates a missing path, revises a diverging one, leaves
   an identical one untouched (so no spurious `ProcessPathUpdated` is
   published), and refuses to silently resurrect a deactivated one.
2. **Flip `deploy_process_path_kafka_source = true`**, which switches both
   sides at once: this service's publisher from the chart's `log` default
   to `kafka`, and each consumer's `pathCatalogue` ConfigMap mount off with
   `PATH_CATALOGUE_SOURCE=kafka` injected.
3. **Roll the consumer Deployments.** Required, not optional: every chart
   pins `image.tag: "local"` with `pullPolicy: IfNotPresent`, so a rebuilt
   image alone never rolls a running pod.

The YAML file is **kept, frozen, and marked SUPERSEDED** rather than
deleted. It is the rollback target, so its content must stay a truthful
description of the building.

## The trap this exposed: the store and the topic can diverge

Every write use case here does `Repo.Save` **then** `Publisher.Publish`,
with no outbox and no compensation. A path defined while this service ran
with `EVENT_PUBLISHER=log` therefore lands in Postgres and is **never
published to Kafka**. Consumers only ever see the topic.

This was not theoretical. On the live cluster the store held only
`STAGE`/`WRAP` while the topic already carried `PICK`/`PACK`/`REBIN`/`SLAM`
from earlier manual testing — the two had diverged **in both directions**
at once. `GET /process-paths` looked healthy and told you nothing about
what any consumer would actually see.

Consequences for anyone operating this service:

- **Never verify a cutover from the REST listing alone.** Check the topic's
  offsets too.
- The seed script's `--republish` flag exists for the store-ahead-of-topic
  case; the default path never republishes.
- Closing this properly needs an outbox (or a transactional
  save-and-publish). That is a **fleet-wide** gap that predates this
  service, is tracked separately, and was deliberately not patched as a
  side effect of this cutover.

## Consequences

### Positive

- A process path is now defined, revised and deactivated through this
  service's API, and the change reaches all three running consumers within
  seconds — **no restart, no redeploy, no file edit.** Verified live
  against pods with 7h40m uptime and 0 restarts: a newly-defined path was
  accepted by wes-work-planning's validation, an unknown id still returned
  400, and after deactivation the same id returned 400 again.
- The `matchPrefix` family rule (`id == prefix` OR `id` starts with
  `prefix + "-"`) survives the move unchanged, so real fleet-shaped ids like
  `pick-zone-a` still resolve to `PICK`.
- Three services no longer need a coordinated redeploy to change one path.

### Negative / accepted

- The boot-time guarantee is now a **readiness** guarantee rather than a
  process-start one: each consumer replays the topic and blocks readiness
  until caught up. A consumer that cannot reach Kafka does not start
  serving, which is the intended equivalent of the old fatal file read.
- An empty topic makes a consumer come up ready with an **empty
  catalogue** — every path id is then unknown. That is why step 1 (seed
  before flip) is ordered first and is not optional.
- This service is now a hard runtime dependency for catalogue *changes*
  (though not for a consumer's steady-state operation, since each holds a
  full local cache).

### Rollback

Pure configuration, no data migration:

1. `deploy_process_path_kafka_source = false`, re-apply.
2. `kubectl rollout restart` the three consumers.

Each one's ConfigMap volume returns and `PATH_CATALOGUE_SOURCE` reverts to
`file`, reading `sortable-fc.yaml` exactly as before. Nothing the seed
script did needs undoing — which is precisely why the YAML is frozen
rather than removed.

## Related

- ADR 0001 — why this bounded context exists and why propagation is
  event-driven rather than a synchronous call.
- `warehouse-infra/scripts/README-process-path-seed.md` — the seed script,
  the store-vs-topic divergence check, and the live verification recipe.
- `warehouse-infra/terraform/variables.tf` —
  `deploy_process_path_kafka_source`'s full rollout-order rationale.
