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

type fakeWorkspaceAPI struct {
	assignmentMode, acknowledgmentMode string
	listMode                           string
	page, pageSize                     int
}

func (f *fakeWorkspaceAPI) GetOverview(assignmentMode, acknowledgmentMode string) (*dto.WorkspaceOverview, error) {
	f.assignmentMode, f.acknowledgmentMode = assignmentMode, acknowledgmentMode
	return &dto.WorkspaceOverview{}, nil
}
func (f *fakeWorkspaceAPI) ListAcknowledgments(mode string, page, pageSize int) (*dto.PagedResult[dto.WorkspaceAcknowledgment], error) {
	f.listMode, f.page, f.pageSize = mode, page, pageSize
	return &dto.PagedResult[dto.WorkspaceAcknowledgment]{}, nil
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

func TestWorkspaceAndStatisticsAPIsRequireSessionAndUsePrincipal(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	statistics := &fakeStatisticsAPI{}
	workspace := &fakeWorkspaceAPI{}
	var workspacePrincipal, statisticsPrincipal uuid.UUID
	api.workspace = func(user *models.User) workspaceAPI { workspacePrincipal = user.ID; return workspace }
	api.statistics = func(user *models.User) statisticsAPI { statisticsPrincipal = user.ID; return statistics }

	unauthorized := httptest.NewRecorder()
	api.Handler().ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v1/workspace/overview", nil))
	assert.Equal(t, http.StatusUnauthorized, unauthorized.Code)

	workspaceRequest := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/overview?assignmentMode=control&acknowledgmentMode=execution", nil)
	workspaceRequest.Header.Set("Authorization", "Bearer "+token)
	workspaceResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(workspaceResponse, workspaceRequest)
	require.Equal(t, http.StatusOK, workspaceResponse.Code, workspaceResponse.Body.String())
	assert.NotEqual(t, uuid.Nil, workspacePrincipal)
	assert.Equal(t, "control", workspace.assignmentMode)
	assert.Equal(t, "execution", workspace.acknowledgmentMode)

	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/acknowledgments?mode=control&page=2&pageSize=7", nil)
	listRequest.Header.Set("Authorization", "Bearer "+token)
	listResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(listResponse, listRequest)
	require.Equal(t, http.StatusOK, listResponse.Code, listResponse.Body.String())
	assert.Equal(t, "control", workspace.listMode)
	assert.Equal(t, 2, workspace.page)
	assert.Equal(t, 7, workspace.pageSize)

	invalidList := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/acknowledgments?pageSize=1000", nil)
	invalidList.Header.Set("Authorization", "Bearer "+token)
	invalidResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(invalidResponse, invalidList)
	require.Equal(t, http.StatusBadRequest, invalidResponse.Code)

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
