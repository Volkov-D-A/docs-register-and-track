package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/security"
)

// Keep user lookup simple while exercising the real transactional write from HTTP.
type lockAuditAuthUsers struct {
	fakeAuthUsers
	repo   *repository.UserRepository
	effect models.OutboxEvent
}

func (s *lockAuditAuthUsers) IncrementFailedLoginAttemptsWithOutbox(id uuid.UUID, effect models.OutboxEvent) (int, bool, error) {
	s.effect = effect
	return s.repo.IncrementFailedLoginAttemptsWithOutbox(id, effect)
}

func TestAuthLockAuditUsesOutboxTransaction(t *testing.T) {
	hash, err := security.HashPassword("Passw0rd!")
	require.NoError(t, err)
	for _, route := range []struct{ path, body string }{
		{"/api/v1/auth/login", `{"login":"user","password":"wrong"}`},
		{"/api/v1/auth/change-required-password", `{"login":"user","oldPassword":"wrong","newPassword":"NewPassw0rd!"}`},
	} {
		for _, outcome := range []string{"lock", "enqueue failure", "already locked", "missing outbox"} {
			t.Run(route.path+"/"+outcome, func(t *testing.T) {
				sqlDB, sql, err := sqlmock.New()
				require.NoError(t, err)
				defer sqlDB.Close()
				db := &database.DB{DB: sqlDB}
				repo := repository.NewUserRepository(db)
				if outcome != "missing outbox" {
					repo.SetOutbox(repository.NewOutboxRepository(db))
				}
				user := &models.User{ID: uuid.New(), Login: "user", FullName: "Test User", PasswordHash: hash, IsActive: true, FailedLoginAttempts: 4}
				attempts := 5
				if outcome == "already locked" {
					user.IsActive, user.FailedLoginAttempts, attempts = false, 5, 6
				}
				users := &lockAuditAuthUsers{fakeAuthUsers: fakeAuthUsers{user: user}, repo: repo}
				audit := &fakeAdminAudit{}
				api := &managementAPI{authUsers: users, audit: audit}
				if outcome != "missing outbox" {
					sql.ExpectBegin()
					sql.ExpectQuery(`UPDATE users SET failed_login_attempts`).WithArgs(user.ID).
						WillReturnRows(sqlmock.NewRows([]string{"failed_login_attempts", "is_active"}).AddRow(attempts, false))
					sql.ExpectExec(`UPDATE server_sessions SET revoked_at`).WithArgs(user.ID).WillReturnResult(sqlmock.NewResult(0, 1))
					if outcome != "already locked" {
						enqueue := sql.ExpectExec(`INSERT INTO event_outbox`).WithArgs(models.OutboxEventAudit, sqlmock.AnyArg(), sqlmock.AnyArg())
						if outcome == "enqueue failure" {
							enqueue.WillReturnError(assert.AnError)
						} else {
							enqueue.WillReturnResult(sqlmock.NewResult(0, 1))
						}
					}
					if outcome == "enqueue failure" {
						sql.ExpectRollback()
					} else {
						sql.ExpectCommit()
					}
				}

				response := httptest.NewRecorder()
				api.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodPost, route.path, strings.NewReader(route.body)))

				status := http.StatusForbidden
				if outcome == "enqueue failure" || outcome == "missing outbox" {
					status = http.StatusInternalServerError
				}
				require.Equal(t, status, response.Code, response.Body.String())
				require.Empty(t, audit.actions, "auth must not write a separate synchronous lock audit")
				var payload models.CreateAdminAuditLogRequest
				require.NoError(t, json.Unmarshal([]byte(users.effect.Payload), &payload))
				require.Equal(t, "USER_LOCKED", payload.Action)
				require.Equal(t, user.ID, payload.UserID)
				require.Equal(t, user.FullName, payload.UserName)
				require.NoError(t, sql.ExpectationsWereMet())
			})
		}
	}
}
