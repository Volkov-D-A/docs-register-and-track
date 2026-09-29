package server

import (
	"net/http"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

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

func (api *managementAPI) userEventService(r *http.Request) userEventAPI {
	return api.userEvents(authenticatedFromContext(r.Context()).User)
}

func (api *managementAPI) administrativeOrderAcknowledgmentService(r *http.Request) administrativeOrderAcknowledgmentAPI {
	return api.administrativeOrderAcknowledgments(authenticatedFromContext(r.Context()).User)
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
	api.events.Publish("user:" + authenticatedFromContext(r.Context()).User.ID.String())
	w.WriteHeader(http.StatusNoContent)
}

func (api *managementAPI) markDocumentUserEventsRead(w http.ResponseWriter, r *http.Request) {
	if err := api.userEventService(r).MarkDocumentRead(r.PathValue("documentId")); err != nil {
		writeUserError(w, err)
		return
	}
	api.events.Publish("user:" + authenticatedFromContext(r.Context()).User.ID.String())
	w.WriteHeader(http.StatusNoContent)
}

func (api *managementAPI) markAllUserEventsRead(w http.ResponseWriter, r *http.Request) {
	if err := api.userEventService(r).MarkAllRead(); err != nil {
		writeUserError(w, err)
		return
	}
	api.events.Publish("user:" + authenticatedFromContext(r.Context()).User.ID.String())
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
