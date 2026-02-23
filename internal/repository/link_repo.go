package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"docflow/internal/database"
	"docflow/internal/models"
)

type LinkRepository struct {
	db *database.DB
}

func NewLinkRepository(db *database.DB) *LinkRepository {
	return &LinkRepository{db: db}
}

// Create — создать новую связь между документами
func (r *LinkRepository) Create(ctx context.Context, link *models.DocumentLink) error {
	query := `
		INSERT INTO document_links (
			source_type, source_id,
			target_type, target_id,
			link_type, created_by
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	return r.db.QueryRowContext(ctx, query,
		link.SourceType, link.SourceID,
		link.TargetType, link.TargetID,
		link.LinkType, link.CreatedBy,
	).Scan(&link.ID, &link.CreatedAt)
}

// Delete — удалить связь по ID
func (r *LinkRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM document_links WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// GetByDocumentID — получить все связи, где документ является источником или целью
func (r *LinkRepository) GetByDocumentID(ctx context.Context, docID uuid.UUID) ([]models.DocumentLink, error) {
	query := `
		SELECT 
			l.id, l.source_type, l.source_id,
			l.target_type, l.target_id,
			l.link_type, l.created_by, l.created_at,
			-- Fetch source document number/subject (simplified, assumes specific tables exist)
			CASE 
				WHEN l.source_type = 'incoming' THEN (SELECT incoming_number FROM incoming_documents WHERE id = l.source_id)
				WHEN l.source_type = 'outgoing' THEN (SELECT outgoing_number FROM outgoing_documents WHERE id = l.source_id)
			END as source_number,
			CASE 
				WHEN l.target_type = 'incoming' THEN (SELECT incoming_number FROM incoming_documents WHERE id = l.target_id)
				WHEN l.target_type = 'outgoing' THEN (SELECT outgoing_number FROM outgoing_documents WHERE id = l.target_id)
			END as target_number,
             CASE 
				WHEN l.target_type = 'incoming' THEN (SELECT subject FROM incoming_documents WHERE id = l.target_id)
				WHEN l.target_type = 'outgoing' THEN (SELECT subject FROM outgoing_documents WHERE id = l.target_id)
			END as target_subject
		FROM document_links l
		WHERE l.source_id = $1 OR l.target_id = $1
		ORDER BY l.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []models.DocumentLink
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

		l.IDStr = l.ID.String()
		l.SourceIDStr = l.SourceID.String()
		l.TargetIDStr = l.TargetID.String()
		l.CreatedByStr = l.CreatedBy.String()

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
				id, source_type, source_id, target_type, target_id, link_type, created_by, created_at,
				1 as depth
			FROM document_links
			WHERE source_id = $1 OR target_id = $1
			
			UNION
			
			-- Recursive step: find links connected to documents in the graph
			SELECT 
				l.id, l.source_type, l.source_id, l.target_type, l.target_id, l.link_type, l.created_by, l.created_at,
				g.depth + 1
			FROM document_links l
			JOIN doc_graph g ON (l.source_id = g.target_id OR l.target_id = g.source_id OR l.source_id = g.source_id OR l.target_id = g.target_id)
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

	var links []models.DocumentLink
	for rows.Next() {
		var l models.DocumentLink
		if err := rows.Scan(
			&l.ID, &l.SourceType, &l.SourceID,
			&l.TargetType, &l.TargetID,
			&l.LinkType, &l.CreatedBy, &l.CreatedAt,
		); err != nil {
			return nil, err
		}
		l.IDStr = l.ID.String()
		l.SourceIDStr = l.SourceID.String()
		l.TargetIDStr = l.TargetID.String()
		l.CreatedByStr = l.CreatedBy.String()
		links = append(links, l)
	}
	return links, nil
}
