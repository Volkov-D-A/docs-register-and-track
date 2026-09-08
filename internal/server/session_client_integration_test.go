package server

import (
	"context"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/background"
	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/services"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
	"github.com/Volkov-D-A/docs-register-and-track/internal/testutil/integrationdb"
)

func TestDesktopSessionInvalidationIntegration(t *testing.T) {
	for _, scenario := range []string{"expiry", "revoke", "deactivate", "password-reset"} {
		t.Run(scenario, func(t *testing.T) {
			db := &database.DB{DB: integrationdb.Open(t)}
			hash, err := security.HashPassword("Passw0rd!")
			require.NoError(t, err)
			adminID := uuid.New()
			_, err = db.Exec(`INSERT INTO users (id, login, password_hash, full_name, is_active, password_change_required)
    VALUES ($1, 'session-admin', $2, 'Session Admin', TRUE, FALSE)`, adminID, hash)
			require.NoError(t, err)
			_, err = db.Exec(`INSERT INTO user_system_permissions (user_id, permission, is_allowed) VALUES ($1, 'admin', TRUE)`, adminID)
			require.NoError(t, err)
			userID := uuid.New()
			_, err = db.Exec(`INSERT INTO users (id, login, password_hash, full_name, is_active, password_change_required)
    VALUES ($1, 'session-client', $2, 'Session Client', TRUE, FALSE)`, userID, hash)
			require.NoError(t, err)
			lifecycle := background.NewLifecycle(func() (*dto.MigrationStatus, error) { return db.GetMigrationStatus(database.DefaultMigrationsPath) }, nil, nil)
			lifecycle.ReconcileSchema()
			api := newManagementAPI(&App{db: db, cfg: validConfig(), lifecycle: lifecycle})
			server := httptest.NewServer(api.Handler())
			defer server.Close()
			client, err := serverclient.New(server.URL)
			require.NoError(t, err)
			adminClient, err := serverclient.New(server.URL)
			require.NoError(t, err)
			_, err = adminClient.Login(context.Background(), "session-admin", "Passw0rd!")
			require.NoError(t, err)
			auth := services.NewAuthService(client, client, nil, nil)
			_, err = auth.Login("session-client", "Passw0rd!")
			require.NoError(t, err)
			require.True(t, auth.IsAuthenticated())
			require.Equal(t, userID.String(), auth.GetSessionState().UserID)
			var notifications atomic.Int32
			client.SetSessionEndedHandler(func(state serverclient.SessionState) {
				notifications.Add(1)
				assert.False(t, state.Authenticated)
			})
			switch scenario {
			case "expiry":
				_, err = db.Exec(`UPDATE server_sessions SET expires_at = CURRENT_TIMESTAMP - INTERVAL '1 second' WHERE user_id=$1`, userID)
			case "revoke":
				_, err = db.Exec(`UPDATE server_sessions SET revoked_at = CURRENT_TIMESTAMP WHERE user_id=$1`, userID)
			case "deactivate":
				_, err = adminClient.UpdateUser(context.Background(), models.UpdateUserRequest{ID: userID.String(), Login: "session-client", FullName: "Session Client", IsActive: false})
			case "password-reset":
				_, err = adminClient.ResetUserPassword(context.Background(), userID.String())
			}
			require.NoError(t, err)
			_, err = client.Me(context.Background())
			require.ErrorIs(t, err, models.ErrUnauthorized)
			assert.False(t, auth.IsAuthenticated())
			assert.Empty(t, auth.GetSessionState().UserID)
			_, err = client.Me(context.Background())
			require.ErrorIs(t, err, models.ErrUnauthorized)
			assert.EqualValues(t, 1, notifications.Load())
		})
	}
}
