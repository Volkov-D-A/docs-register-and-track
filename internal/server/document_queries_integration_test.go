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

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
	"github.com/Volkov-D-A/docs-register-and-track/internal/testutil/integrationdb"
)

func TestDocumentQueryAPIReturnsListAndCardWithServerAccessIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}
	password := "DocumentReadPassw0rd!"
	hash, err := security.HashPassword(password)
	require.NoError(t, err)
	userID, nomenclatureID, organizationID := uuid.New(), uuid.New(), uuid.New()
	_, err = db.Exec(`INSERT INTO users (id, login, password_hash, full_name, is_active, password_change_required)
		VALUES ($1, 'document-reader', $2, 'Document Reader', TRUE, FALSE)`, userID, hash)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO document_permissions (kind_code, subject_type, subject_key, action, is_allowed)
		VALUES ('outgoing_letter', 'user', $1, 'read', TRUE)`, userID.String())
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO nomenclature (id, name, index, year, kind_code, separator, numbering_mode)
		VALUES ($1, 'Outgoing', 'OUT', 2026, 'outgoing_letter', '/', 'index_and_number')`, nomenclatureID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO organizations (id, name) VALUES ($1, 'Document Query Organization')`, organizationID)
	require.NoError(t, err)
	document, err := repository.NewOutgoingDocumentRepository(db).Create(models.CreateOutgoingDocRequest{
		NomenclatureID: nomenclatureID, IdempotencyKey: uuid.New(), DocumentTypeID: models.DocumentTypeLetter,
		RecipientOrgID: organizationID, CreatedBy: userID, OutgoingDate: time.Now().UTC(), Content: "server query integration",
		PagesCount: 1, SenderSignatory: "Signer", SenderExecutor: "Executor", Addressee: "Addressee",
	})
	require.NoError(t, err)

	api := newManagementAPI(&App{
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
