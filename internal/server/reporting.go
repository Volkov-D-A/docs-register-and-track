package server

import (
	"net/http"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func (api *managementAPI) runReport(w http.ResponseWriter, r *http.Request) {
	var req models.ReportRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	result, err := api.reporting.Run(r.Context(), req)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *managementAPI) reportFilterOptions(w http.ResponseWriter, r *http.Request) {
	result, err := api.reporting.FilterOptions()
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
