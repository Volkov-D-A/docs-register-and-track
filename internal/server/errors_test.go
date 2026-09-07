package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const privateErrorDetail = "pq: secret_table password=secret-token postgres://private-host /srv/private/config"

func captureAPILog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &output
}

func TestAPIErrorsSeparatePublicTextAndDiagnosticCause(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		code    string
		err     error
		message string
	}{
		{"raw internal", 500, "operation_failed", errors.New(privateErrorDetail), "Произошла внутренняя ошибка сервера."},
		{"classified internal", 500, "internal_error", models.NewInternal(privateErrorDetail, errors.New("root private cause")), "Произошла внутренняя ошибка сервера."},
		{"unclassified conflict", 409, "conflict", errors.New(privateErrorDetail), "Операция не может быть выполнена из-за конфликта."},
		{"migration conflict", 409, "migration_apply_failed", errors.New(privateErrorDetail), "Не удалось применить миграции базы данных."},
		{"validation", 400, "validation_error", fmt.Errorf("private wrapper: %w", models.NewBadRequestWrapped("Номер обязателен", errors.New(privateErrorDetail))), "Номер обязателен"},
		{"untrusted app error", 400, "validation_error", &models.AppError{Code: 400, Message: privateErrorDetail}, "Проверьте параметры запроса."},
		{"maintenance", 503, "maintenance", errors.New(privateErrorDetail), "Сервис временно недоступен: обслуживание базы данных."},
		{"nil", 500, "operation_failed", nil, "Произошла внутренняя ошибка сервера."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := captureAPILog(t)
			res := httptest.NewRecorder()
			handler := requestLogging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { writeAPIError(w, tt.status, tt.code, tt.err) }), nil)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
			req.Header.Set(requestIDHeader, "client-supplied-secret")
			handler.ServeHTTP(res, req)
			require.Equal(t, tt.status, res.Code)
			var result apiErrorResponse
			require.NoError(t, json.Unmarshal(res.Body.Bytes(), &result))
			require.Equal(t, tt.message, result.Error)
			require.Equal(t, tt.code, result.Code)
			_, err := uuid.Parse(result.RequestID)
			require.NoError(t, err)
			assert.Equal(t, res.Header().Get(requestIDHeader), result.RequestID)
			assert.Contains(t, log.String(), result.RequestID)
			assert.NotContains(t, res.Body.String(), "private")
			assert.NotContains(t, res.Body.String(), "secret")
			if tt.err != nil {
				assert.Contains(t, log.String(), privateErrorDetail)
			}
			if tt.name == "classified internal" {
				assert.Contains(t, log.String(), "root private cause")
			}
		})
	}
}

func TestWriteUserErrorRetainsOriginalCause(t *testing.T) {
	for _, err := range []error{errors.New(privateErrorDetail), models.NewInternal("generic internal", errors.New(privateErrorDetail)), &pq.Error{Code: "23505", Message: privateErrorDetail}} {
		log := captureAPILog(t)
		res := httptest.NewRecorder()
		writeUserError(res, err)
		assert.NotContains(t, res.Body.String(), privateErrorDetail)
		assert.Contains(t, log.String(), privateErrorDetail)
	}
}

func TestManagementEndpointsDoNotExposeDependencyErrors(t *testing.T) {
	for _, scenario := range []string{"status", "ready-maintenance", "ready-dependency", "apply", "rollback"} {
		t.Run(scenario, func(t *testing.T) {
			log := captureAPILog(t)
			api, migrations, lifecycle, _ := testManagementAPI(t)
			path, method, status := "/api/v1/admin/migrations", http.MethodGet, http.StatusInternalServerError
			switch scenario {
			case "status":
				migrations.statusErr = errors.New(privateErrorDetail)
			case "ready-maintenance":
				lifecycle.readyErr = errors.New(privateErrorDetail)
				path, status = "/health/ready", http.StatusServiceUnavailable
			case "ready-dependency":
				api.readinessCheck = func(context.Context, *config.Config) error { return errors.New(privateErrorDetail) }
				path, status = "/health/ready", http.StatusServiceUnavailable
			case "apply":
				migrations.applyErr = errors.New(privateErrorDetail)
				path, method, status = "/api/v1/admin/migrations/apply", http.MethodPost, http.StatusConflict
			case "rollback":
				migrations.rollbackErr = errors.New(privateErrorDetail)
				path, method, status = "/api/v1/admin/migrations/rollback", http.MethodPost, http.StatusConflict
			}
			body := bytes.NewBufferString(`{"backupCompleted":true,"backupReference":"backup","acknowledgedDataLoss":true,"confirmation":"ОТКАТ МИГРАЦИИ"}`)
			req := httptest.NewRequest(method, path, body)
			req.SetBasicAuth("admin", "Passw0rd!")
			res := httptest.NewRecorder()
			api.Handler().ServeHTTP(res, req)
			require.Equal(t, status, res.Code, res.Body.String())
			assert.NotContains(t, res.Body.String(), privateErrorDetail)
			assert.Contains(t, log.String(), privateErrorDetail)
			assert.Contains(t, res.Body.String(), res.Header().Get(requestIDHeader))
			if scenario == "ready-maintenance" {
				assert.Contains(t, res.Body.String(), `"status":"maintenance"`)
			}
			if scenario == "ready-dependency" {
				assert.Contains(t, res.Body.String(), `"status":"not_ready"`)
			}
		})
	}
}

func TestRequestIDsAreUniqueAndIndependentOfCaller(t *testing.T) {
	handler := requestLogging(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }), nil)
	ids := make(map[string]bool)
	for i := 0; i < 3; i++ {
		res := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(requestIDHeader, "same-client-value")
		handler.ServeHTTP(res, req)
		id := res.Header().Get(requestIDHeader)
		_, err := uuid.Parse(id)
		require.NoError(t, err)
		assert.False(t, ids[id])
		ids[id] = true
	}
}

type unavailableAuthUsers struct{ fakeAuthUsers }

func (unavailableAuthUsers) GetByLogin(string) (*models.User, error) {
	return nil, errors.New(privateErrorDetail)
}

type unavailableAuthSessions struct{ fakeAuthSessions }

func (unavailableAuthSessions) GetActiveByTokenHash([]byte, time.Time) (*models.ServerSession, error) {
	return nil, errors.New(privateErrorDetail)
}

func TestAuthenticationDependencyErrorsArePrivate(t *testing.T) {
	for _, route := range []string{"/api/v1/auth/login", "/api/v1/auth/me"} {
		t.Run(route, func(t *testing.T) {
			log := captureAPILog(t)
			api := &managementAPI{authUsers: &unavailableAuthUsers{}, sessions: &unavailableAuthSessions{}}
			method := http.MethodGet
			if route == "/api/v1/auth/login" {
				method = http.MethodPost
			}
			req := httptest.NewRequest(method, route, bytes.NewBufferString(`{"login":"user","password":"not-logged-password"}`))
			req.Header.Set("Authorization", "Bearer not-logged-token")
			res := httptest.NewRecorder()
			api.Handler().ServeHTTP(res, req)
			require.Equal(t, http.StatusInternalServerError, res.Code)
			assert.NotContains(t, res.Body.String(), privateErrorDetail)
			assert.Contains(t, log.String(), privateErrorDetail)
			assert.NotContains(t, log.String(), "not-logged-password")
			assert.NotContains(t, log.String(), "not-logged-token")
		})
	}
}
