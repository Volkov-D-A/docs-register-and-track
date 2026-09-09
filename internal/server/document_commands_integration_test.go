package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
	"github.com/Volkov-D-A/docs-register-and-track/internal/testutil/integrationdb"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDocumentCommandAPIAllKindsIntegration(t *testing.T) {
	db := &database.DB{DB: integrationdb.Open(t)}
	hash, err := security.HashPassword("DocumentCommandPassw0rd!")
	require.NoError(t, err)
	userID := uuid.New()
	_, err = db.Exec(`INSERT INTO users (id,login,password_hash,full_name,is_active,password_change_required) VALUES ($1,'command-user',$2,'Command User',TRUE,FALSE)`, userID, hash)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_system_permissions (user_id,permission,is_allowed) VALUES ($1,'admin',TRUE)`, userID)
	require.NoError(t, err)
	api := newIntegrationManagementAPI(t, &App{db: db, cfg: &config.Config{Server: config.ServerConfig{SessionTTLHours: 12}}, metrics: observability.NewRegistry(32)})
	token := ""
	request := func(method, path, key string, payload any) *httptest.ResponseRecorder {
		data, err := json.Marshal(payload)
		require.NoError(t, err)
		req := httptest.NewRequest(method, path, bytes.NewReader(data))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Idempotency-Key", key)
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, req)
		return response
	}
	login := request(http.MethodPost, "/api/v1/auth/login", "", map[string]any{"login": "command-user", "password": "DocumentCommandPassw0rd!"})
	require.Equal(t, http.StatusOK, login.Code, login.Body.String())
	var session struct {
		AccessToken string `json:"accessToken"`
	}
	require.NoError(t, json.Unmarshal(login.Body.Bytes(), &session))
	token = session.AccessToken
	cases := []struct {
		kind    string
		payload map[string]any
		number  string
	}{
		{"incoming_letter", map[string]any{"incomingDate": "2026-09-09", "documentTypeId": models.DocumentTypeLetter, "content": "original", "pagesCount": 1, "correspondents": []map[string]any{{"correspondentName": "Sender", "registrationNumber": "SRC-1", "registrationDate": "2026-09-08"}}}, "incomingNumber"},
		{"outgoing_letter", map[string]any{"outgoingDate": "2026-09-09", "documentTypeId": models.DocumentTypeLetter, "content": "original", "pagesCount": 1, "recipientOrgName": "Recipient", "senderSignatory": "Signer", "senderExecutor": "Executor"}, "outgoingNumber"},
		{"citizen_appeal", map[string]any{"registrationDate": "2026-09-09", "appealDate": "2026-09-08", "content": "original", "pagesCount": 1, "applicantFullName": "Citizen", "registrationAddress": "Address", "appealType": "жалоба", "applicantCategory": "гражданин"}, "registrationNumber"},
		{"administrative_order", map[string]any{"orderDate": "2026-09-09", "title": "original", "pagesCount": 1, "executionController": "Controller", "isActive": true}, "orderNumber"},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			nomID := uuid.New()
			_, err = db.Exec(`INSERT INTO nomenclature (id,name,index,year,kind_code,separator,numbering_mode) VALUES ($1,$2,'CMD',2026,$2,'/','index_and_number')`, nomID, tc.kind)
			require.NoError(t, err)
			_, err = db.Exec(`INSERT INTO document_permissions (kind_code,subject_type,subject_key,action,is_allowed) SELECT $1,'user',$2,action,TRUE FROM unnest(ARRAY['create','read','update']) action`, tc.kind, userID.String())
			require.NoError(t, err)
			tc.payload["nomenclatureId"] = nomID.String()
			path := "/api/v1/documents/" + tc.kind
			key := uuid.NewString()
			created := request(http.MethodPost, path, key, tc.payload)
			require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
			var document map[string]any
			require.NoError(t, json.Unmarshal(created.Body.Bytes(), &document))
			require.Equal(t, "CMD/1", document[tc.number])
			repeated := request(http.MethodPost, path, key, tc.payload)
			require.Equal(t, http.StatusCreated, repeated.Code, repeated.Body.String())
			require.JSONEq(t, created.Body.String(), repeated.Body.String())
			update := map[string]any{}
			for k, v := range tc.payload {
				if k != "nomenclatureId" && !(tc.kind == "incoming_letter" && k == "incomingDate") {
					update[k] = v
				}
			}
			if tc.kind == "citizen_appeal" {
				update["registrationNumber"] = document[tc.number]
			}
			if tc.kind == "administrative_order" {
				update["title"] = "updated"
			} else {
				update["content"] = "updated"
			}
			changed := request(http.MethodPatch, path+"/"+document["id"].(string), uuid.NewString(), update)
			require.Equal(t, http.StatusOK, changed.Code, changed.Body.String())
			require.Contains(t, changed.Body.String(), "updated")
			var effects int
			require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM event_outbox WHERE COALESCE(payload->>'documentId', payload->>'DocumentID')=$1 AND COALESCE(payload->>'action', payload->>'Action') IN ('CREATE','UPDATE')`, document["id"]).Scan(&effects))
			require.Equal(t, 2, effects, "registration replay must not duplicate journal effects")
			draft := map[string]any{"nomenclatureId": nomID.String(), "registrationDate": "2026-09-09", "adminNumberOverride": map[string]any{"mode": "literal", "number": 7, "suffix": "а"}}
			drafted := request(http.MethodPost, path+"/admin-drafts", uuid.NewString(), draft)
			require.Equal(t, http.StatusCreated, drafted.Code, drafted.Body.String())
			require.Contains(t, drafted.Body.String(), "CMD/7а")
			_, err = db.Exec(`DELETE FROM document_permissions WHERE kind_code=$1 AND subject_key=$2`, tc.kind, userID.String())
			require.NoError(t, err)
			denied := request(http.MethodPost, path, uuid.NewString(), tc.payload)
			require.Equal(t, http.StatusForbidden, denied.Code, denied.Body.String())
		})
	}
}
