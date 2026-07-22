package services

import (
	"errors"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/outbox"
	"github.com/Volkov-D-A/docs-register-and-track/internal/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
	"github.com/Volkov-D-A/docs-register-and-track/internal/testutil/integrationdb"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRepeatedTransitionsProduceDistinctAuditEventsIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}
	outboxRepo := repository.NewOutboxRepository(db)
	userRepo := repository.NewUserRepository(db)
	userRepo.SetOutbox(outboxRepo)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsRepo.SetOutbox(outboxRepo)
	substitutionRepo := repository.NewUserSubstitutionRepository(db)
	substitutionRepo.SetOutbox(outboxRepo)
	auditRepo := repository.NewAdminAuditLogRepository(db)

	adminPassword := "AdminPassw0rd!"
	adminHash, err := security.HashPassword(adminPassword)
	require.NoError(t, err)
	require.NoError(t, userRepo.CreateInitialAdmin(adminHash))

	auth := NewAuthService(db, userRepo)
	auth.SetAccessStore(repository.NewDocumentAccessRepository(db))
	_, err = auth.Login("admin", adminPassword)
	require.NoError(t, err)

	settingsService := NewSettingsService(db, settingsRepo, auth, NewAdminAuditLogService(auditRepo, auth))
	userService := NewUserService(userRepo, auth)
	substitutionService := NewUserSubstitutionService(substitutionRepo, userRepo, auth)

	departmentID := uuid.New()
	_, err = db.Exec(`INSERT INTO departments (id, name) VALUES ($1, $2)`, departmentID, "Audit Integration")
	require.NoError(t, err)

	userHash, err := security.HashPassword("UserPassw0rd!")
	require.NoError(t, err)
	lockedUserID := insertAuditIntegrationUser(t, db, departmentID, userHash, "lock-target", "Lock Target")
	principalID := insertAuditIntegrationUser(t, db, departmentID, userHash, "principal", "Principal")
	substituteID := insertAuditIntegrationUser(t, db, departmentID, userHash, "substitute", "Substitute")

	lockUserFiveTimes(t, auth, "lock-target")
	_, err = userService.UpdateUser(models.UpdateUserRequest{
		ID:                    lockedUserID.String(),
		Login:                 "lock-target",
		FullName:              "Lock Target",
		IsActive:              true,
		DepartmentID:          departmentID.String(),
		IsDocumentParticipant: true,
	})
	require.NoError(t, err)
	lockUserFiveTimes(t, auth, "lock-target")

	setting, err := settingsRepo.Get("max_file_size_mb")
	require.NoError(t, err)
	require.NotNil(t, setting)
	alternativeValue := "16"
	if setting.Value == alternativeValue {
		alternativeValue = "17"
	}
	require.NoError(t, settingsService.Update(setting.Key, alternativeValue))
	require.NoError(t, settingsService.Update(setting.Key, setting.Value))

	setSubstitution := models.UpdateUserSubstitutionRequest{
		PrincipalUserID:  principalID.String(),
		SubstituteUserID: substituteID.String(),
		IsActive:         true,
	}
	clearSubstitution := models.UpdateUserSubstitutionRequest{PrincipalUserID: principalID.String()}
	_, err = substitutionService.UpdateUserSubstitution(setSubstitution)
	require.NoError(t, err)
	_, err = substitutionService.UpdateUserSubstitution(clearSubstitution)
	require.NoError(t, err)
	_, err = substitutionService.UpdateUserSubstitution(setSubstitution)
	require.NoError(t, err)
	_, err = substitutionService.UpdateUserSubstitution(clearSubstitution)
	require.NoError(t, err)

	worker := outbox.NewWorker(outboxRepo, nil, nil, auditRepo, nil, nil)
	require.NoError(t, worker.ProcessOnce())

	assertAuditActionCount(t, db, "USER_LOCKED", 2)
	assertAuditActionCount(t, db, "SETTINGS_UPDATE", 2)
	assertAuditActionCount(t, db, "USER_SUBSTITUTION_UPDATE", 4)
	assertDistinctOutboxKeys(t, db, "user:"+lockedUserID.String()+":locked:%", 2)
	assertDistinctOutboxKeys(t, db, "setting:"+setting.Key+":update:%", 2)
	assertDistinctOutboxKeys(t, db, "user-substitution:"+principalID.String()+":clear:%", 2)
}

func TestOutboxRejectsMismatchedDeduplicationCollisionIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}
	outboxRepo := repository.NewOutboxRepository(db)

	userID := uuid.New()
	_, err := db.Exec(`INSERT INTO users (id, login, password_hash, full_name)
		VALUES ($1, $2, 'integration-hash', 'Collision User')`, userID, "collision-"+uuid.NewString())
	require.NoError(t, err)

	first, err := NewAdminAuditOutboxEvent("collision:"+uuid.NewString(), models.CreateAdminAuditLogRequest{
		UserID: userID, UserName: "Collision User", Action: "COLLISION_TEST", Details: "original",
	})
	require.NoError(t, err)
	require.NoError(t, outboxRepo.Enqueue(first))
	require.NoError(t, outboxRepo.Enqueue(first))

	different, err := NewAdminAuditOutboxEvent(first.DeduplicationKey, models.CreateAdminAuditLogRequest{
		UserID: userID, UserName: "Collision User", Action: "COLLISION_TEST", Details: "different",
	})
	require.NoError(t, err)
	err = outboxRepo.Enqueue(different)
	require.Error(t, err)
	require.True(t, errors.Is(err, repository.ErrOutboxDeduplicationConflict))

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM event_outbox WHERE deduplication_key = $1`, first.DeduplicationKey).Scan(&count))
	require.Equal(t, 1, count)
}

func insertAuditIntegrationUser(t *testing.T, db *database.DB, departmentID uuid.UUID, hash, login, name string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(`INSERT INTO users (
		id, login, password_hash, full_name, department_id,
		is_document_participant, is_active, password_change_required
	) VALUES ($1, $2, $3, $4, $5, TRUE, TRUE, FALSE)`, id, login, hash, name, departmentID)
	require.NoError(t, err)
	return id
}

func lockUserFiveTimes(t *testing.T, auth *AuthService, login string) {
	t.Helper()
	for range 5 {
		_, err := auth.Login(login, "WrongPassw0rd!")
		require.Error(t, err)
	}
}

func assertAuditActionCount(t *testing.T, db *database.DB, action string, expected int) {
	t.Helper()
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM admin_audit_log WHERE action = $1`, action).Scan(&count))
	require.Equal(t, expected, count)
}

func assertDistinctOutboxKeys(t *testing.T, db *database.DB, pattern string, expected int) {
	t.Helper()
	var count, distinctCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*), COUNT(DISTINCT deduplication_key)
		FROM event_outbox WHERE deduplication_key LIKE $1`, pattern).Scan(&count, &distinctCount))
	require.Equal(t, expected, count)
	require.Equal(t, expected, distinctCount)
}
