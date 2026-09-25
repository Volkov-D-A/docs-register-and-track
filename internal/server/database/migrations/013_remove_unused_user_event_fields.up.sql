DROP INDEX idx_user_events_entity;

ALTER TABLE user_events
    DROP COLUMN actor_user_id,
    DROP COLUMN entity_id,
    DROP COLUMN metadata;
