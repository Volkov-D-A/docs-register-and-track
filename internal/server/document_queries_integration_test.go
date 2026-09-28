package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/security"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
)

func TestDocumentQueryAPIReturnsListAndCardWithServerAccessIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := database.Wrap(sqlDB)
	password := "DocumentReadPassw0rd!"
	hash, err := security.HashPassword(password)
	require.NoError(t, err)
	userID, nomenclatureID, organizationID := uuid.New(), uuid.New(), uuid.New()
	_, err = db.Exec(`INSERT INTO users (id, login, password_hash, last_name, first_name, no_patronymic, is_active, password_change_required)
		VALUES ($1, 'document-reader', $2, 'Document', 'Reader', TRUE, TRUE, FALSE)`, userID, hash)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO document_permissions (kind_code, subject_type, subject_key, action, is_allowed)
		VALUES ('outgoing_letter', 'user', $1, 'read', TRUE)`, userID.String())
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO nomenclature (id, name, index, year, kind_code, separator, numbering_mode)
		VALUES ($1, 'Outgoing', 'OUT', 2026, 'outgoing_letter', '/', 'index_and_number')`, nomenclatureID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO organizations (id, name) VALUES ($1, 'Document Query Organization')`, organizationID)
	require.NoError(t, err)
	documentRepo := repository.NewOutgoingDocumentRepository(db)
	documentRepo.SetOutbox(repository.NewOutboxRepository(db))
	document, err := documentRepo.CreateWithJournal(models.CreateOutgoingDocRequest{
		NomenclatureID: nomenclatureID, IdempotencyKey: uuid.New(), DocumentTypeID: models.DocumentTypeLetter,
		RecipientOrgID: organizationID, CreatedBy: userID, OutgoingDate: time.Now().UTC(), Content: "server query integration",
		PagesCount: 1, SenderSignatory: "Signer", SenderExecutor: "Executor", Addressee: "Addressee",
	}, "CREATE", "Created %s")
	require.NoError(t, err)

	api := newIntegrationManagementAPI(t, &App{
		db: db, cfg: &config.Config{Server: config.ServerConfig{SessionTTLHours: 12}},
		metrics: observability.NewRegistry(32),
	})
	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"document-reader","password":"`+password+`"}`))
	loginResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(loginResponse, login)
	require.Equal(t, http.StatusOK, loginResponse.Code, loginResponse.Body.String())
	var session struct {
		AccessToken string `json:"accessToken"`
	}
	require.NoError(t, json.NewDecoder(loginResponse.Body).Decode(&session))

	list := httptest.NewRequest(http.MethodPost, "/api/v1/documents/query", strings.NewReader(`{"kindCode":"outgoing_letter","filter":{"search":"server query","page":1,"pageSize":20}}`))
	list.Header.Set("Authorization", "Bearer "+session.AccessToken)
	listResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(listResponse, list)
	require.Equal(t, http.StatusOK, listResponse.Code, listResponse.Body.String())
	require.Contains(t, listResponse.Body.String(), document.ID.String())

	card := httptest.NewRequest(http.MethodGet, "/api/v1/documents/"+document.ID.String(), nil)
	card.Header.Set("Authorization", "Bearer "+session.AccessToken)
	cardResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(cardResponse, card)
	require.Equal(t, http.StatusOK, cardResponse.Code, cardResponse.Body.String())
	require.Contains(t, cardResponse.Body.String(), `"content":"server query integration"`)
}

func TestDocumentSearchAPIIntegration(t *testing.T) {
	db := database.Wrap(integrationdb.Open(t))
	userID, nomID, orgID := uuid.New(), uuid.New(), uuid.New()
	hash, err := security.HashPassword("SearchPassw0rd!")
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO users (id, login, password_hash, last_name, first_name, no_patronymic, is_active, password_change_required)
        VALUES ($1, 'search-reader', $2, 'Search', 'Reader', TRUE, TRUE, FALSE)`, userID, hash)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO document_permissions (kind_code, subject_type, subject_key, action, is_allowed)
        VALUES ('outgoing_letter', 'user', $1, 'read', TRUE)`, userID.String())
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO nomenclature (id, name, index, year, kind_code, separator, numbering_mode)
        VALUES ($1, 'Search', 'S', 2026, 'outgoing_letter', '/', 'index_and_number')`, nomID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO organizations (id, name) VALUES ($1, 'Администрация')`, orgID)
	require.NoError(t, err)
	repo := repository.NewOutgoingDocumentRepository(db)
	repo.SetOutbox(repository.NewOutboxRepository(db))
	document, err := repo.CreateWithJournal(models.CreateOutgoingDocRequest{
		NomenclatureID: nomID, IdempotencyKey: uuid.New(), DocumentTypeID: models.DocumentTypeLetter,
		RecipientOrgID: orgID, CreatedBy: userID, OutgoingDate: time.Now().UTC(), Content: "О ремонте дороги",
		PagesCount: 1, SenderSignatory: "Иванов", SenderExecutor: "Сидоров", Addressee: "Петров",
	}, "CREATE", "Created %s")
	require.NoError(t, err)
	api := newIntegrationManagementAPI(t, &App{db: db, cfg: &config.Config{Server: config.ServerConfig{SessionTTLHours: 12}}, metrics: observability.NewRegistry(32)})
	login := httptest.NewRecorder()
	api.Handler().ServeHTTP(login, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"search-reader","password":"SearchPassw0rd!"}`)))
	require.Equal(t, http.StatusOK, login.Code, login.Body.String())
	var session struct {
		AccessToken string `json:"accessToken"`
	}
	require.NoError(t, json.NewDecoder(login.Body).Decode(&session))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/documents/search", strings.NewReader(`{"query":"ремонт администрация Иванов Петров","page":1,"pageSize":20}`))
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)
	recorder := httptest.NewRecorder()
	api.Handler().ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Contains(t, recorder.Body.String(), document.ID.String())
	require.Contains(t, recorder.Body.String(), `"totalCount":1`)
}
