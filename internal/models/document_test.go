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

func TestDocumentKindPredicates(t *testing.T) {
	assert.True(t, DocumentKindIncomingLetter.IsIncoming())
	assert.False(t, DocumentKindIncomingLetter.IsOutgoing())

	assert.True(t, DocumentKindOutgoingLetter.IsOutgoing())
	assert.False(t, DocumentKindOutgoingLetter.IsCitizenAppeal())

	assert.True(t, DocumentKindCitizenAppeal.IsCitizenAppeal())
	assert.False(t, DocumentKindCitizenAppeal.IsAdministrativeOrder())

	assert.True(t, DocumentKindAdministrativeOrder.IsAdministrativeOrder())
	assert.False(t, DocumentKindAdministrativeOrder.IsIncoming())
}
