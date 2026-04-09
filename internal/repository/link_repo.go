package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// LinkRepository предоставляет методы для работы со связями между документами в БД.
type LinkRepository struct {
	db *database.DB
}

// NewLinkRepository создает новый экземпляр LinkRepository.
func NewLinkRepository(db *database.DB) *LinkRepository {
	return &LinkRepository{db: db}
}

// Create — создать новую связь между документами
func (r *LinkRepository) Create(ctx context.Context, link *models.DocumentLink) error {
	query := `
		INSERT INTO document_links (
			source_document_id, target_document_id,
			link_type, created_by
		) VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`

	return r.db.QueryRowContext(ctx, query,
		link.SourceID, link.TargetID, link.LinkType, link.CreatedBy,
	).Scan(&link.ID, &link.CreatedAt)
}

// Delete — удалить связь по ID
func (r *LinkRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM document_links WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// GetByID — получить связь по ID
func (r *LinkRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DocumentLink, error) {
	query := `
		SELECT l.id, ds.kind, l.source_document_id, dt.kind, l.target_document_id, l.link_type, l.created_by, l.created_at
		FROM document_links l
		JOIN documents ds ON ds.id = l.source_document_id
		JOIN documents dt ON dt.id = l.target_document_id
		WHERE l.id = $1
	`
	var l models.DocumentLink
	err := r.db.QueryRowContext(ctx, query, id).Scan(&l.ID, &l.SourceType, &l.SourceID, &l.TargetType, &l.TargetID, &l.LinkType, &l.CreatedBy, &l.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// GetByDocumentID — получить все связи, где документ является источником или целью
func (r *LinkRepository) GetByDocumentID(ctx context.Context, docID uuid.UUID) ([]models.DocumentLink, error) {
	query := `
		SELECT 
			l.id, ds.kind, l.source_document_id,
			dt.kind, l.target_document_id,
			l.link_type, l.created_by, l.created_at,
			COALESCE(si.incoming_number, so.outgoing_number) as source_number,
			COALESCE(ti.incoming_number, to2.outgoing_number) as target_number,
			dtarget.content as target_subject
		FROM document_links l
		JOIN documents ds ON ds.id = l.source_document_id
		JOIN documents dt ON dt.id = l.target_document_id
		JOIN documents dtarget ON dtarget.id = l.target_document_id
		LEFT JOIN incoming_document_details si ON si.document_id = l.source_document_id AND ds.kind = 'incoming'
		LEFT JOIN outgoing_document_details so ON so.document_id = l.source_document_id AND ds.kind = 'outgoing'
		LEFT JOIN incoming_document_details ti ON ti.document_id = l.target_document_id AND dt.kind = 'incoming'
		LEFT JOIN outgoing_document_details to2 ON to2.document_id = l.target_document_id AND dt.kind = 'outgoing'
		WHERE l.source_document_id = $1 OR l.target_document_id = $1
		ORDER BY l.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := make([]models.DocumentLink, 0)
	for rows.Next() {
		var l models.DocumentLink
		var sourceNum, targetNum, targetSubj sql.NullString
		if err := rows.Scan(
			&l.ID, &l.SourceType, &l.SourceID,
			&l.TargetType, &l.TargetID,
			&l.LinkType, &l.CreatedBy, &l.CreatedAt,
			&sourceNum, &targetNum, &targetSubj,
		); err != nil {
			return nil, err
		}

		if sourceNum.Valid {
			l.SourceNumber = sourceNum.String
		}
		if targetNum.Valid {
			l.TargetNumber = targetNum.String
		}
		if targetSubj.Valid {
			l.TargetSubject = targetSubj.String
		}

		links = append(links, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return links, nil
}

// GetGraph — получить связанный граф документов с помощью рекурсивного CTE.
// Находит все документы, связанные с rootID, обходя оба направления
func (r *LinkRepository) GetGraph(ctx context.Context, rootID uuid.UUID) ([]models.DocumentLink, error) {
	// Рекурсивный CTE для поиска всех связанных документов
	query := `
		WITH RECURSIVE doc_graph AS (
			-- Base case: direct links to/from the root document
			SELECT 
				l.id,
				ds.kind AS source_type,
				l.source_document_id AS source_id,
				dt.kind AS target_type,
				l.target_document_id AS target_id,
				l.link_type,
				l.created_by,
				l.created_at,
				1 as depth
			FROM document_links l
			JOIN documents ds ON ds.id = l.source_document_id
			JOIN documents dt ON dt.id = l.target_document_id
			WHERE source_document_id = $1 OR target_document_id = $1
			
			UNION
			
			-- Recursive step: find links connected to documents in the graph
			SELECT 
				l.id,
				ds.kind AS source_type,
				l.source_document_id AS source_id,
				dt.kind AS target_type,
				l.target_document_id AS target_id,
				l.link_type,
				l.created_by,
				l.created_at,
				g.depth + 1
			FROM document_links l
			JOIN documents ds ON ds.id = l.source_document_id
			JOIN documents dt ON dt.id = l.target_document_id
			JOIN doc_graph g ON (
				l.source_document_id = g.target_id OR
				l.target_document_id = g.source_id OR
				l.source_document_id = g.source_id OR
				l.target_document_id = g.target_id
			)
			WHERE g.depth < 5 AND l.id != g.id -- Limit depth to prevent infinite loops (though usually DAG)
		)
		SELECT DISTINCT 
			id, source_type, source_id, target_type, target_id, link_type, created_by, created_at
		FROM doc_graph
	`

	rows, err := r.db.QueryContext(ctx, query, rootID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := make([]models.DocumentLink, 0)
	for rows.Next() {
		var l models.DocumentLink
		if err := rows.Scan(
			&l.ID, &l.SourceType, &l.SourceID,
			&l.TargetType, &l.TargetID,
			&l.LinkType, &l.CreatedBy, &l.CreatedAt,
		); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return links, nil
}
