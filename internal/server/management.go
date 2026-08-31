package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
)

const rollbackConfirmationPhrase = "ОТКАТ МИГРАЦИИ"
const maxAuthenticationFailureKeys = 4096

type rollbackRequest struct {
	BackupCompleted      bool   `json:"backupCompleted"`
	BackupReference      string `json:"backupReference"`
	AcknowledgedDataLoss bool   `json:"acknowledgedDataLoss"`
	Confirmation         string `json:"confirmation"`
}

type managementAPI struct {
	cfg           *config.Config
	migrations    migrationStore
	lifecycle     migrationLifecycle
	users         adminUserStore
	userCommands  userManagementStore
	userAccess    userAccessManagementStore
	substitutions userSubstitutionManagementStore
	departments   departmentManagementStore
	references    referenceManagementStore
	nomenclature  nomenclatureManagementStore
	audit         adminAuditStore
	authUsers     authUserStore
	authSettings  authSettingsStore
	sessions      authSessionStore
	acquireLease  func(context.Context) (func(), bool, error)
	migration     sync.Mutex
	authMu        sync.Mutex
	authFailures  map[string]authFailure
}

type authFailure struct {
	count   int
	resetAt time.Time
}

type migrationStore interface {
	RunMigrations(string) error
	GetMigrationStatus(string) (*database.MigrationStatus, error)
	RollbackMigration(string) error
}

type migrationLifecycle interface {
	CheckReady() error
	PrepareRollback() error
	CompleteRollback(bool)
}

type adminUserStore interface {
	GetByLogin(string) (*models.User, error)
}

type adminAuditStore interface {
	Create(models.CreateAdminAuditLogRequest) (uuid.UUID, error)
}

func newManagementAPI(app *App) *managementAPI {
	users := repository.NewUserRepository(app.db)
	outboxRepo := repository.NewOutboxRepository(app.db)
	users.SetOutbox(outboxRepo)
	access := repository.NewDocumentAccessRepository(app.db)
	access.SetOutbox(outboxRepo)
	substitutions := repository.NewUserSubstitutionRepository(app.db)
	substitutions.SetOutbox(outboxRepo)
	departments := repository.NewDepartmentRepository(app.db)
	departments.SetOutbox(outboxRepo)
	references := repository.NewReferenceRepository(app.db)
	references.SetOutbox(outboxRepo)
	nomenclature := repository.NewNomenclatureRepository(app.db)
	nomenclature.SetOutbox(outboxRepo)
	return &managementAPI{
		cfg:           app.cfg,
		migrations:    app.db,
		lifecycle:     app.lifecycle,
		users:         users,
		userCommands:  users,
		userAccess:    access,
		substitutions: substitutions,
		departments:   departments,
		references:    references,
		nomenclature:  nomenclature,
		authUsers:     users,
		authSettings:  repository.NewSettingsRepository(app.db),
		sessions:      repository.NewServerSessionRepository(app.db),
		audit:         repository.NewAdminAuditLogRepository(app.db),
		acquireLease: func(ctx context.Context) (func(), bool, error) {
			lease, acquired, err := app.db.TryAcquireBackgroundWorkerLease(ctx)
			if err != nil || !acquired {
				return nil, acquired, err
			}
			return func() {
				releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := lease.Release(releaseCtx); err != nil {
					slog.Error("failed to release schema migration lease", "error", err)
				}
			}, true, nil
		},
		authFailures: make(map[string]authFailure),
	}
}

