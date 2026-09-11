-- Extends the transactional outbox (ADR 0003) to carry a topic per row, so
-- one event can be enqueued onto more than one topic in the same
-- transaction (ADR 0007: the new analytics topic alongside the existing
-- integration topic). Existing rows are backfilled onto the integration
-- topic they were always meant for.
ALTER TABLE outbox_events ADD COLUMN topic TEXT;
UPDATE outbox_events SET topic = 'warehouse.process-path-management.events' WHERE topic IS NULL;
ALTER TABLE outbox_events ALTER COLUMN topic SET NOT NULL;

-- event_id alone is no longer unique: the same event_id is now
-- deliberately reused across every topic-encoder pair for one event
-- (see postgres.OutboxPublisher.Publish), so uniqueness moves to the
-- (event_id, topic) pair.
ALTER TABLE outbox_events DROP CONSTRAINT outbox_events_event_id_key;
ALTER TABLE outbox_events ADD CONSTRAINT outbox_events_event_id_topic_key UNIQUE (event_id, topic);
