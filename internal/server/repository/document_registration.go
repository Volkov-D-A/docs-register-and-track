package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"
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
			return nil, models.NewNotFound("номенклатура не найдена")
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

func resolveAdminRegistrationNumberTx(tx *sql.Tx, createdBy uuid.UUID, kind models.DocumentKind, nomenclatureID, idempotencyKey uuid.UUID, override *models.AdminNumberOverride) (*registrationNumberResult, error) {
	if override == nil {
		return resolveRegistrationNumberTx(tx, createdBy, kind, nomenclatureID, idempotencyKey, "")
	}
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
	var isActive bool
	if err := tx.QueryRow(`
		SELECT index, separator, numbering_mode, next_number, kind_code, is_active
		FROM nomenclature
		WHERE id = $1
		FOR UPDATE
	`, nomenclatureID).Scan(&index, &separator, &numberingMode, &nextNumber, &kindCode, &isActive); err != nil {
		if err == sql.ErrNoRows {
			return nil, models.NewNotFound("номенклатура не найдена")
		}
		return nil, fmt.Errorf("failed to lock nomenclature: %w", err)
	}
	if kindCode != string(kind) {
		return nil, models.NewBadRequest("выберите номенклатуру для нужного вида документа")
	}
	if !isActive {
		return nil, models.NewBadRequest("выберите действующее дело номенклатуры")
	}
	if override.Number < 1 {
		return nil, models.NewBadRequest("укажите номер документа больше 0")
	}

	switch override.Mode {
	case models.AdminNumberModeInsertShift:
		if strings.TrimSpace(override.Suffix) != "" {
			return nil, models.NewBadRequest("для вставки со сдвигом укажите номер без литеры")
		}
		if err := shiftRegistrationNumbersTx(tx, kind, nomenclatureID, index, separator, numberingMode, override.Number); err != nil {
			return nil, err
		}
		if override.Number >= nextNumber {
			nextNumber = override.Number + 1
		} else {
			nextNumber++
		}
		if _, err := tx.Exec(`
			UPDATE nomenclature
			SET next_number = $2, updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
		`, nomenclatureID, nextNumber); err != nil {
			return nil, fmt.Errorf("failed to update nomenclature number after admin insert: %w", err)
		}
	case models.AdminNumberModeLiteral:
		// Литерный номер не меняет счетчик и существующие документы.
	default:
		return nil, models.NewBadRequest("неверный режим административной нумерации")
	}

	number := formatDocumentNumberWithSuffix(index, separator, numberingMode, override.Number, strings.TrimSpace(override.Suffix))
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

func formatDocumentNumberWithSuffix(index, separator, numberingMode string, number int, suffix string) string {
	switch numberingMode {
	case numberingModeManualOnly:
		return fmt.Sprintf("%d%s", number, suffix)
	case numberingModeNumberOnly:
		return fmt.Sprintf("%d%s", number, suffix)
	default:
		sep := separator
		if sep == "" {
			sep = "-"
		}
		return fmt.Sprintf("%s%s%d%s", index, sep, number, suffix)
	}
}

func shiftRegistrationNumbersTx(tx *sql.Tx, kind models.DocumentKind, nomenclatureID uuid.UUID, index, separator, numberingMode string, startNumber int) error {
	rows, err := tx.Query(`
		SELECT id, registration_number
		FROM documents
		WHERE kind = $1
			AND nomenclature_id = $2
			AND EXTRACT(YEAR FROM registration_date) = (SELECT year FROM nomenclature WHERE id = $2)
		ORDER BY registration_number DESC
		FOR UPDATE
	`, kind, nomenclatureID)
	if err != nil {
		return fmt.Errorf("failed to lock documents for number shift: %w", err)
	}
	defer rows.Close()

	type numberedDocument struct {
		id     uuid.UUID
		number int
	}
	docs := make([]numberedDocument, 0)
	for rows.Next() {
		var id uuid.UUID
		var registrationNumber string
		if err := rows.Scan(&id, &registrationNumber); err != nil {
			return fmt.Errorf("failed to scan document for number shift: %w", err)
		}
		number, ok := parseFormattedDocumentNumber(index, separator, numberingMode, registrationNumber)
		if ok && number >= startNumber {
			docs = append(docs, numberedDocument{id: id, number: number})
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("number shift rows error: %w", err)
	}

	sort.Slice(docs, func(i, j int) bool {
		return docs[i].number > docs[j].number
	})
	for _, doc := range docs {
		newNumber := formatDocumentNumber(index, separator, numberingMode, doc.number+1)
		if err := updateDocumentRegistrationNumberTx(tx, kind, doc.id, newNumber); err != nil {
			return err
		}
	}
	return nil
}

func parseFormattedDocumentNumber(index, separator, numberingMode, registrationNumber string) (int, bool) {
	value := strings.TrimSpace(registrationNumber)
	switch numberingMode {
	case numberingModeNumberOnly, numberingModeManualOnly:
		number, err := strconv.Atoi(value)
		return number, err == nil
	default:
		sep := separator
		if sep == "" {
			sep = "-"
		}
		prefix := index + sep
		if !strings.HasPrefix(value, prefix) {
			return 0, false
		}
		number, err := strconv.Atoi(strings.TrimPrefix(value, prefix))
		return number, err == nil
	}
}

func updateDocumentRegistrationNumberTx(tx *sql.Tx, kind models.DocumentKind, id uuid.UUID, number string) error {
	if _, err := tx.Exec(`
		UPDATE documents
		SET registration_number = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, number, id); err != nil {
		return fmt.Errorf("failed to shift document registration number: %w", err)
	}

	switch kind {
	case models.DocumentKindIncomingLetter:
		_, err := tx.Exec(`UPDATE incoming_document_details SET incoming_number = $1 WHERE document_id = $2`, number, id)
		if err != nil {
			return fmt.Errorf("failed to shift incoming document number: %w", err)
		}
	case models.DocumentKindOutgoingLetter:
		_, err := tx.Exec(`UPDATE outgoing_document_details SET outgoing_number = $1 WHERE document_id = $2`, number, id)
		if err != nil {
			return fmt.Errorf("failed to shift outgoing document number: %w", err)
		}
	case models.DocumentKindAdministrativeOrder:
		_, err := tx.Exec(`UPDATE administrative_order_details SET order_number = $1 WHERE document_id = $2`, number, id)
		if err != nil {
			return fmt.Errorf("failed to shift administrative order number: %w", err)
		}
	}
	return nil
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
