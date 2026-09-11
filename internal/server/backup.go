package server

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/backup"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func (api *managementAPI) backupSnapshot(ctx context.Context, fn func(context.Context) error) error {
	// Serialize against migration authentication/audit as well as schema changes.
	for !api.migration.TryLock() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	defer api.migration.Unlock()
	api.backupPending.Store(true)
	defer api.backupPending.Store(false)
	drain, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	for !api.schemaRequests.TryLock() {
		select {
		case <-drain.Done():
			return fmt.Errorf("request drain timed out")
		case <-time.After(25 * time.Millisecond):
		}
	}
	defer api.schemaRequests.Unlock()
	if err := api.lifecycle.PrepareRollback(); err != nil {
		return err
	}
	defer func() {
		blocked := false
		if api.backupService != nil {
			_, err := os.Stat(filepath.Join(api.backupService.Directory, "recovery-required"))
			blocked = err == nil || !os.IsNotExist(err)
		}
		api.lifecycle.CompleteRollback(blocked)
	}()
	release, acquired, err := api.acquireLease(ctx)
	if err != nil {
		return err
	}
	if !acquired {
		return fmt.Errorf("background worker lease is busy")
	}
	defer release()
	return fn(ctx)
}
func (api *managementAPI) backupRoutes(mux, control *http.ServeMux) {
	for _, kind := range []string{"verify", "restore", "delete"} {
		mux.Handle("POST /api/v1/admin/backups/catalog/"+kind, api.requirePermission(models.SystemPermissionAdmin, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !api.backupAvailable(w) {
				return
			}
			var req models.BackupOperationRequest
			if err := decodeJSON(r, &req); err != nil {
				writeAPIError(w, 400, "invalid_request", err)
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
			defer cancel()
			result, err := api.backupService.StartOperation(ctx, kind, req, authenticatedFromContext(r.Context()).User.ID.String())
			if err != nil {
				writeAPIError(w, 409, "backup_operation_failed", err)
				return
			}
			w.Header().Set("Cache-Control", "no-store")
			writeJSON(w, 202, result)
		})))
	}
	control.HandleFunc("GET /api/v1/admin/backups/operations/{id}", func(w http.ResponseWriter, r *http.Request) {
		if !api.backupAvailable(w) {
			return
		}
		token, _ := bearerToken(r.Header.Get("Authorization"))
		status, err := api.backupService.OperationStatus(r.PathValue("id"), token)
		if err != nil {
			writeAPIError(w, 403, "forbidden", models.ErrForbidden)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, 200, status)
	})
	mux.Handle("POST /api/v1/admin/backups/operations/{id}/cancel", api.requirePermission(models.SystemPermissionAdmin, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !api.backupAvailable(w) {
			return
		}
		if err := api.backupService.CancelOperation(r.PathValue("id")); err != nil {
			writeAPIError(w, 409, "backup_cancel_failed", err)
			return
		}
		w.WriteHeader(204)
	})))
	mux.Handle("GET /api/v1/admin/backups/catalog", api.requirePermission(models.SystemPermissionAdmin, http.HandlerFunc(api.backupCatalog)))
	mux.Handle("GET /api/v1/admin/backups/settings", api.requirePermission(models.SystemPermissionAdmin, http.HandlerFunc(api.backupSettings)))
	mux.Handle("PUT /api/v1/admin/backups/settings", api.requirePermission(models.SystemPermissionAdmin, http.HandlerFunc(api.saveBackupSettings)))
	mux.Handle("POST /api/v1/admin/backups/check", api.requirePermission(models.SystemPermissionAdmin, http.HandlerFunc(api.checkBackup)))
	mux.Handle("POST /api/v1/admin/backups/{id}/retry", api.requirePermission(models.SystemPermissionAdmin, http.HandlerFunc(api.retryBackup)))
	mux.Handle("POST /api/v1/admin/backups", api.requirePermission(models.SystemPermissionAdmin, http.HandlerFunc(api.startBackup)))
	control.Handle("GET /api/v1/admin/backups", http.HandlerFunc(api.backupJobs))
	control.Handle("POST /api/v1/admin/backups/{id}/cancel", http.HandlerFunc(api.cancelBackup))
}
func (api *managementAPI) backupAvailable(w http.ResponseWriter) bool {
	if api.backupService == nil {
		writeAPIError(w, 503, "backup_unavailable", fmt.Errorf("резервирование не настроено"))
		return false
	}
	return true
}

