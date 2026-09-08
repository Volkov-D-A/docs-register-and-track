package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestDocumentCommandTransportNormalization(t *testing.T) {
	for _, tc := range []struct {
		kind             models.DocumentKind
		register, update any
	}{
		{models.DocumentKindIncomingLetter, IncomingLetterRegisterRequest{NomenclatureID: "nom", IdempotencyKey: "key", Correspondents: []IncomingLetterCorrespondentRequest{{RegistrationNumber: "external"}}, AdminNumberOverride: &AdminNumberOverrideRequest{Mode: "literal", Number: 3, Suffix: "a"}}, IncomingLetterUpdateRequest{ID: "doc", Content: "changed"}},
		{models.DocumentKindOutgoingLetter, OutgoingLetterRegisterRequest{NomenclatureID: "nom", IdempotencyKey: "key", RecipientOrgName: "recipient"}, OutgoingLetterUpdateRequest{ID: "doc", Content: "changed"}},
		{models.DocumentKindCitizenAppeal, CitizenAppealRegisterRequest{NomenclatureID: "nom", IdempotencyKey: "key", Correspondents: []CitizenAppealCorrespondentRequest{{CorrespondentName: "sender"}}, Resolutions: []CitizenAppealResolutionRequest{{Resolution: "reply"}}, HasEnvelope: true, ReceivedFromPOS: true}, CitizenAppealUpdateRequest{ID: "doc", Content: "changed"}},
		{models.DocumentKindAdministrativeOrder, AdministrativeOrderRegisterRequest{NomenclatureID: "nom", IdempotencyKey: "key", AcknowledgmentFullNames: []string{"person"}, IsActive: true}, AdministrativeOrderUpdateRequest{ID: "doc", Title: "changed"}},
	} {
		t.Run(string(tc.kind), func(t *testing.T) {
			for _, operation := range []struct {
				name      string
				request   any
				normalize func(models.DocumentKind, any) (any, error)
			}{
				{"register", tc.register, NormalizeDocumentRegisterRequest}, {"update", tc.update, NormalizeDocumentUpdateRequest},
			} {
				t.Run(operation.name, func(t *testing.T) {
					encoded, err := json.Marshal(operation.request)
					require.NoError(t, err)
					var input map[string]any
					require.NoError(t, json.Unmarshal(encoded, &input))
					actual, err := operation.normalize(tc.kind, input)
					require.NoError(t, err)
					require.Equal(t, operation.request, actual)
					actual, err = operation.normalize(tc.kind, operation.request)
					require.NoError(t, err)
					require.Equal(t, operation.request, actual)
					input["injectedPermission"] = "admin"
					_, err = operation.normalize(tc.kind, input)
					require.Error(t, err)
				})
			}
		})
	}
}

func TestDocumentCommandNormalizationDoesNotReplaceServerValidation(t *testing.T) {
	request, err := NormalizeDocumentRegisterRequest(models.DocumentKindIncomingLetter, map[string]any{"nomenclatureId": "invalid", "pagesCount": -1, "adminNumberOverride": map[string]any{"mode": "invalid", "number": -2}})
	require.NoError(t, err)
	require.Equal(t, -1, request.(IncomingLetterRegisterRequest).PagesCount)
	for _, normalize := range []func(models.DocumentKind, any) (any, error){NormalizeDocumentRegisterRequest, NormalizeDocumentUpdateRequest} {
		_, err := normalize(models.DocumentKind("unknown"), map[string]any{})
		require.ErrorIs(t, err, models.ErrForbidden)
		_, err = normalize(models.DocumentKindIncomingLetter, map[string]any{"pagesCount": "wrong type"})
		require.Error(t, err)
		_, err = normalize(models.DocumentKindIncomingLetter, make(chan int))
		require.Error(t, err)
	}
	_, err = NormalizeDocumentRegisterRequest(models.DocumentKindIncomingLetter, map[string]any{"adminNumberOverride": map[string]any{"unexpected": true}})
	require.Error(t, err)
}
