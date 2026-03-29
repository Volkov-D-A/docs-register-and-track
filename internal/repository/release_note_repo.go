package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// ReleaseNoteRepository предоставляет методы для работы с релизами приложения.
type ReleaseNoteRepository struct {
	db *database.DB
}

// NewReleaseNoteRepository создает новый экземпляр ReleaseNoteRepository.
func NewReleaseNoteRepository(db *database.DB) *ReleaseNoteRepository {
	return &ReleaseNoteRepository{db: db}
}

// GetCurrentForUser возвращает текущий релиз и статус ознакомления пользователя.
func (r *ReleaseNoteRepository) GetCurrentForUser(userID uuid.UUID) (*models.ReleaseNote, error) {
	query := `
		SELECT
			rn.id,
			rn.version,
			rn.released_at,
			rn.is_current,
			rn.created_at,
			EXISTS(
				SELECT 1
				FROM user_release_views urv
				WHERE urv.release_note_id = rn.id AND urv.user_id = $1
			) AS is_viewed
		FROM release_notes rn
		WHERE rn.is_current = TRUE
		ORDER BY rn.released_at DESC, rn.created_at DESC
		LIMIT 1
	`

	return r.getSingleByQuery(query, userID)
}

// GetAll возвращает историю всех релизов приложения.
func (r *ReleaseNoteRepository) GetAll() ([]models.ReleaseNote, error) {
	rows, err := r.db.Query(`
		SELECT
			rn.id,
			rn.version,
			rn.released_at,
			rn.is_current,
			rn.created_at,
			FALSE AS is_viewed
		FROM release_notes rn
		ORDER BY rn.released_at DESC, rn.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get release notes: %w", err)
	}
	defer rows.Close()

	notes := make([]models.ReleaseNote, 0)
	noteIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var note models.ReleaseNote
		if err := rows.Scan(
			&note.ID,
			&note.Version,
			&note.ReleasedAt,
			&note.IsCurrent,
			&note.CreatedAt,
			&note.IsViewed,
		); err != nil {
			return nil, fmt.Errorf("failed to scan release note: %w", err)
		}
		notes = append(notes, note)
		noteIDs = append(noteIDs, note.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate release notes: %w", err)
	}

	if len(notes) == 0 {
		return notes, nil
	}

	changesByReleaseID, err := r.getChangesByReleaseNoteIDs(noteIDs)
	if err != nil {
		return nil, err
	}

	for i := range notes {
		notes[i].Changes = changesByReleaseID[notes[i].ID]
	}

	return notes, nil
}

// Create создает новый релиз с перечнем изменений.
func (r *ReleaseNoteRepository) Create(req models.CreateReleaseNoteRequest, releasedAt time.Time) (*models.ReleaseNote, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if req.IsCurrent {
		if _, err := tx.Exec(`UPDATE release_notes SET is_current = FALSE WHERE is_current = TRUE`); err != nil {
			return nil, fmt.Errorf("failed to reset current release note: %w", err)
		}
	}

	var noteID uuid.UUID
	var note models.ReleaseNote
	err = tx.QueryRow(`
		INSERT INTO release_notes (version, released_at, is_current)
		VALUES ($1, $2, $3)
		RETURNING id, version, released_at, is_current, created_at
	`, req.Version, releasedAt, req.IsCurrent).Scan(
		&noteID,
		&note.Version,
		&note.ReleasedAt,
		&note.IsCurrent,
		&note.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create release note: %w", err)
	}

	for index, change := range req.Changes {
		if _, err := tx.Exec(`
			INSERT INTO release_note_changes (release_note_id, sort_order, title, description)
			VALUES ($1, $2, $3, $4)
		`, noteID, index+1, change.Title, change.Description); err != nil {
			return nil, fmt.Errorf("failed to create release note change: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return r.getByID(noteID)
}

// MarkViewed отмечает, что пользователь ознакомился с релизом.
func (r *ReleaseNoteRepository) MarkViewed(releaseNoteID, userID uuid.UUID) error {
	query := `
		INSERT INTO user_release_views (release_note_id, user_id, viewed_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (release_note_id, user_id)
		DO UPDATE SET viewed_at = EXCLUDED.viewed_at
	`
	_, err := r.db.Exec(query, releaseNoteID, userID)
	if err != nil {
		return fmt.Errorf("failed to mark release note as viewed: %w", err)
	}
	return nil
}

// SetCurrent делает указанный релиз текущим.
func (r *ReleaseNoteRepository) SetCurrent(releaseNoteID uuid.UUID) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE release_notes SET is_current = FALSE WHERE is_current = TRUE`); err != nil {
		return fmt.Errorf("failed to reset current release note: %w", err)
	}

	result, err := tx.Exec(`UPDATE release_notes SET is_current = TRUE WHERE id = $1`, releaseNoteID)
	if err != nil {
		return fmt.Errorf("failed to set current release note: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("release note not found")
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *ReleaseNoteRepository) getChanges(releaseNoteID uuid.UUID) ([]models.ReleaseNoteChange, error) {
	rows, err := r.db.Query(`
		SELECT id, release_note_id, sort_order, title, description
		FROM release_note_changes
		WHERE release_note_id = $1
		ORDER BY sort_order ASC, created_at ASC
	`, releaseNoteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get release note changes: %w", err)
	}
	defer rows.Close()

	changes := make([]models.ReleaseNoteChange, 0)
	for rows.Next() {
		var change models.ReleaseNoteChange
		if err := rows.Scan(
			&change.ID,
			&change.ReleaseNoteID,
			&change.SortOrder,
			&change.Title,
			&change.Description,
		); err != nil {
			return nil, fmt.Errorf("failed to scan release note change: %w", err)
		}
		changes = append(changes, change)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate release note changes: %w", err)
	}

	return changes, nil
}

func (r *ReleaseNoteRepository) getByID(releaseNoteID uuid.UUID) (*models.ReleaseNote, error) {
	query := `
		SELECT
			rn.id,
			rn.version,
			rn.released_at,
			rn.is_current,
			rn.created_at,
			FALSE AS is_viewed
		FROM release_notes rn
		WHERE rn.id = $1
	`
	return r.getSingleByQuery(query, releaseNoteID)
}

func (r *ReleaseNoteRepository) getSingleByQuery(query string, args ...any) (*models.ReleaseNote, error) {
	var note models.ReleaseNote
	err := r.db.QueryRow(query, args...).Scan(
		&note.ID,
		&note.Version,
		&note.ReleasedAt,
		&note.IsCurrent,
		&note.CreatedAt,
		&note.IsViewed,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get release note: %w", err)
	}

	changes, err := r.getChanges(note.ID)
	if err != nil {
		return nil, err
	}
	note.Changes = changes

	return &note, nil
}

func (r *ReleaseNoteRepository) getChangesByReleaseNoteIDs(releaseNoteIDs []uuid.UUID) (map[uuid.UUID][]models.ReleaseNoteChange, error) {
	args := make([]any, 0, len(releaseNoteIDs))
	placeholders := make([]string, 0, len(releaseNoteIDs))
	for i, releaseNoteID := range releaseNoteIDs {
		args = append(args, releaseNoteID)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
	}

	rows, err := r.db.Query(fmt.Sprintf(`
		SELECT id, release_note_id, sort_order, title, description
		FROM release_note_changes
		WHERE release_note_id IN (%s)
		ORDER BY release_note_id, sort_order ASC, created_at ASC
	`, strings.Join(placeholders, ", ")), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get release note changes: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]models.ReleaseNoteChange, len(releaseNoteIDs))
	for rows.Next() {
		var change models.ReleaseNoteChange
		if err := rows.Scan(
			&change.ID,
			&change.ReleaseNoteID,
			&change.SortOrder,
			&change.Title,
			&change.Description,
		); err != nil {
			return nil, fmt.Errorf("failed to scan release note change: %w", err)
		}
		result[change.ReleaseNoteID] = append(result[change.ReleaseNoteID], change)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate release note changes: %w", err)
	}

	return result, nil
}
