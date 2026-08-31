package server

import (
	"net/http"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type assignmentAPI interface {
	Create(string, string, string, string, []string) (*dto.Assignment, error)
	CreateSeries(models.AssignmentSeriesRequest) (*dto.AssignmentSeries, error)
	GetSeries(string) (*dto.AssignmentSeries, error)
	GetSeriesHistory(string) ([]dto.Assignment, error)
	UpdateSeries(string, models.AssignmentSeriesRequest) (*dto.AssignmentSeries, error)
	CancelSeries(string) error
	Update(string, string, string, string, []string) (*dto.Assignment, error)
	UpdateStatus(string, string, string) (*dto.Assignment, error)
	GetByID(string) (*dto.Assignment, error)
	GetList(models.AssignmentFilter) (*dto.PagedResult[dto.Assignment], error)
	Delete(string) error
}

type assignmentDetailsRequest struct {
	DocumentID    string   `json:"documentId"`
	ExecutorID    string   `json:"executorId"`
	Content       string   `json:"content"`
	Deadline      string   `json:"deadline"`
	CoExecutorIDs []string `json:"coExecutorIds"`
}

type assignmentStatusRequest struct {
	Status string `json:"status"`
	Report string `json:"report"`
}

func (api *managementAPI) assignmentService(r *http.Request) assignmentAPI {
	return api.assignments(authenticatedFromContext(r.Context()).User)
}

func (api *managementAPI) createAssignment(w http.ResponseWriter, r *http.Request) {
	var req assignmentDetailsRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	result, err := api.assignmentService(r).Create(req.DocumentID, req.ExecutorID, req.Content, req.Deadline, req.CoExecutorIDs)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (api *managementAPI) getAssignment(w http.ResponseWriter, r *http.Request) {
	result, err := api.assignmentService(r).GetByID(r.PathValue("id"))
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *managementAPI) listAssignments(w http.ResponseWriter, r *http.Request) {
	var filter models.AssignmentFilter
	if err := decodeJSON(r, &filter); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	filter.AllowedDocumentKinds = nil
	filter.AccessibleByUserID = ""
	filter.AccessibleByUserIDs = nil
	result, err := api.assignmentService(r).GetList(filter)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *managementAPI) updateAssignment(w http.ResponseWriter, r *http.Request) {
	var req assignmentDetailsRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	result, err := api.assignmentService(r).Update(r.PathValue("id"), req.ExecutorID, req.Content, req.Deadline, req.CoExecutorIDs)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *managementAPI) updateAssignmentStatus(w http.ResponseWriter, r *http.Request) {
	var req assignmentStatusRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	result, err := api.assignmentService(r).UpdateStatus(r.PathValue("id"), req.Status, req.Report)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *managementAPI) deleteAssignment(w http.ResponseWriter, r *http.Request) {
	if err := api.assignmentService(r).Delete(r.PathValue("id")); err != nil {
		writeUserError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *managementAPI) createAssignmentSeries(w http.ResponseWriter, r *http.Request) {
	var req models.AssignmentSeriesRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	result, err := api.assignmentService(r).CreateSeries(req)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (api *managementAPI) getAssignmentSeries(w http.ResponseWriter, r *http.Request) {
	result, err := api.assignmentService(r).GetSeries(r.PathValue("id"))
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *managementAPI) getAssignmentSeriesHistory(w http.ResponseWriter, r *http.Request) {
	result, err := api.assignmentService(r).GetSeriesHistory(r.PathValue("id"))
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *managementAPI) updateAssignmentSeries(w http.ResponseWriter, r *http.Request) {
	var req models.AssignmentSeriesRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	result, err := api.assignmentService(r).UpdateSeries(r.PathValue("id"), req)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *managementAPI) cancelAssignmentSeries(w http.ResponseWriter, r *http.Request) {
	if err := api.assignmentService(r).CancelSeries(r.PathValue("id")); err != nil {
		writeUserError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
