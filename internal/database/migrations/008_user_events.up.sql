CREATE TABLE IF NOT EXISTS user_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipient_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    document_kind VARCHAR(100) NOT NULL,
    document_number VARCHAR(255),
    entity_type VARCHAR(100) NOT NULL,
    entity_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    read_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_user_events_recipient_created
    ON user_events (recipient_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_events_recipient_unread
    ON user_events (recipient_user_id, created_at DESC)
    WHERE read_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_events_entity
    ON user_events (entity_type, entity_id);

CREATE INDEX IF NOT EXISTS idx_user_events_document_recipient_unread
    ON user_events (document_id, recipient_user_id)
    WHERE read_at IS NULL;
