-- Docflow deadline-control baseline. Requires an empty database; no legacy data conversion.

-- 1. Departments
CREATE TABLE IF NOT EXISTS departments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    name TEXT NOT NULL UNIQUE
);

-- 2. Users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    login VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    last_name VARCHAR(100) NOT NULL CHECK (length(btrim(last_name)) > 0),
    first_name VARCHAR(100) NOT NULL CHECK (length(btrim(first_name)) > 0),
    patronymic VARCHAR(100) NOT NULL DEFAULT '',
    no_patronymic BOOLEAN NOT NULL DEFAULT false,
    full_name TEXT GENERATED ALWAYS AS (
        last_name || ' ' || first_name || CASE WHEN no_patronymic THEN '' ELSE ' ' || patronymic END
    ) STORED,
    CONSTRAINT users_patronymic_consistent CHECK (
        (no_patronymic AND patronymic = '') OR
        (NOT no_patronymic AND length(btrim(patronymic)) > 0)
    ),
    department_id UUID REFERENCES departments (id) ON DELETE SET NULL,
    is_document_participant BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    failed_login_attempts INTEGER NOT NULL DEFAULT 0,
    password_changed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    password_change_required BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_system_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    permission VARCHAR(50) NOT NULL CHECK (
        permission IN ('admin', 'references')
    ),
    is_allowed BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, permission)
);

CREATE INDEX idx_user_system_permissions_user_id ON user_system_permissions (user_id);

-- Система должна сохранять хотя бы одного активного администратора.
-- Constraint trigger отложен до COMMIT: ReplaceUserAccessProfile временно удаляет
-- текущие права перед вставкой нового профиля в рамках одной транзакции.
CREATE OR REPLACE FUNCTION ensure_active_administrator_exists()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
	-- Сериализуем конкурирующие изменения пользователей и системных прав.
	-- Без этой блокировки две транзакции могли бы одновременно деактивировать
	-- двух последних администраторов, каждая видя вторую до её commit.
	PERFORM pg_advisory_xact_lock(78652401);

    IF NOT EXISTS (
        SELECT 1
        FROM users u
        JOIN user_system_permissions usp ON usp.user_id = u.id
        WHERE u.is_active = true
          AND usp.permission = 'admin'
          AND usp.is_allowed = true
    ) THEN
        RAISE EXCEPTION 'at least one active administrator must remain'
            USING ERRCODE = 'P0001';
    END IF;

    RETURN NULL;
END;
$$;

CREATE CONSTRAINT TRIGGER active_administrator_required_after_user_update
AFTER UPDATE OF is_active ON users
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION ensure_active_administrator_exists();

CREATE CONSTRAINT TRIGGER active_administrator_required_after_permission_change
AFTER INSERT OR UPDATE OR DELETE ON user_system_permissions
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION ensure_active_administrator_exists();


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
        action IN ('register', 'control_deadlines')
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


-- 8. Common Documents
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind VARCHAR(40) NOT NULL CHECK (kind IN ('incoming_letter', 'outgoing_letter', 'citizen_appeal', 'administrative_order')),
    nomenclature_id UUID NOT NULL REFERENCES nomenclature(id),
    registration_number VARCHAR(100) NOT NULL,
    registration_date DATE NOT NULL,
    document_type VARCHAR(100) NOT NULL CHECK (
        document_type IN (
            'Письмо',
            'Договор',
            'Акт',
            'Счёт',
            'Запрос',
            'Ответ',
            'Уведомление',
            'Обращение',
            'Приказ'
        )
    ),
    content TEXT NOT NULL,
    pages_count INT NOT NULL DEFAULT 1,
    idempotency_key UUID NOT NULL DEFAULT gen_random_uuid(),
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    attachment_pages_count INT NOT NULL DEFAULT 0 CHECK (attachment_pages_count >= 0)
);

CREATE INDEX idx_documents_kind ON documents (kind);
CREATE INDEX idx_documents_nomenclature ON documents (nomenclature_id);
CREATE INDEX idx_documents_registration_date ON documents (registration_date);
CREATE UNIQUE INDEX idx_documents_kind_registration_number_year
    ON documents (kind, registration_number, EXTRACT(YEAR FROM registration_date));
