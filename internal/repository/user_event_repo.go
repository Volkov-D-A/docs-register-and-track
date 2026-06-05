package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// UserEventRepository предоставляет методы для работы с персональными событиями.
type UserEventRepository struct {
	db *database.DB
}

// NewUserEventRepository создает новый экземпляр UserEventRepository.
func NewUserEventRepository(db *database.DB) *UserEventRepository {
	return &UserEventRepository{db: db}
}

// Create создает событие пользователя.
func (r *UserEventRepository) Create(req models.CreateUserEventRequest) (*models.UserEvent, error) {
	metadata := req.Metadata
	if metadata == "" {
		metadata = "{}"
	}

	query := `
		INSERT INTO user_events (
			recipient_user_id, actor_user_id, document_id, document_kind,
			document_number, entity_type, entity_id, event_type,
			title, message, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb)
		RETURNING id
	`
	var id uuid.UUID
	err := r.db.QueryRow(
		query,
		req.RecipientUserID,
		req.ActorUserID,
		req.DocumentID,
		req.DocumentKind,
		req.DocumentNumber,
		req.EntityType,
		req.EntityID,
		req.EventType,
		req.Title,
		req.Message,
		metadata,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to create user event: %w", err)
	}
	return r.GetByID(id)
}

// GetByID возвращает событие по ID.
func (r *UserEventRepository) GetByID(id uuid.UUID) (*models.UserEvent, error) {
	query := `
		SELECT
			e.id, e.recipient_user_id, e.actor_user_id, actor.full_name,
			e.document_id, e.document_kind, e.document_number,
			e.entity_type, e.entity_id, e.event_type,
			e.title, e.message, e.metadata::text,
			e.created_at, e.read_at
		FROM user_events e
		LEFT JOIN users actor ON actor.id = e.actor_user_id
		WHERE e.id = $1
	`
	return scanUserEvent(r.db.QueryRow(query, id))
}

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
			e.id, e.recipient_user_id, e.actor_user_id, actor.full_name,
			e.document_id, e.document_kind, e.document_number,
			e.entity_type, e.entity_id, e.event_type,
			e.title, e.message, e.metadata::text,
			e.created_at, e.read_at
		FROM user_events e
		LEFT JOIN users actor ON actor.id = e.actor_user_id
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
		event, err := scanUserEventRows(rows)
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

type userEventRows interface {
	Scan(dest ...interface{}) error
}

func scanUserEvent(scanner userEventScanner) (*models.UserEvent, error) {
	event, err := scanUserEventValue(scanner)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return event, nil
}

func scanUserEventRows(rows userEventRows) (*models.UserEvent, error) {
	return scanUserEventValue(rows)
}

func scanUserEventValue(scanner userEventScanner) (*models.UserEvent, error) {
	var event models.UserEvent
	var actorUserID sql.NullString
	var actorUserName sql.NullString
	var documentNumber sql.NullString
	var metadata sql.NullString
	var readAt sql.NullTime

	err := scanner.Scan(
		&event.ID,
		&event.RecipientUserID,
		&actorUserID,
		&actorUserName,
		&event.DocumentID,
		&event.DocumentKind,
		&documentNumber,
		&event.EntityType,
		&event.EntityID,
		&event.EventType,
		&event.Title,
		&event.Message,
		&metadata,
		&event.CreatedAt,
		&readAt,
	)
	if err != nil {
		return nil, err
	}

	if actorUserID.Valid {
		uid, err := uuid.Parse(actorUserID.String)
		if err != nil {
			return nil, err
		}
		event.ActorUserID = &uid
	}
	if actorUserName.Valid {
		event.ActorUserName = actorUserName.String
	}
	if documentNumber.Valid {
		event.DocumentNumber = documentNumber.String
	}
	if metadata.Valid {
		event.Metadata = metadata.String
	}
	if readAt.Valid {
		event.ReadAt = &readAt.Time
	}

	return &event, nil
}