func (api *managementAPI) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", api.live)
	mux.HandleFunc("GET /health/ready", api.ready)
	mux.HandleFunc("POST /api/v1/auth/login", api.login)
	mux.HandleFunc("POST /api/v1/auth/change-required-password", api.changeRequiredPassword)
	mux.Handle("POST /api/v1/auth/logout", api.requireSession(http.HandlerFunc(api.logout)))
	mux.Handle("GET /api/v1/auth/me", api.requireSession(http.HandlerFunc(api.me)))
	mux.Handle("POST /api/v1/auth/change-password", api.requireSession(http.HandlerFunc(api.changePassword)))
	mux.Handle("GET /api/v1/users", api.requirePermission(models.SystemPermissionAdmin, http.HandlerFunc(api.listUsers)))
	mux.Handle("POST /api/v1/users", api.requirePermission(models.SystemPermissionAdmin, http.HandlerFunc(api.createUser)))
	mux.Handle("PATCH /api/v1/users/{id}", api.requirePermission(models.SystemPermissionAdmin, http.HandlerFunc(api.updateUser)))
	mux.Handle("POST /api/v1/users/{id}/reset-password", api.requirePermission(models.SystemPermissionAdmin, http.HandlerFunc(api.resetUserPassword)))
	mux.Handle("GET /api/v1/users/{id}/access-profile", api.requirePermission(models.SystemPermissionAdmin, http.HandlerFunc(api.getUserAccessProfile)))
	mux.Handle("PUT /api/v1/users/{id}/access-profile", api.requirePermission(models.SystemPermissionAdmin, http.HandlerFunc(api.updateUserAccessProfile)))
	mux.Handle("GET /api/v1/users/{id}/substitution", api.requirePermission(models.SystemPermissionAdmin, http.HandlerFunc(api.getUserSubstitution)))
	mux.Handle("PUT /api/v1/users/{id}/substitution", api.requirePermission(models.SystemPermissionAdmin, http.HandlerFunc(api.updateUserSubstitution)))
	mux.Handle("GET /api/v1/departments", api.requireSession(http.HandlerFunc(api.listDepartments)))
	mux.Handle("POST /api/v1/departments", api.requirePermission(models.SystemPermissionAdmin, http.HandlerFunc(api.createDepartment)))
	mux.Handle("PATCH /api/v1/departments/{id}", api.requirePermission(models.SystemPermissionAdmin, http.HandlerFunc(api.updateDepartment)))
	mux.Handle("DELETE /api/v1/departments/{id}", api.requirePermission(models.SystemPermissionAdmin, http.HandlerFunc(api.deleteDepartment)))
	mux.Handle("GET /api/v1/references/organizations", api.requireSession(http.HandlerFunc(api.listOrganizations)))
	mux.Handle("POST /api/v1/references/organizations/resolve", api.requireSession(http.HandlerFunc(api.resolveOrganization)))
	mux.Handle("PATCH /api/v1/references/organizations/{id}", api.requirePermission(models.SystemPermissionReferences, http.HandlerFunc(api.updateOrganization)))
	mux.Handle("DELETE /api/v1/references/organizations/{id}", api.requirePermission(models.SystemPermissionReferences, http.HandlerFunc(api.deleteOrganization)))
	mux.Handle("POST /api/v1/references/organizations/{id}/merge", api.requirePermission(models.SystemPermissionReferences, http.HandlerFunc(api.mergeOrganization)))
	mux.Handle("GET /api/v1/references/resolution-executors", api.requireSession(http.HandlerFunc(api.listResolutionExecutors)))
	mux.Handle("POST /api/v1/references/resolution-executors/resolve", api.requireSession(http.HandlerFunc(api.resolveResolutionExecutor)))
	mux.Handle("PATCH /api/v1/references/resolution-executors/{id}", api.requirePermission(models.SystemPermissionReferences, http.HandlerFunc(api.updateResolutionExecutor)))
	mux.Handle("DELETE /api/v1/references/resolution-executors/{id}", api.requirePermission(models.SystemPermissionReferences, http.HandlerFunc(api.deleteResolutionExecutor)))
	mux.Handle("GET /api/v1/nomenclature", api.requireSession(http.HandlerFunc(api.listNomenclature)))
	mux.Handle("GET /api/v1/nomenclature/active", api.requireSession(http.HandlerFunc(api.listActiveNomenclature)))
	mux.Handle("POST /api/v1/nomenclature", api.requirePermission(models.SystemPermissionAdmin, http.HandlerFunc(api.createNomenclature)))
	mux.Handle("PATCH /api/v1/nomenclature/{id}", api.requirePermission(models.SystemPermissionAdmin, http.HandlerFunc(api.updateNomenclature)))
	mux.Handle("DELETE /api/v1/nomenclature/{id}", api.requirePermission(models.SystemPermissionAdmin, http.HandlerFunc(api.deleteNomenclature)))
	mux.Handle("GET /api/v1/access/current", api.requireSession(http.HandlerFunc(api.currentAccessSummary)))
	mux.Handle("PATCH /api/v1/profile", api.requireSession(http.HandlerFunc(api.updateOwnProfile)))
	mux.Handle("GET /api/v1/profile/substitution-candidates", api.requireSession(http.HandlerFunc(api.listOwnSubstitutionCandidates)))
	mux.Handle("GET /api/v1/profile/substitution", api.requireSession(http.HandlerFunc(api.getOwnSubstitution)))
	mux.Handle("PUT /api/v1/profile/substitution", api.requireSession(http.HandlerFunc(api.updateOwnSubstitution)))
	mux.HandleFunc("GET /api/v1/admin/migrations", api.status)
	mux.HandleFunc("POST /api/v1/admin/migrations/apply", api.apply)
	mux.HandleFunc("POST /api/v1/admin/migrations/rollback", api.rollback)
	return requestLogging(mux)
}

