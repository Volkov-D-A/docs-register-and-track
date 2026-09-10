package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Volkov-D-A/docs-register-and-track/internal/backup"
	"github.com/stretchr/testify/require"
)

func TestBackupSnapshotDrainsRequestsAndHoldsLease(t *testing.T) {
	api, _, _, _ := testManagementAPI(t)
	acquired, released := false, false
	api.acquireLease = func(context.Context) (func(), bool, error) {
		acquired = true
		return func() { released = true }, true, nil
	}
	api.schemaRequests.RLock()
	done := make(chan error, 1)
	entered := make(chan struct{})
	go func() {
		done <- api.backupSnapshot(context.Background(), func(context.Context) error {
			require.True(t, acquired)
			require.False(t, released)
			close(entered)
			return nil
		})
	}()
	require.Eventually(t, api.backupPending.Load, time.Second, time.Millisecond)
	select {
	case <-entered:
		t.Fatal("snapshot raced admitted request")
	default:
	}
	res := httptest.NewRecorder()
	api.Handler().ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil))
	require.Equal(t, 503, res.Code)
	api.schemaRequests.RUnlock()
	require.NoError(t, <-done)
	require.True(t, released)
	require.False(t, api.backupPending.Load())
}
func TestBackupStatusUsesReadOnlyAuthentication(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	api, _, _, _ := testManagementAPI(t)
	api.backupService = &backup.Service{DB: db, Directory: t.TempDir()}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/backups", nil)
	req.Header.Set("Authorization", "Bearer test")
	mock.ExpectQuery("SELECT true FROM server_sessions").WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"allowed"}).AddRow(true))
	res := httptest.NewRecorder()
	api.Handler().ServeHTTP(res, req)
	require.Equal(t, 200, res.Code, res.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
	res = httptest.NewRecorder()
	api.Handler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/admin/backups", nil))
	require.Equal(t, 401, res.Code)
	mock.ExpectQuery("SELECT true FROM server_sessions").WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"allowed"}))
	res = httptest.NewRecorder()
	api.Handler().ServeHTTP(res, req)
	require.Equal(t, 403, res.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}
