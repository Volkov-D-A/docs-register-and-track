package models

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// DocumentCursor is an opaque continuation token for document lists ordered by
// created_at DESC, id DESC.
type DocumentCursor struct {
	CreatedAt time.Time `json:"createdAt"`
	ID        uuid.UUID `json:"id"`
}

// EncodeDocumentCursor serializes a continuation position without exposing a
// SQL-specific representation to API callers.
func EncodeDocumentCursor(createdAt time.Time, id uuid.UUID) (string, error) {
	payload, err := json.Marshal(DocumentCursor{CreatedAt: createdAt.UTC(), ID: id})
	if err != nil {
		return "", fmt.Errorf("encode document cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

// DecodeDocumentCursor validates and restores a continuation position.
func DecodeDocumentCursor(value string) (DocumentCursor, error) {
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return DocumentCursor{}, NewBadRequestWrapped("неверный курсор списка документов", err)
	}
	var cursor DocumentCursor
	if err := json.Unmarshal(payload, &cursor); err != nil || cursor.CreatedAt.IsZero() || cursor.ID == uuid.Nil {
		if err == nil {
			err = fmt.Errorf("cursor has empty position")
		}
		return DocumentCursor{}, NewBadRequestWrapped("неверный курсор списка документов", err)
	}
	return cursor, nil
}
