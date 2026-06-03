package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func setupDocumentRepository(t *testing.T) (*DocumentRepository, sqlmock.Sqlmock, func()) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	return NewDocumentRepository(&database.DB{DB: db}), mock, func() { db.Close() }
}

func documentRepositoryRows(now time.Time, rows ...models.Document) *sqlmock.Rows {
	result := sqlmock.NewRows([]string{
		"id", "kind", "nomenclature_id", "registration_number", "registration_date",
		"document_type", "content", "pages_count", "created_by", "created_at", "updated_at",
	})
	for _, doc := range rows {
		createdAt := doc.CreatedAt
		if createdAt.IsZero() {
			createdAt = now
		}
		updatedAt := doc.UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = now
		}
		result.AddRow(
			doc.ID,
			doc.Kind,
			doc.NomenclatureID,
			doc.RegistrationNumber,
			doc.RegistrationDate,
			doc.DocumentTypeID,
			doc.Content,
			doc.PagesCount,
			doc.CreatedBy,
			createdAt,
			updatedAt,
		)
	}
	return result
}

func TestDocumentRepository_GetByID(t *testing.T) {
	repo, mock, cleanup := setupDocumentRepository(t)
	defer cleanup()

	docID := uuid.New()
	now := time.Now()
	expected := models.Document{
		ID:                 docID,
		Kind:               models.DocumentKindIncomingLetter,
		NomenclatureID:     uuid.New(),
		RegistrationNumber: "IN-1",
		RegistrationDate:   now,
		DocumentTypeID:     models.DocumentTypeLetter,
		Content:            "content",
		PagesCount:         3,
		CreatedBy:          uuid.New(),
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	t.Run("success", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, kind, nomenclature_id, registration_number, registration_date`).
			WithArgs(docID).
			WillReturnRows(documentRepositoryRows(now, expected))

		doc, err := repo.GetByID(docID)

		require.NoError(t, err)
		require.NotNil(t, doc)
		assert.Equal(t, expected.ID, doc.ID)
		assert.Equal(t, expected.Kind, doc.Kind)
		assert.Equal(t, expected.RegistrationNumber, doc.RegistrationNumber)
		assert.Equal(t, expected.PagesCount, doc.PagesCount)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, kind, nomenclature_id, registration_number, registration_date`).
			WithArgs(docID).
			WillReturnError(sql.ErrNoRows)

		doc, err := repo.GetByID(docID)

		require.NoError(t, err)
		assert.Nil(t, doc)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDocumentRepository_GetByIDs(t *testing.T) {
	repo, mock, cleanup := setupDocumentRepository(t)
	defer cleanup()

	t.Run("empty ids", func(t *testing.T) {
		docs, err := repo.GetByIDs(nil)

		require.NoError(t, err)
		assert.Empty(t, docs)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success", func(t *testing.T) {
		now := time.Now()
		doc1 := models.Document{
			ID:                 uuid.New(),
			Kind:               models.DocumentKindIncomingLetter,
			NomenclatureID:     uuid.New(),
			RegistrationNumber: "IN-1",
			RegistrationDate:   now,
			DocumentTypeID:     models.DocumentTypeLetter,
			Content:            "incoming",
			PagesCount:         1,
			CreatedBy:          uuid.New(),
		}
		doc2 := models.Document{
			ID:                 uuid.New(),
			Kind:               models.DocumentKindOutgoingLetter,
			NomenclatureID:     uuid.New(),
			RegistrationNumber: "OUT-1",
			RegistrationDate:   now,
			DocumentTypeID:     models.DocumentTypeReply,
			Content:            "outgoing",
			PagesCount:         2,
			CreatedBy:          uuid.New(),
		}

		mock.ExpectQuery(`WHERE id = ANY\(\$1::uuid\[\]\)`).
			WillReturnRows(documentRepositoryRows(now, doc1, doc2))

		docs, err := repo.GetByIDs([]uuid.UUID{doc1.ID, doc2.ID})

		require.NoError(t, err)
		require.Len(t, docs, 2)
		assert.Equal(t, doc1.ID, docs[0].ID)
		assert.Equal(t, doc2.ID, docs[1].ID)
		assert.Equal(t, models.DocumentKindOutgoingLetter, docs[1].Kind)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
