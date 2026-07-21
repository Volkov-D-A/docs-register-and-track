-- Matches the actual document-list scope and ORDER BY created_at DESC.
CREATE INDEX IF NOT EXISTS idx_documents_kind_nomenclature_created_at
    ON documents (kind, nomenclature_id, created_at DESC);

-- pg_trgm is deliberately not enabled here. The current cross-table OR search
-- predicate does not safely benefit from standalone trigram indexes across
-- both selective and broad search terms.
