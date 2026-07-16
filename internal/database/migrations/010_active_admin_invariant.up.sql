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
