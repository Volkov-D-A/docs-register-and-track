package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

type attachmentIntegrationStorage struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func (s *attachmentIntegrationStorage) UploadFile(_ context.Context, name string, reader io.Reader, _ int64, _ string) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[name] = data
	return nil
}
func (s *attachmentIntegrationStorage) DownloadFileToWriter(_ context.Context, name string, writer io.Writer, _ int64) error {
	s.mu.Lock()
	data := append([]byte(nil), s.objects[name]...)
	s.mu.Unlock()
	_, err := writer.Write(data)
	return err
}
func (s *attachmentIntegrationStorage) DeleteFile(_ context.Context, name string) error {
	s.mu.Lock()
	delete(s.objects, name)
	s.mu.Unlock()
	return nil
}
func (s *attachmentIntegrationStorage) RefreshStorageUsage(context.Context) (int, int64, error) {
	return 0, 0, nil
}
func (s *attachmentIntegrationStorage) ListObjectNames(context.Context) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]string, 0, len(s.objects))
	for name := range s.objects {
		result = append(result, name)
	}
	return result, nil
}

func TestAttachmentAPIStreamsAndPersistsLifecycleIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}
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

	storage := &attachmentIntegrationStorage{objects: make(map[string][]byte)}
	api := newManagementAPI(&App{db: db, cfg: &config.Config{Server: config.ServerConfig{SessionTTLHours: 12}}, metrics: observability.NewRegistry(32), storage: storage})
	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"attachment-user","password":"`+password+`"}`))
	loginResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(loginResponse, login)
	require.Equal(t, http.StatusOK, loginResponse.Code, loginResponse.Body.String())
	var session struct {
		AccessToken string `json:"accessToken"`
	}
	require.NoError(t, json.NewDecoder(loginResponse.Body).Decode(&session))

	content := []byte("%PDF integration")
	upload := httptest.NewRequest(http.MethodPost, "/api/v1/documents/"+document.ID.String()+"/attachments", bytes.NewReader(content))
	upload.Header.Set("Authorization", "Bearer "+session.AccessToken)
	upload.Header.Set("Content-Disposition", `attachment; filename="report.pdf"`)
	uploadResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(uploadResponse, upload)
	require.Equal(t, http.StatusCreated, uploadResponse.Code, uploadResponse.Body.String())
	var uploaded struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(uploadResponse.Body).Decode(&uploaded))
	require.NotEmpty(t, uploaded.ID)

	download := httptest.NewRequest(http.MethodGet, "/api/v1/attachments/"+uploaded.ID+"/content", nil)
	download.Header.Set("Authorization", "Bearer "+session.AccessToken)
	downloadResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(downloadResponse, download)
	require.Equal(t, http.StatusOK, downloadResponse.Code)
	require.Equal(t, content, downloadResponse.Body.Bytes())

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/attachments/"+uploaded.ID, nil)
	deleteRequest.Header.Set("Authorization", "Bearer "+session.AccessToken)
	deleteResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(deleteResponse, deleteRequest)
	require.Equal(t, http.StatusNoContent, deleteResponse.Code, deleteResponse.Body.String())
	afterDelete := httptest.NewRequest(http.MethodGet, "/api/v1/attachments/"+uploaded.ID+"/content", nil)
	afterDelete.Header.Set("Authorization", "Bearer "+session.AccessToken)
	afterDeleteResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(afterDeleteResponse, afterDelete)
	require.Equal(t, http.StatusNotFound, afterDeleteResponse.Code, afterDeleteResponse.Body.String())

	var deletionRequested bool
	require.NoError(t, db.QueryRow(`SELECT deletion_requested_at IS NOT NULL FROM attachments WHERE id=$1`, uuid.MustParse(uploaded.ID)).Scan(&deletionRequested))
	require.True(t, deletionRequested)
	var outboxEvents int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM event_outbox WHERE processed_at IS NULL AND event_type IN ($1, $2)`, models.OutboxEventFileDelete, models.OutboxEventJournal).Scan(&outboxEvents))
	require.GreaterOrEqual(t, outboxEvents, 3)
}
