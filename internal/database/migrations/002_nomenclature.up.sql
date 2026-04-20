-- 4. Nomenclature
CREATE TABLE nomenclature (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    name VARCHAR(500) NOT NULL,
    index VARCHAR(50) NOT NULL,
    year INT NOT NULL,
    kind_code VARCHAR(40) NOT NULL CHECK (
        kind_code IN ('incoming_letter', 'outgoing_letter')
    ),
    separator VARCHAR(10) NOT NULL DEFAULT '/',
    numbering_mode VARCHAR(30) NOT NULL DEFAULT 'index_and_number' CHECK (
        numbering_mode IN ('index_and_number', 'number_only', 'manual_only')
    ),
    next_number INT NOT NULL DEFAULT 1,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (index, year, kind_code)
);

CREATE INDEX idx_nomenclature_year ON nomenclature (year);

CREATE INDEX idx_nomenclature_kind_code ON nomenclature (kind_code);

CREATE TABLE document_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    kind_code VARCHAR(40) NOT NULL CHECK (
        kind_code IN ('incoming_letter', 'outgoing_letter')
    ),
    subject_type VARCHAR(20) NOT NULL CHECK (
        subject_type IN ('role', 'department', 'user')
    ),
    subject_key VARCHAR(100) NOT NULL,
    action VARCHAR(30) NOT NULL CHECK (
        action IN (
            'create',
            'read',
            'update',
            'delete',
            'assign',
            'acknowledge',
            'upload',
            'link',
            'view_journal'
        )
    ),
    is_allowed BOOLEAN NOT NULL DEFAULT true,
    UNIQUE (kind_code, subject_type, subject_key, action)
);

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

-- 8. Resolution Executors (справочник исполнителей резолюции)
CREATE TABLE resolution_executors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(500) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
