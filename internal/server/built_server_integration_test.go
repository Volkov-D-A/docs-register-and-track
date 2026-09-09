package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
	"github.com/Volkov-D-A/docs-register-and-track/internal/testutil/integrationdb"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Invoked by integration-smoke.sh only after building and starting the real server.
func TestBuiltServerAttachmentsIntegration(t *testing.T) {
	base := os.Getenv("DOCFLOW_INTEGRATION_SERVER_URL")
	if base == "" {
		t.Skip("run make storage-smoke-test for the built-server and backup/restore smoke")
	}
	dsn := os.Getenv("DOCFLOW_INTEGRATION_DSN")
	require.NoError(t, integrationdb.ValidateDSN(dsn))
	sqlDB, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer sqlDB.Close()
	db := &database.DB{DB: sqlDB}
	request := func(method, path string, body io.Reader, token string) *http.Response {
		req, err := http.NewRequest(method, base+path, body)
		require.NoError(t, err)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if method == http.MethodPost && strings.HasSuffix(path, "/attachments") {
			req.Header.Set("Content-Disposition", `attachment; filename="report.pdf"`)
		}
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		t.Cleanup(func() { resp.Body.Close() })
		return resp
	}
	login := func() string {
		resp := request(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"attachment-user","password":"AttachmentPassw0rd!"}`), "")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var session struct {
			AccessToken string `json:"accessToken"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&session))
		return session.AccessToken
	}
	content := []byte("%PDF built server persistence and backup smoke\n")
	if os.Getenv("DOCFLOW_SMOKE_VERIFY") == "1" {
		token := login()
		var attachmentID string
		require.NoError(t, db.QueryRow(`SELECT id FROM attachments WHERE filename='report.pdf' AND deletion_requested_at IS NULL`).Scan(&attachmentID))
		resp := request(http.MethodGet, "/api/v1/attachments/"+attachmentID+"/content", nil, token)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, content, body)
		resp = request(http.MethodGet, "/api/v1/admin/attachments/reconciliation", nil, token)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var result models.AttachmentStorageReconciliation
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		require.Empty(t, result.MissingObjects)
		require.Empty(t, result.OrphanObjects)
		return
	}
	password := "AttachmentPassw0rd!"
	hash, err := security.HashPassword(password)
	require.NoError(t, err)
	userID, nomenclatureID, organizationID := uuid.New(), uuid.New(), uuid.New()
	_, err = db.Exec(`INSERT INTO users (id, login, password_hash, full_name, is_document_participant, is_active, password_change_required)
		VALUES ($1, 'attachment-user', $2, 'Attachment User', TRUE, TRUE, FALSE)`, userID, hash)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO document_permissions (kind_code, subject_type, subject_key, action, is_allowed)
		VALUES ('outgoing_letter', 'user', $1, 'upload', TRUE),
		       ('outgoing_letter', 'user', $1, 'read', TRUE)`, userID.String())
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO nomenclature (id, name, index, year, kind_code, separator, numbering_mode)
		VALUES ($1, 'Outgoing', 'OUT', 2026, 'outgoing_letter', '/', 'index_and_number')`, nomenclatureID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO organizations (id, name) VALUES ($1, 'Attachment Organization')`, organizationID)
	require.NoError(t, err)
	document, err := repository.NewOutgoingDocumentRepository(db).Create(models.CreateOutgoingDocRequest{
		NomenclatureID: nomenclatureID, IdempotencyKey: uuid.New(), DocumentTypeID: models.DocumentTypeLetter,
		RecipientOrgID: organizationID, CreatedBy: userID, OutgoingDate: time.Now().UTC(), Content: "attachment integration",
		PagesCount: 1, SenderSignatory: "Signer", SenderExecutor: "Executor", Addressee: "Addressee",
	})
	require.NoError(t, err)
	allowed, err := repository.NewDocumentAccessRepository(db).HasPermission(string(models.DocumentKindOutgoingLetter), "upload", "", userID.String())
	require.NoError(t, err)
	require.True(t, allowed)

	_, err = db.Exec(`INSERT INTO user_system_permissions(user_id,permission,is_allowed) VALUES($1,'admin',TRUE)`, userID)
	require.NoError(t, err)
	token := login()
	upload := func() string {
		resp := request(http.MethodPost, "/api/v1/documents/"+document.ID.String()+"/attachments", bytes.NewReader(content), token)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var result struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		return result.ID
	}
	id := upload()
	resp := request(http.MethodGet, "/api/v1/attachments/"+id+"/content", nil, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, content, body)
	resp = request(http.MethodDelete, "/api/v1/attachments/"+id, nil, token)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.Eventually(t, func() bool {
		var n int
		err := db.QueryRow(`SELECT COUNT(*) FROM attachments WHERE id=$1`, id).Scan(&n)
		return err == nil && n == 0
	}, 10*time.Second, 100*time.Millisecond)
	upload() // Leave a real attachment for restart and backup/restore verification.
}
