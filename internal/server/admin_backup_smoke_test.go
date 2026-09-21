package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/backup"
	"github.com/stretchr/testify/require"
)

func TestBuiltAdminRestoreIntegration(t *testing.T) {
	base, dir := os.Getenv("DOCFLOW_INTEGRATION_SERVER_URL"), os.Getenv("DOCFLOW_SMOKE_ARCHIVE_DIR")
	if base == "" || dir == "" {
		t.Skip("run make storage-smoke-test")
	}
	request := func(method, path string, value any, token string, status int) []byte {
		raw, err := json.Marshal(value)
		require.NoError(t, err)
		req, err := http.NewRequest(method, base+path, bytes.NewReader(raw))
		require.NoError(t, err)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := (&http.Client{Timeout: 45 * time.Second}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, err = io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, status, resp.StatusCode, string(raw))
		return raw
	}
	login := func(user, password string) string {
		var result struct {
			AccessToken string `json:"accessToken"`
		}
		require.NoError(t, json.Unmarshal(request("POST", "/api/v1/auth/login", map[string]string{"login": user, "password": password}, "", 200), &result))
		return result.AccessToken
	}
	var token string
	reset := os.Getenv("DOCFLOW_SMOKE_RESET") != ""
	if reset {
		request("POST", "/api/v1/auth/setup", map[string]string{"password": "NewAdminPassw0rd!"}, "", 204)
		token = login("admin", "NewAdminPassw0rd!")
	} else {
		token = login("attachment-user", "AttachmentPassw0rd!")
	}
	settings := backup.DefaultSettings()
	settings.KeepCopies = 1
	settings.SMB = models.SMBDestination{Host: "samba", Share: "backups", User: "docflow"}
	request("PUT", "/api/v1/admin/backups/settings", backup.SettingsUpdate{Settings: settings, Password: "integration-password"}, token, 204)
	archives, err := filepath.Glob(filepath.Join(dir, "*.tar.gz"))
	require.NoError(t, err)
	require.Len(t, archives, 1)
	id := strings.TrimSuffix(filepath.Base(archives[0]), ".tar.gz")
	var copies []models.BackupCopy
	require.NoError(t, json.Unmarshal(request("GET", "/api/v1/admin/backups/catalog", nil, token, 200), &copies))
	var selected models.BackupCopy
	for _, copy := range copies {
		if copy.ID == id {
			selected = copy
		}
	}
	require.Equal(t, id, selected.ID)
	wait := func(start models.BackupOperationStarted) models.BackupOperation {
		deadline := time.Now().Add(3 * time.Minute)
		for time.Now().Before(deadline) {
			var op models.BackupOperation
			require.NoError(t, json.Unmarshal(request("GET", "/api/v1/admin/backups/operations/"+start.Job.ID, nil, start.StatusToken, 200), &op))
			if op.State == "completed" || op.State == "failed" || op.State == "rolled_back" || op.State == "rollback_failed" {
				return op
			}
			time.Sleep(250 * time.Millisecond)
		}
		t.Fatal("operation did not finish")
		return models.BackupOperation{}
	}
	var verification models.BackupOperationStarted
	require.NoError(t, json.Unmarshal(request("POST", "/api/v1/admin/backups/catalog/verify", models.BackupOperationRequest{CopyID: id, Confirmation: selected.RestoreConfirmation}, token, 202), &verification))
	op := wait(verification)
	require.Equal(t, "completed", op.State, op.Error)
	db, err := sql.Open("postgres", os.Getenv("DOCFLOW_INTEGRATION_DSN"))
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(`INSERT INTO users(login,password_hash,full_name,is_active) VALUES('replace-extra','unused','Must disappear',true)`)
	require.NoError(t, err)
	var restore models.BackupOperationStarted
	require.NoError(t, json.Unmarshal(request("POST", "/api/v1/admin/backups/catalog/restore", models.BackupOperationRequest{CopyID: id, VerificationID: verification.Job.ID, Confirmation: selected.RestoreConfirmation}, token, 202), &restore))
	request("GET", "/api/v1/admin/backups/operations/"+restore.Job.ID, nil, verification.StatusToken, 403)
	op = wait(restore)
	require.Equal(t, "completed", op.State, op.Error)
	request("GET", "/api/v1/auth/me", nil, token, 401)
	token = login("attachment-user", "AttachmentPassw0rd!")
	var count int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM users WHERE login='replace-extra'").Scan(&count))
	require.Zero(t, count)
	var response models.BackupSettingsResponse
	require.NoError(t, json.Unmarshal(request("GET", "/api/v1/admin/backups/settings", nil, token, 200), &response))
	require.False(t, response.Settings.Enabled)
	require.False(t, response.Settings.PasswordSet)
	// Keep the restored copy while creating another current-format backup, then
	// exercise deletion without depending on an obsolete archive format.
	if reset {
		settings.KeepCopies = 2
	}
	request("PUT", "/api/v1/admin/backups/settings", backup.SettingsUpdate{Settings: settings, Password: "integration-password"}, token, 204)
	if reset {
		var job models.BackupJob
		require.NoError(t, json.Unmarshal(request("POST", "/api/v1/admin/backups", nil, token, 202), &job))
		deadline := time.Now().Add(2 * time.Minute)
		for time.Now().Before(deadline) {
			var jobs []models.BackupJob
			require.NoError(t, json.Unmarshal(request("GET", "/api/v1/admin/backups", nil, token, 200), &jobs))
			for _, candidate := range jobs {
				if candidate.ID == job.ID {
					job = candidate
				}
			}
			if job.State == "completed" {
				break
			}
			require.NotContains(t, []string{"failed", "cancelled", "interrupted"}, job.State, job.Error)
			time.Sleep(500 * time.Millisecond)
		}
		require.Equal(t, "completed", job.State, job.Error)
		settings.KeepCopies = 1
		request("PUT", "/api/v1/admin/backups/settings", backup.SettingsUpdate{Settings: settings}, token, 204)
		var deletion models.BackupOperationStarted
		require.NoError(t, json.Unmarshal(request("POST", "/api/v1/admin/backups/catalog/delete", models.BackupOperationRequest{CopyID: id, Confirmation: selected.DeleteConfirmation}, token, 202), &deletion))
		deleted := wait(deletion)
		require.Equal(t, "completed", deleted.State, deleted.Error)
		var remote []models.BackupCopy
		require.NoError(t, json.Unmarshal(request("GET", "/api/v1/admin/backups/catalog", nil, token, 200), &remote))
		require.Len(t, remote, 1)
		require.Equal(t, job.ID, remote[0].ID)
		require.Equal(t, 3, remote[0].Format)
	} else {
		// The only existing backup is protected by the configured minimum.
		var deletion models.BackupOperationStarted
		require.NoError(t, json.Unmarshal(request("POST", "/api/v1/admin/backups/catalog/delete", models.BackupOperationRequest{CopyID: id, Confirmation: selected.DeleteConfirmation}, token, 202), &deletion))
		op = wait(deletion)
		require.Equal(t, "failed", op.State)
		require.Contains(t, op.Error, "минимум")
	}
}
