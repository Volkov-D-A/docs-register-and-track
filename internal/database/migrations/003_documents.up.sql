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