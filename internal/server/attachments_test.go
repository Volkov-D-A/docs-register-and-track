package server

import (
	"bytes"
	"context"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type fakeAttachmentAPI struct {
	uploadedFilename string
	uploadedContent  string
	uploadedDocument string
	maxSize          int64
	download         *models.Attachment
}

func (f *fakeAttachmentAPI) MaxUploadSize() int64 { return f.maxSize }
func (f *fakeAttachmentAPI) UploadContent(documentID string, _ *uuid.UUID, filename string, _ int64, body io.Reader) (*dto.Attachment, error) {
	data, err := io.ReadAll(body)
	f.uploadedDocument, f.uploadedFilename, f.uploadedContent = documentID, filename, string(data)
	return &dto.Attachment{ID: uuid.NewString(), DocumentID: documentID, Filename: filename}, err
}
func (f *fakeAttachmentAPI) UploadAssignmentContent(string, string, int64, io.Reader) (*dto.Attachment, error) {
	return &dto.Attachment{}, nil
}
func (*fakeAttachmentAPI) GetList(string) ([]dto.Attachment, error) { return []dto.Attachment{}, nil }
func (*fakeAttachmentAPI) GetAssignmentFiles(string) ([]dto.Attachment, error) {
	return []dto.Attachment{}, nil
}
func (f *fakeAttachmentAPI) AuthorizeDownload(string) (*models.Attachment, error) {
	return f.download, nil
}
func (*fakeAttachmentAPI) StreamAttachment(_ context.Context, _ *models.Attachment, writer io.Writer) error {
	_, err := io.WriteString(writer, "file")
	return err
}
func (*fakeAttachmentAPI) Delete(string) error                     { return nil }
func (*fakeAttachmentAPI) BulkDeleteOlderThan(string) (int, error) { return 2, nil }
func (*fakeAttachmentAPI) ReconcileStorage() (*models.AttachmentStorageReconciliation, error) {
	return &models.AttachmentStorageReconciliation{MissingObjects: []string{"missing"}}, nil
}

func TestAttachmentAPIStreamsAuthenticatedUpload(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	service := &fakeAttachmentAPI{maxSize: 4}
	api.attachments = func(*models.User) attachmentAPI { return service }
	documentID := uuid.NewString()

	unauthorized := httptest.NewRecorder()
	api.Handler().ServeHTTP(unauthorized, httptest.NewRequest(http.MethodPost, "/api/v1/documents/"+documentID+"/attachments", bytes.NewReader([]byte("abc"))))
	assert.Equal(t, http.StatusUnauthorized, unauthorized.Code)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/documents/"+documentID+"/attachments", bytes.NewReader([]byte("abc")))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": "отчёт.txt"}))
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	assert.Equal(t, documentID, service.uploadedDocument)
	assert.Equal(t, "отчёт.txt", service.uploadedFilename)
	assert.Equal(t, "abc", service.uploadedContent)
}

func TestAttachmentAPIRejectsBodyAboveApplicationLimit(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	service := &fakeAttachmentAPI{maxSize: 3}
	api.attachments = func(*models.User) attachmentAPI { return service }
	request := httptest.NewRequest(http.MethodPost, "/api/v1/documents/"+uuid.NewString()+"/attachments", bytes.NewReader([]byte("four")))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Disposition", `attachment; filename="file.txt"`)
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, request)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, service.uploadedContent)
}

func TestAttachmentAPIDownloadUsesSafeHeadersAndStreamsBody(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	service := &fakeAttachmentAPI{maxSize: 10, download: &models.Attachment{ID: uuid.New(), Filename: "report.txt", FileSize: 4, ContentType: "text/plain"}}
	api.attachments = func(*models.User) attachmentAPI { return service }
	request := httptest.NewRequest(http.MethodGet, "/api/v1/attachments/"+service.download.ID.String()+"/content", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "file", response.Body.String())
	assert.Equal(t, "text/plain", response.Header().Get("Content-Type"))
	_, params, err := mime.ParseMediaType(response.Header().Get("Content-Disposition"))
	require.NoError(t, err)
	assert.Equal(t, "report.txt", params["filename"])
}

func TestAttachmentAdminAPIsRequireAdminPermission(t *testing.T) {
	regularAPI, _, regularToken := authenticatedUserAPI(t, nil)
	regularAPI.attachments = func(*models.User) attachmentAPI { return &fakeAttachmentAPI{maxSize: 10} }
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/attachments/reconciliation", nil)
	request.Header.Set("Authorization", "Bearer "+regularToken)
	response := httptest.NewRecorder()
	regularAPI.Handler().ServeHTTP(response, request)
	assert.Equal(t, http.StatusForbidden, response.Code)

	adminAPI, _, adminToken := authenticatedUserAPI(t, []string{models.SystemPermissionAdmin})
	adminAPI.attachments = func(*models.User) attachmentAPI { return &fakeAttachmentAPI{maxSize: 10} }
	request = httptest.NewRequest(http.MethodDelete, "/api/v1/admin/attachments?before=2026-01-01T00:00:00Z", nil)
	request.Header.Set("Authorization", "Bearer "+adminToken)
	response = httptest.NewRecorder()
	adminAPI.Handler().ServeHTTP(response, request)
	assert.Equal(t, http.StatusOK, response.Code)
}
