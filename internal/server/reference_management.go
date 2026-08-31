package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type referenceManagementStore interface {
	GetAllOrganizations() ([]models.Organization, error)
	FindOrCreateOrganization(string) (*models.Organization, error)
	SearchOrganizations(string) ([]models.Organization, error)
	UpdateOrganizationWithOutbox(uuid.UUID, string, []models.OutboxEvent) error
	DeleteOrganizationWithOutbox(uuid.UUID, []models.OutboxEvent) error
	MergeOrganizationsWithOutbox(uuid.UUID, uuid.UUID, []models.OutboxEvent) error
	GetAllResolutionExecutors() ([]models.ResolutionExecutor, error)
	FindOrCreateResolutionExecutor(string) (*models.ResolutionExecutor, error)
	SearchResolutionExecutors(string) ([]models.ResolutionExecutor, error)
	UpdateResolutionExecutorWithOutbox(uuid.UUID, string, []models.OutboxEvent) error
	DeleteResolutionExecutorWithOutbox(uuid.UUID, []models.OutboxEvent) error
}

type referenceNameRequest struct {
	Name string `json:"name"`
}

type organizationMergeRequest struct {
	TargetID string `json:"targetId"`
}

func (api *managementAPI) listOrganizations(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	var items []models.Organization
	var err error
	if query == "" {
		items, err = api.references.GetAllOrganizations()
	} else {
		items, err = api.references.SearchOrganizations(query)
	}
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.MapOrganizations(items))
}

func (api *managementAPI) resolveOrganization(w http.ResponseWriter, r *http.Request) {
	var req referenceNameRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	item, err := api.references.FindOrCreateOrganization(req.Name)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.MapOrganization(item))
}

func (api *managementAPI) updateOrganization(w http.ResponseWriter, r *http.Request) {
	id, ok := parseReferenceID(w, r.PathValue("id"), "неверный ID записи справочника")
	if !ok {
		return
	}
	var req referenceNameRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	effect, err := referenceAuditEffect(r, "organization:"+id.String()+":update:"+uuid.NewString(), "ORG_UPDATE", fmt.Sprintf("Обновлена организация «%s»", req.Name))
	if err != nil {
		writeUserError(w, err)
		return
	}
	if err := api.references.UpdateOrganizationWithOutbox(id, req.Name, []models.OutboxEvent{effect}); err != nil {
		writeUserError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *managementAPI) deleteOrganization(w http.ResponseWriter, r *http.Request) {
	id, ok := parseReferenceID(w, r.PathValue("id"), "неверный ID записи справочника")
	if !ok {
		return
	}
	effect, err := referenceAuditEffect(r, "organization:"+id.String()+":delete", "ORG_DELETE", fmt.Sprintf("Удалена организация (ID: %s)", id))
	if err != nil {
		writeUserError(w, err)
		return
	}
	if err := api.references.DeleteOrganizationWithOutbox(id, []models.OutboxEvent{effect}); err != nil {
		writeUserError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *managementAPI) mergeOrganization(w http.ResponseWriter, r *http.Request) {
	sourceID, ok := parseReferenceID(w, r.PathValue("id"), "неверный ID исходной организации")
	if !ok {
		return
	}
	var req organizationMergeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	targetID, ok := parseReferenceID(w, req.TargetID, "неверный ID целевой организации")
	if !ok {
		return
	}
	if sourceID == targetID {
		writeUserError(w, models.NewBadRequest("нельзя объединить организацию саму с собой"))
		return
	}
	effect, err := referenceAuditEffect(r, "organization:"+sourceID.String()+":merge:"+targetID.String(), "ORG_MERGE", fmt.Sprintf("Объединены организации: %s -> %s", sourceID, targetID))
	if err != nil {
		writeUserError(w, err)
		return
	}
	if err := api.references.MergeOrganizationsWithOutbox(sourceID, targetID, []models.OutboxEvent{effect}); err != nil {
		writeUserError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *managementAPI) listResolutionExecutors(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	var items []models.ResolutionExecutor
	var err error
	if query == "" {
		items, err = api.references.GetAllResolutionExecutors()
	} else {
		items, err = api.references.SearchResolutionExecutors(query)
	}
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.MapResolutionExecutors(items))
}

func (api *managementAPI) resolveResolutionExecutor(w http.ResponseWriter, r *http.Request) {
	var req referenceNameRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	item, err := api.references.FindOrCreateResolutionExecutor(req.Name)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.MapResolutionExecutor(item))
}

func (api *managementAPI) updateResolutionExecutor(w http.ResponseWriter, r *http.Request) {
	id, ok := parseReferenceID(w, r.PathValue("id"), "неверный ID записи справочника")
	if !ok {
		return
	}
	var req referenceNameRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	effect, err := referenceAuditEffect(r, "resolution-executor:"+id.String()+":update:"+uuid.NewString(), "RESEXEC_UPDATE", fmt.Sprintf("Обновлен исполнитель резолюции «%s»", req.Name))
	if err != nil {
		writeUserError(w, err)
		return
	}
	if err := api.references.UpdateResolutionExecutorWithOutbox(id, req.Name, []models.OutboxEvent{effect}); err != nil {
		writeUserError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *managementAPI) deleteResolutionExecutor(w http.ResponseWriter, r *http.Request) {
	id, ok := parseReferenceID(w, r.PathValue("id"), "неверный ID записи справочника")
	if !ok {
		return
	}
	effect, err := referenceAuditEffect(r, "resolution-executor:"+id.String()+":delete", "RESEXEC_DELETE", fmt.Sprintf("Удален исполнитель резолюции (ID: %s)", id))
	if err != nil {
		writeUserError(w, err)
		return
	}
	if err := api.references.DeleteResolutionExecutorWithOutbox(id, []models.OutboxEvent{effect}); err != nil {
		writeUserError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseReferenceID(w http.ResponseWriter, raw, message string) (uuid.UUID, bool) {
	id, err := uuid.Parse(raw)
	if err != nil {
		writeUserError(w, models.NewBadRequestWrapped(message, err))
		return uuid.Nil, false
	}
	return id, true
}

func referenceAuditEffect(r *http.Request, key, action, details string) (models.OutboxEvent, error) {
	auth := authenticatedFromContext(r.Context())
	return userAuditEffect(auth.User, key, action, details)
}
