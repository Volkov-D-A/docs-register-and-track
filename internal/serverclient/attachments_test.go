package serverclient

import (
	"context"
	"io"
	"mime"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttachmentClientStreamsUploadAndDownload(t *testing.T) {
	documentID, attachmentID := uuid.NewString(), uuid.NewString()
	requestNumber := 0
	client := userClientWithToken(t, func(r *http.Request) (*http.Response, error) {
		requestNumber++
		assert.Equal(t, "Bearer session-token", r.Header.Get("Authorization"))
		switch requestNumber {
		case 1:
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/api/v1/documents/"+documentID+"/attachments", r.URL.Path)
			assert.Equal(t, int64(3), r.ContentLength)
			_, params, err := mime.ParseMediaType(r.Header.Get("Content-Disposition"))
			require.NoError(t, err)
			assert.Equal(t, "отчёт.txt", params["filename"])
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.Equal(t, "abc", string(body))
			return response(http.StatusCreated, `{"id":"`+attachmentID+`","filename":"отчёт.txt"}`), nil
		case 2:
			assert.Equal(t, "/api/v1/attachments/"+attachmentID+"/content", r.URL.Path)
			header := make(http.Header)
			header.Set("Content-Type", "text/plain")
			header.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": "отчёт.txt"}))
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("abc")), Header: header, ContentLength: 3}, nil
		default:
			t.Fatalf("unexpected request %d", requestNumber)
			return nil, nil
		}
	})

	uploaded, err := client.UploadAttachment(context.Background(), documentID, "", "отчёт.txt", 3, strings.NewReader("abc"))
	require.NoError(t, err)
	assert.Equal(t, attachmentID, uploaded.ID)

	metadata, content, err := client.GetAttachmentContent(context.Background(), attachmentID)
	require.NoError(t, err)
	defer content.Close()
	data, err := io.ReadAll(content)
	require.NoError(t, err)
	assert.Equal(t, "отчёт.txt", metadata.Filename)
	assert.Equal(t, "abc", string(data))
}

func TestAttachmentClientUsesAssignmentAndAdminEndpoints(t *testing.T) {
	assignmentID := uuid.NewString()
	requestNumber := 0
	client := userClientWithToken(t, func(r *http.Request) (*http.Response, error) {
		requestNumber++
		switch requestNumber {
		case 1:
			assert.Equal(t, "/api/v1/assignments/"+assignmentID+"/attachments", r.URL.Path)
			return response(http.StatusCreated, `{"id":"`+uuid.NewString()+`"}`), nil
		case 2:
			assert.Equal(t, http.MethodDelete, r.Method)
			assert.Equal(t, "/api/v1/admin/attachments", r.URL.Path)
			assert.Equal(t, "2026-01-01T00:00:00Z", r.URL.Query().Get("before"))
			return response(http.StatusOK, `{"count":2}`), nil
		case 3:
			assert.Equal(t, "/api/v1/admin/attachments/reconciliation", r.URL.Path)
			return response(http.StatusOK, `{"missingObjects":[],"orphanObjects":[]}`), nil
		default:
			t.Fatalf("unexpected request %d", requestNumber)
			return nil, nil
		}
	})

	_, err := client.UploadAttachment(context.Background(), "", assignmentID, "file.txt", 1, strings.NewReader("x"))
	require.NoError(t, err)
	count, err := client.BulkDeleteAttachments(context.Background(), "2026-01-01T00:00:00Z")
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	_, err = client.ReconcileAttachmentStorage(context.Background())
	require.NoError(t, err)
}
