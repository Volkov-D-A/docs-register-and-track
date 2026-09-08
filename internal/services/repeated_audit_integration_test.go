package services

import (
	"errors"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/testutil/integrationdb"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

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
