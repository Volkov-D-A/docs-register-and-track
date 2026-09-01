package serverclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type AttachmentClient interface {
	UploadAttachment(context.Context, string, string, string, int64, io.Reader) (*dto.Attachment, error)
	ListDocumentAttachments(context.Context, string) ([]dto.Attachment, error)
	ListAssignmentAttachments(context.Context, string) ([]dto.Attachment, error)
	GetAttachmentContent(context.Context, string) (*dto.Attachment, io.ReadCloser, error)
	DeleteAttachment(context.Context, string) error
	BulkDeleteAttachments(context.Context, string) (int, error)
	ReconcileAttachmentStorage(context.Context) (*models.AttachmentStorageReconciliation, error)
}

const maximumAttachmentResponseSize = int64(1 << 30)

func (c *Client) UploadAttachment(ctx context.Context, documentID, assignmentID, filename string, size int64, content io.Reader) (*dto.Attachment, error) {
	if filename == "" || size < 0 || content == nil {
		return nil, models.NewBadRequest("файл для загрузки указан некорректно")
	}
	var path string
	if assignmentID != "" {
		path = "/api/v1/assignments/" + url.PathEscape(assignmentID) + "/attachments"
	} else {
		if documentID == "" {
			return nil, models.NewBadRequest("идентификатор документа обязателен")
		}
		path = "/api/v1/documents/" + url.PathEscape(documentID) + "/attachments"
	}
	req, err := c.authenticatedRequestWithBody(ctx, http.MethodPost, path, content)
	if err != nil {
		return nil, err
	}
	req.ContentLength = size
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("docflow-server is unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return nil, decodeAuthError(resp)
	}
	var result dto.Attachment
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode attachment upload response: %w", err)
	}
	return &result, nil
}

func (c *Client) ListDocumentAttachments(ctx context.Context, documentID string) ([]dto.Attachment, error) {
	return c.listAttachments(ctx, "/api/v1/documents/"+url.PathEscape(documentID)+"/attachments")
}

func (c *Client) ListAssignmentAttachments(ctx context.Context, assignmentID string) ([]dto.Attachment, error) {
	return c.listAttachments(ctx, "/api/v1/assignments/"+url.PathEscape(assignmentID)+"/attachments")
}

func (c *Client) listAttachments(ctx context.Context, path string) ([]dto.Attachment, error) {
	var result []dto.Attachment
	if err := c.doUserRequest(ctx, http.MethodGet, path, nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetAttachmentContent(ctx context.Context, id string) (*dto.Attachment, io.ReadCloser, error) {
	req, err := c.authenticatedRequest(ctx, http.MethodGet, "/api/v1/attachments/"+url.PathEscape(id)+"/content")
	if err != nil {
		return nil, nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("docflow-server is unavailable: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, nil, decodeAuthError(resp)
	}
	if resp.ContentLength < 0 || resp.ContentLength > maximumAttachmentResponseSize {
		resp.Body.Close()
		return nil, nil, fmt.Errorf("docflow-server returned an invalid attachment content length")
	}
	_, params, parseErr := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
	if parseErr != nil || params["filename"] == "" {
		resp.Body.Close()
		return nil, nil, fmt.Errorf("docflow-server returned attachment content without a safe filename")
	}
	return &dto.Attachment{ID: id, Filename: params["filename"], FileSize: resp.ContentLength, ContentType: resp.Header.Get("Content-Type")}, resp.Body, nil
}

func (c *Client) DeleteAttachment(ctx context.Context, id string) error {
	return c.doUserRequest(ctx, http.MethodDelete, "/api/v1/attachments/"+url.PathEscape(id), nil, http.StatusNoContent, nil)
}

func (c *Client) BulkDeleteAttachments(ctx context.Context, before string) (int, error) {
	var result struct {
		Count int `json:"count"`
	}
	path := "/api/v1/admin/attachments?before=" + url.QueryEscape(before)
	if err := c.doUserRequest(ctx, http.MethodDelete, path, nil, http.StatusOK, &result); err != nil {
		return 0, err
	}
	return result.Count, nil
}

func (c *Client) ReconcileAttachmentStorage(ctx context.Context) (*models.AttachmentStorageReconciliation, error) {
	var result models.AttachmentStorageReconciliation
	if err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/admin/attachments/reconciliation", nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
