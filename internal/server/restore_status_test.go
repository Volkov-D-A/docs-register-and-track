package server

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/backup"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRestoreStatusRemainsScopedWhileBusinessAPIIsBlocked(t *testing.T) {
	dir := t.TempDir()
	id := uuid.NewString()
	token := "status-capability"
	hash := sha256.Sum256([]byte(token))
	raw, err := json.Marshal(map[string]any{"id": id, "copyId": uuid.NewString(), "kind": "restore", "state": "restoring", "tokenHash": hash[:], "expiresAt": time.Now().Add(time.Hour)})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, id+".operation.json"), raw, 0600))
	api, _, _, _ := testManagementAPI(t)
	api.backupService = &backup.Service{Directory: dir}
	api.replacementPending.Store(true)
	check := func(handler http.Handler, path, credential string, status int) {
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("Authorization", "Bearer "+credential)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		require.Equal(t, status, res.Code, res.Body.String())
		if status == 200 {
			require.Equal(t, "no-store", res.Header().Get("Cache-Control"))
			require.NotContains(t, res.Body.String(), token)
		}
	}
	check(api.Handler(), "/api/v1/admin/backups/operations/"+id, token, 200)
	check(api.Handler(), "/api/v1/admin/backups/operations/"+id, "", 403)
	check(api.Handler(), "/api/v1/admin/backups/operations/"+uuid.NewString(), token, 403)
	check(api.Handler(), "/api/v1/admin/migrations", token, 503)
	check(api.Handler(), "/api/v1/auth/me", token, 503)
	cfg := validConfig()
	cfg.Backup.Directory = dir
	require.NoError(t, os.WriteFile(filepath.Join(dir, "recovery-required"), []byte(id), 0600))
	app, err := New(cfg)
	require.NoError(t, err)
	require.Nil(t, app.db)
	check(app.http.Handler, "/api/v1/admin/backups/operations/"+id, token, 200)
	check(app.http.Handler, "/api/v1/auth/me", token, 503)
	for _, phase := range []string{"clearing_database", "clearing_objects", "restoring", "migrating", "rollback_restoring", "rollback_failed", "finalizing"} {
		state, err := json.Marshal(map[string]any{"id": id, "copyId": uuid.NewString(), "kind": "restore", "state": phase, "tokenHash": hash[:], "expiresAt": time.Now().Add(time.Hour)})
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, id+".operation.json"), state, 0600))
		restarted, err := New(cfg)
		require.NoError(t, err)
		require.Nil(t, restarted.db, phase)
		check(restarted.http.Handler, "/api/v1/admin/backups/operations/"+id, token, 200)
		check(restarted.http.Handler, "/api/v1/admin/migrations", token, 503)
	}
}

func TestBackupManagementRequiresAdministrator(t *testing.T) {
	hash := sha256.Sum256([]byte("ordinary-session"))
	user := &models.User{ID: uuid.New(), IsActive: true}
	api := &managementAPI{authUsers: &fakeAuthUsers{user: user}, authSettings: fakeAuthSettings{}, sessions: &fakeAuthSessions{hash: hash[:], session: &models.ServerSession{UserID: user.ID, ExpiresAt: time.Now().Add(time.Hour)}}}
	for _, path := range []string{"/api/v1/admin/backups/catalog/verify", "/api/v1/admin/backups/catalog/restore", "/api/v1/admin/backups/catalog/delete", "/api/v1/admin/backups/operations/" + uuid.NewString() + "/cancel"} {
		for _, token := range []string{"", "ordinary-session", "status-capability"} {
			req := httptest.NewRequest("POST", path, nil)
			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}
			res := httptest.NewRecorder()
			api.Handler().ServeHTTP(res, req)
			if token == "ordinary-session" {
				require.Equal(t, 403, res.Code, res.Body.String())
			} else {
				require.Equal(t, 401, res.Code, res.Body.String())
			}
		}
	}
}

func TestLifetimeLeaseSurvivesPoolReplacementIntegration(t *testing.T) {
	dsn := os.Getenv("DOCFLOW_INTEGRATION_DSN")
	if dsn == "" {
		t.Skip("run make integration-test")
	}
	pool, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer pool.Close()
	lease, err := acquireInstance(context.Background(), pool)
	require.NoError(t, err)
	defer releaseInstance(lease)
	require.NoError(t, pool.Close())
	other, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer other.Close()
	conn, err := other.Conn(context.Background())
	require.NoError(t, err)
	defer conn.Close()
	var acquired bool
	require.NoError(t, conn.QueryRowContext(context.Background(), "SELECT pg_try_advisory_lock($1)", instanceLeaseID).Scan(&acquired))
	require.False(t, acquired)
}
