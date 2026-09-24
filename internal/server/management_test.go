package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/security"
)

type fakeManagementMigrations struct {
	status        dto.MigrationStatus
	statusErr     error
	applyErr      error
	rollbackErr   error
	applyCalls    int
	rollbackCalls int
}

func (m *fakeManagementMigrations) RunMigrations(string) error {
	m.applyCalls++
	return m.applyErr
}
func (m *fakeManagementMigrations) GetMigrationStatus(string) (*dto.MigrationStatus, error) {
	return &m.status, m.statusErr
}
func (m *fakeManagementMigrations) RollbackMigration(string) error {
	m.rollbackCalls++
	return m.rollbackErr
}

type fakeManagementLifecycle struct {
	prepareCalls int
	complete     []bool
	readyErr     error
}

func (l *fakeManagementLifecycle) CheckReady() error { return l.readyErr }
func (l *fakeManagementLifecycle) PrepareRollback() error {
	l.prepareCalls++
	return nil
}
func (l *fakeManagementLifecycle) CompleteRollback(success bool) {
	l.complete = append(l.complete, success)
}

type fakeAdminUsers struct{ user *models.User }

func (s fakeAdminUsers) GetByLogin(login string) (*models.User, error) {
	if s.user != nil && s.user.Login == login {
		return s.user, nil
	}
	return nil, nil
}

type fakeAdminAudit struct{ actions []string }

func (a *fakeAdminAudit) Create(req models.CreateAdminAuditLogRequest) (uuid.UUID, error) {
	a.actions = append(a.actions, req.Action)
	return uuid.New(), nil
}

func testManagementAPI(t *testing.T) (*managementAPI, *fakeManagementMigrations, *fakeManagementLifecycle, *fakeAdminAudit) {
	t.Helper()
	hash, err := security.HashPassword("Passw0rd!")
	require.NoError(t, err)
	migrations := &fakeManagementMigrations{status: dto.MigrationStatus{CurrentVersion: 8, LatestAvailableVersion: 8, UpToDate: true, Compatible: true}}
	lifecycle := &fakeManagementLifecycle{}
	audit := &fakeAdminAudit{}
	api := &managementAPI{
		migrations:    migrations,
		lifecycle:     lifecycle,
		serverVersion: "1.0.6",
		authSettings:  fakeAuthSettings{},
		users: fakeAdminUsers{user: &models.User{
			ID:                uuid.New(),
			Login:             "admin",
			PasswordHash:      hash,
			FullName:          "Administrator",
			IsActive:          true,
			SystemPermissions: []string{models.SystemPermissionAdmin},
		}},
		audit: audit,
		acquireLease: func(context.Context) (func(), bool, error) {
			return func() {}, true, nil
		},
	}
	return api, migrations, lifecycle, audit
}

func TestManagementAPISystemStatusReady(t *testing.T) {
	api, _, _, _ := testManagementAPI(t)
	res := httptest.NewRecorder()
	api.Handler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/system/status", nil))

	require.Equal(t, http.StatusOK, res.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &payload))
	assert.Equal(t, "ready", payload["status"])
	assert.Equal(t, "ready", payload["code"])
	assert.Equal(t, "1.0.6", payload["serverVersion"])
}

func TestManagementAPISystemStatusDoesNotLeakInternalError(t *testing.T) {
	api, migrations, _, _ := testManagementAPI(t)
	migrations.statusErr = errors.New("password=secret database detail")
	res := httptest.NewRecorder()
	api.Handler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/system/status", nil))

	require.Equal(t, http.StatusOK, res.Code)
	assert.Contains(t, res.Body.String(), `"code":"status_unavailable"`)
	assert.NotContains(t, res.Body.String(), "secret")
}

func TestManagementAPISystemStatusReportsMaintenance(t *testing.T) {
	api, _, lifecycle, _ := testManagementAPI(t)
	lifecycle.readyErr = errors.New("private maintenance detail")
	res := httptest.NewRecorder()
	api.Handler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/system/status", nil))

	require.Equal(t, http.StatusOK, res.Code)
	assert.Contains(t, res.Body.String(), `"status":"maintenance"`)
	assert.NotContains(t, res.Body.String(), "private")
}

func TestManagementAPICompatibilityUsesExactReleaseVersion(t *testing.T) {
	api, _, _, _ := testManagementAPI(t)
	tests := []struct {
		version string
		code    string
		ok      bool
	}{
		{"1.0.6", "build_identity_required", false},
		{"1.0.5", "client_too_old", false},
		{"1.0.7", "client_too_new", false},
	}
	for _, test := range tests {
		t.Run(test.version, func(t *testing.T) {
			res := httptest.NewRecorder()
			path := "/api/v1/system/compatibility?clientVersion=" + test.version
			api.Handler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, path, nil))
			require.Equal(t, http.StatusOK, res.Code)
			assert.Contains(t, res.Body.String(), `"code":"`+test.code+`"`)
			assert.Contains(t, res.Body.String(), `"compatible":`+map[bool]string{true: "true", false: "false"}[test.ok])
		})
	}
}

func TestManagementAPICompatibilityRejectsMalformedVersion(t *testing.T) {
	api, _, _, _ := testManagementAPI(t)
	for _, version := range []string{"1.0", "+1.0.6", "1.0.6-beta", "01.0.6"} {
		res := httptest.NewRecorder()
		path := "/api/v1/system/compatibility?clientVersion=" + url.QueryEscape(version)
		api.Handler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, path, nil))

		assert.Equal(t, http.StatusBadRequest, res.Code)
		assert.Contains(t, res.Body.String(), `"code":"invalid_client_version"`)
	}
}

