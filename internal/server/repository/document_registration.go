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

func resolveRegistrationNumberTx(tx *sql.Tx, kind models.DocumentKind, nomenclatureID uuid.UUID, requestedNumber string) (string, error) {
	var index, separator, numberingMode, kindCode string
	var nextNumber int
	if err := tx.QueryRow(`
		SELECT index, separator, numbering_mode, next_number, kind_code
		FROM nomenclature
		WHERE id = $1
		FOR UPDATE
	`, nomenclatureID).Scan(&index, &separator, &numberingMode, &nextNumber, &kindCode); err != nil {
		if err == sql.ErrNoRows {
			return "", models.NewNotFound("номенклатура не найдена")
		}
		return "", fmt.Errorf("failed to lock nomenclature: %w", err)
	}

	if kindCode != string(kind) {
		return "", models.NewBadRequest("выберите номенклатуру для нужного вида документа")
	}

	number := strings.TrimSpace(requestedNumber)
	if numberingMode == numberingModeManualOnly {
		if number == "" {
			return "", models.NewBadRequest("укажите регистрационный номер вручную")
		}
		return number, nil
	}

	number = formatDocumentNumber(index, separator, numberingMode, nextNumber)
	if _, err := tx.Exec(`
		UPDATE nomenclature
		SET next_number = next_number + 1
		WHERE id = $1
	`, nomenclatureID); err != nil {
		return "", fmt.Errorf("failed to increment nomenclature number: %w", err)
	}

	return number, nil
}

func resolveAdminRegistrationNumberTx(tx *sql.Tx, kind models.DocumentKind, nomenclatureID uuid.UUID, override *models.AdminNumberOverride) (string, error) {
	if override == nil {
		return resolveRegistrationNumberTx(tx, kind, nomenclatureID, "")
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
			return "", models.NewNotFound("номенклатура не найдена")
		}
		return "", fmt.Errorf("failed to lock nomenclature: %w", err)
	}
	if kindCode != string(kind) {
		return "", models.NewBadRequest("выберите номенклатуру для нужного вида документа")
	}
	if !isActive {
		return "", models.NewBadRequest("выберите действующее дело номенклатуры")
	}
	if override.Number < 1 {
		return "", models.NewBadRequest("укажите номер документа больше 0")
	}

	switch override.Mode {
	case models.AdminNumberModeInsertShift:
		if strings.TrimSpace(override.Suffix) != "" {
			return "", models.NewBadRequest("для вставки со сдвигом укажите номер без литеры")
		}
		if err := shiftRegistrationNumbersTx(tx, kind, nomenclatureID, index, separator, numberingMode, override.Number); err != nil {
			return "", err
		}
		if override.Number >= nextNumber {
			nextNumber = override.Number + 1
		} else {
			nextNumber++
		}
		if _, err := tx.Exec(`
			UPDATE nomenclature
			SET next_number = $2
			WHERE id = $1
		`, nomenclatureID, nextNumber); err != nil {
			return "", fmt.Errorf("failed to update nomenclature number after admin insert: %w", err)
		}
	case models.AdminNumberModeLiteral:
		// Литерный номер не меняет счетчик и существующие документы.
	default:
		return "", models.NewBadRequest("неверный режим административной нумерации")
	}

	number := formatDocumentNumberWithSuffix(index, separator, numberingMode, override.Number, strings.TrimSpace(override.Suffix))
	return number, nil
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
		if err := updateDocumentRegistrationNumberTx(tx, doc.id, newNumber); err != nil {
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

func updateDocumentRegistrationNumberTx(tx *sql.Tx, id uuid.UUID, number string) error {
	if _, err := tx.Exec(`
		UPDATE documents
		SET registration_number = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, number, id); err != nil {
		return fmt.Errorf("failed to shift document registration number: %w", err)
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
