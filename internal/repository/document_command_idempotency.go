package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func reserveDocumentCommandTx(tx *sql.Tx, principalID uuid.UUID, operation string, key uuid.UUID, requestHash string) (uuid.UUID, error) {
	if requestHash == "" {
		return uuid.Nil, nil
	}
	if principalID == uuid.Nil || key == uuid.Nil || operation == "" {
		return uuid.Nil, models.NewBadRequest("неверный ключ идемпотентности команды")
	}

	var storedHash string
	var documentID *uuid.UUID
	err := tx.QueryRow(`
		SELECT request_hash, document_id
		FROM document_command_idempotency
		WHERE principal_id = $1 AND operation = $2 AND idempotency_key = $3
		FOR UPDATE
	`, principalID, operation, key).Scan(&storedHash, &documentID)
	if err == nil {
		if storedHash != requestHash {
			return uuid.Nil, models.NewConflict("ключ идемпотентности уже использован для другой команды")
		}
		if documentID == nil {
			return uuid.Nil, fmt.Errorf("idempotent command has no document result")
		}
		return *documentID, nil
	}
	if err != sql.ErrNoRows {
		return uuid.Nil, fmt.Errorf("read document command idempotency: %w", err)
	}
	result, err := tx.Exec(`
		INSERT INTO document_command_idempotency (principal_id, operation, idempotency_key, request_hash)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (principal_id, operation, idempotency_key) DO NOTHING
	`, principalID, operation, key, requestHash)
	if err != nil {
		return uuid.Nil, fmt.Errorf("reserve document command idempotency: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return uuid.Nil, fmt.Errorf("read document command reservation result: %w", err)
	}
	if rows == 0 {
		err := tx.QueryRow(`
			SELECT request_hash, document_id
			FROM document_command_idempotency
			WHERE principal_id = $1 AND operation = $2 AND idempotency_key = $3
			FOR UPDATE
		`, principalID, operation, key).Scan(&storedHash, &documentID)
		if err != nil {
			return uuid.Nil, fmt.Errorf("resolve concurrent document command idempotency: %w", err)
		}
		if storedHash != requestHash {
			return uuid.Nil, models.NewConflict("ключ идемпотентности уже использован для другой команды")
		}
		if documentID == nil {
			return uuid.Nil, fmt.Errorf("idempotent command has no document result")
		}
		return *documentID, nil
	}
	return uuid.Nil, nil
}

func completeDocumentCommandTx(tx *sql.Tx, principalID uuid.UUID, operation string, key uuid.UUID, requestHash string, documentID uuid.UUID) error {
	if requestHash == "" {
		return nil
	}
	result, err := tx.Exec(`
		UPDATE document_command_idempotency
		SET document_id = $5
		WHERE principal_id = $1 AND operation = $2 AND idempotency_key = $3 AND request_hash = $4
	`, principalID, operation, key, requestHash, documentID)
	if err != nil {
		return fmt.Errorf("complete document command idempotency: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil || rows != 1 {
		return fmt.Errorf("document command idempotency reservation was not completed")
	}
	return nil
}
