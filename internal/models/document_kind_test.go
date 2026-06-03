package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentKindSpecs(t *testing.T) {
	specs := AllDocumentKindSpecs()
	require.Len(t, specs, 4)

	specs[0].Name = "mutated"
	freshSpecs := AllDocumentKindSpecs()
	assert.NotEqual(t, "mutated", freshSpecs[0].Name)

	spec, ok := GetDocumentKindSpec(DocumentKindIncomingLetter)
	require.True(t, ok)
	assert.Equal(t, "Входящее письмо", spec.Name)
	assert.Equal(t, "incoming_letter_form", spec.RegistrationFormCode)

	_, ok = GetDocumentKindSpec(DocumentKind("unknown"))
	assert.False(t, ok)
}

func TestDocumentKindSupportsAction(t *testing.T) {
	assert.True(t, DocumentKindIncomingLetter.SupportsAction(string(DocumentActionRead)))
	assert.True(t, DocumentKindAdministrativeOrder.SupportsAction(string(DocumentActionViewJournal)))
	assert.False(t, DocumentKindIncomingLetter.SupportsAction("delete"))
	assert.False(t, DocumentKind("unknown").SupportsAction(string(DocumentActionRead)))
}

func TestNormalizeDocumentKindAndLabel(t *testing.T) {
	assert.Equal(t, DocumentKindIncomingLetter, NormalizeDocumentKind("incoming"))
	assert.Equal(t, DocumentKindIncomingLetter, NormalizeDocumentKind(string(DocumentKindIncomingLetter)))
	assert.Equal(t, DocumentKindOutgoingLetter, NormalizeDocumentKind("outgoing"))
	assert.Equal(t, DocumentKindOutgoingLetter, NormalizeDocumentKind(string(DocumentKindOutgoingLetter)))
	assert.Equal(t, DocumentKindCitizenAppeal, NormalizeDocumentKind(string(DocumentKindCitizenAppeal)))
	assert.Equal(t, DocumentKindAdministrativeOrder, NormalizeDocumentKind(string(DocumentKindAdministrativeOrder)))
	assert.Equal(t, DocumentKind("custom"), NormalizeDocumentKind("custom"))

	assert.Equal(t, "Входящее письмо", DocumentKindIncomingLetter.Label())
	assert.Equal(t, "custom", DocumentKind("custom").Label())
}
