ALTER TABLE outbox_events DROP CONSTRAINT IF EXISTS outbox_events_event_id_topic_key;
ALTER TABLE outbox_events ADD CONSTRAINT outbox_events_event_id_key UNIQUE (event_id);
ALTER TABLE outbox_events DROP COLUMN IF EXISTS topic;
