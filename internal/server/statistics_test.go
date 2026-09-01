package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type fakeDashboardAPI struct{}

func (*fakeDashboardAPI) GetActivity() (*dto.DashboardActivity, error) {
	return &dto.DashboardActivity{ExpiringAssignments: []dto.Assignment{}}, nil
}

type fakeStatisticsAPI struct {
	documentReport documentStatisticsReportRequest
}

func (*fakeStatisticsAPI) GetDocumentStatistics() (*models.DocumentStatistics, error) {
	return &models.DocumentStatistics{}, nil
}
func (f *fakeStatisticsAPI) GetDocumentReport(a, b, c, d, e, g string) (*models.DocumentStatisticsReport, error) {
	f.documentReport = documentStatisticsReportRequest{StartDate: a, EndDate: b, GroupBy: c, KindCode: d, NomenclatureID: e, UserID: g}
	return &models.DocumentStatisticsReport{}, nil
}
func (*fakeStatisticsAPI) GetDocumentFilterOptions() (*models.DocumentStatisticsFilters, error) {
	return &models.DocumentStatisticsFilters{}, nil
}
func (*fakeStatisticsAPI) GetAssignmentStatistics() (*models.AssignmentStatistics, error) {
	return &models.AssignmentStatistics{}, nil
}
func (*fakeStatisticsAPI) GetAssignmentReport(string, string, bool, string) (*models.AssignmentStatisticsReport, error) {
	return &models.AssignmentStatisticsReport{}, nil
}
func (*fakeStatisticsAPI) GetAssignmentFilterOptions() (*models.AssignmentStatisticsFilters, error) {
	return &models.AssignmentStatisticsFilters{}, nil
}
func (*fakeStatisticsAPI) GetSystemStatistics() (*models.SystemStatistics, error) {
	return &models.SystemStatistics{}, nil
}
func (*fakeStatisticsAPI) GetStorageStatisticsStatus() (*models.StorageStatisticsStatus, error) {
	return &models.StorageStatisticsStatus{}, nil
}
func (*fakeStatisticsAPI) RetryStorageStatisticsRefresh() (*models.StorageStatisticsStatus, error) {
	return &models.StorageStatisticsStatus{}, nil
}

func TestDashboardAndStatisticsAPIsRequireSessionAndUsePrincipal(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	statistics := &fakeStatisticsAPI{}
	var dashboardPrincipal, statisticsPrincipal uuid.UUID
	api.dashboard = func(user *models.User) dashboardAPI { dashboardPrincipal = user.ID; return &fakeDashboardAPI{} }
	api.statistics = func(user *models.User) statisticsAPI { statisticsPrincipal = user.ID; return statistics }

	unauthorized := httptest.NewRecorder()
	api.Handler().ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/activity", nil))
	assert.Equal(t, http.StatusUnauthorized, unauthorized.Code)

	dashboardRequest := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/activity", nil)
	dashboardRequest.Header.Set("Authorization", "Bearer "+token)
	dashboardResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(dashboardResponse, dashboardRequest)
	require.Equal(t, http.StatusOK, dashboardResponse.Code, dashboardResponse.Body.String())
	assert.NotEqual(t, uuid.Nil, dashboardPrincipal)

	reportRequest := httptest.NewRequest(http.MethodPost, "/api/v1/statistics/documents/report", strings.NewReader(`{"startDate":"2026-01-01","endDate":"2026-09-01","groupBy":"kind","accessScope":{"restricted":false}}`))
	reportRequest.Header.Set("Authorization", "Bearer "+token)
	reportResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(reportResponse, reportRequest)
	require.Equal(t, http.StatusBadRequest, reportResponse.Code, reportResponse.Body.String())
	assert.Equal(t, uuid.Nil, statisticsPrincipal)

	reportRequest = httptest.NewRequest(http.MethodPost, "/api/v1/statistics/documents/report", strings.NewReader(`{"startDate":"2026-01-01","endDate":"2026-09-01","groupBy":"kind"}`))
	reportRequest.Header.Set("Authorization", "Bearer "+token)
	reportResponse = httptest.NewRecorder()
	api.Handler().ServeHTTP(reportResponse, reportRequest)
	require.Equal(t, http.StatusOK, reportResponse.Code, reportResponse.Body.String())
	assert.NotEqual(t, uuid.Nil, statisticsPrincipal)
	assert.Equal(t, "kind", statistics.documentReport.GroupBy)
}

func TestRequestPrincipalChecksSessionPermissions(t *testing.T) {
	principal := requestDocumentPrincipal{user: &models.User{ID: uuid.New(), IsActive: true, SystemPermissions: []string{models.SystemPermissionStatsDocuments}}}
	assert.True(t, principal.HasSystemPermission(models.SystemPermissionStatsDocuments))
	assert.False(t, principal.HasSystemPermission(models.SystemPermissionStatsSystem))
}
