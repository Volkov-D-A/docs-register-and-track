package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
	"github.com/Volkov-D-A/docs-register-and-track/internal/testutil/integrationdb"
)

func TestLinkAndJournalAPIUseServerPrincipalIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}
	password := "LinksPassw0rd!"
	hash, err := security.HashPassword(password)
	require.NoError(t, err)
	managerID, outsiderID := uuid.New(), uuid.New()
	for _, user := range []struct {
		id    uuid.UUID
		login string
		name  string
	}{{managerID, "links-manager", "Links Manager"}, {outsiderID, "links-outsider", "Links Outsider"}} {
		_, err = db.Exec(`INSERT INTO users (id, login, password_hash, full_name, is_active, is_document_participant, password_change_required)
			VALUES ($1, $2, $3, $4, TRUE, TRUE, FALSE)`, user.id, user.login, hash, user.name)
		require.NoError(t, err)
	}
	_, err = db.Exec(`INSERT INTO document_permissions (kind_code, subject_type, subject_key, action, is_allowed)
		VALUES ('outgoing_letter', 'user', $1, 'read', TRUE),
		       ('outgoing_letter', 'user', $1, 'link', TRUE),
		       ('outgoing_letter', 'user', $1, 'view_journal', TRUE)`, managerID.String())
	require.NoError(t, err)
	nomenclatureID, organizationID := uuid.New(), uuid.New()
	_, err = db.Exec(`INSERT INTO nomenclature (id, name, index, year, kind_code, separator, numbering_mode)
		VALUES ($1, 'Link integration', 'OUT-L', 2026, 'outgoing_letter', '/', 'index_and_number')`, nomenclatureID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO organizations (id, name) VALUES ($1, 'Link Organization')`, organizationID)
	require.NoError(t, err)
	documents := repository.NewOutgoingDocumentRepository(db)
	createDocument := func(content string) *models.OutgoingDocument {
		document, createErr := documents.Create(models.CreateOutgoingDocRequest{
			NomenclatureID: nomenclatureID, IdempotencyKey: uuid.New(), DocumentTypeID: models.DocumentTypeLetter,
			RecipientOrgID: organizationID, CreatedBy: managerID, OutgoingDate: time.Now().UTC(), Content: content,
			PagesCount: 1, SenderSignatory: "Signer", SenderExecutor: "Executor", Addressee: "Addressee",
		})
		require.NoError(t, createErr)
		return document
	}
	source, target := createDocument("link source"), createDocument("link target")

	api := newManagementAPI(&App{db: db, cfg: &config.Config{Server: config.ServerConfig{SessionTTLHours: 12}}, metrics: observability.NewRegistry(32)})
	login := func(login string) string {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"`+login+`","password":"`+password+`"}`))
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, request)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		var session struct {
			AccessToken string `json:"accessToken"`
		}
		require.NoError(t, json.NewDecoder(response.Body).Decode(&session))
		return session.AccessToken
	}
	requestWithToken := func(method, path, body, token string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, request)
		return response
	}

	managerToken := login("links-manager")
	createBody := `{"sourceId":"` + source.ID.String() + `","targetId":"` + target.ID.String() + `","linkType":"related"}`
	createResponse := requestWithToken(http.MethodPost, "/api/v1/document-links", createBody, managerToken)
	require.Equal(t, http.StatusCreated, createResponse.Code, createResponse.Body.String())
	var created struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(createResponse.Body).Decode(&created))
	var createdBy uuid.UUID
	require.NoError(t, db.QueryRow(`SELECT created_by FROM document_links WHERE id=$1`, created.ID).Scan(&createdBy))
	require.Equal(t, managerID, createdBy)

	linksResponse := requestWithToken(http.MethodGet, "/api/v1/documents/"+source.ID.String()+"/links", "", managerToken)
	require.Equal(t, http.StatusOK, linksResponse.Code, linksResponse.Body.String())
	require.Contains(t, linksResponse.Body.String(), created.ID)

	journalID, err := repository.NewJournalRepository(db).Create(context.Background(), models.CreateJournalEntryRequest{
		DocumentID: source.ID, UserID: managerID, Action: "LINK_INTEGRATION", Details: "server-owned journal",
	})
	require.NoError(t, err)
	journalResponse := requestWithToken(http.MethodGet, "/api/v1/documents/"+source.ID.String()+"/journal", "", managerToken)
	require.Equal(t, http.StatusOK, journalResponse.Code, journalResponse.Body.String())
	require.Contains(t, journalResponse.Body.String(), journalID.String())

	outsiderResponse := requestWithToken(http.MethodGet, "/api/v1/documents/"+source.ID.String()+"/journal", "", login("links-outsider"))
	require.Equal(t, http.StatusForbidden, outsiderResponse.Code, outsiderResponse.Body.String())
}
