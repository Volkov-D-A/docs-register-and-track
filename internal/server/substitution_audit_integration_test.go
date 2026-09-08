package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/outbox"
	"github.com/Volkov-D-A/docs-register-and-track/internal/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
	"github.com/Volkov-D-A/docs-register-and-track/internal/testutil/integrationdb"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRepeatedSubstitutionTransitionsAuditIntegration(t *testing.T) {
	db := &database.DB{DB: integrationdb.Open(t)}
	department, adminID, principalID, substituteID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	hash, err := security.HashPassword("AuditPassw0rd!")
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO departments(id,name) VALUES ($1,'Substitution Audit')`, department)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO users(id,login,password_hash,full_name,is_active,is_document_participant,department_id,password_change_required) VALUES
 ($1,'audit-admin',$4,'Admin',TRUE,FALSE,$5,FALSE),($2,'audit-principal',$4,'Principal',TRUE,TRUE,$5,FALSE),($3,'audit-substitute',$4,'Substitute',TRUE,FALSE,$5,FALSE)`, adminID, principalID, substituteID, hash, department)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_system_permissions(user_id,permission,is_allowed) VALUES($1,'admin',TRUE)`, adminID)
	require.NoError(t, err)
	api := newIntegrationManagementAPI(t, &App{db: db, cfg: validConfig()})
	handler := api.Handler()
	for _, tc := range []struct {
		login, path, action string
		actor               uuid.UUID
	}{
		{"audit-admin", "/api/v1/users/" + principalID.String() + "/substitution", "USER_SUBSTITUTION_UPDATE", adminID},
		{"audit-principal", "/api/v1/profile/substitution", "USER_SUBSTITUTION_SELF_UPDATE", principalID},
	} {
		t.Run(tc.login, func(t *testing.T) {
			login := httptest.NewRecorder()
			handler.ServeHTTP(login, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"login":"`+tc.login+`","password":"AuditPassw0rd!"}`)))
			require.Equal(t, http.StatusOK, login.Code, login.Body.String())
			var session struct {
				AccessToken string `json:"accessToken"`
			}
			require.NoError(t, json.Unmarshal(login.Body.Bytes(), &session))
			for _, substitute := range []string{substituteID.String(), "", substituteID.String(), ""} {
				payload, err := json.Marshal(models.UpdateUserSubstitutionRequest{PrincipalUserID: uuid.NewString(), SubstituteUserID: substitute, IsActive: true})
				require.NoError(t, err)
				request := httptest.NewRequest(http.MethodPut, tc.path, bytes.NewReader(payload))
				request.Header.Set("Authorization", "Bearer "+session.AccessToken)
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				require.Equal(t, http.StatusOK, response.Code, response.Body.String())
				item, err := repository.NewUserSubstitutionRepository(db).GetByPrincipalID(principalID)
				require.NoError(t, err)
				if substitute == "" {
					require.Nil(t, item)
				} else {
					require.NotNil(t, item)
					require.Equal(t, substituteID, item.SubstituteUserID)
				}
			}
			var effects, keys int
			require.NoError(t, db.QueryRow(`SELECT COUNT(*), COUNT(DISTINCT deduplication_key) FROM event_outbox WHERE event_type=$1 AND payload->>'Action'=$2`, models.OutboxEventAudit, tc.action).Scan(&effects, &keys))
			require.Equal(t, 4, effects)
			require.Equal(t, 4, keys)
			worker := outbox.NewWorker(repository.NewOutboxRepository(db), nil, nil, repository.NewAdminAuditLogRepository(db), nil, nil)
			require.NoError(t, worker.ProcessOnce())
			var auditCount int
			require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM admin_audit_log WHERE action=$1 AND user_id=$2`, tc.action, tc.actor).Scan(&auditCount))
			require.Equal(t, 4, auditCount)
		})
	}
}