func TestManagementAPILivenessDoesNotRequireReadySchema(t *testing.T) {
	api, _, _, _ := testManagementAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	res := httptest.NewRecorder()

	api.Handler().ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.JSONEq(t, `{"status":"live"}`, res.Body.String())
}

func TestRequestLoggingRecordsLowCardinalityHTTPMetrics(t *testing.T) {
	metrics := observability.NewRegistry(16)
	handler := requestLogging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Pattern = "GET /api/v1/items/{id}"
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
	}), metrics)

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/items/secret-id", nil))

	operations := operationMetrics(metrics.Snapshot())
	require.Contains(t, operations, "http.request")
	assert.EqualValues(t, 1, operations["http.request"].Count)
	assert.EqualValues(t, 1, operations["http.request"].Errors)
	assert.Contains(t, operations, "http.GET /api/v1/items/{id}")
	assert.NotContains(t, operations, "secret-id")
	counters := counterMetrics(metrics.Counters())
	assert.EqualValues(t, 1, counters["http.responses.5xx"])
	gauges := gaugeMetrics(metrics.Gauges())
	assert.Zero(t, gauges["http.in_flight"])
}

func TestManagementAPIApplyAuthenticatesAdminAndReconcilesWorker(t *testing.T) {
	api, migrations, lifecycle, audit := testManagementAPI(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/migrations/apply", nil)
	req.SetBasicAuth("admin", "Passw0rd!")
	res := httptest.NewRecorder()

	api.Handler().ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, 1, migrations.applyCalls)
	assert.Equal(t, 1, lifecycle.prepareCalls)
	assert.Equal(t, []bool{false}, lifecycle.complete)
	assert.Equal(t, []string{"MIGRATION_RUN"}, audit.actions)
}

func TestManagementAPIApplyRejectsInvalidCredentials(t *testing.T) {
	api, migrations, _, _ := testManagementAPI(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/migrations/apply", nil)
	req.SetBasicAuth("admin", "wrong")
	res := httptest.NewRecorder()

	api.Handler().ServeHTTP(res, req)

	assert.Equal(t, http.StatusUnauthorized, res.Code)
	assert.Zero(t, migrations.applyCalls)
}

func TestMigrationAPIEnforcesPasswordLifetime(t *testing.T) {
	expired := time.Now().AddDate(0, 0, -30)
	recent := time.Now().Add(-time.Hour)
	for _, operation := range []string{"apply", "rollback"} {
		for _, tc := range []struct {
			name     string
			lifetime string
			changed  *time.Time
			required bool
			allowed  bool
		}{
			{name: "expired without required flag", lifetime: "1", changed: &expired},
			{name: "missing change date", lifetime: "1"},
			{name: "unexpired password", lifetime: "1", changed: &recent, allowed: true},
			{name: "expiration disabled", lifetime: "0", changed: &expired, allowed: true},
			{name: "explicit change required", lifetime: "0", changed: &recent, required: true},
		} {
			t.Run(operation+"/"+tc.name, func(t *testing.T) {
				api, migrations, lifecycle, audit := testManagementAPI(t)
				api.authSettings = scenarioAuthSettings{value: tc.lifetime}
				user := api.users.(fakeAdminUsers).user
				user.PasswordChangedAt = tc.changed
				user.PasswordChangeRequired = tc.required
				body := strings.NewReader(`{"backupCompleted":true,"backupReference":"backup-42","acknowledgedDataLoss":true,"confirmation":"ОТКАТ МИГРАЦИИ"}`)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/migrations/"+operation, body)
				req.SetBasicAuth("admin", "Passw0rd!")
				res := httptest.NewRecorder()

				api.Handler().ServeHTTP(res, req)

				if tc.allowed {
					require.Equal(t, http.StatusOK, res.Code)
					require.Equal(t, 1, migrations.applyCalls+migrations.rollbackCalls)
				} else {
					require.Equal(t, http.StatusUnauthorized, res.Code)
					require.Contains(t, res.Body.String(), `"code":"invalid_credentials"`)
					require.Zero(t, migrations.applyCalls)
					require.Zero(t, migrations.rollbackCalls)
					require.Zero(t, lifecycle.prepareCalls)
					require.Empty(t, audit.actions)
				}
			})
		}
	}
}

func TestManagementAPIReadinessReportsMaintenance(t *testing.T) {
	api, _, lifecycle, _ := testManagementAPI(t)
	lifecycle.readyErr = models.NewConflict("schema update required")
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	res := httptest.NewRecorder()

	api.Handler().ServeHTTP(res, req)

	assert.Equal(t, http.StatusServiceUnavailable, res.Code)
	assert.Contains(t, res.Body.String(), `"status":"maintenance"`)
}

func TestManagementAPIRollbackKeepsWorkerInMaintenance(t *testing.T) {
	api, migrations, lifecycle, audit := testManagementAPI(t)
	body := strings.NewReader(`{"backupCompleted":true,"backupReference":"backup-42","acknowledgedDataLoss":true,"confirmation":"ОТКАТ МИГРАЦИИ"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/migrations/rollback", body)
	req.SetBasicAuth("admin", "Passw0rd!")
	res := httptest.NewRecorder()

	api.Handler().ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, 1, migrations.rollbackCalls)
	assert.Equal(t, []bool{true}, lifecycle.complete)
	assert.Equal(t, []string{"MIGRATION_ROLLBACK_REQUESTED", "MIGRATION_ROLLBACK"}, audit.actions)
}

func TestRemoteHostIgnoresEphemeralPort(t *testing.T) {
	assert.Equal(t, "192.0.2.10", remoteHost("192.0.2.10:54321"))
}
