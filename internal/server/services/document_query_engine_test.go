package services

import (
	"errors"
	"strings"
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

func TestDocumentQueryEngine_GetByID(t *testing.T) {
	t.Run("returns card after access checks", func(t *testing.T) {
		user := documentAccessUser(true, nil)
		deps := setupDocumentAccessService(t, user, allowDocumentActions(models.DocumentKindIncomingLetter, "register"))
		documentID := uuid.New()
		deps.accessRepo.readable[documentID] = true
		deps.docRepo.docs[documentID] = documentAccessDoc(documentID, models.DocumentKindIncomingLetter)

		expected := &dto.DocumentCard{ID: documentID.String(), KindCode: string(models.DocumentKindIncomingLetter)}
		handler := &stubDocumentKindQueryHandler{
			kind: models.DocumentKindIncomingLetter,
			card: expected,
		}
		svc := NewDocumentQueryEngine(NewDocumentKindQueryRegistry(handler), deps.service, nil, nil)

		card, err := svc.GetByID(documentID.String())

		require.NoError(t, err)
		assert.Same(t, expected, card)
		assert.Equal(t, documentID, handler.lastCardID)
	})

	t.Run("rejects invalid id before document lookup", func(t *testing.T) {
		deps := setupDocumentAccessService(t, documentAccessUser(true, nil), allowDocumentActions(models.DocumentKindIncomingLetter, "register"))
		handler := &stubDocumentKindQueryHandler{kind: models.DocumentKindIncomingLetter}
		svc := NewDocumentQueryEngine(NewDocumentKindQueryRegistry(handler), deps.service, nil, nil)

		card, err := svc.GetByID("not-a-uuid")

		require.Error(t, err)
		assert.Nil(t, card)
		requireAppError(t, err, "VALIDATION_ERROR", 400, "неверный ID документа")
	})

	t.Run("returns forbidden for missing handler", func(t *testing.T) {
		user := documentAccessUser(true, nil)
		deps := setupDocumentAccessService(t, user, allowDocumentActions(models.DocumentKindIncomingLetter, "register"))
		documentID := uuid.New()
		deps.accessRepo.readable[documentID] = true
		deps.docRepo.docs[documentID] = documentAccessDoc(documentID, models.DocumentKindIncomingLetter)
		svc := NewDocumentQueryEngine(NewDocumentKindQueryRegistry(), deps.service, nil, nil)

		card, err := svc.GetByID(documentID.String())

		require.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, card)
	})

	t.Run("returns handler error", func(t *testing.T) {
		user := documentAccessUser(true, nil)
		deps := setupDocumentAccessService(t, user, allowDocumentActions(models.DocumentKindIncomingLetter, "register"))
		documentID := uuid.New()
		deps.accessRepo.readable[documentID] = true
		deps.docRepo.docs[documentID] = documentAccessDoc(documentID, models.DocumentKindIncomingLetter)
		handlerErr := errors.New("card failed")
		handler := &stubDocumentKindQueryHandler{
			kind:    models.DocumentKindIncomingLetter,
			cardErr: handlerErr,
		}
		svc := NewDocumentQueryEngine(NewDocumentKindQueryRegistry(handler), deps.service, nil, nil)

		card, err := svc.GetByID(documentID.String())

		require.ErrorIs(t, err, handlerErr)
		assert.Nil(t, card)
	})

	t.Run("returns access error before handler lookup", func(t *testing.T) {
		user := documentAccessUser(false, nil)
		deps := setupDocumentAccessService(t, user, nil)
		documentID := uuid.New()
		deps.docRepo.docs[documentID] = documentAccessDoc(documentID, models.DocumentKindIncomingLetter)
		handler := &stubDocumentKindQueryHandler{kind: models.DocumentKindIncomingLetter}
		svc := NewDocumentQueryEngine(NewDocumentKindQueryRegistry(handler), deps.service, nil, nil)

		card, err := svc.GetByID(documentID.String())

		require.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, card)
		assert.Equal(t, uuid.Nil, handler.lastCardID)
	})
}

