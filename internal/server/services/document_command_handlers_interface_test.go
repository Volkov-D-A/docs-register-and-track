package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestDocumentCommandHandlers_InterfaceMethods(t *testing.T) {
	tests := []struct {
		name        string
		kind        models.DocumentKind
		handler     DocumentKindCommandHandler
		registerReq any
		updateReq   any
	}{
		{
			name:        "incoming letter",
			kind:        models.DocumentKindIncomingLetter,
			handler:     setupIncomingLetterCommandHandler(t, nil).handler,
			registerReq: dto.IncomingLetterRegisterRequest{},
			updateReq:   dto.IncomingLetterUpdateRequest{ID: "bad-id"},
		},
		{
			name:        "outgoing letter",
			kind:        models.DocumentKindOutgoingLetter,
			handler:     setupOutgoingLetterCommandHandler(t, nil).handler,
			registerReq: dto.OutgoingLetterRegisterRequest{},
			updateReq:   dto.OutgoingLetterUpdateRequest{ID: "bad-id"},
		},
		{
			name:        "citizen appeal",
			kind:        models.DocumentKindCitizenAppeal,
			handler:     setupCitizenAppealCommandHandler(t, nil).handler,
			registerReq: dto.CitizenAppealRegisterRequest{},
			updateReq:   dto.CitizenAppealUpdateRequest{ID: "bad-id"},
		},
		{
			name:        "administrative order",
			kind:        models.DocumentKindAdministrativeOrder,
			handler:     setupAdministrativeOrderCommandHandler(t, nil).handler,
			registerReq: dto.AdministrativeOrderRegisterRequest{},
			updateReq:   dto.AdministrativeOrderUpdateRequest{ID: "bad-id"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.kind, tt.handler.Kind())

			result, err := tt.handler.RegisterDocument("bad request")
			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "invalid register request")

			result, err = tt.handler.RegisterDocument(tt.registerReq)
			require.ErrorIs(t, err, models.ErrForbidden)
			assert.Nil(t, result)

			result, err = tt.handler.UpdateDocument("bad request")
			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "invalid update request")

			result, err = tt.handler.UpdateDocument(tt.updateReq)
			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "неверный ID документа")
		})
	}
}
