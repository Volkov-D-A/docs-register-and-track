package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDocumentTypes(t *testing.T) {
	types := AllowedDocumentTypes()

	assert.Contains(t, types, DocumentTypeLetter)
	assert.Contains(t, types, DocumentTypeAdministrativeOrder)
	assert.Equal(t, "Письмо", NormalizeDocumentType("  Письмо  "))
	assert.True(t, IsAllowedDocumentType(" Письмо "))
	assert.False(t, IsAllowedDocumentType("Неизвестный тип"))
}
