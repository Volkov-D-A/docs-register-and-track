package repository

import (
	"fmt"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// applyDocumentCursor adds the continuation predicate for the common stable
// document ordering: created_at DESC, id DESC.
func applyDocumentCursor(where *[]string, args *[]interface{}, argIdx *int, enabled bool, value string) error {
	if !enabled || value == "" {
		return nil
	}
	cursor, err := models.DecodeDocumentCursor(value)
	if err != nil {
		return err
	}
	*where = append(*where, fmt.Sprintf("(d.created_at, d.id) < ($%d, $%d)", *argIdx, *argIdx+1))
	*args = append(*args, cursor.CreatedAt, cursor.ID)
	*argIdx += 2
	return nil
}
