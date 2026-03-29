package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinkRepository_Create(t *testing.T) {
	// Создание новой связи между документами
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewLinkRepository(&database.DB{DB: db})
	ctx := context.Background()

	link := &models.DocumentLink{
		SourceType: "incoming",
		SourceID:   uuid.New(),
		TargetType: "outgoing",
		TargetID:   uuid.New(),
		LinkType:   "reply",
		CreatedBy:  uuid.New(),
	}

	query := `INSERT INTO document_links \( source_type, source_id, target_type, target_id, link_type, created_by \) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6\) RETURNING id, created_at`

	now := time.Now()
	newID := uuid.New()
	mock.ExpectQuery(query).
		WithArgs(link.SourceType, link.SourceID, link.TargetType, link.TargetID, link.LinkType, link.CreatedBy).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(newID, now))

	err = repo.Create(ctx, link)
	require.NoError(t, err)
	assert.Equal(t, newID, link.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLinkRepository_Delete(t *testing.T) {
	// Удаление связи между документами по её ID
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewLinkRepository(&database.DB{DB: db})
	ctx := context.Background()
	id := uuid.New()

	query := `DELETE FROM document_links WHERE id = \$1`
	mock.ExpectExec(query).WithArgs(id).WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Delete(ctx, id)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLinkRepository_GetByDocumentID(t *testing.T) {
	// Получение списка всех связей для конкретного документа (как для входящих, так и для исходящих)
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewLinkRepository(&database.DB{DB: db})
	ctx := context.Background()
	docID := uuid.New()
	now := time.Now()

	query := `SELECT 
			l.id, l.source_type, l.source_id,
			l.target_type, l.target_id,
			l.link_type, l.created_by, l.created_at,
			-- Fetch source document number/subject \(simplified, assumes specific tables exist\)
			CASE 
				WHEN l.source_type = 'incoming' THEN \(SELECT incoming_number FROM incoming_documents WHERE id = l.source_id\)
				WHEN l.source_type = 'outgoing' THEN \(SELECT outgoing_number FROM outgoing_documents WHERE id = l.source_id\)
			END as source_number,
			CASE 
				WHEN l.target_type = 'incoming' THEN \(SELECT incoming_number FROM incoming_documents WHERE id = l.target_id\)
				WHEN l.target_type = 'outgoing' THEN \(SELECT outgoing_number FROM outgoing_documents WHERE id = l.target_id\)
			END as target_number,
             CASE 
				WHEN l.target_type = 'incoming' THEN \(SELECT content FROM incoming_documents WHERE id = l.target_id\)
				WHEN l.target_type = 'outgoing' THEN \(SELECT content FROM outgoing_documents WHERE id = l.target_id\)
			END as target_subject
		FROM document_links l
		WHERE l.source_id = \$1 OR l.target_id = \$1
		ORDER BY l.created_at DESC`

	rows := sqlmock.NewRows([]string{
		"id", "source_type", "source_id", "target_type", "target_id",
		"link_type", "created_by", "created_at",
		"source_number", "target_number", "target_subject",
	}).AddRow(
		uuid.New(), "incoming", docID, "outgoing", uuid.New(),
		"reply", uuid.New(), now,
		"INC-001", "OUT-002", "Subject Test",
	)

	mock.ExpectQuery(query).WithArgs(docID).WillReturnRows(rows)

	links, err := repo.GetByDocumentID(ctx, docID)
	require.NoError(t, err)
	require.Len(t, links, 1)
	assert.Equal(t, "INC-001", links[0].SourceNumber)
	assert.Equal(t, "OUT-002", links[0].TargetNumber)
	assert.Equal(t, "Subject Test", links[0].TargetSubject)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLinkRepository_GetGraph(t *testing.T) {
	// Получение полного графа связанных документов через рекурсивный SQL-запрос
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewLinkRepository(&database.DB{DB: db})
	ctx := context.Background()
	rootID := uuid.New()
	now := time.Now()

	query := `WITH RECURSIVE doc_graph AS \(
			-- Base case: direct links to/from the root document
			SELECT 
				id, source_type, source_id, target_type, target_id, link_type, created_by, created_at,
				1 as depth
			FROM document_links
			WHERE source_id = \$1 OR target_id = \$1
			
			UNION
			
			-- Recursive step: find links connected to documents in the graph
			SELECT 
				l.id, l.source_type, l.source_id, l.target_type, l.target_id, l.link_type, l.created_by, l.created_at,
				g.depth \+ 1
			FROM document_links l
			JOIN doc_graph g ON \(l.source_id = g.target_id OR l.target_id = g.source_id OR l.source_id = g.source_id OR l.target_id = g.target_id\)
			WHERE g.depth < 5 AND l.id != g.id -- Limit depth to prevent infinite loops \(though usually DAG\)
		\)
		SELECT DISTINCT 
			id, source_type, source_id, target_type, target_id, link_type, created_by, created_at 
		FROM doc_graph`

	rows := sqlmock.NewRows([]string{
		"id", "source_type", "source_id", "target_type", "target_id",
		"link_type", "created_by", "created_at",
	}).AddRow(uuid.New(), "incoming", rootID, "outgoing", uuid.New(), "reply", uuid.New(), now)

	mock.ExpectQuery(query).WithArgs(rootID).WillReturnRows(rows)

	links, err := repo.GetGraph(ctx, rootID)
	require.NoError(t, err)
	require.Len(t, links, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}
