package services

import (
	"errors"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
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