func TestDocumentQueryEngine_GetList(t *testing.T) {
	t.Run("passes search and paging with server read scope", func(t *testing.T) {
		deps := setupDocumentAccessService(t, documentAccessUser(true, nil), allowDocumentActions(models.DocumentKindIncomingLetter, "register"))
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
		svc := NewDocumentQueryEngine(NewDocumentKindQueryRegistry(handler), deps.service, nil, nil)

		res, err := svc.GetList(string(models.DocumentKindIncomingLetter), models.DocumentFilter{Search: "abc", Page: 2, PageSize: 25})

		require.NoError(t, err)
		assert.Same(t, expected, res)
		assert.Equal(t, "abc", handler.lastFilter.Search)
		require.NotNil(t, handler.lastFilter.AccessScope)
	})

	t.Run("replaces client scope with current user identity", func(t *testing.T) {
		departmentID := uuid.New()
		user := documentAccessUser(true, &departmentID)
		deps := setupDocumentAccessService(t, user, nil)
		handler := &stubDocumentKindQueryHandler{
			kind: models.DocumentKindIncomingLetter,
			list: &dto.PagedResult[dto.DocumentListItem]{},
		}
		svc := NewDocumentQueryEngine(NewDocumentKindQueryRegistry(handler), deps.service, nil, nil)

		res, err := svc.GetList(string(models.DocumentKindIncomingLetter), models.DocumentFilter{Page: 1, PageSize: 10, AccessScope: &models.DocumentAccessScope{}})

		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, handler.lastFilter.AccessScope)
		assert.Equal(t, user.ID.String(), handler.lastFilter.AccessScope.UserID)
	})

	t.Run("returns forbidden for unsupported kind", func(t *testing.T) {
		deps := setupDocumentAccessService(t, documentAccessUser(true, nil), allowDocumentActions(models.DocumentKindIncomingLetter, "register"))
		svc := NewDocumentQueryEngine(NewDocumentKindQueryRegistry(), deps.service, nil, nil)

		res, err := svc.GetList(string(models.DocumentKindIncomingLetter), models.DocumentFilter{})

		require.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, res)
	})

	t.Run("returns handler list error", func(t *testing.T) {
		deps := setupDocumentAccessService(t, documentAccessUser(true, nil), allowDocumentActions(models.DocumentKindIncomingLetter, "register"))
		handlerErr := errors.New("list failed")
		handler := &stubDocumentKindQueryHandler{
			kind:    models.DocumentKindIncomingLetter,
			listErr: handlerErr,
		}
		svc := NewDocumentQueryEngine(NewDocumentKindQueryRegistry(handler), deps.service, nil, nil)

		res, err := svc.GetList(string(models.DocumentKindIncomingLetter), models.DocumentFilter{Search: "abc"})

		require.ErrorIs(t, err, handlerErr)
		assert.Nil(t, res)
		assert.Equal(t, "abc", handler.lastFilter.Search)
	})
}

type stubDocumentSearchStore struct {
	request dto.DocumentSearchRequest
	scopes  map[models.DocumentKind]models.DocumentAccessScope
	calls   int
}

func (s *stubDocumentSearchStore) SearchDocuments(request dto.DocumentSearchRequest, scopes map[models.DocumentKind]models.DocumentAccessScope) (*dto.DocumentSearchResult, error) {
	s.request, s.scopes = request, scopes
	s.calls++
	return &dto.DocumentSearchResult{Items: []dto.DocumentSearchItem{}, Page: request.Page, PageSize: request.PageSize}, nil
}

func TestDocumentQueryEngine_Search(t *testing.T) {
	deps := setupDocumentAccessService(t, documentAccessUser(true, nil), allowDocumentActions(models.DocumentKindIncomingLetter, "register"))
	store := &stubDocumentSearchStore{}
	svc := NewDocumentQueryEngine(NewDocumentKindQueryRegistry(
		&stubDocumentKindQueryHandler{kind: models.DocumentKindIncomingLetter},
		&stubDocumentKindQueryHandler{kind: models.DocumentKindOutgoingLetter},
	), deps.service, nil, store)
	result, err := svc.Search(dto.DocumentSearchRequest{Query: "  ремонт\n Иванов  ", PageSize: 1000})
	require.NoError(t, err)
	require.Equal(t, "ремонт Иванов", store.request.Query)
	require.Equal(t, 1, result.Page)
	require.Equal(t, 100, result.PageSize)
	require.NotContains(t, store.scopes, models.DocumentKindAdministrativeOrder)
	for _, request := range []dto.DocumentSearchRequest{
		{Query: "  "}, {Query: strings.Repeat("я", 501)},
		{Query: strings.Repeat("слово ", 33)}, {Query: "ремонт", Page: 10001},
	} {
		_, err := svc.Search(request)
		require.Error(t, err)
	}
	require.Equal(t, 1, store.calls)
	forbidden := setupDocumentAccessService(t, documentAccessUser(false, nil), allowDocumentActions(models.DocumentKindIncomingLetter))
	svc = NewDocumentQueryEngine(NewDocumentKindQueryRegistry(), forbidden.service, nil, store)
	result, err = svc.Search(dto.DocumentSearchRequest{Query: "ремонт"})
	require.NoError(t, err)
	require.Empty(t, result.Items)
	require.Empty(t, store.scopes)
	require.Equal(t, 2, store.calls)
}
