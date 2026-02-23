-- 12. Attachments
CREATE TABLE IF NOT EXISTS attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    document_id UUID NOT NULL, -- Generic link to incoming or outgoing document
    document_type VARCHAR(50) NOT NULL, -- 'incoming' or 'outgoing'
    filename VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    content_type VARCHAR(100),
    content BYTEA,
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
        'НАША ОРГАНИЗАЦИЯ',
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