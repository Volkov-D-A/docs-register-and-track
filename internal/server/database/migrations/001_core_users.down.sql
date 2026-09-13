DROP TRIGGER IF EXISTS active_administrator_required_after_permission_change ON user_system_permissions;
DROP TRIGGER IF EXISTS active_administrator_required_after_user_update ON users;
DROP FUNCTION IF EXISTS ensure_active_administrator_exists();

DROP TABLE IF EXISTS user_system_permissions;

DROP TABLE IF EXISTS users;

DROP TABLE IF EXISTS departments;
