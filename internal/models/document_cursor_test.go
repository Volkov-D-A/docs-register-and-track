package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDocumentCursorRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, 7, 21, 10, 11, 12, 123456789, time.FixedZone("UTC+5", 5*60*60))
	id := uuid.New()

	encoded, err := EncodeDocumentCursor(createdAt, id)
	require.NoError(t, err)
	cursor, err := DecodeDocumentCursor(encoded)
	require.NoError(t, err)
	require.Equal(t, createdAt.UTC(), cursor.CreatedAt)
	require.Equal(t, id, cursor.ID)
}

func TestDecodeDocumentCursorRejectsMalformedValue(t *testing.T) {
	_, err := DecodeDocumentCursor("not-a-cursor")
	require.Error(t, err)
}
