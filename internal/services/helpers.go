package services

import (
	"fmt"

	"github.com/google/uuid"
)

// parseUUID — парсинг строки UUID с обработкой ошибок
func parseUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid UUID: %s", s)
	}
	return id, nil
}

// formatDocumentNumber — форматирование номера документа
func formatDocumentNumber(index string, number int) string {
	return fmt.Sprintf("%s/%d", index, number)
}
