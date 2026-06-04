package services

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type stubDocumentKindQueryHandler struct {
	kind       models.DocumentKind
	card       *dto.DocumentCard
	cardErr    error
	list       *dto.PagedResult[dto.DocumentListItem]
	listErr    error
	lastCardID uuid.UUID
	lastFilter models.DocumentFilter
}

func (h *stubDocumentKindQueryHandler) Kind() models.DocumentKind {
	return h.kind
}

func (h *stubDocumentKindQueryHandler) GetCard(id uuid.UUID) (*dto.DocumentCard, error) {
	h.lastCardID = id
	return h.card, h.cardErr
}

func (h *stubDocumentKindQueryHandler) GetList(filter models.DocumentFilter) (*dto.PagedResult[dto.DocumentListItem], error) {
	h.lastFilter = filter
	return h.list, h.listErr
}

func TestDocumentKindQueryRegistry(t *testing.T) {
	handler := &stubDocumentKindQueryHandler{kind: models.DocumentKindIncomingLetter}
	registry := NewDocumentKindQueryRegistry(handler)

	got, err := registry.Get(models.DocumentKindIncomingLetter)
	require.NoError(t, err)
	assert.Same(t, handler, got)

	got, err = registry.Get(models.DocumentKind("unsupported"))
	require.Error(t, err)
	assert.Nil(t, got)
	requireAppError(t, err, "VALIDATION_ERROR", 400, "неподдерживаемый вид документа")
}

func TestDocumentQueryService_GetByID(t *testing.T) {
	t.Run("returns card after access checks", func(t *testing.T) {
		user := documentAccessUser(true, nil)
		deps := setupDocumentAccessService(t, user, allowDocumentActions(models.DocumentKindIncomingLetter, "read"))
		documentID := uuid.New()
		deps.docRepo.docs[documentID] = documentAccessDoc(documentID, uuid.New(), models.DocumentKindIncomingLetter)

		expected := &dto.DocumentCard{ID: documentID.String(), KindCode: string(models.DocumentKindIncomingLetter)}
		handler := &stubDocumentKindQueryHandler{
			kind: models.DocumentKindIncomingLetter,
			card: expected,
		}
		svc := NewDocumentQueryService(NewDocumentKindQueryRegistry(handler), deps.service)

		card, err := svc.GetByID(documentID.String())

		require.NoError(t, err)
		assert.Same(t, expected, card)
		assert.Equal(t, documentID, handler.lastCardID)
	})

	t.Run("rejects invalid id before document lookup", func(t *testing.T) {
		deps := setupDocumentAccessService(t, documentAccessUser(true, nil), allowDocumentActions(models.DocumentKindIncomingLetter, "read"))
		handler := &stubDocumentKindQueryHandler{kind: models.DocumentKindIncomingLetter}
		svc := NewDocumentQueryService(NewDocumentKindQueryRegistry(handler), deps.service)

		card, err := svc.GetByID("not-a-uuid")

		require.Error(t, err)
		assert.Nil(t, card)
		requireAppError(t, err, "VALIDATION_ERROR", 400, "неверный ID документа")
	})

	t.Run("returns forbidden for missing handler", func(t *testing.T) {
		user := documentAccessUser(true, nil)
		deps := setupDocumentAccessService(t, user, allowDocumentActions(models.DocumentKindIncomingLetter, "read"))
		documentID := uuid.New()
		deps.docRepo.docs[documentID] = documentAccessDoc(documentID, uuid.New(), models.DocumentKindIncomingLetter)
		svc := NewDocumentQueryService(NewDocumentKindQueryRegistry(), deps.service)

		card, err := svc.GetByID(documentID.String())

		require.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, card)
	})

	t.Run("returns handler error", func(t *testing.T) {
		user := documentAccessUser(true, nil)
		deps := setupDocumentAccessService(t, user, allowDocumentActions(models.DocumentKindIncomingLetter, "read"))
		documentID := uuid.New()
		deps.docRepo.docs[documentID] = documentAccessDoc(documentID, uuid.New(), models.DocumentKindIncomingLetter)
		handlerErr := errors.New("card failed")
		handler := &stubDocumentKindQueryHandler{
			kind:    models.DocumentKindIncomingLetter,
			cardErr: handlerErr,
		}
		svc := NewDocumentQueryService(NewDocumentKindQueryRegistry(handler), deps.service)

		card, err := svc.GetByID(documentID.String())

		require.ErrorIs(t, err, handlerErr)
		assert.Nil(t, card)
	})

	t.Run("returns access error before handler lookup", func(t *testing.T) {
		user := documentAccessUser(false, nil)
		deps := setupDocumentAccessService(t, user, nil)
		documentID := uuid.New()
		deps.docRepo.docs[documentID] = documentAccessDoc(documentID, uuid.New(), models.DocumentKindIncomingLetter)
		handler := &stubDocumentKindQueryHandler{kind: models.DocumentKindIncomingLetter}
		svc := NewDocumentQueryService(NewDocumentKindQueryRegistry(handler), deps.service)

		card, err := svc.GetByID(documentID.String())

		require.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, card)
		assert.Equal(t, uuid.Nil, handler.lastCardID)
	})
}

