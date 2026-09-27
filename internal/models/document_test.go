package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDocumentTypeValidation(t *testing.T) {
	assert.True(t, IsAllowedDocumentType(DocumentTypeLetter))
	assert.True(t, IsAllowedDocumentType(DocumentTypeAdministrativeOrder))
	assert.Equal(t, "Письмо", NormalizeDocumentType("  Письмо  "))
	assert.True(t, IsAllowedDocumentType(" Письмо "))
	assert.False(t, IsAllowedDocumentType("Неизвестный тип"))
}
