package server

import (
	"context"
	"net/http"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type deadlineControlAPI interface {
	PreviewResume(context.Context, string) (*models.DeadlineSeriesResumePreview, error)
	ListExecutors(context.Context, bool) ([]models.DeadlineExecutor, error)
	SaveExternalExecutor(context.Context, string, models.ExternalDeadlineExecutorRequest) (*models.DeadlineExecutor, error)
	Get(context.Context, string) (*models.DocumentDeadlineControl, error)
	List(context.Context, models.DeadlineControlFilter) (*models.DeadlineControlPage, error)
	Create(context.Context, models.DeadlineControlRequest) (*models.DocumentDeadlineControl, error)
	Update(context.Context, string, models.DeadlineControlRequest) (*models.DocumentDeadlineControl, error)
	Complete(context.Context, string, models.CompleteDeadlineRequest) (*models.DocumentDeadlineControl, error)
	Cancel(context.Context, string, models.CancelDeadlineRequest) (*models.DocumentDeadlineControl, error)
	CreateSeries(context.Context, models.DeadlineSeriesRequest) (*models.DocumentDeadlineSeries, error)
	GetSeries(context.Context, string) (*models.DocumentDeadlineSeries, error)
	UpdateSeries(context.Context, string, models.DeadlineSeriesRequest) (*models.DocumentDeadlineSeries, error)
	StopSeries(context.Context, string, models.DeadlineSeriesActionRequest) (*models.DocumentDeadlineSeries, error)
	ResumeSeries(context.Context, string, models.DeadlineSeriesActionRequest) (*models.DocumentDeadlineSeries, error)
}

func (api *managementAPI) deadlineService(r *http.Request) deadlineControlAPI {
	return api.deadlines(authenticatedFromContext(r.Context()).User)
}

func deadlineRequest[Request any, Result any](w http.ResponseWriter, r *http.Request, status int, fn func(Request) (Result, error)) {
	var request Request
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	result, err := fn(request)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, status, result)
}
func deadlineResponse[Result any](w http.ResponseWriter, result Result, err error) {
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (api *managementAPI) listDeadlineExecutors(w http.ResponseWriter, r *http.Request) {
	value, err := api.deadlineService(r).ListExecutors(r.Context(), r.URL.Query().Get("includeInactive") == "true")
	deadlineResponse(w, value, err)
}
func (api *managementAPI) createDeadlineExecutor(w http.ResponseWriter, r *http.Request) {
	deadlineRequest(w, r, http.StatusCreated, func(req models.ExternalDeadlineExecutorRequest) (*models.DeadlineExecutor, error) {
		return api.deadlineService(r).SaveExternalExecutor(r.Context(), "", req)
	})
}
func (api *managementAPI) updateDeadlineExecutor(w http.ResponseWriter, r *http.Request) {
	deadlineRequest(w, r, http.StatusOK, func(req models.ExternalDeadlineExecutorRequest) (*models.DeadlineExecutor, error) {
		return api.deadlineService(r).SaveExternalExecutor(r.Context(), r.PathValue("id"), req)
	})
}
func (api *managementAPI) getDeadlineControl(w http.ResponseWriter, r *http.Request) {
	value, err := api.deadlineService(r).Get(r.Context(), r.PathValue("id"))
	deadlineResponse(w, value, err)
}
func (api *managementAPI) queryDeadlineControls(w http.ResponseWriter, r *http.Request) {
	deadlineRequest(w, r, http.StatusOK, func(req models.DeadlineControlFilter) (*models.DeadlineControlPage, error) {
		return api.deadlineService(r).List(r.Context(), req)
	})
}
func (api *managementAPI) createDeadlineControl(w http.ResponseWriter, r *http.Request) {
	deadlineRequest(w, r, http.StatusCreated, func(req models.DeadlineControlRequest) (*models.DocumentDeadlineControl, error) {
		return api.deadlineService(r).Create(r.Context(), req)
	})
}
func (api *managementAPI) updateDeadlineControl(w http.ResponseWriter, r *http.Request) {
	deadlineRequest(w, r, http.StatusOK, func(req models.DeadlineControlRequest) (*models.DocumentDeadlineControl, error) {
		return api.deadlineService(r).Update(r.Context(), r.PathValue("id"), req)
	})
}
func (api *managementAPI) completeDeadlineControl(w http.ResponseWriter, r *http.Request) {
	deadlineRequest(w, r, http.StatusOK, func(req models.CompleteDeadlineRequest) (*models.DocumentDeadlineControl, error) {
		return api.deadlineService(r).Complete(r.Context(), r.PathValue("id"), req)
	})
}
func (api *managementAPI) cancelDeadlineControl(w http.ResponseWriter, r *http.Request) {
	deadlineRequest(w, r, http.StatusOK, func(req models.CancelDeadlineRequest) (*models.DocumentDeadlineControl, error) {
		return api.deadlineService(r).Cancel(r.Context(), r.PathValue("id"), req)
	})
}
func (api *managementAPI) createDeadlineSeries(w http.ResponseWriter, r *http.Request) {
	deadlineRequest(w, r, http.StatusCreated, func(req models.DeadlineSeriesRequest) (*models.DocumentDeadlineSeries, error) {
		return api.deadlineService(r).CreateSeries(r.Context(), req)
	})
}
func (api *managementAPI) getDeadlineSeries(w http.ResponseWriter, r *http.Request) {
	value, err := api.deadlineService(r).GetSeries(r.Context(), r.PathValue("id"))
	deadlineResponse(w, value, err)
}
func (api *managementAPI) updateDeadlineSeries(w http.ResponseWriter, r *http.Request) {
	deadlineRequest(w, r, http.StatusOK, func(req models.DeadlineSeriesRequest) (*models.DocumentDeadlineSeries, error) {
		return api.deadlineService(r).UpdateSeries(r.Context(), r.PathValue("id"), req)
	})
}
func (api *managementAPI) stopDeadlineSeries(w http.ResponseWriter, r *http.Request) {
	deadlineRequest(w, r, http.StatusOK, func(req models.DeadlineSeriesActionRequest) (*models.DocumentDeadlineSeries, error) {
		return api.deadlineService(r).StopSeries(r.Context(), r.PathValue("id"), req)
	})
}
func (api *managementAPI) resumeDeadlineSeries(w http.ResponseWriter, r *http.Request) {
	deadlineRequest(w, r, http.StatusOK, func(req models.DeadlineSeriesActionRequest) (*models.DocumentDeadlineSeries, error) {
		return api.deadlineService(r).ResumeSeries(r.Context(), r.PathValue("id"), req)
	})
}

func (api *managementAPI) previewDeadlineSeriesResume(w http.ResponseWriter, r *http.Request) {
	value, err := api.deadlineService(r).PreviewResume(r.Context(), r.PathValue("id"))
	deadlineResponse(w, value, err)
}
