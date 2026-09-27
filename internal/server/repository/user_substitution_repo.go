package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
)

// UserSubstitutionRepository предоставляет методы работы с замещениями пользователей.
type UserSubstitutionRepository struct {
	db     *database.DB
	outbox *OutboxRepository
}

func (r *UserSubstitutionRepository) SetOutbox(outbox *OutboxRepository) { r.outbox = outbox }

// NewUserSubstitutionRepository создает репозиторий замещений.
func NewUserSubstitutionRepository(db *database.DB) *UserSubstitutionRepository {
	return &UserSubstitutionRepository{db: db}
}

func scanUserSubstitution(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.UserSubstitution, error) {
	var item models.UserSubstitution
	var startsAt sql.NullTime
	var endsAt sql.NullTime
	if err := scanner.Scan(&item.SubstituteUserID, &startsAt, &endsAt, &item.IsActive); err != nil {
		return nil, err
	}
	if startsAt.Valid {
		item.StartsAt = &startsAt.Time
	}
	if endsAt.Valid {
		item.EndsAt = &endsAt.Time
	}
	return &item, nil
}

const userSubstitutionSelect = `
	SELECT us.substitute_user_id, us.starts_at, us.ends_at, us.is_active
	FROM user_substitutions us`

// GetByPrincipalID возвращает настройку замещения пользователя.
func (r *UserSubstitutionRepository) GetByPrincipalID(principalUserID uuid.UUID) (*models.UserSubstitution, error) {
	item, err := scanUserSubstitution(r.db.QueryRow(userSubstitutionSelect+` WHERE us.principal_user_id = $1`, principalUserID))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user substitution: %w", err)
	}
	return item, nil
}

// GetActivePrincipalIDs возвращает пользователей, которых сейчас замещает substituteUserID.
func (r *UserSubstitutionRepository) GetActivePrincipalIDs(substituteUserID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.Query(`
		SELECT principal_user_id
		FROM user_substitutions
		WHERE substitute_user_id = $1
		  AND is_active = true
		  AND (starts_at IS NULL OR starts_at <= CURRENT_DATE)
		  AND (ends_at IS NULL OR ends_at >= CURRENT_DATE)
	`, substituteUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active principals: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// IsActiveSubstitute проверяет, может ли substituteUserID действовать за principalUserID.
func (r *UserSubstitutionRepository) IsActiveSubstitute(substituteUserID, principalUserID uuid.UUID) (bool, error) {
	var ok bool
	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM user_substitutions
			WHERE substitute_user_id = $1
			  AND principal_user_id = $2
			  AND is_active = true
			  AND (starts_at IS NULL OR starts_at <= CURRENT_DATE)
			  AND (ends_at IS NULL OR ends_at >= CURRENT_DATE)
		)
	`, substituteUserID, principalUserID).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("failed to check active substitution: %w", err)
	}
	return ok, nil
}

// ReplaceForPrincipalWithOutbox persists an administrative change and its audit
// event in one transaction.
func (r *UserSubstitutionRepository) ReplaceForPrincipalWithOutbox(
	principalUserID uuid.UUID,
	substituteUserID *uuid.UUID,
	startsAt *time.Time,
	endsAt *time.Time,
	isActive bool,
	effects []models.OutboxEvent,
) (*models.UserSubstitution, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if substituteUserID == nil || *substituteUserID == uuid.Nil {
		if _, err := tx.Exec(`DELETE FROM user_substitutions WHERE principal_user_id = $1`, principalUserID); err != nil {
			return nil, fmt.Errorf("failed to delete user substitution: %w", err)
		}
	} else if _, err := tx.Exec(`
		INSERT INTO user_substitutions (
			principal_user_id, substitute_user_id, starts_at, ends_at, is_active
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (principal_user_id) DO UPDATE
		SET substitute_user_id = EXCLUDED.substitute_user_id,
		    starts_at = EXCLUDED.starts_at,
		    ends_at = EXCLUDED.ends_at,
		    is_active = EXCLUDED.is_active
		`, principalUserID, *substituteUserID, startsAt, endsAt, isActive); err != nil {
		return nil, fmt.Errorf("failed to replace user substitution: %w", err)
	}
	if err := enqueueOutboxEffects(r.outbox, tx, effects); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	if substituteUserID == nil || *substituteUserID == uuid.Nil {
		return nil, nil
	}
	return r.GetByPrincipalID(principalUserID)
}
