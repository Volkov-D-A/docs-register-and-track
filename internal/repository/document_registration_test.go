package repository

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatDocumentNumber(t *testing.T) {
	assert.Equal(t, "7", formatDocumentNumber("IDX", "/", numberingModeNumberOnly, 7))
	assert.Equal(t, "", formatDocumentNumber("IDX", "/", numberingModeManualOnly, 7))
	assert.Equal(t, "IDX/7", formatDocumentNumber("IDX", "/", numberingModeIndexAndNumber, 7))
	assert.Equal(t, "IDX-7", formatDocumentNumber("IDX", "", numberingModeIndexAndNumber, 7))
	assert.Equal(t, "IDX-7", formatDocumentNumber("IDX", "", "unknown", 7))
}

func TestIsUniqueViolation(t *testing.T) {
	err := &pq.Error{Code: "23505", Constraint: "documents_number_key"}

	assert.True(t, isUniqueViolation(err, "documents_number_key"))
	assert.True(t, isUniqueViolation(err, ""))
	assert.False(t, isUniqueViolation(err, "other_key"))
	assert.False(t, isUniqueViolation(&pq.Error{Code: "23503", Constraint: "documents_number_key"}, "documents_number_key"))
	assert.False(t, isUniqueViolation(errors.New("plain error"), "documents_number_key"))
	assert.False(t, isUniqueViolation(nil, "documents_number_key"))
}

func TestFindExistingDocumentIDByIdempotency(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		createdBy := uuid.New()
		idempotencyKey := uuid.New()
		docID := uuid.New()

		mock.ExpectQuery(`SELECT id\s+FROM documents\s+WHERE created_by = \$1 AND kind = \$2 AND idempotency_key = \$3`).
			WithArgs(createdBy, models.DocumentKindIncomingLetter, idempotencyKey).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(docID))

		result, err := findExistingDocumentIDByIdempotency(db, createdBy, models.DocumentKindIncomingLetter, idempotencyKey)
		require.NoError(t, err)
		assert.Equal(t, docID, result)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		createdBy := uuid.New()
		idempotencyKey := uuid.New()

		mock.ExpectQuery(`SELECT id\s+FROM documents\s+WHERE created_by = \$1 AND kind = \$2 AND idempotency_key = \$3`).
			WithArgs(createdBy, models.DocumentKindOutgoingLetter, idempotencyKey).
			WillReturnError(sql.ErrNoRows)

		result, err := findExistingDocumentIDByIdempotency(db, createdBy, models.DocumentKindOutgoingLetter, idempotencyKey)
		require.ErrorIs(t, err, sql.ErrNoRows)
		assert.Equal(t, uuid.Nil, result)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
