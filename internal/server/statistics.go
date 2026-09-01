package server

import (
	"net/http"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type dashboardAPI interface {
	GetActivity() (*dto.DashboardActivity, error)
}

type statisticsAPI interface {
	GetDocumentStatistics() (*models.DocumentStatistics, error)
	GetDocumentReport(string, string, string, string, string, string) (*models.DocumentStatisticsReport, error)
	GetDocumentFilterOptions() (*models.DocumentStatisticsFilters, error)
	GetAssignmentStatistics() (*models.AssignmentStatistics, error)
	GetAssignmentReport(string, string, bool, string) (*models.AssignmentStatisticsReport, error)
	GetAssignmentFilterOptions() (*models.AssignmentStatisticsFilters, error)
	GetSystemStatistics() (*models.SystemStatistics, error)
	GetStorageStatisticsStatus() (*models.StorageStatisticsStatus, error)
	RetryStorageStatisticsRefresh() (*models.StorageStatisticsStatus, error)
}

type documentStatisticsReportRequest struct {
	StartDate      string `json:"startDate"`
	EndDate        string `json:"endDate"`
	GroupBy        string `json:"groupBy"`
	KindCode       string `json:"kindCode"`
	NomenclatureID string `json:"nomenclatureId"`
	UserID         string `json:"userId"`
}
type assignmentStatisticsReportRequest struct {
	StartDate   string `json:"startDate"`
	EndDate     string `json:"endDate"`
	OnlyOverdue bool   `json:"onlyOverdue"`
	UserID      string `json:"userId"`
}

func (api *managementAPI) dashboardService(r *http.Request) dashboardAPI {
	return api.dashboard(authenticatedFromContext(r.Context()).User)
}
func (api *managementAPI) statisticsService(r *http.Request) statisticsAPI {
	return api.statistics(authenticatedFromContext(r.Context()).User)
}

func writeStatisticsResult(w http.ResponseWriter, result any, err error) {
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *managementAPI) getDashboardActivity(w http.ResponseWriter, r *http.Request) {
	result, err := api.dashboardService(r).GetActivity()
	writeStatisticsResult(w, result, err)
}
func (api *managementAPI) getDocumentStatistics(w http.ResponseWriter, r *http.Request) {
	result, err := api.statisticsService(r).GetDocumentStatistics()
	writeStatisticsResult(w, result, err)
}
func (api *managementAPI) getDocumentStatisticsFilters(w http.ResponseWriter, r *http.Request) {
	result, err := api.statisticsService(r).GetDocumentFilterOptions()
	writeStatisticsResult(w, result, err)
}
func (api *managementAPI) getAssignmentStatistics(w http.ResponseWriter, r *http.Request) {
	result, err := api.statisticsService(r).GetAssignmentStatistics()
	writeStatisticsResult(w, result, err)
}
func (api *managementAPI) getAssignmentStatisticsFilters(w http.ResponseWriter, r *http.Request) {
	result, err := api.statisticsService(r).GetAssignmentFilterOptions()
	writeStatisticsResult(w, result, err)
}
func (api *managementAPI) getSystemStatistics(w http.ResponseWriter, r *http.Request) {
	result, err := api.statisticsService(r).GetSystemStatistics()
	writeStatisticsResult(w, result, err)
}
func (api *managementAPI) getStorageStatisticsStatus(w http.ResponseWriter, r *http.Request) {
	result, err := api.statisticsService(r).GetStorageStatisticsStatus()
	writeStatisticsResult(w, result, err)
}
func (api *managementAPI) retryStorageStatisticsRefresh(w http.ResponseWriter, r *http.Request) {
	result, err := api.statisticsService(r).RetryStorageStatisticsRefresh()
	writeStatisticsResult(w, result, err)
}

func (api *managementAPI) getDocumentStatisticsReport(w http.ResponseWriter, r *http.Request) {
	var req documentStatisticsReportRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	result, err := api.statisticsService(r).GetDocumentReport(req.StartDate, req.EndDate, req.GroupBy, req.KindCode, req.NomenclatureID, req.UserID)
	writeStatisticsResult(w, result, err)
}

func (api *managementAPI) getAssignmentStatisticsReport(w http.ResponseWriter, r *http.Request) {
	var req assignmentStatisticsReportRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	result, err := api.statisticsService(r).GetAssignmentReport(req.StartDate, req.EndDate, req.OnlyOverdue, req.UserID)
	writeStatisticsResult(w, result, err)
}
