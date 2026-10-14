package server

import (
	"fmt"
	"mime"
	"net/http"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func (api *managementAPI) createReportSchedule(w http.ResponseWriter, r *http.Request) {
	var req models.ReportScheduleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	owner := authenticatedFromContext(r.Context()).User.ID
	item, err := api.reportSchedules.Create(r.Context(), owner, req)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (api *managementAPI) listReportSchedules(w http.ResponseWriter, r *http.Request) {
	owner := authenticatedFromContext(r.Context()).User.ID
	items, err := api.reportSchedules.List(r.Context(), owner)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (api *managementAPI) disableReportSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := validReportID(r.PathValue("id"))
	if err != nil {
		writeUserError(w, err)
		return
	}
	owner := authenticatedFromContext(r.Context()).User.ID
	if err = api.reportSchedules.Disable(r.Context(), owner, id); err != nil {
		writeUserError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *managementAPI) listReportRuns(w http.ResponseWriter, r *http.Request) {
	id, err := validReportID(r.PathValue("id"))
	if err != nil {
		writeUserError(w, err)
		return
	}
	owner := authenticatedFromContext(r.Context()).User.ID
	items, err := api.reportSchedules.Runs(r.Context(), owner, id)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (api *managementAPI) retryReportRun(w http.ResponseWriter, r *http.Request) {
	id, err := validReportID(r.PathValue("id"))
	if err != nil {
		writeUserError(w, err)
		return
	}
	owner := authenticatedFromContext(r.Context()).User.ID
	if err = api.reportSchedules.RetryRun(r.Context(), owner, id); err != nil {
		writeUserError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *managementAPI) downloadReportRun(w http.ResponseWriter, r *http.Request) {
	id, err := validReportID(r.PathValue("id"))
	if err != nil {
		writeUserError(w, err)
		return
	}
	owner := authenticatedFromContext(r.Context()).User.ID
	key, format, size, err := api.reportSchedules.DownloadInfo(r.Context(), owner, id)
	if err != nil {
		writeUserError(w, err)
		return
	}
	contentType := "application/pdf"
	if format == "xlsx" {
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": fmt.Sprintf("scheduled-report-%s.%s", id, format)}))
	w.Header().Set("Content-Length", fmt.Sprint(size))
	if err := api.reportSchedules.files.DownloadFileToWriter(r.Context(), key, w, 50<<20); err != nil {
		// Response may already be committed; the client detects truncated transfers.
		return
	}
}
