package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
	"github.com/stretchr/testify/require"
)

func TestUserNameSetupAndProfileIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := database.Wrap(sqlDB)
	users := repository.NewUserRepository(db)
	users.SetOutbox(repository.NewOutboxRepository(db))
	api := &managementAPI{cfg: &config.Config{Server: config.ServerConfig{SessionTTLHours: 12}}, initialSetup: users, authUsers: users, authSettings: repository.NewSettingsRepository(db), sessions: repository.NewServerSessionRepository(db), audit: repository.NewAdminAuditLogRepository(db), userCommands: users}
	request := func(method, path, body, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		result := httptest.NewRecorder()
		api.Handler().ServeHTTP(result, req)
		return result
	}
	setup := request(http.MethodPost, "/api/v1/auth/setup", `{"password":"AdminPassw0rd!","lastName":" Иванов ","firstName":" Иван ","patronymic":" Иванович "}`, "")
	require.Equal(t, http.StatusNoContent, setup.Code, setup.Body.String())
	admin, err := users.GetByLogin("admin")
	require.NoError(t, err)
	require.NotNil(t, admin)
	require.Equal(t, "Иванов Иван Иванович", admin.FullName())
	require.Contains(t, admin.SystemPermissions, models.SystemPermissionAdmin)
	require.False(t, admin.NoPatronymic)
	login := request(http.MethodPost, "/api/v1/auth/login", `{"login":"admin","password":"AdminPassw0rd!"}`, "")
	require.Equal(t, http.StatusOK, login.Code, login.Body.String())
	var session struct {
		AccessToken string   `json:"accessToken"`
		User        dto.User `json:"user"`
	}
	require.NoError(t, json.Unmarshal(login.Body.Bytes(), &session))
	require.Equal(t, "Иванович", session.User.Patronymic)
	longPart := strings.Repeat("Я", 100)
	longName := longPart + " " + longPart + " " + longPart
	for _, tc := range []struct {
		body, want string
		none       bool
	}{
		{`{"login":"admin","lastName":" де ла Крус ","firstName":" Анна-Мария ","patronymic":"must clear","noPatronymic":true}`, "де ла Крус Анна-Мария", true},
		{`{"login":"admin","lastName":"Иванов","firstName":"Иван","patronymic":" Иванович ","noPatronymic":false}`, "Иванов Иван Иванович", false},
		{`{"login":"admin","lastName":"` + longPart + `","firstName":"` + longPart + `","patronymic":"` + longPart + `"}`, longName, false},
	} {
		result := request(http.MethodPatch, "/api/v1/profile", tc.body, session.AccessToken)
		require.Equal(t, http.StatusOK, result.Code, result.Body.String())
		var updated dto.User
		require.NoError(t, json.Unmarshal(result.Body.Bytes(), &updated))
		require.Equal(t, tc.want, updated.FullName)
		require.Equal(t, tc.none, updated.NoPatronymic)
		stored, err := users.GetByID(admin.ID)
		require.NoError(t, err)
		require.Equal(t, tc.want, stored.FullName())
		require.Equal(t, updated.Patronymic, stored.Patronymic)
		var sqlName string
		require.NoError(t, db.QueryRow(`SELECT full_name FROM users WHERE id=$1`, admin.ID).Scan(&sqlName))
		require.Equal(t, stored.FullName(), sqlName)
		me := request(http.MethodGet, "/api/v1/auth/me", "", session.AccessToken)
		require.Equal(t, http.StatusOK, me.Code)
		var current dto.User
		require.NoError(t, json.Unmarshal(me.Body.Bytes(), &current))
		require.Equal(t, updated, current)
	}
	var audits int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM event_outbox WHERE event_type=$1`, models.OutboxEventAudit).Scan(&audits))
	require.Equal(t, 3, audits)
	login = request(http.MethodPost, "/api/v1/auth/login", `{"login":"admin","password":"AdminPassw0rd!"}`, "")
	require.Equal(t, http.StatusOK, login.Code, login.Body.String())
	var auditedName string
	require.NoError(t, db.QueryRow(`SELECT user_name FROM admin_audit_log WHERE action='LOGIN' ORDER BY created_at DESC LIMIT 1`).Scan(&auditedName))
	require.Equal(t, longName, auditedName)
	_, err = db.Exec(`UPDATE users SET full_name='independent copy' WHERE id=$1`, admin.ID)
	require.Error(t, err)
	_, err = db.Exec(`UPDATE users SET no_patronymic=true WHERE id=$1`, admin.ID)
	require.Error(t, err)
	_, err = db.Exec(`UPDATE users SET patronymic='' WHERE id=$1`, admin.ID)
	require.Error(t, err)
	_, err = db.Exec(`UPDATE users SET first_name='' WHERE id=$1`, admin.ID)
	require.Error(t, err)
	_, err = db.Exec(`UPDATE users SET last_name=$1 WHERE id=$2`, strings.Repeat("Я", 101), admin.ID)
	require.Error(t, err)
}