func (api *managementAPI) backupCatalog(w http.ResponseWriter, r *http.Request) {
	if !api.backupAvailable(w) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	copies, err := api.backupService.Catalog(ctx)
	if err != nil {
		writeAPIError(w, 503, "backup_catalog_failed", err)
		return
	}
	writeJSON(w, 200, copies)
}
func (api *managementAPI) backupSettings(w http.ResponseWriter, r *http.Request) {
	if !api.backupAvailable(w) {
		return
	}
	settings, err := api.backupService.Settings(r.Context())
	if err != nil {
		writeUserError(w, err)
		return
	}
	issue := ""
	if err = api.backupService.Ready(); err != nil {
		issue = err.Error()
	}
	next := ""
	if run := settings.Next(time.Now()); !run.IsZero() {
		next = run.Format(time.RFC3339)
	}
	writeJSON(w, 200, models.BackupSettingsResponse{Settings: models.BackupSettings(settings), Issue: issue, NextRun: next})
}
func (api *managementAPI) saveBackupSettings(w http.ResponseWriter, r *http.Request) {
	if !api.backupAvailable(w) {
		return
	}
	var req backup.SettingsUpdate
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, 400, "invalid_request", err)
		return
	}
	if err := api.backupService.SaveSettings(r.Context(), req, authenticatedFromContext(r.Context()).User.ID.String()); err != nil {
		writeAPIError(w, 400, "backup_settings_failed", err)
		return
	}
	w.WriteHeader(204)
}
func (api *managementAPI) checkBackup(w http.ResponseWriter, r *http.Request) {
	if !api.backupAvailable(w) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if err := api.backupService.Check(ctx); err != nil {
		writeAPIError(w, 400, "backup_check_failed", err)
		return
	}
	w.WriteHeader(204)
}
func (api *managementAPI) startBackup(w http.ResponseWriter, r *http.Request) {
	if !api.backupAvailable(w) {
		return
	}
	job, err := api.backupService.Start(r.Context(), authenticatedFromContext(r.Context()).User.ID.String(), time.Time{})
	if err != nil {
		writeAPIError(w, 409, "backup_start_failed", err)
		return
	}
	writeJSON(w, 202, job)
}

// The usual session lookup updates last_seen_at. These maintenance controls
// deliberately authenticate with SELECT only so they cannot change the snapshot.
func (api *managementAPI) backupReadOnlyAdmin(w http.ResponseWriter, r *http.Request) bool {
	if !api.backupAvailable(w) {
		return false
	}
	token, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		writeAPIError(w, 401, "authentication_required", models.ErrUnauthorized)
		return false
	}
	hash := sha256.Sum256([]byte(token))
	var allowed bool
	err := api.backupService.DB.QueryRowContext(r.Context(), `SELECT true FROM server_sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=$1 AND s.revoked_at IS NULL AND s.expires_at>now() AND u.is_active AND EXISTS (SELECT 1 FROM user_system_permissions p WHERE p.user_id=u.id AND p.permission='admin' AND p.is_allowed)`, hash[:]).Scan(&allowed)
	if err != nil || !allowed {
		if err != sql.ErrNoRows && err != nil {
			writeUserError(w, err)
		} else {
			writeAPIError(w, 403, "forbidden", models.ErrForbidden)
		}
		return false
	}
	return true
}
func (api *managementAPI) backupJobs(w http.ResponseWriter, r *http.Request) {
	if !api.backupReadOnlyAdmin(w, r) {
		return
	}
	jobs, err := api.backupService.Jobs()
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, 200, jobs)
}
func (api *managementAPI) cancelBackup(w http.ResponseWriter, r *http.Request) {
	if !api.backupReadOnlyAdmin(w, r) {
		return
	}
	if err := api.backupService.Cancel(r.PathValue("id")); err != nil {
		writeAPIError(w, 409, "backup_cancel_failed", err)
		return
	}
	w.WriteHeader(204)
}

func (api *managementAPI) retryBackup(w http.ResponseWriter, r *http.Request) {
	if !api.backupAvailable(w) {
		return
	}
	if err := api.backupService.Retry(r.Context(), r.PathValue("id")); err != nil {
		writeAPIError(w, 409, "backup_retry_failed", err)
		return
	}
	w.WriteHeader(204)
}
