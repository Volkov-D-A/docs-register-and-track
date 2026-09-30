DELETE FROM user_system_permissions WHERE permission = 'reports';
ALTER TABLE user_system_permissions DROP CONSTRAINT user_system_permissions_permission_check;
ALTER TABLE user_system_permissions ADD CONSTRAINT user_system_permissions_permission_check
    CHECK (permission IN ('admin', 'references', 'stats_documents', 'stats_assignments', 'stats_system'));