func (api *managementAPI) live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "live"})
}

func (api *managementAPI) ready(w http.ResponseWriter, r *http.Request) {
	if err := api.lifecycle.CheckReady(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "maintenance", "error": err.Error()})
		return
	}
	if err := HealthCheck(r.Context(), api.cfg); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready", "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (api *managementAPI) status(w http.ResponseWriter, _ *http.Request) {
	status, err := api.migrations.GetMigrationStatus(database.DefaultMigrationsPath)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "migration_status_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (api *managementAPI) apply(w http.ResponseWriter, r *http.Request) {
	user, ok := api.authenticateAdmin(w, r)
	if !ok {
		return
	}
	api.migration.Lock()
	defer api.migration.Unlock()

	if err := api.lifecycle.PrepareRollback(); err != nil {
		writeAPIError(w, http.StatusConflict, "worker_stop_failed", err)
		return
	}
	defer api.lifecycle.CompleteRollback(false)
	releaseLease, err := api.acquireSchemaLease(r)
	if err != nil {
		writeAPIError(w, http.StatusConflict, "migration_lock_busy", err)
		return
	}
	defer releaseLease()
	if err := api.migrations.RunMigrations(database.DefaultMigrationsPath); err != nil {
		writeAPIError(w, http.StatusConflict, "migration_apply_failed", err)
		return
	}
	api.auditAction(user, "MIGRATION_RUN", "Применены миграции БД через docflow-server")
	slog.Info("database migrations applied", "admin_user_id", user.ID)
	status, err := api.migrations.GetMigrationStatus(database.DefaultMigrationsPath)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "migration_status_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (api *managementAPI) rollback(w http.ResponseWriter, r *http.Request) {
	user, ok := api.authenticateAdmin(w, r)
	if !ok {
		return
	}
	var req rollbackRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	if err := validateRollback(req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_rollback_confirmation", err)
		return
	}

	api.migration.Lock()
	defer api.migration.Unlock()
	if err := api.lifecycle.PrepareRollback(); err != nil {
		writeAPIError(w, http.StatusConflict, "worker_stop_failed", err)
		return
	}
	succeeded := false
	defer func() { api.lifecycle.CompleteRollback(succeeded) }()
	releaseLease, err := api.acquireSchemaLease(r)
	if err != nil {
		writeAPIError(w, http.StatusConflict, "migration_lock_busy", err)
		return
	}
	defer releaseLease()
	api.auditAction(user, "MIGRATION_ROLLBACK_REQUESTED", fmt.Sprintf("Запрошен откат; backup: %s", strings.TrimSpace(req.BackupReference)))
	if err := api.migrations.RollbackMigration(database.DefaultMigrationsPath); err != nil {
		writeAPIError(w, http.StatusConflict, "migration_rollback_failed", err)
		return
	}
	succeeded = true
	api.auditAction(user, "MIGRATION_ROLLBACK", fmt.Sprintf("Откачена последняя миграция; backup: %s", strings.TrimSpace(req.BackupReference)))
	slog.Warn("database migration rolled back", "admin_user_id", user.ID, "backup_reference", strings.TrimSpace(req.BackupReference))
	status, err := api.migrations.GetMigrationStatus(database.DefaultMigrationsPath)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "migration_status_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (api *managementAPI) acquireSchemaLease(r *http.Request) (func(), error) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	release, acquired, err := api.acquireLease(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire schema migration lease: %w", err)
	}
	if !acquired {
		return nil, errors.New("another worker or migration owns the schema lease")
	}
	return release, nil
}

