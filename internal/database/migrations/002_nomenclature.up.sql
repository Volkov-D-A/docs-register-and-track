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