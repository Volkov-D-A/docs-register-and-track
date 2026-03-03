package repository

import (
	"context"
	"docflow/internal/database"
	"docflow/internal/models"

	"github.com/google/uuid"
)

// JournalRepository предоставляет методы для работы с журналом действий.
type JournalRepository struct {
	db *database.DB
}

// NewJournalRepository создает новый экземпляр JournalRepository.
func NewJournalRepository(db *database.DB) *JournalRepository {
	return &JournalRepository{db: db}
}

func (r *JournalRepository) Create(ctx context.Context, req models.CreateJournalEntryRequest) (uuid.UUID, error) {
	query := `
		INSERT INTO document_journal (document_id, document_type, user_id, action, details)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx, query, req.DocumentID, req.DocumentType, req.UserID, req.Action, req.Details).Scan(&id)
	return id, err
}

func (r *JournalRepository) GetByDocumentID(ctx context.Context, documentID uuid.UUID, documentType string) ([]models.JournalEntry, error) {
	query := `
		SELECT j.id, j.document_id, j.document_type, j.user_id, 
		       u.full_name, 
		       j.action, j.details, j.created_at
		FROM document_journal j
		JOIN users u ON j.user_id = u.id
		WHERE j.document_id = $1 AND j.document_type = $2
		ORDER BY j.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, documentID, documentType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.JournalEntry
	for rows.Next() {
		var entry models.JournalEntry
		err := rows.Scan(
			&entry.ID,
			&entry.DocumentID,
			&entry.DocumentType,
			&entry.UserID,
			&entry.UserName,
			&entry.Action,
			&entry.Details,
			&entry.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Возвращаем пустой массив вместо nil для корректной сериализации в JSON
	if entries == nil {
		entries = []models.JournalEntry{}
	}

	return entries, nil
}
