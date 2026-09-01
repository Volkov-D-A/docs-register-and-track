package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type attachmentAPI interface {
	MaxUploadSize() int64
	UploadContent(string, *uuid.UUID, string, int64, io.Reader) (*dto.Attachment, error)
	UploadAssignmentContent(string, string, int64, io.Reader) (*dto.Attachment, error)
	GetList(string) ([]dto.Attachment, error)
	GetAssignmentFiles(string) ([]dto.Attachment, error)
	AuthorizeDownload(string) (*models.Attachment, error)
	StreamAttachment(context.Context, *models.Attachment, io.Writer) error
	Delete(string) error
	BulkDeleteOlderThan(string) (int, error)
	ReconcileStorage() (*models.AttachmentStorageReconciliation, error)
}

func (api *managementAPI) attachmentService(r *http.Request) attachmentAPI {
	return api.attachments(authenticatedFromContext(r.Context()).User)
}

func uploadFilename(r *http.Request) (string, error) {
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Disposition"))
	if err != nil || mediaType != "attachment" || strings.TrimSpace(params["filename"]) == "" {
		return "", models.NewBadRequest("имя загружаемого файла не указано")
	}
	return params["filename"], nil
}

func (api *managementAPI) uploadDocumentAttachment(w http.ResponseWriter, r *http.Request) {
	service := api.attachmentService(r)
	filename, err := uploadFilename(r)
	if err != nil {
		writeUserError(w, err)
		return
	}
	if !guardAttachmentBody(w, r, service.MaxUploadSize()) {
		return
	}
	result, err := service.UploadContent(r.PathValue("id"), nil, filename, r.ContentLength, r.Body)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (api *managementAPI) uploadAssignmentAttachment(w http.ResponseWriter, r *http.Request) {
	service := api.attachmentService(r)
	filename, err := uploadFilename(r)
	if err != nil {
		writeUserError(w, err)
		return
	}
	if !guardAttachmentBody(w, r, service.MaxUploadSize()) {
		return
	}
	result, err := service.UploadAssignmentContent(r.PathValue("id"), filename, r.ContentLength, r.Body)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func guardAttachmentBody(w http.ResponseWriter, r *http.Request, maxSize int64) bool {
	if r.ContentLength < 0 {
		writeUserError(w, models.NewBadRequest("размер загружаемого файла не указан"))
		return false
	}
	if r.ContentLength > maxSize {
		writeUserError(w, models.NewBadRequest(fmt.Sprintf("размер файла превышает максимально допустимый (%d МБ)", maxSize/(1024*1024))))
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)
	return true
}

func (api *managementAPI) listDocumentAttachments(w http.ResponseWriter, r *http.Request) {
	result, err := api.attachmentService(r).GetList(r.PathValue("id"))
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *managementAPI) listAssignmentAttachments(w http.ResponseWriter, r *http.Request) {
	result, err := api.attachmentService(r).GetAssignmentFiles(r.PathValue("id"))
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *managementAPI) downloadAttachment(w http.ResponseWriter, r *http.Request) {
	service := api.attachmentService(r)
	attachment, err := service.AuthorizeDownload(r.PathValue("id"))
	if err != nil {
		writeUserError(w, err)
		return
	}
	filename := filepath.Base(strings.ReplaceAll(attachment.Filename, "\\", "/"))
	w.Header().Set("Content-Type", attachment.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", attachment.FileSize))
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	if err := service.StreamAttachment(r.Context(), attachment, w); err != nil {
		slog.Warn("attachment download stream failed", "attachment_id", attachment.ID, "error", err)
	}
}

func (api *managementAPI) deleteAttachment(w http.ResponseWriter, r *http.Request) {
	if err := api.attachmentService(r).Delete(r.PathValue("id")); err != nil {
		writeUserError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *managementAPI) bulkDeleteAttachments(w http.ResponseWriter, r *http.Request) {
	count, err := api.attachmentService(r).BulkDeleteOlderThan(r.URL.Query().Get("before"))
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"count": count})
}

func (api *managementAPI) reconcileAttachments(w http.ResponseWriter, r *http.Request) {
	result, err := api.attachmentService(r).ReconcileStorage()
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
