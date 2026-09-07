-- Transactional outbox (ADR 0003). Every domain event a use case raises is
-- written here in the SAME transaction as the process_paths change that
-- caused it; a relay drains unpublished rows onto Kafka afterwards. The
-- store and the topic can therefore never diverge the way ADR 0002
-- documents they once did.
CREATE TABLE outbox_events (
    id           BIGSERIAL PRIMARY KEY,
    event_id     UUID        NOT NULL UNIQUE,
    event_type   TEXT        NOT NULL,
    aggregate_id TEXT        NOT NULL,
    payload      JSONB       NOT NULL,
    occurred_at  TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ,
    attempts     INTEGER     NOT NULL DEFAULT 0,
    last_error   TEXT
);

-- The relay only ever asks "what is still unpublished, oldest first"; a
-- partial index keeps that scan tiny no matter how much published history
-- accumulates.
CREATE INDEX idx_outbox_events_unpublished ON outbox_events (id) WHERE published_at IS NULL;
