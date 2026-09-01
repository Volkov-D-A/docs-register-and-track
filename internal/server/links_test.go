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

type fakeLinkAPI struct {
	sourceID, targetID, linkType string
	listDocumentID               string
}

func (f *fakeLinkAPI) LinkDocuments(sourceID, targetID, linkType string) (*dto.DocumentLink, error) {
	f.sourceID, f.targetID, f.linkType = sourceID, targetID, linkType
	return &dto.DocumentLink{ID: uuid.NewString()}, nil
}
func (*fakeLinkAPI) UnlinkDocument(string) error { return nil }
func (f *fakeLinkAPI) GetDocumentLinks(id string) ([]dto.DocumentLink, error) {
	f.listDocumentID = id
	return []dto.DocumentLink{}, nil
}
func (*fakeLinkAPI) GetDocumentFlow(string) (*models.GraphData, error) {
	return &models.GraphData{Nodes: []models.GraphNode{}, Edges: []models.GraphEdge{}}, nil
}

type fakeJournalAPI struct{ documentID string }

func (f *fakeJournalAPI) GetByDocumentID(id string) ([]dto.JournalEntry, error) {
	f.documentID = id
	return []dto.JournalEntry{}, nil
}

func TestLinkAndJournalAPIsRequireSessionAndUseRequestPrincipal(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	links, journal := &fakeLinkAPI{}, &fakeJournalAPI{}
	var linkPrincipal, journalPrincipal uuid.UUID
	api.links = func(user *models.User) linkAPI { linkPrincipal = user.ID; return links }
	api.journal = func(user *models.User) journalAPI { journalPrincipal = user.ID; return journal }

	unauthorized := httptest.NewRecorder()
	api.Handler().ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v1/documents/doc/journal", nil))
	assert.Equal(t, http.StatusUnauthorized, unauthorized.Code)

	malicious := httptest.NewRequest(http.MethodPost, "/api/v1/document-links", strings.NewReader(`{"sourceId":"source","targetId":"target","linkType":"related","userId":"forged"}`))
	malicious.Header.Set("Authorization", "Bearer "+token)
	maliciousResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(maliciousResponse, malicious)
	require.Equal(t, http.StatusBadRequest, maliciousResponse.Code, maliciousResponse.Body.String())

	create := httptest.NewRequest(http.MethodPost, "/api/v1/document-links", strings.NewReader(`{"sourceId":"source","targetId":"target","linkType":"related"}`))
	create.Header.Set("Authorization", "Bearer "+token)
	createResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(createResponse, create)
	require.Equal(t, http.StatusCreated, createResponse.Code, createResponse.Body.String())
	assert.Equal(t, "source", links.sourceID)
	assert.Equal(t, "target", links.targetID)
	assert.NotEqual(t, uuid.Nil, linkPrincipal)

	journalRequest := httptest.NewRequest(http.MethodGet, "/api/v1/documents/doc/journal", nil)
	journalRequest.Header.Set("Authorization", "Bearer "+token)
	journalResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(journalResponse, journalRequest)
	require.Equal(t, http.StatusOK, journalResponse.Code, journalResponse.Body.String())
	assert.Equal(t, "doc", journal.documentID)
	assert.NotEqual(t, uuid.Nil, journalPrincipal)
}

func TestDocumentLinkReadRoutes(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	links := &fakeLinkAPI{}
	api.links = func(*models.User) linkAPI { return links }
	for _, path := range []string{"/api/v1/documents/doc/links", "/api/v1/documents/doc/link-graph"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, request)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	}
	assert.Equal(t, "doc", links.listDocumentID)
}
