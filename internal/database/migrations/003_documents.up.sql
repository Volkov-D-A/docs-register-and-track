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
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_documents_kind ON documents (kind);
CREATE INDEX idx_documents_nomenclature ON documents (nomenclature_id);
CREATE INDEX idx_documents_registration_date ON documents (registration_date);
CREATE UNIQUE INDEX idx_documents_kind_registration_number_year
    ON documents (kind, registration_number, EXTRACT(YEAR FROM registration_date));
CREATE INDEX idx_documents_created_at ON documents (created_at);
CREATE INDEX idx_documents_kind_created_at ON documents (kind, created_at DESC);
CREATE UNIQUE INDEX idx_documents_created_by_kind_idempotency
    ON documents (created_by, kind, idempotency_key);

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
    appeal_pages_count INT NOT NULL DEFAULT 1,
    attachment_pages_count INT NOT NULL DEFAULT 0,
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
