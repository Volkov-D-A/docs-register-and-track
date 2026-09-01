package server

import (
	"net/http"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type acknowledgmentAPI interface {
	Create(string, string, []string) (*dto.Acknowledgment, error)
	GetList(string) ([]dto.Acknowledgment, error)
	GetPendingForCurrentUser() ([]dto.Acknowledgment, error)
	GetCurrentUserPendingByDocument(string) ([]dto.Acknowledgment, error)
	GetAllActive() ([]dto.Acknowledgment, error)
	MarkViewed(string) error
	MarkConfirmed(string) error
	Delete(string) error
}

type userEventAPI interface {
	GetCurrentUserEvents(models.UserEventFilter) (*dto.PagedResult[dto.UserEvent], error)
	GetUnreadCount() (int, error)
	MarkRead(string) error
	MarkDocumentRead(string) error
	MarkAllRead() error
}

type administrativeOrderAcknowledgmentAPI interface {
	MarkAcknowledged(string) (*dto.AdministrativeOrderAcknowledgmentPerson, error)
}

type createAcknowledgmentRequest struct {
	DocumentID string   `json:"documentId"`
	Content    string   `json:"content"`
	UserIDs    []string `json:"userIds"`
}

func (api *managementAPI) acknowledgmentService(r *http.Request) acknowledgmentAPI {
	return api.acknowledgments(authenticatedFromContext(r.Context()).User)
}

func (api *managementAPI) userEventService(r *http.Request) userEventAPI {
	return api.userEvents(authenticatedFromContext(r.Context()).User)
}

func (api *managementAPI) administrativeOrderAcknowledgmentService(r *http.Request) administrativeOrderAcknowledgmentAPI {
	return api.administrativeOrderAcknowledgments(authenticatedFromContext(r.Context()).User)
}

func (api *managementAPI) createAcknowledgment(w http.ResponseWriter, r *http.Request) {
	var req createAcknowledgmentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	result, err := api.acknowledgmentService(r).Create(req.DocumentID, req.Content, req.UserIDs)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (api *managementAPI) listAcknowledgments(w http.ResponseWriter, r *http.Request) {
	result, err := api.acknowledgmentService(r).GetList(r.URL.Query().Get("documentId"))
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *managementAPI) listPendingAcknowledgments(w http.ResponseWriter, r *http.Request) {
	result, err := api.acknowledgmentService(r).GetPendingForCurrentUser()
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *managementAPI) listPendingAcknowledgmentsByDocument(w http.ResponseWriter, r *http.Request) {
	result, err := api.acknowledgmentService(r).GetCurrentUserPendingByDocument(r.PathValue("documentId"))
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *managementAPI) listActiveAcknowledgments(w http.ResponseWriter, r *http.Request) {
	result, err := api.acknowledgmentService(r).GetAllActive()
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *managementAPI) markAcknowledgmentViewed(w http.ResponseWriter, r *http.Request) {
	if err := api.acknowledgmentService(r).MarkViewed(r.PathValue("id")); err != nil {
		writeUserError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *managementAPI) markAcknowledgmentConfirmed(w http.ResponseWriter, r *http.Request) {
	if err := api.acknowledgmentService(r).MarkConfirmed(r.PathValue("id")); err != nil {
		writeUserError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *managementAPI) deleteAcknowledgment(w http.ResponseWriter, r *http.Request) {
	if err := api.acknowledgmentService(r).Delete(r.PathValue("id")); err != nil {
		writeUserError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *managementAPI) listUserEvents(w http.ResponseWriter, r *http.Request) {
	var filter models.UserEventFilter
	if err := decodeJSON(r, &filter); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	result, err := api.userEventService(r).GetCurrentUserEvents(filter)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *managementAPI) getUnreadUserEventCount(w http.ResponseWriter, r *http.Request) {
	count, err := api.userEventService(r).GetUnreadCount()
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"count": count})
}

func (api *managementAPI) markUserEventRead(w http.ResponseWriter, r *http.Request) {
	if err := api.userEventService(r).MarkRead(r.PathValue("id")); err != nil {
		writeUserError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *managementAPI) markDocumentUserEventsRead(w http.ResponseWriter, r *http.Request) {
	if err := api.userEventService(r).MarkDocumentRead(r.PathValue("documentId")); err != nil {
		writeUserError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *managementAPI) markAllUserEventsRead(w http.ResponseWriter, r *http.Request) {
	if err := api.userEventService(r).MarkAllRead(); err != nil {
		writeUserError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *managementAPI) markAdministrativeOrderAcknowledged(w http.ResponseWriter, r *http.Request) {
	result, err := api.administrativeOrderAcknowledgmentService(r).MarkAcknowledged(r.PathValue("id"))
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
