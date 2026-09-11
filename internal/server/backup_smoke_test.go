package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/backup"
	"github.com/Volkov-D-A/docs-register-and-track/internal/backup/smb"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/stretchr/testify/require"
)

func TestBuiltServerBackupIntegration(t *testing.T) {
	base := os.Getenv("DOCFLOW_INTEGRATION_SERVER_URL")
	directory := os.Getenv("DOCFLOW_SMOKE_ARCHIVE_DIR")
	if base == "" || directory == "" {
		t.Skip("run backup smoke test")
	}
	request := func(method, path string, value any, token string, status int) []byte {
		var body io.Reader
		if value != nil {
			raw, err := json.Marshal(value)
			require.NoError(t, err)
			body = bytes.NewReader(raw)
		}
		req, err := http.NewRequest(method, base+path, body)
		require.NoError(t, err)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		client := http.Client{Timeout: 45 * time.Second}
		res, err := client.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()
		raw, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, status, res.StatusCode, string(raw))
		return raw
	}
	var session struct {
		AccessToken string `json:"accessToken"`
	}
	require.NoError(t, json.Unmarshal(request("POST", "/api/v1/auth/login", map[string]string{"login": "attachment-user", "password": "AttachmentPassw0rd!"}, "", 200), &session))
	token := session.AccessToken
	settings := backup.DefaultSettings()
	settings.SMB = models.SMBDestination{Host: "samba", Share: "backups", User: "docflow"}
	request("PUT", "/api/v1/admin/backups/settings", backup.SettingsUpdate{Settings: settings, Password: "integration-password"}, token, 204)
	request("POST", "/api/v1/admin/backups/check", nil, token, 204)
	raw := request("GET", "/api/v1/admin/backups/settings", nil, token, 200)
	require.NotContains(t, string(raw), "integration-password")
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
	db, err := sql.Open("postgres", os.Getenv("DOCFLOW_INTEGRATION_DSN"))
	require.NoError(t, err)
	defer db.Close()
	var stored string
	require.NoError(t, db.QueryRow("SELECT settings::text FROM backup_settings").Scan(&stored))
	require.NotContains(t, stored, "integration-password")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	client, err := smb.Open(ctx, smb.Config{Host: "127.0.0.1", Port: 5445, Share: "backups", User: "docflow"}, "integration-password")
	require.NoError(t, err)
	defer client.Close()
	archive, err := backup.DownloadCopy(ctx, client, job.ID, directory, 1<<30)
	require.NoError(t, err)
	require.NoError(t, os.Chmod(archive, 0644))
}
