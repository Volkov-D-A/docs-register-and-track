package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/background"
)

var maintenanceBusinessRoutes = []struct{ method, path string }{
	{"POST", "/api/v1/auth/login"},
	{"GET", "/api/v1/auth/setup-required"},
	{"POST", "/api/v1/auth/setup"},
	{"POST", "/api/v1/auth/change-required-password"},
	{"GET", "/api/v1/auth/me"},
	{"GET", "/api/v1/documents/00000000-0000-0000-0000-000000000001"},
	{"POST", "/api/v1/documents/incoming"},
	{"POST", "/api/v1/documents/00000000-0000-0000-0000-000000000001/attachments"},
	{"POST", "/api/v1/assignments/00000000-0000-0000-0000-000000000001/attachments"},
	{"GET", "/api/v1/attachments/00000000-0000-0000-0000-000000000001/content"},
	{"PATCH", "/api/v1/settings/example"},
	{"POST", "/api/v1/telemetry/logs"},
	{"GET", "/api/v1/admin/outbox/stats"},
	{"POST", "/api/v1/future-business-route"},
}

func assertBusinessMaintenance(t *testing.T, handler http.Handler) {
	t.Helper()
	// The API has no session, document, or attachment dependencies: reaching any
	// of those layers with a bearer token would panic instead of returning 503.
	for _, route := range maintenanceBusinessRoutes {
		req := httptest.NewRequest(route.method, route.path, strings.NewReader(`{}`))
		req.Header.Set("Authorization", "Bearer test-token")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		require.Equal(t, http.StatusServiceUnavailable, res.Code, "%s %s: %s", route.method, route.path, res.Body.String())
		assert.Contains(t, res.Body.String(), `"code":"maintenance"`)
		assert.NotContains(t, res.Body.String(), "private")
	}
}

func assertMaintenanceControls(t *testing.T, handler http.Handler) {
	t.Helper()
	for _, path := range []string{"/health/live", "/api/v1/system/status", "/api/v1/system/compatibility?clientVersion=1.0.6", "/api/v1/admin/migrations"} {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, path, nil))
		assert.Equal(t, http.StatusOK, res.Code, path)
	}
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	assert.Equal(t, http.StatusServiceUnavailable, res.Code)
}

func TestMaintenanceBlocksBusinessBeforeAuthentication(t *testing.T) {
	api, _, lifecycle, _ := testManagementAPI(t)
	lifecycle.readyErr = errors.New("private schema detail")
	handler := api.Handler()
	assertBusinessMaintenance(t, handler)
	assertMaintenanceControls(t, handler)
	for _, action := range []string{"apply", "rollback"} {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/admin/migrations/"+action, nil))
		assert.Equal(t, http.StatusUnauthorized, res.Code)
	}
	lifecycle.readyErr = nil
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil))
	assert.Equal(t, http.StatusUnauthorized, res.Code)
}

type heldSchemaMigration struct {
	*fakeManagementMigrations
	entered chan struct{}
	release chan struct{}
	err     error
}

func (m *heldSchemaMigration) RunMigrations(string) error {
	close(m.entered)
	<-m.release
	return m.err
}
func (m *heldSchemaMigration) RollbackMigration(string) error {
	close(m.entered)
	<-m.release
	return m.err
}

func TestMaintenanceDrainsRequestsAndBlocksDuringSchemaChanges(t *testing.T) {
	for _, action := range []string{"apply", "rollback"} {
		for _, fail := range []bool{false, true} {
			name := action
			if fail {
				name += "-failure"
			}
			t.Run(name, func(t *testing.T) {
				api, migrations, _, _ := testManagementAPI(t)
				held := &heldSchemaMigration{fakeManagementMigrations: migrations, entered: make(chan struct{}), release: make(chan struct{})}
				if fail {
					held.err = errors.New("migration failed")
				}
				api.migrations = held
				lifecycle := background.NewLifecycle(migrations, nil, nil)
				lifecycle.ReconcileSchema()
				api.lifecycle = lifecycle
				handler := api.Handler()

				requestEntered := make(chan struct{})
				releaseRequest := make(chan struct{})
				requestDone := make(chan struct{})
				draining := api.requireReadySchema(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
					close(requestEntered)
					<-releaseRequest
				}))
				go func() {
					defer close(requestDone)
					draining.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil))
				}()
				<-requestEntered
				// Always unblock goroutines, including when an assertion fails.
				defer func() { close(held.release) }()
				requestReleased := false
				defer func() {
					if !requestReleased {
						close(releaseRequest)
					}
				}()

				body := strings.NewReader(`{"backupCompleted":true,"backupReference":"backup-42","acknowledgedDataLoss":true,"confirmation":"ОТКАТ МИГРАЦИИ"}`)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/migrations/"+action, body)
				req.SetBasicAuth("admin", "Passw0rd!")
				res := httptest.NewRecorder()
				migrationDone := make(chan struct{})
				go func() { defer close(migrationDone); handler.ServeHTTP(res, req) }()

				require.Eventually(t, func() bool {
					probe := httptest.NewRecorder()
					handler.ServeHTTP(probe, httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil))
					return probe.Code == http.StatusServiceUnavailable
				}, 5*time.Second, time.Millisecond)
				select {
				case <-held.entered:
					t.Fatal("schema change began before the admitted request completed")
				default:
				}
				close(releaseRequest)
				requestReleased = true
				<-requestDone
				select {
				case <-held.entered:
				case <-time.After(5 * time.Second):
					t.Fatal("migration did not start after requests drained")
				}
				assertBusinessMaintenance(t, handler)
				assertMaintenanceControls(t, handler)
				// Sending releases the migration without closing the channel twice.
				held.release <- struct{}{}
				select {
				case <-migrationDone:
				case <-time.After(5 * time.Second):
					t.Fatal("migration did not finish")
				}
				expected := http.StatusOK
				if fail {
					expected = http.StatusConflict
				}
				require.Equal(t, expected, res.Code, res.Body.String())
				if action == "rollback" && !fail {
					assertBusinessMaintenance(t, handler)
				} else {
					require.NoError(t, lifecycle.CheckReady())
					probe := httptest.NewRecorder()
					handler.ServeHTTP(probe, httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil))
					assert.Equal(t, http.StatusUnauthorized, probe.Code)
				}
			})
		}
	}
}
