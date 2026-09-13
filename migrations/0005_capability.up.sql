-- ADR 0010: fulfillment capability contract.
--
-- 1. process_paths gains cycle_time_p95 (interval, NOT NULL) and
--    eligibility (jsonb, NOT NULL, default '{}' -- the permissive
--    zero-value Eligibility{} ADR 0010 requires to be valid). Existing
--    rows are backfilled with a 2-hour default cycle time -- a
--    reasonable stand-in for the retired YAML catalogue's implied
--    PROMISE_PATH_LEAD_TIMES lead times (no single value was universal
--    across all path families there, so this migration picks the same
--    conservative default this service's own DefinePath validation
--    would accept, and operators are expected to revise it per path via
--    PUT /process-paths/{pathId} once this lands).
-- 2. A new cpt_schedules aggregate (ADR 0010, section 2): one row per
--    site, holding the schedule's timezone; its cutoffs live in a child
--    table (cpt_schedule_cutoffs) since a schedule has a variable number
--    of cutoffs, each with its own eligiblePathIds set. eligible_path_ids
--    is a text[] rather than a further child table since it is always
--    read/written as one atomic list per cutoff and never queried
--    independently -- the same modelling choice this table's sibling
--    process_paths.required_capabilities already made.

ALTER TABLE process_paths
    ADD COLUMN cycle_time_p95 INTERVAL NOT NULL DEFAULT '2 hours',
    ADD COLUMN eligibility JSONB NOT NULL DEFAULT '{}';

-- Drop the DEFAULT once existing rows are backfilled: cycle_time_p95 is
-- a required field on every new Define/Revise call going forward, so a
-- future INSERT with no explicit value should fail loudly (NOT NULL,
-- no default) rather than silently take the migration-era default.
ALTER TABLE process_paths ALTER COLUMN cycle_time_p95 DROP DEFAULT;
ALTER TABLE process_paths ALTER COLUMN eligibility DROP DEFAULT;

CREATE TABLE cpt_schedules (
    site_id    TEXT PRIMARY KEY,
    timezone   TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE cpt_schedule_cutoffs (
    schedule_site_id  TEXT NOT NULL REFERENCES cpt_schedules (site_id) ON DELETE CASCADE,
    cpt_id            TEXT NOT NULL,
    local_time        TEXT NOT NULL,
    days_of_week      TEXT[] NOT NULL,
    ship_method       TEXT NOT NULL,
    eligible_path_ids TEXT[] NOT NULL,
    PRIMARY KEY (schedule_site_id, cpt_id)
);
