package server

import (
	"encoding/json"
	"fmt"
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
	"github.com/Volkov-D-A/docs-register-and-track/internal/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
	"github.com/Volkov-D-A/docs-register-and-track/internal/testutil/integrationdb"
)

func TestNomenclatureAPIPersistsCRUDWithAuditOutboxIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}
	outbox := repository.NewOutboxRepository(db)
	users := repository.NewUserRepository(db)
	users.SetOutbox(outbox)
	nomenclature := repository.NewNomenclatureRepository(db)
	nomenclature.SetOutbox(outbox)
	sessions := repository.NewServerSessionRepository(db)
	hash, err := security.HashPassword("NomenclaturePassw0rd!")
	require.NoError(t, err)
	userID := uuid.New()
	_, err = db.Exec(`INSERT INTO users (id, login, password_hash, full_name, is_active, password_change_required)
		VALUES ($1, 'nomenclature-admin', $2, 'Nomenclature Admin', TRUE, FALSE)`, userID, hash)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_system_permissions (user_id, permission, is_allowed) VALUES ($1, $2, TRUE)`, userID, models.SystemPermissionAdmin)
	require.NoError(t, err)

	api := &managementAPI{
		cfg:       &config.Config{Server: config.ServerConfig{SessionTTLHours: 12}},
		authUsers: users, authSettings: repository.NewSettingsRepository(db), sessions: sessions,
		nomenclature: nomenclature,
	}
	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"nomenclature-admin","password":"NomenclaturePassw0rd!"}`))
	loginResult := httptest.NewRecorder()
	api.Handler().ServeHTTP(loginResult, login)
	require.Equal(t, http.StatusOK, loginResult.Code, loginResult.Body.String())
	var loginBody struct {
		AccessToken string `json:"accessToken"`
	}
	require.NoError(t, json.NewDecoder(loginResult.Body).Decode(&loginBody))
	token := loginBody.AccessToken
	year := time.Now().Year()

	createBody := fmt.Sprintf(`{"name":"Incoming","index":"01-01","year":%d,"kindCode":"incoming_letter","separator":"/","numberingMode":"index_and_number","startNumber":0}`, year)
	create := httptest.NewRequest(http.MethodPost, "/api/v1/nomenclature", strings.NewReader(createBody))
	create.Header.Set("Authorization", "Bearer "+token)
	createResult := httptest.NewRecorder()
	api.Handler().ServeHTTP(createResult, create)
	require.Equal(t, http.StatusCreated, createResult.Code, createResult.Body.String())
	var created struct {
		ID         string `json:"id"`
		NextNumber int    `json:"nextNumber"`
	}
	require.NoError(t, json.NewDecoder(createResult.Body).Decode(&created))
	require.Equal(t, 1, created.NextNumber)

	active := httptest.NewRequest(http.MethodGet, "/api/v1/nomenclature/active?kindCode=incoming_letter", nil)
	active.Header.Set("Authorization", "Bearer "+token)
	activeResult := httptest.NewRecorder()
	api.Handler().ServeHTTP(activeResult, active)
	require.Equal(t, http.StatusOK, activeResult.Code, activeResult.Body.String())
	require.Contains(t, activeResult.Body.String(), created.ID)

	updateBody := fmt.Sprintf(`{"name":"Incoming updated","index":"01-02","year":%d,"kindCode":"incoming_letter","separator":"-","numberingMode":"number_only","isActive":false}`, year)
	update := httptest.NewRequest(http.MethodPatch, "/api/v1/nomenclature/"+created.ID, strings.NewReader(updateBody))
	update.Header.Set("Authorization", "Bearer "+token)
	updateResult := httptest.NewRecorder()
	api.Handler().ServeHTTP(updateResult, update)
	require.Equal(t, http.StatusOK, updateResult.Code, updateResult.Body.String())

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/nomenclature/"+created.ID, nil)
	deleteRequest.Header.Set("Authorization", "Bearer "+token)
	deleteResult := httptest.NewRecorder()
	api.Handler().ServeHTTP(deleteResult, deleteRequest)
	require.Equal(t, http.StatusNoContent, deleteResult.Code, deleteResult.Body.String())

	var rows, effects int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM nomenclature WHERE id=$1`, uuid.MustParse(created.ID)).Scan(&rows))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM event_outbox WHERE processed_at IS NULL AND event_type=$1`, models.OutboxEventAudit).Scan(&effects))
	require.Zero(t, rows)
	require.Equal(t, 3, effects)
}
