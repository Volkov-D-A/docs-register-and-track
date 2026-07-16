DROP INDEX IF EXISTS idx_event_outbox_failed;

ALTER TABLE event_outbox
    DROP COLUMN IF EXISTS failed_at;
