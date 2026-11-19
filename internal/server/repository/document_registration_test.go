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

func TestResolveRegistrationNumberTx(t *testing.T) {
	t.Run("nomenclature not found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		nomenclatureID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT index, separator, numbering_mode, next_number, kind_code, is_active\s+FROM nomenclature\s+WHERE id = \$1\s+FOR UPDATE`).
			WithArgs(nomenclatureID).
			WillReturnError(sql.ErrNoRows)

		mock.ExpectRollback()
		tx, err := db.Begin()
		require.NoError(t, err)
		result, err := resolveRegistrationNumberTx(tx, models.DocumentKindIncomingLetter, nomenclatureID, "")
		require.Error(t, err)
		assert.Empty(t, result)
		assert.Contains(t, err.Error(), "номенклатура не найдена")
		require.NoError(t, tx.Rollback())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("nomenclature kind mismatch", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		nomenclatureID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT index, separator, numbering_mode, next_number, kind_code, is_active\s+FROM nomenclature\s+WHERE id = \$1\s+FOR UPDATE`).
			WithArgs(nomenclatureID).
			WillReturnRows(sqlmock.NewRows([]string{"index", "separator", "numbering_mode", "next_number", "kind_code", "is_active"}).
				AddRow("01-01", "/", numberingModeManualOnly, 1, string(models.DocumentKindOutgoingLetter), true))

		mock.ExpectRollback()
		tx, err := db.Begin()
		require.NoError(t, err)
		result, err := resolveRegistrationNumberTx(tx, models.DocumentKindIncomingLetter, nomenclatureID, "ВХ-1")
		require.Error(t, err)
		assert.Empty(t, result)
		assert.Contains(t, err.Error(), "выберите номенклатуру")
		require.NoError(t, tx.Rollback())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("manual numbering requires requested number", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		nomenclatureID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT index, separator, numbering_mode, next_number, kind_code, is_active\s+FROM nomenclature\s+WHERE id = \$1\s+FOR UPDATE`).
			WithArgs(nomenclatureID).
			WillReturnRows(sqlmock.NewRows([]string{"index", "separator", "numbering_mode", "next_number", "kind_code", "is_active"}).
				AddRow("01-01", "/", numberingModeManualOnly, 1, string(models.DocumentKindIncomingLetter), true))

		mock.ExpectRollback()
		tx, err := db.Begin()
		require.NoError(t, err)
		result, err := resolveRegistrationNumberTx(tx, models.DocumentKindIncomingLetter, nomenclatureID, "   ")
		require.Error(t, err)
		assert.Empty(t, result)
		assert.Contains(t, err.Error(), "укажите регистрационный номер")
		require.NoError(t, tx.Rollback())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("auto numbering increments nomenclature", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		nomenclatureID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT index, separator, numbering_mode, next_number, kind_code, is_active\s+FROM nomenclature\s+WHERE id = \$1\s+FOR UPDATE`).
			WithArgs(nomenclatureID).
			WillReturnRows(sqlmock.NewRows([]string{"index", "separator", "numbering_mode", "next_number", "kind_code", "is_active"}).
				AddRow("01-01", "/", numberingModeIndexAndNumber, 7, string(models.DocumentKindIncomingLetter), true))
		mock.ExpectExec(`UPDATE nomenclature\s+SET next_number = next_number \+ 1\s+WHERE id = \$1`).
			WithArgs(nomenclatureID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectRollback()
		tx, err := db.Begin()
		require.NoError(t, err)
		result, err := resolveRegistrationNumberTx(tx, models.DocumentKindIncomingLetter, nomenclatureID, "")
		require.NoError(t, err)
		assert.Equal(t, "01-01/7", result)

		require.NoError(t, tx.Rollback())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("auto numbering increment error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		nomenclatureID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT index, separator, numbering_mode, next_number, kind_code, is_active\s+FROM nomenclature\s+WHERE id = \$1\s+FOR UPDATE`).
			WithArgs(nomenclatureID).
			WillReturnRows(sqlmock.NewRows([]string{"index", "separator", "numbering_mode", "next_number", "kind_code", "is_active"}).
				AddRow("01-01", "/", numberingModeNumberOnly, 7, string(models.DocumentKindIncomingLetter), true))
		mock.ExpectExec(`UPDATE nomenclature\s+SET next_number = next_number \+ 1\s+WHERE id = \$1`).
			WithArgs(nomenclatureID).
			WillReturnError(sql.ErrConnDone)

		mock.ExpectRollback()
		tx, err := db.Begin()
		require.NoError(t, err)
		result, err := resolveRegistrationNumberTx(tx, models.DocumentKindIncomingLetter, nomenclatureID, "")
		require.Error(t, err)
		assert.Empty(t, result)
		assert.Contains(t, err.Error(), "failed to increment nomenclature number")
		require.NoError(t, tx.Rollback())
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestResolveAdminRegistrationNumberTx(t *testing.T) {
	t.Run("insert shift shifts existing numbers and updates next number", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		nomenclatureID := uuid.New()
		doc15 := uuid.New()
		doc30 := uuid.New()

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT index, separator, numbering_mode, next_number, kind_code, is_active\s+FROM nomenclature\s+WHERE id = \$1\s+FOR UPDATE`).
			WithArgs(nomenclatureID).
			WillReturnRows(sqlmock.NewRows([]string{"index", "separator", "numbering_mode", "next_number", "kind_code", "is_active"}).
				AddRow("26-01-27", "/", numberingModeIndexAndNumber, 31, string(models.DocumentKindIncomingLetter), true))
		mock.ExpectQuery(`SELECT id, registration_number\s+FROM documents\s+WHERE kind = \$1\s+AND nomenclature_id = \$2\s+AND EXTRACT\(YEAR FROM registration_date\) = \(SELECT year FROM nomenclature WHERE id = \$2\)\s+ORDER BY registration_number DESC\s+FOR UPDATE`).
			WithArgs(models.DocumentKindIncomingLetter, nomenclatureID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "registration_number"}).
				AddRow(doc30, "26-01-27/30").
				AddRow(doc15, "26-01-27/15").
				AddRow(uuid.New(), "26-01-27/15А").
				AddRow(uuid.New(), "26-01-27/14"))
		mock.ExpectExec(`UPDATE documents\s+SET registration_number = \$1, updated_at = CURRENT_TIMESTAMP\s+WHERE id = \$2`).
			WithArgs("26-01-27/31", doc30).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE documents\s+SET registration_number = \$1, updated_at = CURRENT_TIMESTAMP\s+WHERE id = \$2`).
			WithArgs("26-01-27/16", doc15).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE nomenclature\s+SET next_number = \$2\s+WHERE id = \$1`).
			WithArgs(nomenclatureID, 32).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectRollback()
		tx, err := db.Begin()
		require.NoError(t, err)
		result, err := resolveAdminRegistrationNumberTx(tx, models.DocumentKindIncomingLetter, nomenclatureID, &models.AdminNumberOverride{
			Mode:   models.AdminNumberModeInsertShift,
			Number: 15,
		})
		require.NoError(t, err)
		assert.Equal(t, "26-01-27/15", result)

		require.NoError(t, tx.Rollback())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("literal number does not shift existing numbers", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		nomenclatureID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT index, separator, numbering_mode, next_number, kind_code, is_active\s+FROM nomenclature\s+WHERE id = \$1\s+FOR UPDATE`).
			WithArgs(nomenclatureID).
			WillReturnRows(sqlmock.NewRows([]string{"index", "separator", "numbering_mode", "next_number", "kind_code", "is_active"}).
				AddRow("26-01-27", "/", numberingModeIndexAndNumber, 31, string(models.DocumentKindIncomingLetter), true))

		mock.ExpectRollback()
		tx, err := db.Begin()
		require.NoError(t, err)
		result, err := resolveAdminRegistrationNumberTx(tx, models.DocumentKindIncomingLetter, nomenclatureID, &models.AdminNumberOverride{
			Mode:   models.AdminNumberModeLiteral,
			Number: 15,
			Suffix: "А",
		})
		require.NoError(t, err)
		assert.Equal(t, "26-01-27/15А", result)

		require.NoError(t, tx.Rollback())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rejects inactive nomenclature", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		nomenclatureID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT index, separator, numbering_mode, next_number, kind_code, is_active\s+FROM nomenclature\s+WHERE id = \$1\s+FOR UPDATE`).
			WithArgs(nomenclatureID).
			WillReturnRows(sqlmock.NewRows([]string{"index", "separator", "numbering_mode", "next_number", "kind_code", "is_active"}).
				AddRow("26-01-27", "/", numberingModeIndexAndNumber, 31, string(models.DocumentKindIncomingLetter), false))

		mock.ExpectRollback()
		tx, err := db.Begin()
		require.NoError(t, err)
		result, err := resolveAdminRegistrationNumberTx(tx, models.DocumentKindIncomingLetter, nomenclatureID, &models.AdminNumberOverride{
			Mode:   models.AdminNumberModeLiteral,
			Number: 15,
			Suffix: "А",
		})
		require.Error(t, err)
		assert.Empty(t, result)
		assert.Contains(t, err.Error(), "действующее дело номенклатуры")
		require.NoError(t, tx.Rollback())
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
