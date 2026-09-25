-- Historical field values cannot be recovered. Existing rows receive a
-- sentinel entity ID so the previous NOT NULL schema can be restored.
ALTER TABLE user_events
    ADD COLUMN actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN entity_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    ADD COLUMN metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE user_events ALTER COLUMN entity_id DROP DEFAULT;

CREATE INDEX idx_user_events_entity ON user_events (entity_type, entity_id);
