-- 1. Departments
CREATE TABLE IF NOT EXISTS departments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. Users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    login VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
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
        permission IN ('admin', 'references', 'stats_documents', 'stats_assignments', 'stats_system')
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
