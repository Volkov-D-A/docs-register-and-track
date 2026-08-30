package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
)

type fakeManagementMigrations struct {
	status        database.MigrationStatus
	applyErr      error
	rollbackErr   error
	applyCalls    int
	rollbackCalls int
}

func (m *fakeManagementMigrations) RunMigrations(string) error {
	m.applyCalls++
	return m.applyErr
}
func (m *fakeManagementMigrations) GetMigrationStatus(string) (*database.MigrationStatus, error) {
	return &m.status, nil
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
	migrations := &fakeManagementMigrations{status: database.MigrationStatus{CurrentVersion: 8, LatestAvailableVersion: 8, UpToDate: true, Compatible: true}}
	lifecycle := &fakeManagementLifecycle{}
	audit := &fakeAdminAudit{}
	api := &managementAPI{
		migrations: migrations,
		lifecycle:  lifecycle,
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

func TestManagementAPILivenessDoesNotRequireReadySchema(t *testing.T) {
	api, _, _, _ := testManagementAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	res := httptest.NewRecorder()

	api.Handler().ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.JSONEq(t, `{"status":"live"}`, res.Body.String())
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
