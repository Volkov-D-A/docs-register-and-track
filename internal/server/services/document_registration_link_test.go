package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestRegistrationLinkRequiresAccessToBothDocuments(t *testing.T) {
	for _, missing := range []string{"none", "existing read", "existing link", "new read", "new link"} {
		t.Run(missing, func(t *testing.T) {
			allowed := allowDocumentActions(models.DocumentKindIncomingLetter, "read", "link")
			addDocumentActions(allowed, models.DocumentKindOutgoingLetter, "create", "read", "link")
			switch missing {
			case "existing read":
				delete(allowed[models.DocumentKindIncomingLetter], "read")
			case "existing link":
				delete(allowed[models.DocumentKindIncomingLetter], "link")
			case "new read":
				delete(allowed[models.DocumentKindOutgoingLetter], "read")
			case "new link":
				delete(allowed[models.DocumentKindOutgoingLetter], "link")
			}
			deps := setupDocumentAccessService(t, documentAccessUser(false, nil), allowed)
			id := uuid.New()
			deps.docRepo.docs[id] = documentAccessDoc(id, uuid.New(), models.DocumentKindIncomingLetter)
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
	allowed := allowDocumentActions(models.DocumentKindIncomingLetter, "read", "link")
	addDocumentActions(allowed, models.DocumentKindAdministrativeOrder, "create", "read", "link")
	deps := setupDocumentAccessService(t, documentAccessUser(false, nil), allowed)
	id := uuid.New()
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
