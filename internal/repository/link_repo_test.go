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
		SourceID:  uuid.New(),
		TargetID:  uuid.New(),
		LinkType:  "reply",
		CreatedBy: uuid.New(),
	}

	query := `INSERT INTO document_links \( source_document_id, target_document_id, link_type, created_by \) VALUES \(\$1, \$2, \$3, \$4\) RETURNING id, created_at`

	now := time.Now()
	newID := uuid.New()
	mock.ExpectQuery(query).
		WithArgs(link.SourceID, link.TargetID, link.LinkType, link.CreatedBy).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(newID, now))

	err = repo.Create(ctx, link)
	require.NoError(t, err)
	assert.Equal(t, newID, link.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLinkRepository_CreateAndCancelOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewLinkRepository(&database.DB{DB: db})
	link := &models.DocumentLink{SourceID: uuid.New(), TargetID: uuid.New(), LinkType: "order_cancels", CreatedBy: uuid.New()}
	createdAt := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO document_links`).
		WithArgs(link.SourceID, link.TargetID, link.LinkType, link.CreatedBy).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(uuid.New(), createdAt))
	mock.ExpectExec(`UPDATE documents d`).
		WithArgs(link.TargetID, models.DocumentKindAdministrativeOrder, createdAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE administrative_order_details`).
		WithArgs(link.TargetID, createdAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = repo.CreateAndCancelOrder(context.Background(), link)
	require.NoError(t, err)
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

func TestLinkRepository_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewLinkRepository(&database.DB{DB: db})
		ctx := context.Background()
		id := uuid.New()
		sourceID := uuid.New()
		targetID := uuid.New()
		createdBy := uuid.New()
		now := time.Now()

		query := `SELECT l\.id, ds\.kind, l\.source_document_id, dt\.kind, l\.target_document_id, l\.link_type, l\.created_by, l\.created_at FROM document_links l JOIN documents ds ON ds\.id = l\.source_document_id JOIN documents dt ON dt\.id = l\.target_document_id WHERE l\.id = \$1`
		rows := sqlmock.NewRows([]string{
			"id", "source_kind", "source_document_id", "target_kind", "target_document_id",
			"link_type", "created_by", "created_at",
		}).AddRow(id, models.DocumentKindIncomingLetter, sourceID, models.DocumentKindOutgoingLetter, targetID, "reply", createdBy, now)

		mock.ExpectQuery(query).WithArgs(id).WillReturnRows(rows)

		link, err := repo.GetByID(ctx, id)
		require.NoError(t, err)
		require.NotNil(t, link)
		assert.Equal(t, id, link.ID)
		assert.Equal(t, models.DocumentKindIncomingLetter, link.SourceKind)
		assert.Equal(t, sourceID, link.SourceID)
		assert.Equal(t, models.DocumentKindOutgoingLetter, link.TargetKind)
		assert.Equal(t, targetID, link.TargetID)
		assert.Equal(t, "reply", link.LinkType)
		assert.Equal(t, createdBy, link.CreatedBy)
		assert.Equal(t, now, link.CreatedAt)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewLinkRepository(&database.DB{DB: db})
		ctx := context.Background()
		id := uuid.New()

		query := `SELECT l\.id, ds\.kind, l\.source_document_id, dt\.kind, l\.target_document_id, l\.link_type, l\.created_by, l\.created_at FROM document_links l JOIN documents ds ON ds\.id = l\.source_document_id JOIN documents dt ON dt\.id = l\.target_document_id WHERE l\.id = \$1`
		mock.ExpectQuery(query).WithArgs(id).WillReturnError(sqlmock.ErrCancelled)

		link, err := repo.GetByID(ctx, id)
		require.Error(t, err)
		assert.Nil(t, link)
		require.NoError(t, mock.ExpectationsWereMet())
	})
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

	query := `SELECT(.*)FROM document_links l(.*)JOIN documents ds ON ds.id = l.source_document_id(.*)WHERE l.source_document_id = \$1 OR l.target_document_id = \$1(.*)`

	rows := sqlmock.NewRows([]string{
		"id", "kind", "source_document_id", "kind", "target_document_id",
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

	query := `WITH RECURSIVE doc_graph AS \(.*ds\.kind AS source_type.*l\.source_document_id AS source_id.*dt\.kind AS target_type.*l\.target_document_id AS target_id.*source_document_id = \$1 OR target_document_id = \$1.*SELECT DISTINCT.*source_type, source_id, target_type, target_id, link_type, created_by, created_at.*FROM doc_graph`

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
