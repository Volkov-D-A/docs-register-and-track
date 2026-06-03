package services

import (
	"errors"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubDocumentKindCommandHandler struct {
	kind         models.DocumentKind
	registerFunc func(req any) (any, error)
	updateFunc   func(req any) (any, error)
}

func (h stubDocumentKindCommandHandler) Kind() models.DocumentKind {
	return h.kind
}

func (h stubDocumentKindCommandHandler) RegisterDocument(req any) (any, error) {
	return h.registerFunc(req)
}

func (h stubDocumentKindCommandHandler) UpdateDocument(req any) (any, error) {
	return h.updateFunc(req)
}

func TestDocumentRegistrationService_RegisterRoutesToKindHandler(t *testing.T) {
	t.Parallel()

	expectedReq := IncomingLetterRegisterRequest{Content: "test"}
	handler := stubDocumentKindCommandHandler{
		kind: models.DocumentKindIncomingLetter,
		registerFunc: func(req any) (any, error) {
			typedReq, ok := req.(IncomingLetterRegisterRequest)
			if !ok {
				t.Fatalf("unexpected request type %T", req)
			}
			if typedReq.Content != expectedReq.Content {
				t.Fatalf("unexpected request content: %q", typedReq.Content)
			}
			return "ok", nil
		},
		updateFunc: func(req any) (any, error) {
			t.Fatalf("update should not be called")
			return nil, nil
		},
	}

	service := NewDocumentRegistrationService(NewDocumentKindCommandRegistry(handler))

	got, err := service.Register(string(models.DocumentKindIncomingLetter), expectedReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "ok" {
		t.Fatalf("unexpected result: %v", got)
	}
}

func TestDocumentRegistrationService_UpdateRoutesToKindHandler(t *testing.T) {
	t.Parallel()

	expectedReq := OutgoingLetterUpdateRequest{ID: "doc-id"}
	handler := stubDocumentKindCommandHandler{
		kind: models.DocumentKindOutgoingLetter,
		registerFunc: func(req any) (any, error) {
			t.Fatalf("register should not be called")
			return nil, nil
		},
		updateFunc: func(req any) (any, error) {
			typedReq, ok := req.(OutgoingLetterUpdateRequest)
			if !ok {
				t.Fatalf("unexpected request type %T", req)
			}
			if typedReq.ID != expectedReq.ID {
				t.Fatalf("unexpected request id: %q", typedReq.ID)
			}
			return 123, nil
		},
	}

	service := NewDocumentRegistrationService(NewDocumentKindCommandRegistry(handler))

	got, err := service.Update(string(models.DocumentKindOutgoingLetter), expectedReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 123 {
		t.Fatalf("unexpected result: %v", got)
	}
}

func TestDocumentRegistrationService_UnsupportedKindReturnsForbidden(t *testing.T) {
	t.Parallel()

	service := NewDocumentRegistrationService(NewDocumentKindCommandRegistry())

	_, err := service.Register("unknown_kind", nil)
	if !errors.Is(err, models.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestDocumentRegistrationService_NormalizesRegisterRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		kind         models.DocumentKind
		payload      map[string]any
		assertTyped  func(t *testing.T, req any)
		expectedResp string
	}{
		{
			name: "incoming",
			kind: models.DocumentKindIncomingLetter,
			payload: map[string]any{
				"nomenclatureId":  "nom-1",
				"idempotencyKey":  "idem-1",
				"incomingDate":    "2026-06-03",
				"content":         "Входящее",
				"senderSignatory": "Подписант",
				"correspondents":  []map[string]any{{"correspondentName": "Организация"}},
			},
			assertTyped: func(t *testing.T, req any) {
				t.Helper()
				typed, ok := req.(IncomingLetterRegisterRequest)
				require.True(t, ok)
				assert.Equal(t, "Входящее", typed.Content)
				require.Len(t, typed.Correspondents, 1)
				assert.Equal(t, "Организация", typed.Correspondents[0].CorrespondentName)
			},
			expectedResp: "incoming ok",
		},
		{
			name: "outgoing",
			kind: models.DocumentKindOutgoingLetter,
			payload: map[string]any{
				"nomenclatureId":   "nom-1",
				"idempotencyKey":   "idem-1",
				"outgoingDate":     "2026-06-03",
				"content":          "Исходящее",
				"recipientOrgName": "Получатель",
				"senderExecutor":   "Исполнитель",
				"senderSignatory":  "Подписант",
			},
			assertTyped: func(t *testing.T, req any) {
				t.Helper()
				typed, ok := req.(OutgoingLetterRegisterRequest)
				require.True(t, ok)
				assert.Equal(t, "Исходящее", typed.Content)
				assert.Equal(t, "Получатель", typed.RecipientOrgName)
			},
			expectedResp: "outgoing ok",
		},
		{
			name: "citizen appeal",
			kind: models.DocumentKindCitizenAppeal,
			payload: map[string]any{
				"nomenclatureId":       "nom-1",
				"idempotencyKey":       "idem-1",
				"registrationDate":     "2026-06-03",
				"appealDate":           "2026-06-02",
				"content":              "Обращение",
				"applicantFullName":    "Иван Иванов",
				"registrationAddress":  "Адрес",
				"appealType":           "жалоба",
				"applicantCategory":    "гражданин",
				"appealPagesCount":     1,
				"attachmentPagesCount": 2,
				"hasEnvelope":          true,
				"receivedFromPos":      false,
				"correspondents":       []map[string]any{{"correspondentName": "Администрация"}},
				"resolutions":          []map[string]any{{"resolution": "Подготовить ответ", "resolutionAuthor": "Руководитель", "resolutionExecutors": "Исполнитель"}},
			},
			assertTyped: func(t *testing.T, req any) {
				t.Helper()
				typed, ok := req.(CitizenAppealRegisterRequest)
				require.True(t, ok)
				assert.Equal(t, "Иван Иванов", typed.ApplicantFullName)
				assert.True(t, typed.HasEnvelope)
			},
			expectedResp: "appeal ok",
		},
		{
			name: "administrative order",
			kind: models.DocumentKindAdministrativeOrder,
			payload: map[string]any{
				"nomenclatureId":          "nom-1",
				"idempotencyKey":          "idem-1",
				"orderDate":               "2026-06-03",
				"title":                   "Приказ",
				"executionController":     "Контроль",
				"executionDeadline":       "2026-06-30",
				"isActive":                true,
				"acknowledgmentFullNames": []string{"Иван Иванов"},
			},
			assertTyped: func(t *testing.T, req any) {
				t.Helper()
				typed, ok := req.(AdministrativeOrderRegisterRequest)
				require.True(t, ok)
				assert.Equal(t, "Приказ", typed.Title)
				assert.Equal(t, []string{"Иван Иванов"}, typed.AcknowledgmentFullNames)
			},
			expectedResp: "order ok",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler := stubDocumentKindCommandHandler{
				kind: tt.kind,
				registerFunc: func(req any) (any, error) {
					tt.assertTyped(t, req)
					return tt.expectedResp, nil
				},
				updateFunc: func(req any) (any, error) {
					t.Fatalf("update should not be called")
					return nil, nil
				},
			}
			service := NewDocumentRegistrationService(NewDocumentKindCommandRegistry(handler))

			got, err := service.Register(string(tt.kind), tt.payload)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedResp, got)
		})
	}
}

func TestDocumentRegistrationService_NormalizesUpdateRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		kind        models.DocumentKind
		payload     map[string]any
		assertTyped func(t *testing.T, req any)
	}{
		{
			name: "incoming",
			kind: models.DocumentKindIncomingLetter,
			payload: map[string]any{
				"id":              "doc-1",
				"content":         "Входящее",
				"senderSignatory": "Подписант",
			},
			assertTyped: func(t *testing.T, req any) {
				t.Helper()
				typed, ok := req.(IncomingLetterUpdateRequest)
				require.True(t, ok)
				assert.Equal(t, "doc-1", typed.ID)
			},
		},
		{
			name: "outgoing",
			kind: models.DocumentKindOutgoingLetter,
			payload: map[string]any{
				"id":               "doc-1",
				"outgoingDate":     "2026-06-03",
				"content":          "Исходящее",
				"recipientOrgName": "Получатель",
				"senderExecutor":   "Исполнитель",
			},
			assertTyped: func(t *testing.T, req any) {
				t.Helper()
				typed, ok := req.(OutgoingLetterUpdateRequest)
				require.True(t, ok)
				assert.Equal(t, "Получатель", typed.RecipientOrgName)
			},
		},
		{
			name: "citizen appeal",
			kind: models.DocumentKindCitizenAppeal,
			payload: map[string]any{
				"id":                   "doc-1",
				"registrationDate":     "2026-06-03",
				"appealDate":           "2026-06-02",
				"content":              "Обращение",
				"applicantFullName":    "Иван Иванов",
				"registrationAddress":  "Адрес",
				"appealType":           "жалоба",
				"applicantCategory":    "гражданин",
				"appealPagesCount":     1,
				"attachmentPagesCount": 2,
			},
			assertTyped: func(t *testing.T, req any) {
				t.Helper()
				typed, ok := req.(CitizenAppealUpdateRequest)
				require.True(t, ok)
				assert.Equal(t, "жалоба", typed.AppealType)
			},
		},
		{
			name: "administrative order",
			kind: models.DocumentKindAdministrativeOrder,
			payload: map[string]any{
				"id":                      "doc-1",
				"orderDate":               "2026-06-03",
				"title":                   "Приказ",
				"executionController":     "Контроль",
				"isActive":                true,
				"acknowledgmentFullNames": []string{"Иван Иванов"},
			},
			assertTyped: func(t *testing.T, req any) {
				t.Helper()
				typed, ok := req.(AdministrativeOrderUpdateRequest)
				require.True(t, ok)
				assert.Equal(t, "Приказ", typed.Title)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler := stubDocumentKindCommandHandler{
				kind: tt.kind,
				registerFunc: func(req any) (any, error) {
					t.Fatalf("register should not be called")
					return nil, nil
				},
				updateFunc: func(req any) (any, error) {
					tt.assertTyped(t, req)
					return "updated", nil
				},
			}
			service := NewDocumentRegistrationService(NewDocumentKindCommandRegistry(handler))

			got, err := service.Update(string(tt.kind), tt.payload)

			require.NoError(t, err)
			assert.Equal(t, "updated", got)
		})
	}
}

func TestDocumentRegistrationService_RejectsMalformedCommandPayload(t *testing.T) {
	t.Parallel()

	handler := stubDocumentKindCommandHandler{
		kind: models.DocumentKindIncomingLetter,
		registerFunc: func(req any) (any, error) {
			t.Fatalf("handler should not be called")
			return nil, nil
		},
		updateFunc: func(req any) (any, error) {
			t.Fatalf("handler should not be called")
			return nil, nil
		},
	}
	service := NewDocumentRegistrationService(NewDocumentKindCommandRegistry(handler))

	_, err := service.Register(string(models.DocumentKindIncomingLetter), map[string]any{
		"content":      "test",
		"unknownField": "boom",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "неверные поля команды документа")
}

func TestDocumentRegistrationService_UnsupportedUpdateKindReturnsForbidden(t *testing.T) {
	t.Parallel()

	service := NewDocumentRegistrationService(NewDocumentKindCommandRegistry())

	_, err := service.Update("unknown_kind", nil)

	require.ErrorIs(t, err, models.ErrForbidden)
}
