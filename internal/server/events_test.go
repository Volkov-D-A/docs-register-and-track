package server

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"github.com/Volkov-D-A/docs-register-and-track/internal/backup"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/liveevents"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type eventTestWriter struct {
	*httptest.ResponseRecorder
	onFlush func()
}

func (w *eventTestWriter) Flush() { w.ResponseRecorder.Flush(); w.onFlush() }

func TestSessionEventsReleaseMaintenanceLocksAndFilterAdminTopics(t *testing.T) {
	for _, admin := range []bool{false, true} {
		t.Run(map[bool]string{false: "user", true: "admin"}[admin], func(t *testing.T) {
			user := &models.User{ID: uuid.New(), IsActive: true}
			if admin {
				user.SystemPermissions = []string{models.SystemPermissionAdmin}
			}
			hash := sha256.Sum256([]byte("secret"))
			sessions := &fakeAuthSessions{hash: hash[:], session: &models.ServerSession{UserID: user.ID}}
			api := &managementAPI{events: &liveevents.Bus{}, authUsers: &fakeAuthUsers{user: user}, sessions: sessions}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil).WithContext(ctx)
			req.Header.Set("Authorization", "Bearer secret")
			recorder := httptest.NewRecorder()
			flushes := 0
			w := &eventTestWriter{ResponseRecorder: recorder}
			w.onFlush = func() {
				flushes++
				require.True(t, api.schemaRequests.TryLock())
				api.schemaRequests.Unlock()
				require.True(t, api.replacementRequests.TryLock())
				api.replacementRequests.Unlock()
				if flushes == 1 {
					api.events.Publish("user:" + user.ID.String())
					api.events.Publish("backups")
				} else if (!admin && strings.Contains(recorder.Body.String(), "event: user-events")) || (admin && strings.Contains(recorder.Body.String(), "event: user-events") && strings.Contains(recorder.Body.String(), "event: backups")) {
					cancel()
				}
			}
			// Prevent a regression from hanging the suite.
			timer := time.AfterFunc(time.Second, cancel)
			defer timer.Stop()
			api.Handler().ServeHTTP(w, req)
			require.Contains(t, recorder.Body.String(), "event: resync")
			require.Contains(t, recorder.Body.String(), "event: user-events")
			if admin {
				require.Contains(t, recorder.Body.String(), "event: backups")
			} else {
				require.NotContains(t, recorder.Body.String(), "event: backups")
			}
			require.NotContains(t, recorder.Body.String(), "secret")
		})
	}
}

func TestSessionEventsRejectMissingSessionAndMaintenance(t *testing.T) {
	api := &managementAPI{sessions: &fakeAuthSessions{}}
	for _, maintenance := range []bool{false, true} {
		api.replacementPending.Store(maintenance)
		w := httptest.NewRecorder()
		api.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/events", nil))
		if maintenance {
			require.Equal(t, 503, w.Code)
		} else {
			require.Equal(t, 401, w.Code)
		}
		require.NotContains(t, w.Header().Get("Content-Type"), "text/event-stream")
	}
}

func TestOperationEventsUseOnlyScopedCapabilityDuringReplacement(t *testing.T) {
	id := uuid.NewString()
	hash := sha256.Sum256([]byte("operation-token"))
	directory := t.TempDir()
	raw, err := json.Marshal(map[string]any{
		"id": id, "copyId": uuid.NewString(), "kind": "restore", "state": "completed", "tokenHash": hash[:], "expiresAt": time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(directory, id+".operation.json"), raw, 0600))
	api := &managementAPI{backupService: &backup.Service{Directory: directory}}
	api.replacementPending.Store(true)
	api.schemaRequests.Lock()
	defer api.schemaRequests.Unlock()
	api.replacementRequests.Lock()
	defer api.replacementRequests.Unlock()
	for _, token := range []string{"operation-token", "ordinary-login-token", ""} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/backups/operations/"+id+"/events", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		api.Handler().ServeHTTP(w, req)
		if token == "operation-token" {
			require.Equal(t, 200, w.Code)
			require.Contains(t, w.Body.String(), `"state":"completed"`)
			require.NotContains(t, w.Body.String(), "tokenHash")
		} else {
			require.Equal(t, 403, w.Code)
		}
	}
}

func TestSessionEventsRevalidateSessionBeforeDelivery(t *testing.T) {
	user := &models.User{ID: uuid.New(), IsActive: true}
	hash := sha256.Sum256([]byte("session"))
	sessions := &fakeAuthSessions{hash: hash[:], session: &models.ServerSession{UserID: user.ID}}
	api := &managementAPI{events: &liveevents.Bus{}, sessions: sessions, authUsers: &fakeAuthUsers{user: user}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	req.Header.Set("Authorization", "Bearer session")
	recorder := httptest.NewRecorder()
	w := &eventTestWriter{ResponseRecorder: recorder, onFlush: func() {
		sessions.session = nil
		api.events.Publish("user:" + user.ID.String())
	}}
	api.Handler().ServeHTTP(w, req)
	require.Contains(t, recorder.Body.String(), "event: resync")
	require.NotContains(t, recorder.Body.String(), "event: user-events")
}
