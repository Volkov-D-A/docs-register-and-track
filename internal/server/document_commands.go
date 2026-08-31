package server

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/services"
)

type documentCommandAPI interface {
	Register(string, any) (any, error)
	Update(string, any) (any, error)
	CreateAdminDraft(string, services.AdminDraftCreateRequest) (any, error)
}

func documentIdempotencyKey(r *http.Request) (string, error) {
	value := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	key, err := uuid.Parse(value)
	if err != nil || key == uuid.Nil {
		return "", models.NewBadRequest("укажите корректный Idempotency-Key")
	}
	return key.String(), nil
}

func (api *managementAPI) registerDocument(w http.ResponseWriter, r *http.Request) {
	key, err := documentIdempotencyKey(r)
	if err != nil {
		writeUserError(w, err)
		return
	}
	var req map[string]any
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	req["idempotencyKey"] = key
	result, err := api.documentCommands(authenticatedFromContext(r.Context()).User).Register(r.PathValue("kind"), req)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (api *managementAPI) updateDocument(w http.ResponseWriter, r *http.Request) {
	key, err := documentIdempotencyKey(r)
	if err != nil {
		writeUserError(w, err)
		return
	}
	var req map[string]any
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	req["id"] = r.PathValue("id")
	req["idempotencyKey"] = key
	result, err := api.documentCommands(authenticatedFromContext(r.Context()).User).Update(r.PathValue("kind"), req)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *managementAPI) createAdminDocumentDraft(w http.ResponseWriter, r *http.Request) {
	key, err := documentIdempotencyKey(r)
	if err != nil {
		writeUserError(w, err)
		return
	}
	var req services.AdminDraftCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	req.IdempotencyKey = key
	result, err := api.documentCommands(authenticatedFromContext(r.Context()).User).CreateAdminDraft(r.PathValue("kind"), req)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}
