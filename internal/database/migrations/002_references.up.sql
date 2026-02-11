-- Номенклатура дел
CREATE TABLE nomenclature (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(500) NOT NULL,
    index VARCHAR(50) NOT NULL,
    year INT NOT NULL,
    direction VARCHAR(20) NOT NULL CHECK (direction IN ('incoming', 'outgoing')),
    next_number INT NOT NULL DEFAULT 1,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(index, year, direction)
);

-- Организации (автозаполняемый справочник)
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(500) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Типы документов
CREATE TABLE document_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Начальные типы документов
INSERT INTO document_types (name) VALUES 
    ('Письмо'),
    ('Договор'),
    ('Акт'),
    ('Счёт'),
    ('Запрос'),
    ('Ответ'),
    ('Уведомление');

-- Индексы
CREATE INDEX idx_nomenclature_year ON nomenclature(year);
CREATE INDEX idx_nomenclature_direction ON nomenclature(direction);
