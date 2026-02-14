-- 12. Attachments
CREATE TABLE IF NOT EXISTS attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    document_id UUID NOT NULL, -- Generic link to incoming or outgoing document
    document_type VARCHAR(50) NOT NULL, -- 'incoming' or 'outgoing'
    filename VARCHAR(255) NOT NULL,
    filepath VARCHAR(1024) NOT NULL, -- Path relative to app root or absolute path
    file_size BIGINT NOT NULL,
    content_type VARCHAR(100),
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
        '10',
        'Максимальный размер файла (МБ)'
    ),
    (
        'allowed_file_types',
        '.pdf,.doc,.docx,.xls,.xlsx,.jpg,.png,.zip,.rar',
        'Разрешенные типы файлов (через запятую)'
    );