-- 1. Departments
CREATE TABLE IF NOT EXISTS departments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. Users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    login VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    department_id UUID REFERENCES departments (id) ON DELETE SET NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_login ON users (login);

-- 3. User Roles
CREATE TABLE user_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL CHECK (
        role IN ('admin', 'clerk', 'executor')
    ),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, role)
);

CREATE INDEX idx_user_roles_user_id ON user_roles (user_id);

-- 4. Nomenclature
CREATE TABLE nomenclature (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    name VARCHAR(500) NOT NULL,
    index VARCHAR(50) NOT NULL,
    year INT NOT NULL,
    direction VARCHAR(20) NOT NULL CHECK (
        direction IN ('incoming', 'outgoing')
    ),
    next_number INT NOT NULL DEFAULT 1,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (index, year, direction)
);

CREATE INDEX idx_nomenclature_year ON nomenclature (year);

CREATE INDEX idx_nomenclature_direction ON nomenclature (direction);

-- 5. Department Nomenclature
CREATE TABLE IF NOT EXISTS department_nomenclature (
    department_id UUID NOT NULL REFERENCES departments (id) ON DELETE CASCADE,
    nomenclature_id UUID NOT NULL REFERENCES nomenclature (id) ON DELETE CASCADE,
    PRIMARY KEY (
        department_id,
        nomenclature_id
    )
);

-- 6. Organizations
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    name VARCHAR(500) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 7. Document Types
CREATE TABLE document_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    name VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO
    document_types (name)
VALUES ('Письмо'),
    ('Договор'),
    ('Акт'),
    ('Счёт'),
    ('Запрос'),
    ('Ответ'),
    ('Уведомление');

-- 8. Incoming Documents
CREATE TABLE incoming_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nomenclature_id UUID NOT NULL REFERENCES nomenclature(id),

-- Numbers and dates
incoming_number VARCHAR(50) NOT NULL,
incoming_date DATE NOT NULL,
outgoing_number_sender VARCHAR(100) NOT NULL,
outgoing_date_sender DATE NOT NULL,
intermediate_number VARCHAR(100),
intermediate_date DATE,

-- About document
document_type_id UUID NOT NULL REFERENCES document_types (id),
subject VARCHAR(1000) NOT NULL,
pages_count INT NOT NULL DEFAULT 1,
content TEXT NOT NULL,

-- Sender
sender_org_id UUID NOT NULL REFERENCES organizations (id),
sender_signatory VARCHAR(255) NOT NULL,
sender_executor VARCHAR(255) NOT NULL,

-- Recipient
recipient_org_id UUID NOT NULL REFERENCES organizations (id),
addressee VARCHAR(255) NOT NULL,

-- Resolution
resolution TEXT,

-- Metadata
created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_incoming_docs_nomenclature ON incoming_documents (nomenclature_id);

CREATE INDEX idx_incoming_docs_date ON incoming_documents (incoming_date);

CREATE INDEX idx_incoming_docs_sender ON incoming_documents (sender_org_id);

-- 9. Outgoing Documents
CREATE TABLE outgoing_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nomenclature_id UUID NOT NULL REFERENCES nomenclature(id),

-- Numbers and dates
outgoing_number VARCHAR(50) NOT NULL, outgoing_date DATE NOT NULL,

-- About document
document_type_id UUID NOT NULL REFERENCES document_types (id),
subject VARCHAR(1000) NOT NULL,
pages_count INT NOT NULL DEFAULT 1,
content TEXT NOT NULL,

-- Sender
sender_org_id UUID NOT NULL REFERENCES organizations (id),
sender_signatory VARCHAR(255) NOT NULL,
sender_executor VARCHAR(255) NOT NULL,

-- Recipient
recipient_org_id UUID NOT NULL REFERENCES organizations (id),
addressee VARCHAR(255) NOT NULL,

-- Metadata
created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_outgoing_docs_nomenclature ON outgoing_documents (nomenclature_id);

CREATE INDEX idx_outgoing_docs_date ON outgoing_documents (outgoing_date);

-- 10. Document Links
CREATE TABLE document_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    source_type VARCHAR(20) NOT NULL CHECK (
        source_type IN ('incoming', 'outgoing')
    ),
    source_id UUID NOT NULL,
    target_type VARCHAR(20) NOT NULL CHECK (
        target_type IN ('incoming', 'outgoing')
    ),
    target_id UUID NOT NULL,
    link_type VARCHAR(50) NOT NULL CHECK (
        link_type IN (
            'reply',
            'follow_up',
            'related',
            'clarification'
        )
    ),
    created_by UUID NOT NULL REFERENCES users (id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (
        source_type,
        source_id,
        target_type,
        target_id
    )
);

CREATE INDEX idx_document_links_source ON document_links (source_type, source_id);

CREATE INDEX idx_document_links_target ON document_links (target_type, target_id);

-- 11. Assignments
CREATE TABLE assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    document_id UUID NOT NULL,
    document_type VARCHAR(50) NOT NULL, -- 'incoming' or 'outgoing'
    executor_id UUID NOT NULL REFERENCES users (id),
    content TEXT NOT NULL,
    deadline DATE,
    status VARCHAR(50) NOT NULL DEFAULT 'new', -- new, in_progress, completed, cancelled
    report TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_assignments_executor ON assignments (executor_id);

CREATE INDEX idx_assignments_document ON assignments (document_id);

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

-- 14. Assignment Co-Executors
CREATE TABLE assignment_co_executors (
    assignment_id UUID NOT NULL REFERENCES assignments (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (assignment_id, user_id)
);

CREATE INDEX idx_assignment_co_executors_assignment ON assignment_co_executors (assignment_id);

CREATE INDEX idx_assignment_co_executors_user ON assignment_co_executors (user_id);

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