func TestDocumentQueryService_GetList(t *testing.T) {
	t.Run("passes unrestricted filter when read is allowed", func(t *testing.T) {
		deps := setupDocumentAccessService(t, documentAccessUser(true, nil), allowDocumentActions(models.DocumentKindIncomingLetter, "read"))
		expected := &dto.PagedResult[dto.DocumentListItem]{
			Items:      []dto.DocumentListItem{{KindCode: string(models.DocumentKindIncomingLetter)}},
			TotalCount: 1,
			Page:       2,
			PageSize:   25,
		}
		handler := &stubDocumentKindQueryHandler{
			kind: models.DocumentKindIncomingLetter,
			list: expected,
		}
		svc := NewDocumentQueryService(NewDocumentKindQueryRegistry(handler), deps.service)

		res, err := svc.GetList(string(models.DocumentKindIncomingLetter), models.DocumentFilter{Search: "abc", Page: 2, PageSize: 25})

		require.NoError(t, err)
		assert.Same(t, expected, res)
		assert.Equal(t, "abc", handler.lastFilter.Search)
		assert.Empty(t, handler.lastFilter.AllowedNomenclatureIDs)
		assert.Empty(t, handler.lastFilter.AccessibleByUserID)
	})

	t.Run("applies restricted scope to filter", func(t *testing.T) {
		departmentID := uuid.New()
		user := documentAccessUser(true, &departmentID)
		deps := setupDocumentAccessService(t, user, allowDocumentActions(models.DocumentKindIncomingLetter, "upload"))
		deps.depRepo.nomenclatureIDs = []string{"nom-1", "nom-2"}
		handler := &stubDocumentKindQueryHandler{
			kind: models.DocumentKindIncomingLetter,
			list: &dto.PagedResult[dto.DocumentListItem]{},
		}
		svc := NewDocumentQueryService(NewDocumentKindQueryRegistry(handler), deps.service)

		res, err := svc.GetList(string(models.DocumentKindIncomingLetter), models.DocumentFilter{Page: 1, PageSize: 10})

		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, []string{"nom-1", "nom-2"}, handler.lastFilter.AllowedNomenclatureIDs)
		assert.Equal(t, user.ID.String(), handler.lastFilter.AccessibleByUserID)
	})

	t.Run("returns forbidden for unsupported kind", func(t *testing.T) {
		deps := setupDocumentAccessService(t, documentAccessUser(true, nil), allowDocumentActions(models.DocumentKindIncomingLetter, "read"))
		svc := NewDocumentQueryService(NewDocumentKindQueryRegistry(), deps.service)

		res, err := svc.GetList(string(models.DocumentKindIncomingLetter), models.DocumentFilter{})

		require.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, res)
	})

	t.Run("returns handler list error", func(t *testing.T) {
		deps := setupDocumentAccessService(t, documentAccessUser(true, nil), allowDocumentActions(models.DocumentKindIncomingLetter, "read"))
		handlerErr := errors.New("list failed")
		handler := &stubDocumentKindQueryHandler{
			kind:    models.DocumentKindIncomingLetter,
			listErr: handlerErr,
		}
		svc := NewDocumentQueryService(NewDocumentKindQueryRegistry(handler), deps.service)

		res, err := svc.GetList(string(models.DocumentKindIncomingLetter), models.DocumentFilter{Search: "abc"})

		require.ErrorIs(t, err, handlerErr)
		assert.Nil(t, res)
		assert.Equal(t, "abc", handler.lastFilter.Search)
	})
}