CREATE INDEX idx_documents_created_at ON documents (created_at);
CREATE INDEX idx_documents_kind_created_at ON documents (kind, created_at DESC);
-- Matches the actual document-list scope and ORDER BY created_at DESC.
CREATE INDEX idx_documents_kind_nomenclature_created_at
    ON documents (kind, nomenclature_id, created_at DESC);
CREATE UNIQUE INDEX idx_documents_created_by_kind_idempotency
    ON documents (created_by, kind, idempotency_key);

-- Stores the durable result of document commands independently from the
-- document registration key. This also covers updates and admin drafts.
CREATE TABLE document_command_idempotency (
    principal_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    operation TEXT NOT NULL,
    idempotency_key UUID NOT NULL,
    request_hash TEXT NOT NULL,
    document_id UUID REFERENCES documents(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (principal_id, operation, idempotency_key)
);

CREATE INDEX idx_document_command_idempotency_created_at
    ON document_command_idempotency(created_at);

-- 9. Document Correspondent Registrations
CREATE TABLE document_correspondent_registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    registration_number VARCHAR(100) NOT NULL,
    registration_date DATE NOT NULL,
    correspondent_org_id UUID NOT NULL REFERENCES organizations(id),
    position INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_doc_corr_reg_document ON document_correspondent_registrations (document_id);
CREATE INDEX idx_doc_corr_reg_number ON document_correspondent_registrations (registration_number);
CREATE INDEX idx_doc_corr_reg_date ON document_correspondent_registrations (registration_date);
CREATE INDEX idx_doc_corr_reg_org ON document_correspondent_registrations (correspondent_org_id);

-- 10. Document Resolutions
CREATE TABLE document_resolutions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    resolution TEXT,
    resolution_author VARCHAR(255),
    resolution_executors TEXT,
    position INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_document_resolutions_document ON document_resolutions (document_id);
CREATE INDEX idx_document_resolutions_document_position ON document_resolutions (document_id, position);

-- 11. Incoming Document Details
CREATE TABLE incoming_document_details (
    document_id UUID PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE,
    incoming_number VARCHAR(50) NOT NULL,
    incoming_date DATE NOT NULL,
    sender_signatory VARCHAR(255) NOT NULL
);

CREATE INDEX idx_incoming_doc_details_number ON incoming_document_details (incoming_number);
CREATE INDEX idx_incoming_doc_details_date ON incoming_document_details (incoming_date);

-- 12. Outgoing Document Details
CREATE TABLE outgoing_document_details (
    document_id UUID PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE,
    outgoing_number VARCHAR(50) NOT NULL,
    outgoing_date DATE NOT NULL,
    sender_signatory VARCHAR(255) NOT NULL,
    sender_executor VARCHAR(255) NOT NULL,
    recipient_org_id UUID NOT NULL REFERENCES organizations(id),
    addressee VARCHAR(255) NOT NULL
);

CREATE INDEX idx_outgoing_doc_details_number ON outgoing_document_details (outgoing_number);
CREATE INDEX idx_outgoing_doc_details_date ON outgoing_document_details (outgoing_date);
CREATE INDEX idx_outgoing_doc_details_recipient ON outgoing_document_details (recipient_org_id);

-- 13. Citizen Appeal Details
CREATE TABLE citizen_appeal_details (
    document_id UUID PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE,
    appeal_date DATE NOT NULL,
    applicant_full_name VARCHAR(255) NOT NULL,
    registration_address TEXT NOT NULL,
    appeal_type VARCHAR(30) NOT NULL CHECK (
        appeal_type IN ('предложение', 'заявление', 'жалоба')
    ),
    applicant_category VARCHAR(255) NOT NULL,
    has_envelope BOOLEAN NOT NULL DEFAULT false,
    received_from_pos BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX idx_citizen_appeal_details_appeal_date ON citizen_appeal_details (appeal_date);
CREATE INDEX idx_citizen_appeal_details_applicant ON citizen_appeal_details (applicant_full_name);
CREATE INDEX idx_citizen_appeal_details_type ON citizen_appeal_details (appeal_type);

-- 14. Administrative Order Details
CREATE TABLE administrative_order_details (
    document_id UUID PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE,
    order_number VARCHAR(50) NOT NULL,
    order_date DATE NOT NULL,
    title TEXT NOT NULL,
    execution_controller VARCHAR(255) NOT NULL DEFAULT '',
    execution_deadline DATE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    cancelled_at TIMESTAMP WITH TIME ZONE,
    CHECK (
        (is_active = true AND cancelled_at IS NULL)
        OR (is_active = false AND cancelled_at IS NOT NULL)
    )
);

CREATE INDEX idx_administrative_order_details_number ON administrative_order_details (order_number);
CREATE INDEX idx_administrative_order_details_date ON administrative_order_details (order_date);
CREATE INDEX idx_administrative_order_details_deadline ON administrative_order_details (execution_deadline);
CREATE INDEX idx_administrative_order_details_active ON administrative_order_details (is_active);

CREATE TABLE administrative_order_acknowledgment_people (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    full_name VARCHAR(255) NOT NULL,
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    acknowledged_by UUID REFERENCES users(id),
    position INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_admin_order_ack_people_document ON administrative_order_acknowledgment_people (document_id);
CREATE INDEX idx_admin_order_ack_people_pending ON administrative_order_acknowledgment_people (document_id) WHERE acknowledged_at IS NULL;

-- 14. Document Links
CREATE TABLE document_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    source_document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    target_document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    link_type VARCHAR(50) NOT NULL CHECK (
        link_type IN (
            'reply',
            'follow_up',
            'related',
            'clarification',
            'order_amends',
            'order_cancels'
        )
    ),
    created_by UUID NOT NULL REFERENCES users (id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (source_document_id, target_document_id)
);

CREATE INDEX idx_document_links_source ON document_links (source_document_id);
CREATE INDEX idx_document_links_target ON document_links (target_document_id);


CREATE TABLE document_deadline_series (
 id UUID PRIMARY KEY,
 document_id UUID NOT NULL REFERENCES documents(id) ON DELETE RESTRICT,
 created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
 executor_id UUID NOT NULL REFERENCES deadline_executors(id) ON DELETE RESTRICT,
 co_executor_ids UUID[] NOT NULL DEFAULT '{}',
 subject TEXT NOT NULL CHECK (length(btrim(subject)) <= 2000),
 interval_unit TEXT NOT NULL CHECK (interval_unit IN ('day','week','month','year')),
 interval_value INTEGER NOT NULL CHECK (interval_value BETWEEN 1 AND 3650),
 day_rule TEXT NOT NULL,
 day_of_month INTEGER NOT NULL DEFAULT 0,
 first_due_date DATE NOT NULL,
 active BOOLEAN NOT NULL DEFAULT true,
 stop_reason TEXT NOT NULL DEFAULT '',
 current_control_id UUID NOT NULL,
 current_iteration INTEGER NOT NULL CHECK (current_iteration > 0),
 version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
 CHECK ((interval_unit IN ('day','week') AND day_rule='same_day' AND day_of_month=0)
    OR (interval_unit IN ('month','year') AND day_rule='fixed' AND day_of_month BETWEEN 1 AND 31)
    OR (interval_unit IN ('month','year') AND day_rule='last_day' AND day_of_month=0))
);
CREATE TABLE document_deadline_controls (
 id UUID PRIMARY KEY,
 document_id UUID NOT NULL REFERENCES documents(id) ON DELETE RESTRICT,
 created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
 executor_id UUID NOT NULL REFERENCES deadline_executors(id) ON DELETE RESTRICT,
 co_executor_ids UUID[] NOT NULL DEFAULT '{}',
 subject TEXT NOT NULL CHECK (length(btrim(subject)) <= 2000),
 due_date DATE NOT NULL,
 status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','completed','cancelled')),
 executed_on DATE,
 completed_at TIMESTAMPTZ,
 completed_by UUID REFERENCES users(id) ON DELETE RESTRICT,
 comment TEXT NOT NULL DEFAULT '',
 cancellation_reason TEXT NOT NULL DEFAULT '',
 series_id UUID REFERENCES document_deadline_series(id) ON DELETE RESTRICT,
 iteration INTEGER NOT NULL DEFAULT 0,
 planned_due_date DATE,
 version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 CHECK ((status='completed' AND executed_on IS NOT NULL AND completed_at IS NOT NULL AND completed_by IS NOT NULL AND cancellation_reason='')
     OR (status='active' AND executed_on IS NULL AND completed_at IS NULL AND completed_by IS NULL AND cancellation_reason='')
     OR (status='cancelled' AND executed_on IS NULL AND completed_at IS NULL AND completed_by IS NULL AND length(btrim(cancellation_reason))>0)),
 CHECK ((series_id IS NULL AND iteration=0 AND planned_due_date IS NULL)
     OR (series_id IS NOT NULL AND iteration>0 AND planned_due_date IS NOT NULL)),
 UNIQUE(series_id,iteration),
 UNIQUE(id,series_id,document_id,iteration)
);
ALTER TABLE document_deadline_series ADD CONSTRAINT deadline_series_current_control
 FOREIGN KEY(current_control_id,id,document_id,current_iteration)
 REFERENCES document_deadline_controls(id,series_id,document_id,iteration) DEFERRABLE INITIALLY DEFERRED;
CREATE UNIQUE INDEX deadline_series_one_active_iteration ON document_deadline_controls(series_id) WHERE status='active';
CREATE INDEX deadline_controls_document ON document_deadline_controls(document_id);
CREATE INDEX deadline_controls_author ON document_deadline_controls(created_by);
CREATE INDEX deadline_controls_executor ON document_deadline_controls(executor_id);
CREATE INDEX deadline_controls_co_executors ON document_deadline_controls USING gin(co_executor_ids);
CREATE INDEX deadline_controls_active_due ON document_deadline_controls(due_date) WHERE status='active';
CREATE INDEX deadline_series_document ON document_deadline_series(document_id);
CREATE INDEX deadline_series_author ON document_deadline_series(created_by);

CREATE INDEX deadline_series_executor ON document_deadline_series(executor_id);
CREATE INDEX deadline_series_co_executors ON document_deadline_series USING gin(co_executor_ids);
CREATE FUNCTION deadline_executor_set_valid(main UUID, co UUID[]) RETURNS BOOLEAN
LANGUAGE SQL IMMUTABLE AS $$
 SELECT NOT main=ANY(co) AND array_position(co,NULL) IS NULL
    AND cardinality(co)=(SELECT count(DISTINCT id) FROM unnest(co) AS id);
$$;
ALTER TABLE document_deadline_controls ADD CHECK(deadline_executor_set_valid(executor_id,co_executor_ids));
ALTER TABLE document_deadline_series ADD CHECK(deadline_executor_set_valid(executor_id,co_executor_ids));
ALTER TABLE document_deadline_controls ADD CHECK(executed_on IS NULL OR executed_on <= (completed_at AT TIME ZONE 'Asia/Yekaterinburg')::date);

-- Preserve referential integrity for co-executor arrays as well as the primary FK.
CREATE FUNCTION verify_deadline_co_executors() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
 PERFORM id FROM deadline_executors WHERE id=ANY(NEW.co_executor_ids) FOR KEY SHARE;
 IF EXISTS(SELECT 1 FROM unnest(NEW.co_executor_ids) AS co(executor_id) WHERE NOT EXISTS(SELECT 1 FROM deadline_executors e WHERE e.id=co.executor_id)) THEN
  RAISE EXCEPTION 'co-executor does not exist' USING ERRCODE='23503';
 END IF;
 RETURN NEW;
END;
$$;
CREATE CONSTRAINT TRIGGER deadline_control_co_executor_refs AFTER INSERT OR UPDATE OF co_executor_ids ON document_deadline_controls
 DEFERRABLE INITIALLY IMMEDIATE FOR EACH ROW EXECUTE FUNCTION verify_deadline_co_executors();
CREATE CONSTRAINT TRIGGER deadline_series_co_executor_refs AFTER INSERT OR UPDATE OF co_executor_ids ON document_deadline_series
 DEFERRABLE INITIALLY IMMEDIATE FOR EACH ROW EXECUTE FUNCTION verify_deadline_co_executors();
CREATE FUNCTION restrict_deadline_co_executor_deletion() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
 IF EXISTS(SELECT 1 FROM document_deadline_controls WHERE OLD.id=ANY(co_executor_ids))
    OR EXISTS(SELECT 1 FROM document_deadline_series WHERE OLD.id=ANY(co_executor_ids)) THEN
  RAISE EXCEPTION 'executor is referenced by deadline controls' USING ERRCODE='23503';
 END IF;
 RETURN OLD;
END;
$$;
CREATE TRIGGER deadline_co_executor_deletion BEFORE DELETE ON deadline_executors
 FOR EACH ROW EXECUTE FUNCTION restrict_deadline_co_executor_deletion();


-- 14. System Settings
CREATE TABLE IF NOT EXISTS system_settings (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT
);

-- Default settings
INSERT INTO
    system_settings (key, value, description)
VALUES (
        'organization_name',
        '',
        'Название организации-отправителя для исходящих документов'
    ),
    (
        'organization_short_name',
        '',
        'Краткое название организации'
    ),
    (
        'password_lifetime_days',
        '0',
        'Срок жизни пароля (дней, 0 - без ограничения)'
    );


CREATE TABLE IF NOT EXISTS document_journal (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE RESTRICT,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action VARCHAR(100) NOT NULL, -- 'CREATE', 'UPDATE', 'DELETE', 'STATUS_CHANGE', etc.
    details TEXT,
    outbox_deduplication_key VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_document_journal_doc_id ON document_journal(document_id);
CREATE INDEX IF NOT EXISTS idx_document_journal_created_at ON document_journal(created_at DESC);
CREATE UNIQUE INDEX idx_document_journal_outbox_dedup
    ON document_journal (outbox_deduplication_key)
    WHERE outbox_deduplication_key IS NOT NULL;


CREATE TABLE IF NOT EXISTS admin_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Scheduled operations and actors absent after a restore have no current user.
    user_id UUID REFERENCES users(id) ON DELETE RESTRICT,
    user_name TEXT NOT NULL,
    action VARCHAR(100) NOT NULL,
    details TEXT,
    outbox_deduplication_key VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_admin_audit_log_created_at ON admin_audit_log(created_at DESC);
CREATE UNIQUE INDEX idx_admin_audit_log_outbox_dedup
    ON admin_audit_log (outbox_deduplication_key)
    WHERE outbox_deduplication_key IS NOT NULL;


CREATE TABLE IF NOT EXISTS user_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipient_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    document_kind VARCHAR(100) NOT NULL,
    document_number VARCHAR(255),
    entity_type VARCHAR(100) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    outbox_deduplication_key VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    read_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_user_events_recipient_created
    ON user_events (recipient_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_events_recipient_unread
    ON user_events (recipient_user_id, created_at DESC)
    WHERE read_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_events_document_recipient_unread
    ON user_events (document_id, recipient_user_id)
    WHERE read_at IS NULL;

CREATE UNIQUE INDEX idx_user_events_outbox_deduplication
    ON user_events (outbox_deduplication_key)
    WHERE outbox_deduplication_key IS NOT NULL;


CREATE TABLE event_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    deduplication_key VARCHAR(255) NOT NULL UNIQUE,
    payload JSONB NOT NULL,
    available_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processing_started_at TIMESTAMP WITH TIME ZONE,
    processed_at TIMESTAMP WITH TIME ZONE,
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    last_error TEXT,
    failed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_event_outbox_pending
    ON event_outbox (available_at, created_at)
    WHERE processed_at IS NULL;

CREATE INDEX idx_event_outbox_failed
    ON event_outbox (failed_at, created_at)
    WHERE failed_at IS NOT NULL;

CREATE INDEX idx_event_outbox_processed_cleanup
    ON event_outbox (processed_at, id)
    WHERE processed_at IS NOT NULL;


CREATE TABLE server_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash BYTEA NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked_at TIMESTAMP WITH TIME ZONE,
    last_seen_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_server_sessions_user_active
    ON server_sessions(user_id, expires_at)
    WHERE revoked_at IS NULL;

CREATE INDEX idx_server_sessions_expiry ON server_sessions(expires_at);


CREATE TABLE backup_settings (
    id BOOLEAN PRIMARY KEY DEFAULT true CHECK (id),
    settings JSONB NOT NULL
);
