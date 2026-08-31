package repository

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestReserveDocumentCommandRejectsChangedPayload(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectBegin()
	tx, err := db.Begin()
	require.NoError(t, err)
	principalID, key, documentID := uuid.New(), uuid.New(), uuid.New()
	mock.ExpectQuery("SELECT request_hash, document_id").
		WithArgs(principalID, "documents.update:incoming_letter", key).
		WillReturnRows(sqlmock.NewRows([]string{"request_hash", "document_id"}).AddRow("old-hash", documentID))
	mock.ExpectRollback()

	_, err = reserveDocumentCommandTx(tx, principalID, "documents.update:incoming_letter", key, "new-hash")
	require.Error(t, err)
	appErr, ok := models.AsAppError(err)
	require.True(t, ok)
	require.Equal(t, 409, appErr.StatusCode())
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReserveAndCompleteDocumentCommand(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectBegin()
	tx, err := db.Begin()
	require.NoError(t, err)
	principalID, key, documentID := uuid.New(), uuid.New(), uuid.New()
	operation := "documents.register:incoming_letter"
	mock.ExpectQuery("SELECT request_hash, document_id").WithArgs(principalID, operation, key).WillReturnError(sql.ErrNoRows)
	mock.ExpectExec("INSERT INTO document_command_idempotency").WithArgs(principalID, operation, key, "hash").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE document_command_idempotency").WithArgs(principalID, operation, key, "hash", documentID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectRollback()

	existing, err := reserveDocumentCommandTx(tx, principalID, operation, key, "hash")
	require.NoError(t, err)
	require.Equal(t, uuid.Nil, existing)
	require.NoError(t, completeDocumentCommandTx(tx, principalID, operation, key, "hash", documentID))
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}
