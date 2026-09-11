package server

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/backup"
	"github.com/Volkov-D-A/docs-register-and-track/internal/backup/smb"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
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
	reset := os.Getenv("DOCFLOW_SMOKE_RESET")
	if reset != "" {
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
	if reset == "v2" {
		id = "backup_20260910_090000"
	}
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
	if reset == "v2" {
		request("PUT", "/api/v1/admin/backups/settings", backup.SettingsUpdate{Settings: settings, Password: "integration-password"}, token, 204)
		var remote []models.BackupCopy
		require.NoError(t, json.Unmarshal(request("GET", "/api/v1/admin/backups/catalog", nil, token, 200), &remote))
		for _, copy := range remote {
			if copy.Format == 3 {
				var deletion models.BackupOperationStarted
				require.NoError(t, json.Unmarshal(request("POST", "/api/v1/admin/backups/catalog/delete", models.BackupOperationRequest{CopyID: copy.ID, Confirmation: copy.DeleteConfirmation}, token, 202), &deletion))
				deleted := wait(deletion)
				require.Equal(t, "completed", deleted.State, deleted.Error)
			}
		}
		require.NoError(t, json.Unmarshal(request("GET", "/api/v1/admin/backups/catalog", nil, token, 200), &remote))
		require.Len(t, remote, 1)
		require.Equal(t, 2, remote[0].Format)
	}
	if reset == "" {
		// Publish an original-contract v2 sidecar and archive for the reset run.
		publishLegacySmoke(t, archives[0])
		request("PUT", "/api/v1/admin/backups/settings", backup.SettingsUpdate{Settings: settings, Password: "integration-password"}, token, 204)
		// Both valid sets are protected when the configured minimum is two.
		settings.KeepCopies = 2
		request("PUT", "/api/v1/admin/backups/settings", backup.SettingsUpdate{Settings: settings}, token, 204)
		var deletion models.BackupOperationStarted
		require.NoError(t, json.Unmarshal(request("POST", "/api/v1/admin/backups/catalog/delete", models.BackupOperationRequest{CopyID: id, Confirmation: selected.DeleteConfirmation}, token, 202), &deletion))
		op = wait(deletion)
		require.Equal(t, "failed", op.State)
		require.Contains(t, op.Error, "минимум")
	}
}

func publishLegacySmoke(t *testing.T, archive string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	work := t.TempDir()
	m, err := backup.Unpack(ctx, archive, filepath.Join(work, "contents"), 1<<30)
	require.NoError(t, err)
	// Build a genuine older-schema v2 dump in a disposable database in the
	// isolated smoke PostgreSQL, retaining the attachment objects from v3.
	root, err := sql.Open("postgres", os.Getenv("DOCFLOW_INTEGRATION_DSN"))
	require.NoError(t, err)
	defer root.Close()
	_, err = root.ExecContext(ctx, "CREATE DATABASE docflow_legacy_smoke")
	require.NoError(t, err)
	defer root.ExecContext(context.Background(), "DROP DATABASE docflow_legacy_smoke WITH (FORCE)")
	dumpPath := filepath.Join(work, "contents", "database.dump")
	dump, err := os.Open(dumpPath)
	require.NoError(t, err)
	load := exec.CommandContext(ctx, "docker", "exec", "-i", "docflow-smoke-postgres-1", "pg_restore", "--username=docflow_integration", "--dbname=docflow_legacy_smoke", "--no-owner", "--no-acl", "--clean", "--if-exists", "--exit-on-error")
	load.Stdin = dump
	output, loadErr := load.CombinedOutput()
	dump.Close()
	require.NoError(t, loadErr, string(output))
	legacy, err := sql.Open("postgres", strings.Replace(os.Getenv("DOCFLOW_INTEGRATION_DSN"), "/docflow_test_outbox", "/docflow_legacy_smoke", 1))
	require.NoError(t, err)
	_, err = legacy.ExecContext(ctx, "DROP TABLE backup_audit,backup_jobs,backup_settings; UPDATE schema_migrations SET version=11")
	legacy.Close()
	require.NoError(t, err)
	dump, err = os.OpenFile(dumpPath, os.O_WRONLY|os.O_TRUNC, 0600)
	require.NoError(t, err)
	export := exec.CommandContext(ctx, "docker", "exec", "docflow-smoke-postgres-1", "pg_dump", "--username=docflow_integration", "--dbname=docflow_legacy_smoke", "--format=custom", "--no-owner", "--no-acl")
	export.Stdout = dump
	exportErr := export.Run()
	dump.Close()
	require.NoError(t, exportErr)
	var payload bytes.Buffer
	gz := gzip.NewWriter(&payload)
	tw := tar.NewWriter(gz)
	add := func(name, source string) {
		info, err := os.Stat(source)
		require.NoError(t, err)
		require.NoError(t, tw.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: info.Size()}))
		f, err := os.Open(source)
		require.NoError(t, err)
		_, err = io.Copy(tw, f)
		f.Close()
		require.NoError(t, err)
	}
	add("database.dump", filepath.Join(work, "contents", "database.dump"))
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "objects/", Mode: 0700, Typeflag: tar.TypeDir}))
	for _, obj := range m.Objects {
		add("objects/"+obj.Key, filepath.Join(work, "contents", obj.File))
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	file := filepath.Join(work, "legacy.tar.gz")
	require.NoError(t, os.WriteFile(file, payload.Bytes(), 0600))
	digest, size, err := backup.FileDigest(file)
	require.NoError(t, err)
	client, err := smb.Open(ctx, smb.Config{Host: "127.0.0.1", Port: 5445, Share: "backups", User: "docflow"}, "integration-password")
	require.NoError(t, err)
	defer client.Close()
	name := "backup_20260910_090000.tar.gz"
	require.NoError(t, client.Write(ctx, name, bytes.NewReader(payload.Bytes())))
	require.NoError(t, client.Write(ctx, name+".manifest", strings.NewReader(fmt.Sprintf("format_version=2\narchive=%s\nsize_bytes=%d\nsha256=%s\n", name, size, digest))))
}
