package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestRegistrationLinkRequiresExistingReadAndNewRegistration(t *testing.T) {
	for _, missing := range []string{"none", "existing read", "new registration"} {
		t.Run(missing, func(t *testing.T) {
			allowed := allowDocumentActions(models.DocumentKindOutgoingLetter, "register")
			if missing == "new registration" {
				delete(allowed[models.DocumentKindOutgoingLetter], "register")
			}
			deps := setupDocumentAccessService(t, documentAccessUser(false, nil), allowed)
			id := uuid.New()
			deps.docRepo.docs[id] = documentAccessDoc(id, uuid.New(), models.DocumentKindIncomingLetter)
			deps.accessRepo.readable[id] = missing != "existing read"
			link, err := buildRegistrationLink(deps.service, models.DocumentKindOutgoingLetter, uuid.New(), &dto.DocumentRegistrationLinkRequest{DocumentID: id.String(), LinkType: "reply"})
			if missing == "none" {
				require.NoError(t, err)
				require.Equal(t, id, link.DocumentID)
			} else {
				require.ErrorIs(t, err, models.ErrForbidden)
				require.Nil(t, link)
			}
		})
	}
}

func TestRegistrationLinkRejectsInvalidTypeAndOrderEndpoints(t *testing.T) {
	allowed := allowDocumentActions(models.DocumentKindIncomingLetter, "register")
	addDocumentActions(allowed, models.DocumentKindAdministrativeOrder, "register")
	deps := setupDocumentAccessService(t, documentAccessUser(false, nil), allowed)
	id := uuid.New()
	deps.accessRepo.readable[id] = true
	deps.docRepo.docs[id] = documentAccessDoc(id, uuid.New(), models.DocumentKindIncomingLetter)
	for _, linkType := range []string{"", "unknown", "order_amends", "order_cancels"} {
		_, err := buildRegistrationLink(deps.service, models.DocumentKindAdministrativeOrder, uuid.New(), &dto.DocumentRegistrationLinkRequest{DocumentID: id.String(), LinkType: linkType})
		require.Error(t, err)
	}
	deps.docRepo.docs[id] = documentAccessDoc(id, uuid.New(), models.DocumentKindAdministrativeOrder)
	_, err := buildRegistrationLink(deps.service, models.DocumentKindAdministrativeOrder, uuid.New(), &dto.DocumentRegistrationLinkRequest{DocumentID: id.String(), LinkType: "order_cancels"})
	require.NoError(t, err)
}

func TestRegisterCommandsPreserveAndHashLink(t *testing.T) {
	for _, kind := range []models.DocumentKind{models.DocumentKindIncomingLetter, models.DocumentKindOutgoingLetter, models.DocumentKindCitizenAppeal, models.DocumentKindAdministrativeOrder} {
		t.Run(string(kind), func(t *testing.T) {
			request := map[string]any{"idempotencyKey": uuid.NewString()}
			plain, err := dto.NormalizeDocumentRegisterRequest(kind, request)
			require.NoError(t, err)
			plainHash, err := documentCommandHash(plain)
			require.NoError(t, err)
			request["link"] = map[string]any{"documentId": uuid.NewString(), "linkType": "related"}
			linked, err := dto.NormalizeDocumentRegisterRequest(kind, request)
			require.NoError(t, err)
			linkedHash, err := documentCommandHash(linked)
			require.NoError(t, err)
			require.NotEqual(t, plainHash, linkedHash)
			request["link"] = map[string]any{"documentId": uuid.NewString(), "linkType": "related"}
			changed, err := dto.NormalizeDocumentRegisterRequest(kind, request)
			require.NoError(t, err)
			changedHash, err := documentCommandHash(changed)
			require.NoError(t, err)
			require.NotEqual(t, linkedHash, changedHash)
		})
	}
}
