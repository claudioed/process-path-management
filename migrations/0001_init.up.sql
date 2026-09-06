CREATE TABLE process_paths (
    id                    TEXT PRIMARY KEY,
    match_prefix          TEXT NOT NULL,
    direct                BOOLEAN NOT NULL,
    required_capabilities TEXT[] NOT NULL,
    status                TEXT NOT NULL CHECK (status IN ('ACTIVE', 'DEACTIVATED')),
    created_at            TIMESTAMPTZ NOT NULL,
    updated_at            TIMESTAMPTZ NOT NULL
);

-- The operator SPA's default view and every consumer's read model query
-- active paths far more often than the full audit list, so this partial
-- index keeps that hot path small regardless of how many deactivated
-- paths accumulate over time.
CREATE INDEX idx_process_paths_active ON process_paths (id) WHERE status = 'ACTIVE';
