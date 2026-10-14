DROP TABLE backup_settings;


DROP TABLE IF EXISTS server_sessions;


DROP INDEX IF EXISTS idx_event_outbox_pending;
DROP INDEX IF EXISTS idx_event_outbox_failed;
DROP INDEX IF EXISTS idx_event_outbox_processed_cleanup;
DROP TABLE IF EXISTS event_outbox;


DROP TABLE IF EXISTS user_events;


DROP INDEX IF EXISTS idx_admin_audit_log_created_at;
DROP INDEX IF EXISTS idx_admin_audit_log_outbox_dedup;
DROP TABLE IF EXISTS admin_audit_log;


DROP INDEX IF EXISTS idx_document_journal_created_at;
DROP INDEX IF EXISTS idx_document_journal_doc_id;
DROP INDEX IF EXISTS idx_document_journal_outbox_dedup;

DROP TABLE IF EXISTS document_journal;


DROP TABLE IF EXISTS system_settings;


DROP TRIGGER IF EXISTS deadline_co_executor_deletion ON deadline_executors;
DROP FUNCTION IF EXISTS restrict_deadline_co_executor_deletion();
ALTER TABLE IF EXISTS document_deadline_series DROP CONSTRAINT IF EXISTS deadline_series_current_control;
DROP TABLE IF EXISTS document_deadline_controls;
DROP TABLE IF EXISTS document_deadline_series;
DROP FUNCTION IF EXISTS verify_deadline_co_executors();
DROP FUNCTION IF EXISTS deadline_executor_set_valid(UUID,UUID[]);


DROP INDEX IF EXISTS idx_documents_kind_nomenclature_created_at;

DROP TABLE IF EXISTS document_links;

DROP TABLE IF EXISTS administrative_order_acknowledgment_people;

DROP TABLE IF EXISTS administrative_order_details;

DROP TABLE IF EXISTS citizen_appeal_details;

DROP TABLE IF EXISTS outgoing_document_details;

DROP TABLE IF EXISTS incoming_document_details;

DROP TABLE IF EXISTS document_resolutions;

DROP TABLE IF EXISTS document_correspondent_registrations;

DROP TABLE IF EXISTS document_command_idempotency;

DROP TABLE IF EXISTS documents;


DROP TRIGGER IF EXISTS user_deadline_executor ON users;
DROP FUNCTION IF EXISTS create_user_deadline_executor();
DROP TABLE IF EXISTS deadline_executors;
DROP TABLE IF EXISTS resolution_executors;

DROP TABLE IF EXISTS organizations;

DROP TABLE IF EXISTS department_nomenclature;

DROP TABLE IF EXISTS document_permissions;

DROP TABLE IF EXISTS nomenclature;


DROP TRIGGER IF EXISTS active_administrator_required_after_permission_change ON user_system_permissions;
DROP TRIGGER IF EXISTS active_administrator_required_after_user_update ON users;
DROP FUNCTION IF EXISTS ensure_active_administrator_exists();

DROP TABLE IF EXISTS user_system_permissions;

DROP TABLE IF EXISTS users;

DROP TABLE IF EXISTS departments;
