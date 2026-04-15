-- 8. Common Documents
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind VARCHAR(40) NOT NULL CHECK (kind IN ('incoming_letter', 'outgoing_letter')),
    nomenclature_id UUID NOT NULL REFERENCES nomenclature(id),
    registration_number VARCHAR(100) NOT NULL,
    registration_date DATE NOT NULL,
    document_type_id UUID NOT NULL REFERENCES document_types(id),
    content TEXT NOT NULL,
    pages_count INT NOT NULL DEFAULT 1,
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

-- 9. Incoming Document Details
CREATE TABLE incoming_document_details (
    document_id UUID PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE,
    incoming_number VARCHAR(50) NOT NULL,
    incoming_date DATE NOT NULL,
    outgoing_number_sender VARCHAR(100) NOT NULL,
    outgoing_date_sender DATE NOT NULL,
    intermediate_number VARCHAR(100),
    intermediate_date DATE,
    sender_org_id UUID NOT NULL REFERENCES organizations(id),
    sender_signatory VARCHAR(255) NOT NULL,
    resolution TEXT,
    resolution_author VARCHAR(255),
    resolution_executors TEXT
);

CREATE INDEX idx_incoming_doc_details_number ON incoming_document_details (incoming_number);
CREATE INDEX idx_incoming_doc_details_date ON incoming_document_details (incoming_date);
CREATE INDEX idx_incoming_doc_details_sender ON incoming_document_details (sender_org_id);

-- 10. Outgoing Document Details
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

-- 11. Document Links
CREATE TABLE document_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    source_document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    target_document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
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
    UNIQUE (source_document_id, target_document_id)
);

CREATE INDEX idx_document_links_source ON document_links (source_document_id);
CREATE INDEX idx_document_links_target ON document_links (target_document_id);
