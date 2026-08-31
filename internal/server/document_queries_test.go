package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type fakeDocumentQueryAPI struct {
	card       *dto.DocumentCard
	list       *dto.PagedResult[dto.DocumentListItem]
	lastID     string
	lastKind   string
	lastFilter models.DocumentFilter
}

func (q *fakeDocumentQueryAPI) GetByID(id string) (*dto.DocumentCard, error) {
	q.lastID = id
	return q.card, nil
}

func (q *fakeDocumentQueryAPI) GetList(kind string, filter models.DocumentFilter) (*dto.PagedResult[dto.DocumentListItem], error) {
	q.lastKind, q.lastFilter = kind, filter
	return q.list, nil
}

func TestDocumentQueryAPIRequiresSessionAndUsesRequestPrincipal(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	id := uuid.NewString()
	query := &fakeDocumentQueryAPI{
		card: &dto.DocumentCard{ID: id, KindCode: string(models.DocumentKindIncomingLetter)},
		list: &dto.PagedResult[dto.DocumentListItem]{Items: []dto.DocumentListItem{{ID: id}}, TotalCount: 1},
	}
	var principalID uuid.UUID
	api.documentQueries = func(user *models.User) documentQueryAPI {
		principalID = user.ID
		return query
	}

	unauthorized := httptest.NewRecorder()
	api.Handler().ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v1/documents/"+id, nil))
	assert.Equal(t, http.StatusUnauthorized, unauthorized.Code)

	cardRequest := httptest.NewRequest(http.MethodGet, "/api/v1/documents/"+id, nil)
	cardRequest.Header.Set("Authorization", "Bearer "+token)
	cardResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(cardResponse, cardRequest)
	require.Equal(t, http.StatusOK, cardResponse.Code, cardResponse.Body.String())
	assert.Equal(t, id, query.lastID)
	assert.NotEqual(t, uuid.Nil, principalID)

	maliciousRequest := httptest.NewRequest(http.MethodPost, "/api/v1/documents/query", strings.NewReader(`{"kindCode":"incoming_letter","filter":{"search":"needle","accessScope":{"restricted":false}}}`))
	maliciousRequest.Header.Set("Authorization", "Bearer "+token)
	maliciousResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(maliciousResponse, maliciousRequest)
	require.Equal(t, http.StatusBadRequest, maliciousResponse.Code, maliciousResponse.Body.String())
	assert.Empty(t, query.lastKind)

	listRequest := httptest.NewRequest(http.MethodPost, "/api/v1/documents/query", strings.NewReader(`{"kindCode":"incoming_letter","filter":{"search":"needle"}}`))
	listRequest.Header.Set("Authorization", "Bearer "+token)
	listResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(listResponse, listRequest)
	require.Equal(t, http.StatusOK, listResponse.Code, listResponse.Body.String())
	assert.Equal(t, "incoming_letter", query.lastKind)
	assert.Equal(t, "needle", query.lastFilter.Search)
	assert.Nil(t, query.lastFilter.AccessScope)
}

func TestRequestDocumentPrincipalIsImmutablePerRequest(t *testing.T) {
	user := &models.User{ID: uuid.New(), IsActive: true, FullName: "Reader"}
	principal := requestDocumentPrincipal{user: user}
	got, err := principal.GetCurrentUserUUID()
	require.NoError(t, err)
	assert.Equal(t, user.ID, got)
	dtoUser, err := principal.GetCurrentUser()
	require.NoError(t, err)
	assert.Equal(t, user.ID.String(), dtoUser.ID)
}
