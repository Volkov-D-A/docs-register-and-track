package server

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/security"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Invoked by the built-server persistence and backup/restore smoke scripts.
func TestBuiltServerControlsIntegration(t *testing.T) {
	base := os.Getenv("DOCFLOW_INTEGRATION_SERVER_URL")
	if base == "" {
		t.Skip("set DOCFLOW_INTEGRATION_SERVER_URL for the built-server smoke")
	}
	dsn := os.Getenv("DOCFLOW_INTEGRATION_DSN")
	require.NoError(t, integrationdb.ValidateDSN(dsn))
	sqlDB, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer sqlDB.Close()
	db := database.Wrap(sqlDB)
	request := func(method, path string, body io.Reader, token string) *http.Response {
		req, err := http.NewRequest(method, base+path, body)
		require.NoError(t, err)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		response, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
		require.NoError(t, err)
		t.Cleanup(func() { response.Body.Close() })
		return response
	}
	login := func() string {
		response := request(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"control-user","password":"ControlPassw0rd!"}`), "")
		require.Equal(t, http.StatusOK, response.StatusCode)
		var session struct {
			AccessToken string `json:"accessToken"`
		}
		require.NoError(t, json.NewDecoder(response.Body).Decode(&session))
		return session.AccessToken
	}
	if os.Getenv("DOCFLOW_SMOKE_VERIFY") != "1" {
		hash, err := security.HashPassword("ControlPassw0rd!")
		require.NoError(t, err)
		user, nom, org := uuid.New(), uuid.New(), uuid.New()
		_, err = db.Exec(`INSERT INTO users(id,login,password_hash,last_name,first_name,no_patronymic,is_document_participant,password_change_required) VALUES($1,'control-user',$2,'Control','User',true,true,false)`, user, hash)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO user_system_permissions(user_id,permission) VALUES($1,'admin')`, user)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO document_permissions(kind_code,subject_type,subject_key,action) VALUES('outgoing_letter','user',$1,'register'),('outgoing_letter','user',$1,'control_deadlines')`, user.String())
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO nomenclature(id,name,index,year,kind_code) VALUES($1,'Outgoing','OUT',2026,'outgoing_letter')`, nom)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO organizations(id,name) VALUES($1,'Control Organization')`, org)
		require.NoError(t, err)
		documents := repository.NewOutgoingDocumentRepository(db)
		documents.SetOutbox(repository.NewOutboxRepository(db))
		doc, err := documents.CreateWithJournal(models.CreateOutgoingDocRequest{NomenclatureID: nom, IdempotencyKey: uuid.New(), DocumentTypeID: models.DocumentTypeLetter, RecipientOrgID: org, CreatedBy: user, OutgoingDate: time.Now().UTC(), Content: "Control persistence", PagesCount: 1, SenderSignatory: "Signer", SenderExecutor: "Executor", Addressee: "Addressee"}, "CREATE", "Created %s")
		require.NoError(t, err)
		body, err := json.Marshal(models.DeadlineControlRequest{DocumentID: doc.ID.String(), ExecutorID: user.String(), Subject: "Проверить сохранность", DueDate: "2026-10-20"})
		require.NoError(t, err)
		response := request(http.MethodPost, "/api/v1/deadline-controls", strings.NewReader(string(body)), login())
		require.Equal(t, http.StatusCreated, response.StatusCode)
	}
	token := login()
	var id string
	require.NoError(t, db.QueryRow(`SELECT id::text FROM document_deadline_controls WHERE subject='Проверить сохранность'`).Scan(&id))
	response := request(http.MethodGet, "/api/v1/deadline-controls/"+id, nil, token)
	require.Equal(t, http.StatusOK, response.StatusCode)
	var control models.DocumentDeadlineControl
	require.NoError(t, json.NewDecoder(response.Body).Decode(&control))
	require.Equal(t, models.DeadlineActive, control.Status)
	require.Equal(t, "Проверить сохранность", control.Subject)
	require.True(t, control.CanManage)
	response = request(http.MethodGet, "/api/v1/documents/"+control.DocumentID+"/attachments", nil, token)
	require.Contains(t, []int{http.StatusNotFound, http.StatusMethodNotAllowed}, response.StatusCode)
}
