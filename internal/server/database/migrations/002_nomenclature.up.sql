-- 4. Nomenclature
CREATE TABLE nomenclature (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    name VARCHAR(500) NOT NULL,
    index VARCHAR(50) NOT NULL,
    year INT NOT NULL,
    kind_code VARCHAR(40) NOT NULL CHECK (
        kind_code IN ('incoming_letter', 'outgoing_letter', 'citizen_appeal', 'administrative_order')
    ),
    separator VARCHAR(10) NOT NULL DEFAULT '/',
    numbering_mode VARCHAR(30) NOT NULL DEFAULT 'index_and_number' CHECK (
        numbering_mode IN ('index_and_number', 'number_only', 'manual_only')
    ),
    next_number INT NOT NULL DEFAULT 1,
    is_active BOOLEAN NOT NULL DEFAULT true,
    UNIQUE (index, year, kind_code)
);

CREATE INDEX idx_nomenclature_year ON nomenclature (year);

CREATE INDEX idx_nomenclature_kind_code ON nomenclature (kind_code);

CREATE TABLE document_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    kind_code VARCHAR(40) NOT NULL CHECK (
        kind_code IN ('incoming_letter', 'outgoing_letter', 'citizen_appeal', 'administrative_order')
    ),
    subject_type VARCHAR(20) NOT NULL CHECK (
        subject_type IN ('role', 'department', 'user')
    ),
    subject_key VARCHAR(100) NOT NULL,
    action VARCHAR(30) NOT NULL CHECK (
        action IN (
            'register',
            'control_deadlines',
            'create',
            'read',
            'update',
            'delete',
            'assign',
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
    name VARCHAR(500) NOT NULL UNIQUE
);

-- 8. Resolution Executors (справочник исполнителей резолюции)
CREATE TABLE resolution_executors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(500) NOT NULL UNIQUE
);

-- Executors of deadline controls. Linked users are represented without copying names.
CREATE TABLE deadline_executors (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 user_id UUID UNIQUE REFERENCES users(id) ON DELETE RESTRICT,
 name VARCHAR(500),
 institution VARCHAR(500) NOT NULL DEFAULT '',
 position VARCHAR(500) NOT NULL DEFAULT '',
 is_active BOOLEAN NOT NULL DEFAULT true,
 version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
 CHECK ((user_id IS NOT NULL AND name IS NULL AND institution='' AND position='')
     OR (user_id IS NULL AND name IS NOT NULL AND length(btrim(name))>0))
);
CREATE FUNCTION create_user_deadline_executor() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
 INSERT INTO deadline_executors(id,user_id) VALUES(NEW.id,NEW.id);
 RETURN NEW;
END;
$$;
CREATE TRIGGER user_deadline_executor AFTER INSERT ON users
 FOR EACH ROW EXECUTE FUNCTION create_user_deadline_executor();
