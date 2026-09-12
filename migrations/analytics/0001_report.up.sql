-- process-path-management "Process Path Catalogue Growth & Change" analytics
-- read model (ADR 0007, mirroring facility-layout's ADR-0010).
--
-- This is the ANALYTICAL database, separate from the OLTP database. It is
-- written only by cmd/pathmgmt-projector and read (read-only) by
-- cmd/pathmgmt-reports. The tables here are projections derived from the
-- analytics event stream, not sources of truth.
--
-- Unlike facility-layout's site/zone-scoped rollup, this service's
-- catalogue has no spatial dimension (a process path is a single flat
-- identity, PathId), so the rollup is bucketed by DAY alone.

-- Idempotency + freshness: every applied analytics event id is recorded
-- here exactly once. applied_at is wall-clock insert time; occurred_at is
-- the event's business time, used to compute the projection's freshness
-- lag.
CREATE TABLE analytics_processed_events (
    event_id    TEXT PRIMARY KEY,
    occurred_at TIMESTAMPTZ NOT NULL,
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_analytics_processed_events_occurred_at
    ON analytics_processed_events (occurred_at DESC);

-- Consumer-level dedupe set, used by the inbound consumer's idempotency
-- gate. Kept SEPARATE from analytics_processed_events (which the
-- projection UPSERT claims) so the two idempotency layers do not race to
-- claim the same event_id: the consumer gate admits the event, the
-- projection then records its effect.
CREATE TABLE analytics_consumed_events (
    event_id     TEXT PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- The catalogue-growth rollup fact table: one row per day_bucket.
-- Counters are UPSERTed as events arrive. Because the process-path
-- catalogue is a slow-changing, operator-configured reference catalogue,
-- rows are bucketed by DAY (midnight UTC), not by hour.
CREATE TABLE catalogue_growth_rollup (
    day_bucket        TIMESTAMPTZ PRIMARY KEY,
    paths_defined     BIGINT NOT NULL DEFAULT 0,
    paths_revised     BIGINT NOT NULL DEFAULT 0,
    paths_deactivated BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_catalogue_growth_rollup_day_bucket
    ON catalogue_growth_rollup (day_bucket);
