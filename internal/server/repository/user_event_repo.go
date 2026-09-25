package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
)

// UserEventRepository предоставляет методы для работы с персональными событиями.
type UserEventRepository struct {
	db *database.DB
}

// NewUserEventRepository создает новый экземпляр UserEventRepository.
func NewUserEventRepository(db *database.DB) *UserEventRepository {
	return &UserEventRepository{db: db}
}

// CreateFromOutbox is idempotent across worker crashes and retries.
func (r *UserEventRepository) CreateFromOutbox(req models.CreateUserEventRequest, deduplicationKey string) error {
	_, err := r.db.Exec(userEventInsertQuery, req.RecipientUserID, req.DocumentID, req.DocumentKind,
		req.DocumentNumber, req.EntityType, req.EventType, req.Title, req.Message, deduplicationKey)
	if err != nil {
		return fmt.Errorf("failed to create user event: %w", err)
	}
	return nil
}

const userEventInsertQuery = `
		INSERT INTO user_events (
			recipient_user_id, document_id, document_kind, document_number,
			entity_type, event_type, title, message, outbox_deduplication_key
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9, ''))
		ON CONFLICT (outbox_deduplication_key) WHERE outbox_deduplication_key IS NOT NULL DO NOTHING`

// GetList возвращает список событий пользователя.
func (r *UserEventRepository) GetList(userID uuid.UUID, filter models.UserEventFilter) (*models.PagedResult[models.UserEvent], error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	where := "WHERE e.recipient_user_id = $1"
	if filter.UnreadOnly {
		where += " AND e.read_at IS NULL"
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM user_events e %s", where)
	var total int
	if err := r.db.QueryRow(countQuery, userID).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count user events: %w", err)
	}

	offset := (filter.Page - 1) * filter.PageSize
	query := fmt.Sprintf(`
		SELECT
			e.id, e.recipient_user_id,
			e.document_id, e.document_kind, e.document_number,
			e.entity_type, e.event_type,
			e.title, e.message,
			e.created_at, e.read_at
		FROM user_events e
		%s
		ORDER BY e.created_at DESC
		LIMIT $2 OFFSET $3
	`, where)

	rows, err := r.db.Query(query, userID, filter.PageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get user events: %w", err)
	}
	defer rows.Close()

	items := make([]models.UserEvent, 0)
	for rows.Next() {
		event, err := scanUserEvent(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &models.PagedResult[models.UserEvent]{
		Items:      items,
		TotalCount: total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	}, nil
}

// CountUnread возвращает количество непрочитанных событий пользователя.
func (r *UserEventRepository) CountUnread(userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM user_events WHERE recipient_user_id = $1 AND read_at IS NULL",
		userID,
	).Scan(&count)
	return count, err
}

// MarkRead отмечает одно событие пользователя прочитанным.
func (r *UserEventRepository) MarkRead(id, userID uuid.UUID, readAt time.Time) error {
	result, err := r.db.Exec(
		"UPDATE user_events SET read_at = COALESCE(read_at, $3) WHERE id = $1 AND recipient_user_id = $2",
		id,
		userID,
		readAt,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return models.NewNotFound("событие не найдено")
	}
	return nil
}

// MarkDocumentRead отмечает все события пользователя по документу прочитанными.
func (r *UserEventRepository) MarkDocumentRead(documentID, userID uuid.UUID, readAt time.Time) error {
	_, err := r.db.Exec(
		"UPDATE user_events SET read_at = COALESCE(read_at, $3) WHERE document_id = $1 AND recipient_user_id = $2 AND read_at IS NULL",
		documentID,
		userID,
		readAt,
	)
	return err
}

// MarkAllRead отмечает все события пользователя прочитанными.
func (r *UserEventRepository) MarkAllRead(userID uuid.UUID, readAt time.Time) error {
	_, err := r.db.Exec(
		"UPDATE user_events SET read_at = COALESCE(read_at, $2) WHERE recipient_user_id = $1 AND read_at IS NULL",
		userID,
		readAt,
	)
	return err
}

type userEventScanner interface {
	Scan(dest ...interface{}) error
}

func scanUserEvent(scanner userEventScanner) (*models.UserEvent, error) {
	var event models.UserEvent
	var documentNumber sql.NullString
	var readAt sql.NullTime

	err := scanner.Scan(
		&event.ID,
		&event.RecipientUserID,
		&event.DocumentID,
		&event.DocumentKind,
		&documentNumber,
		&event.EntityType,
		&event.EventType,
		&event.Title,
		&event.Message,
		&event.CreatedAt,
		&readAt,
	)
	if err != nil {
		return nil, err
	}

	if documentNumber.Valid {
		event.DocumentNumber = documentNumber.String
	}
	if readAt.Valid {
		event.ReadAt = &readAt.Time
	}

	return &event, nil
}
