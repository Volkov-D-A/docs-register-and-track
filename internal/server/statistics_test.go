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

type fakeWorkspaceAPI struct{}

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
func (*fakeStatisticsAPI) GetDeadlineStatistics() (*models.DeadlineStatistics, error) {
	return &models.DeadlineStatistics{}, nil
}
func (*fakeStatisticsAPI) GetDeadlineReport(string, string, bool, string) (*models.DeadlineStatisticsReport, error) {
	return &models.DeadlineStatisticsReport{}, nil
}
func (*fakeStatisticsAPI) GetDeadlineFilterOptions() (*models.DeadlineStatisticsFilters, error) {
	return &models.DeadlineStatisticsFilters{}, nil
}
func (*fakeStatisticsAPI) GetSystemStatistics() (*models.SystemStatistics, error) {
	return &models.SystemStatistics{}, nil
}
func TestWorkspaceAndStatisticsAPIsRequireSessionAndUsePrincipal(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	statistics := &fakeStatisticsAPI{}
	var statisticsPrincipal uuid.UUID
	api.statistics = func(user *models.User) statisticsAPI { statisticsPrincipal = user.ID; return statistics }
	unauthorized := httptest.NewRecorder()
	api.Handler().ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v1/statistics/documents", nil))
	assert.Equal(t, http.StatusUnauthorized, unauthorized.Code)

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
	principal := requestDocumentPrincipal{user: &models.User{ID: uuid.New(), IsActive: true, SystemPermissions: []string{models.SystemPermissionReferences}}}
	assert.True(t, principal.HasSystemPermission(models.SystemPermissionReferences))
	assert.False(t, principal.HasSystemPermission(models.SystemPermissionAdmin))
}

func (f *fakeWorkspaceAPI) GetRecentDocuments() (*dto.WorkspaceDocuments, error) {
	return &dto.WorkspaceDocuments{Available: true, Items: []dto.WorkspaceDocument{}}, nil
}

func TestWorkspaceRecentDocumentsEndpointUsesAuthenticatedPrincipal(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	var principal uuid.UUID
	api.workspace = func(user *models.User) workspaceAPI { principal = user.ID; return &fakeWorkspaceAPI{} }
	unauthorized := httptest.NewRecorder()
	api.Handler().ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v1/workspace/documents", nil))
	require.Equal(t, http.StatusUnauthorized, unauthorized.Code)
	require.Equal(t, uuid.Nil, principal)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/documents", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, req)
	require.Equal(t, http.StatusOK, response.Code)
	require.NotEqual(t, uuid.Nil, principal)
	require.JSONEq(t, `{"available":true,"items":[]}`, response.Body.String())
}
