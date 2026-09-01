package server

import (
	"net/http"
	"strconv"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type adminAuditAPI interface {
	GetAll(int, int) (*dto.AdminAuditLogPage, error)
}
type outboxAdminAPI interface {
	GetStats() (models.OutboxStats, error)
	GetFailed(int) ([]models.FailedOutboxEvent, error)
	Requeue(string) error
}

func parsePositiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}

func (api *managementAPI) getAdminAuditLog(w http.ResponseWriter, r *http.Request) {
	result, err := api.adminAudit(authenticatedFromContext(r.Context()).User).GetAll(parsePositiveInt(r.URL.Query().Get("page"), 1), parsePositiveInt(r.URL.Query().Get("pageSize"), 50))
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *managementAPI) getOutboxStats(w http.ResponseWriter, r *http.Request) {
	result, err := api.outboxAdmin(authenticatedFromContext(r.Context()).User).GetStats()
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *managementAPI) getFailedOutboxEvents(w http.ResponseWriter, r *http.Request) {
	result, err := api.outboxAdmin(authenticatedFromContext(r.Context()).User).GetFailed(parsePositiveInt(r.URL.Query().Get("limit"), 50))
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *managementAPI) requeueOutboxEvent(w http.ResponseWriter, r *http.Request) {
	if err := api.outboxAdmin(authenticatedFromContext(r.Context()).User).Requeue(r.PathValue("id")); err != nil {
		writeUserError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
