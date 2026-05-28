package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

const (
	numberingModeIndexAndNumber = "index_and_number"
	numberingModeNumberOnly     = "number_only"
	numberingModeManualOnly     = "manual_only"
)

type registrationNumberResult struct {
	Number   string
	Existing uuid.UUID
}

type documentIDLookup interface {
	QueryRow(query string, args ...any) *sql.Row
}

func resolveRegistrationNumberTx(tx *sql.Tx, createdBy uuid.UUID, kind models.DocumentKind, nomenclatureID, idempotencyKey uuid.UUID, requestedNumber string) (*registrationNumberResult, error) {
	if idempotencyKey == uuid.Nil {
		return nil, models.NewBadRequest("отсутствует ключ идемпотентности")
	}

	var existingID uuid.UUID
	err := tx.QueryRow(`
		SELECT id
		FROM documents
		WHERE created_by = $1 AND kind = $2 AND idempotency_key = $3
	`, createdBy, kind, idempotencyKey).Scan(&existingID)
	if err == nil {
		return &registrationNumberResult{Existing: existingID}, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check document idempotency: %w", err)
	}

	var index, separator, numberingMode, kindCode string
	var nextNumber int
	if err := tx.QueryRow(`
		SELECT index, separator, numbering_mode, next_number, kind_code
		FROM nomenclature
		WHERE id = $1
		FOR UPDATE
	`, nomenclatureID).Scan(&index, &separator, &numberingMode, &nextNumber, &kindCode); err != nil {
		if err == sql.ErrNoRows {
			return nil, models.NewBadRequest("номенклатура не найдена")
		}
		return nil, fmt.Errorf("failed to lock nomenclature: %w", err)
	}

	if kindCode != string(kind) {
		return nil, models.NewBadRequest("выберите номенклатуру для нужного вида документа")
	}

	number := strings.TrimSpace(requestedNumber)
	if numberingMode == numberingModeManualOnly {
		if number == "" {
			return nil, models.NewBadRequest("укажите регистрационный номер вручную")
		}
		return &registrationNumberResult{Number: number}, nil
	}

	number = formatDocumentNumber(index, separator, numberingMode, nextNumber)
	if _, err := tx.Exec(`
		UPDATE nomenclature
		SET next_number = next_number + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, nomenclatureID); err != nil {
		return nil, fmt.Errorf("failed to increment nomenclature number: %w", err)
	}

	return &registrationNumberResult{Number: number}, nil
}

func formatDocumentNumber(index, separator, numberingMode string, number int) string {
	switch numberingMode {
	case numberingModeManualOnly:
		return ""
	case numberingModeNumberOnly:
		return fmt.Sprintf("%d", number)
	default:
		sep := separator
		if sep == "" {
			sep = "-"
		}
		return fmt.Sprintf("%s%s%d", index, sep, number)
	}
}

func isUniqueViolation(err error, constraint string) bool {
	var pqErr *pq.Error
	if err == nil || !errors.As(err, &pqErr) {
		return false
	}
	if pqErr.Code != "23505" {
		return false
	}
	return constraint == "" || pqErr.Constraint == constraint
}

func findExistingDocumentIDByIdempotency(db documentIDLookup, createdBy uuid.UUID, kind models.DocumentKind, idempotencyKey uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := db.QueryRow(`
		SELECT id
		FROM documents
		WHERE created_by = $1 AND kind = $2 AND idempotency_key = $3
	`, createdBy, kind, idempotencyKey).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}
