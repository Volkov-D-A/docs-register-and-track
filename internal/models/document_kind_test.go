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

	_, ok = GetDocumentKindSpec(DocumentKind("unknown"))
	assert.False(t, ok)
}

func TestDocumentKindSupportsAction(t *testing.T) {
	assert.True(t, DocumentKindIncomingLetter.SupportsAction(string(DocumentActionRead)))
	assert.True(t, DocumentKindAdministrativeOrder.SupportsAction(string(DocumentActionViewJournal)))
	assert.False(t, DocumentKindIncomingLetter.SupportsAction("delete"))
	assert.False(t, DocumentKind("unknown").SupportsAction(string(DocumentActionRead)))
}

func TestDocumentKindLabel(t *testing.T) {

	assert.Equal(t, "Входящее письмо", DocumentKindIncomingLetter.Label())
	assert.Equal(t, "custom", DocumentKind("custom").Label())
}
