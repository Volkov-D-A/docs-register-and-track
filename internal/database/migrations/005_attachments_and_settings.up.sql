-- 12. Attachments
CREATE TABLE IF NOT EXISTS attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    document_id UUID NOT NULL, -- Generic link to incoming or outgoing document
    document_type VARCHAR(50) NOT NULL, -- 'incoming' or 'outgoing'
    filename VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    content_type VARCHAR(100),
    storage_path VARCHAR(512) NOT NULL,
    uploaded_by UUID NOT NULL REFERENCES users (id),
    uploaded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_attachments_document ON attachments (document_id);

-- 13. System Settings
CREATE TABLE IF NOT EXISTS system_settings (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Default settings
INSERT INTO
    system_settings (key, value, description)
VALUES (
        'max_file_size_mb',
        '15',
        'Максимальный размер файла (МБ)'
    ),
    (
        'allowed_file_types',
        '.pdf,.doc,.docx,.odt,.xls,.xlsx,.ods,.jpg,.png',
        'Разрешенные типы файлов (через запятую)'
    ),
    (
        'organization_name',
        'Управление социальной защиты населения администрации Озерского городского округа Челябинской области',
        'Название организации-отправителя для исходящих документов'
    );

-- 15. Acknowledgments
CREATE TABLE acknowledgments (
    id UUID PRIMARY KEY,
    document_id UUID NOT NULL,
    document_type VARCHAR(50) NOT NULL, -- 'incoming' or 'outgoing'
    creator_id UUID NOT NULL REFERENCES users (id),
    content TEXT DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE acknowledgment_users (
    id UUID PRIMARY KEY,
    acknowledgment_id UUID NOT NULL REFERENCES acknowledgments (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    viewed_at TIMESTAMP WITH TIME ZONE,
    confirmed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (acknowledgment_id, user_id)
);

-- 16. Release Notes
CREATE TABLE IF NOT EXISTS release_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version VARCHAR(50) NOT NULL UNIQUE,
    released_at TIMESTAMP WITH TIME ZONE NOT NULL,
    is_current BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_release_notes_current_true
    ON release_notes (is_current)
    WHERE is_current = TRUE;

CREATE TABLE IF NOT EXISTS release_note_changes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    release_note_id UUID NOT NULL REFERENCES release_notes(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_release_note_changes_release_note_id
    ON release_note_changes (release_note_id, sort_order);

CREATE TABLE IF NOT EXISTS user_release_views (
    release_note_id UUID NOT NULL REFERENCES release_notes(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    viewed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (release_note_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_user_release_views_user_id
    ON user_release_views (user_id, viewed_at DESC);
