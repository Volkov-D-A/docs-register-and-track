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

func TestResolveRegistrationNumberTx(t *testing.T) {
	t.Run("returns existing document for idempotency key", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		createdBy := uuid.New()
		idempotencyKey := uuid.New()
		existingID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT id\s+FROM documents\s+WHERE created_by = \$1 AND kind = \$2 AND idempotency_key = \$3`).
			WithArgs(createdBy, models.DocumentKindIncomingLetter, idempotencyKey).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(existingID))

		mock.ExpectRollback()
		tx, err := db.Begin()
		require.NoError(t, err)
		result, err := resolveRegistrationNumberTx(tx, createdBy, models.DocumentKindIncomingLetter, uuid.New(), idempotencyKey, "")
		require.NoError(t, err)
		assert.Equal(t, existingID, result.Existing)
		assert.Empty(t, result.Number)
		require.NoError(t, tx.Rollback())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("idempotency check database error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		createdBy := uuid.New()
		idempotencyKey := uuid.New()

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT id\s+FROM documents\s+WHERE created_by = \$1 AND kind = \$2 AND idempotency_key = \$3`).
			WithArgs(createdBy, models.DocumentKindIncomingLetter, idempotencyKey).
			WillReturnError(sql.ErrConnDone)

		mock.ExpectRollback()
		tx, err := db.Begin()
		require.NoError(t, err)
		result, err := resolveRegistrationNumberTx(tx, createdBy, models.DocumentKindIncomingLetter, uuid.New(), idempotencyKey, "")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to check document idempotency")
		require.NoError(t, tx.Rollback())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("nomenclature not found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		createdBy := uuid.New()
		idempotencyKey := uuid.New()
		nomenclatureID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT id\s+FROM documents\s+WHERE created_by = \$1 AND kind = \$2 AND idempotency_key = \$3`).
			WithArgs(createdBy, models.DocumentKindIncomingLetter, idempotencyKey).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`SELECT index, separator, numbering_mode, next_number, kind_code\s+FROM nomenclature\s+WHERE id = \$1\s+FOR UPDATE`).
			WithArgs(nomenclatureID).
			WillReturnError(sql.ErrNoRows)

		mock.ExpectRollback()
		tx, err := db.Begin()
		require.NoError(t, err)
		result, err := resolveRegistrationNumberTx(tx, createdBy, models.DocumentKindIncomingLetter, nomenclatureID, idempotencyKey, "")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "номенклатура не найдена")
		require.NoError(t, tx.Rollback())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("nomenclature kind mismatch", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		createdBy := uuid.New()
		idempotencyKey := uuid.New()
		nomenclatureID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT id\s+FROM documents\s+WHERE created_by = \$1 AND kind = \$2 AND idempotency_key = \$3`).
			WithArgs(createdBy, models.DocumentKindIncomingLetter, idempotencyKey).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`SELECT index, separator, numbering_mode, next_number, kind_code\s+FROM nomenclature\s+WHERE id = \$1\s+FOR UPDATE`).
			WithArgs(nomenclatureID).
			WillReturnRows(sqlmock.NewRows([]string{"index", "separator", "numbering_mode", "next_number", "kind_code"}).
				AddRow("01-01", "/", numberingModeManualOnly, 1, string(models.DocumentKindOutgoingLetter)))

		mock.ExpectRollback()
		tx, err := db.Begin()
		require.NoError(t, err)
		result, err := resolveRegistrationNumberTx(tx, createdBy, models.DocumentKindIncomingLetter, nomenclatureID, idempotencyKey, "ВХ-1")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "выберите номенклатуру")
		require.NoError(t, tx.Rollback())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("manual numbering requires requested number", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		createdBy := uuid.New()
		idempotencyKey := uuid.New()
		nomenclatureID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT id\s+FROM documents\s+WHERE created_by = \$1 AND kind = \$2 AND idempotency_key = \$3`).
			WithArgs(createdBy, models.DocumentKindIncomingLetter, idempotencyKey).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`SELECT index, separator, numbering_mode, next_number, kind_code\s+FROM nomenclature\s+WHERE id = \$1\s+FOR UPDATE`).
			WithArgs(nomenclatureID).
			WillReturnRows(sqlmock.NewRows([]string{"index", "separator", "numbering_mode", "next_number", "kind_code"}).
				AddRow("01-01", "/", numberingModeManualOnly, 1, string(models.DocumentKindIncomingLetter)))

		mock.ExpectRollback()
		tx, err := db.Begin()
		require.NoError(t, err)
		result, err := resolveRegistrationNumberTx(tx, createdBy, models.DocumentKindIncomingLetter, nomenclatureID, idempotencyKey, "   ")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "укажите регистрационный номер")
		require.NoError(t, tx.Rollback())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("auto numbering increments nomenclature", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		createdBy := uuid.New()
		idempotencyKey := uuid.New()
		nomenclatureID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT id\s+FROM documents\s+WHERE created_by = \$1 AND kind = \$2 AND idempotency_key = \$3`).
			WithArgs(createdBy, models.DocumentKindIncomingLetter, idempotencyKey).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`SELECT index, separator, numbering_mode, next_number, kind_code\s+FROM nomenclature\s+WHERE id = \$1\s+FOR UPDATE`).
			WithArgs(nomenclatureID).
			WillReturnRows(sqlmock.NewRows([]string{"index", "separator", "numbering_mode", "next_number", "kind_code"}).
				AddRow("01-01", "/", numberingModeIndexAndNumber, 7, string(models.DocumentKindIncomingLetter)))
		mock.ExpectExec(`UPDATE nomenclature\s+SET next_number = next_number \+ 1, updated_at = CURRENT_TIMESTAMP\s+WHERE id = \$1`).
			WithArgs(nomenclatureID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectRollback()
		tx, err := db.Begin()
		require.NoError(t, err)
		result, err := resolveRegistrationNumberTx(tx, createdBy, models.DocumentKindIncomingLetter, nomenclatureID, idempotencyKey, "")
		require.NoError(t, err)
		assert.Equal(t, "01-01/7", result.Number)
		assert.Equal(t, uuid.Nil, result.Existing)
		require.NoError(t, tx.Rollback())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("auto numbering increment error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		createdBy := uuid.New()
		idempotencyKey := uuid.New()
		nomenclatureID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT id\s+FROM documents\s+WHERE created_by = \$1 AND kind = \$2 AND idempotency_key = \$3`).
			WithArgs(createdBy, models.DocumentKindIncomingLetter, idempotencyKey).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`SELECT index, separator, numbering_mode, next_number, kind_code\s+FROM nomenclature\s+WHERE id = \$1\s+FOR UPDATE`).
			WithArgs(nomenclatureID).
			WillReturnRows(sqlmock.NewRows([]string{"index", "separator", "numbering_mode", "next_number", "kind_code"}).
				AddRow("01-01", "/", numberingModeNumberOnly, 7, string(models.DocumentKindIncomingLetter)))
		mock.ExpectExec(`UPDATE nomenclature\s+SET next_number = next_number \+ 1, updated_at = CURRENT_TIMESTAMP\s+WHERE id = \$1`).
			WithArgs(nomenclatureID).
			WillReturnError(sql.ErrConnDone)

		mock.ExpectRollback()
		tx, err := db.Begin()
		require.NoError(t, err)
		result, err := resolveRegistrationNumberTx(tx, createdBy, models.DocumentKindIncomingLetter, nomenclatureID, idempotencyKey, "")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to increment nomenclature number")
		require.NoError(t, tx.Rollback())
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