func (api *managementAPI) authenticateAdmin(w http.ResponseWriter, r *http.Request) (*models.User, bool) {
	login, password, ok := r.BasicAuth()
	if !ok || strings.TrimSpace(login) == "" || password == "" {
		w.Header().Set("WWW-Authenticate", `Basic realm="docflow migrations"`)
		writeAPIError(w, http.StatusUnauthorized, "authentication_required", errors.New("administrator credentials are required"))
		return nil, false
	}
	authKey := remoteHost(r.RemoteAddr) + "\x00" + strings.TrimSpace(login)
	if !api.authenticationAllowed(authKey, time.Now()) {
		writeAPIError(w, http.StatusTooManyRequests, "authentication_rate_limited", errors.New("too many authentication failures; retry later"))
		return nil, false
	}
	user, err := api.users.GetByLogin(strings.TrimSpace(login))
	if err != nil || user == nil || !security.VerifyPassword(user.PasswordHash, password) || !user.IsActive || user.PasswordChangeRequired || !contains(user.SystemPermissions, models.SystemPermissionAdmin) {
		api.recordAuthenticationFailure(authKey, time.Now())
		time.Sleep(200 * time.Millisecond)
		writeAPIError(w, http.StatusUnauthorized, "invalid_credentials", errors.New("invalid administrator credentials"))
		return nil, false
	}
	api.clearAuthenticationFailures(authKey)
	return user, true
}

func remoteHost(remoteAddress string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddress))
	if err == nil {
		return host
	}
	return strings.TrimSpace(remoteAddress)
}

func (api *managementAPI) authenticationAllowed(key string, now time.Time) bool {
	api.authMu.Lock()
	defer api.authMu.Unlock()
	if api.authFailures == nil {
		api.authFailures = make(map[string]authFailure)
	}
	failure, ok := api.authFailures[key]
	if !ok || !now.Before(failure.resetAt) {
		delete(api.authFailures, key)
		return true
	}
	return failure.count < 5
}

func (api *managementAPI) recordAuthenticationFailure(key string, now time.Time) {
	api.authMu.Lock()
	defer api.authMu.Unlock()
	if api.authFailures == nil {
		api.authFailures = make(map[string]authFailure)
	}
	if len(api.authFailures) >= maxAuthenticationFailureKeys {
		for existingKey, existingFailure := range api.authFailures {
			if !now.Before(existingFailure.resetAt) {
				delete(api.authFailures, existingKey)
			}
		}
	}
	if _, exists := api.authFailures[key]; !exists && len(api.authFailures) >= maxAuthenticationFailureKeys {
		return
	}
	failure := api.authFailures[key]
	if !now.Before(failure.resetAt) {
		failure = authFailure{resetAt: now.Add(time.Minute)}
	}
	failure.count++
	api.authFailures[key] = failure
}

func (api *managementAPI) clearAuthenticationFailures(key string) {
	api.authMu.Lock()
	delete(api.authFailures, key)
	api.authMu.Unlock()
}

func (api *managementAPI) auditAction(user *models.User, action, details string) {
	if user == nil || api.audit == nil {
		return
	}
	if _, err := api.audit.Create(models.CreateAdminAuditLogRequest{UserID: user.ID, UserName: user.FullName, Action: action, Details: details}); err != nil {
		slog.Error("failed to write migration audit", "action", action, "error", err)
	}
}

func validateRollback(req rollbackRequest) error {
	if !req.BackupCompleted || strings.TrimSpace(req.BackupReference) == "" {
		return errors.New("fresh backup reference is required")
	}
	if !req.AcknowledgedDataLoss {
		return errors.New("data loss acknowledgement is required")
	}
	if strings.TrimSpace(req.Confirmation) != rollbackConfirmationPhrase {
		return fmt.Errorf("confirmation must equal %q", rollbackConfirmationPhrase)
	}
	return nil
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode request: %w", err)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeAPIError(w http.ResponseWriter, status int, code string, err error) {
	if status == http.StatusConflict || status >= http.StatusInternalServerError {
		slog.Warn("management API operation failed", "code", code, "status", status, "error", err)
	}
	writeJSON(w, status, map[string]string{"code": code, "error": err.Error()})
}

func requestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
