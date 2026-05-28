ALTER TABLE document_journal
    DROP CONSTRAINT IF EXISTS document_journal_document_id_fkey,
    ADD CONSTRAINT document_journal_document_id_fkey
        FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE RESTRICT;

ALTER TABLE document_journal
    DROP CONSTRAINT IF EXISTS document_journal_user_id_fkey,
    ADD CONSTRAINT document_journal_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE admin_audit_log
    DROP CONSTRAINT IF EXISTS admin_audit_log_user_id_fkey,
    ADD CONSTRAINT admin_audit_log_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT;
