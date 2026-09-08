package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/operations"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func authHTTPService(t *testing.T, handler http.HandlerFunc, lifecycle *operations.Lifecycle) *AuthService {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := serverclient.New(server.URL)
	require.NoError(t, err)
	return NewAuthService(client, client, lifecycle, nil)
}
func writeLogin(w http.ResponseWriter, id string) {
	_ = json.NewEncoder(w).Encode(map[string]any{"accessToken": id, "user": dto.User{ID: id, IsActive: true}})
}
func awaitAuthCall(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("auth call did not finish")
		return nil
	}
}
func awaitRequest(t *testing.T, entered <-chan struct{}) {
	t.Helper()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("request did not start")
	}
}

func TestAuthAdapterSlowLogoutPreservesNewSession(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	oldID, newID := uuid.NewString(), uuid.NewString()
	service := authHTTPService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/login" {
			var req struct{ Login string }
			_ = json.NewDecoder(r.Body).Decode(&req)
			writeLogin(w, req.Login)
			return
		}
		close(entered)
		<-release
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":"session_invalid","error":"session invalid"}`))
	}, nil)
	t.Cleanup(func() { once.Do(func() { close(release) }) })
	_, err := service.Login(oldID, "password")
	require.NoError(t, err)
	before := service.GetSessionState()
	done := make(chan error, 1)
	go func() { done <- service.Logout() }()
	awaitRequest(t, entered)
	require.False(t, service.IsAuthenticated())
	_, err = service.Login(newID, "password")
	require.NoError(t, err)
	once.Do(func() { close(release) })
	require.ErrorIs(t, awaitAuthCall(t, done), models.ErrUnauthorized)
	after := service.GetSessionState()
	require.True(t, after.Authenticated)
	require.Equal(t, newID, after.UserID)
	require.Greater(t, after.Revision, before.Revision)
}

func TestAuthAdapterLogoutDiscardsPendingLogin(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	service := authHTTPService(t, func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		writeLogin(w, uuid.NewString())
	}, nil)
	t.Cleanup(func() { once.Do(func() { close(release) }) })
	done := make(chan error, 1)
	go func() { _, err := service.Login("user", "password"); done <- err }()
	awaitRequest(t, entered)
	require.NoError(t, service.Logout())
	revision := service.GetSessionState().Revision
	once.Do(func() { close(release) })
	require.ErrorIs(t, awaitAuthCall(t, done), models.ErrUnauthorized)
	require.False(t, service.IsAuthenticated())
	require.Equal(t, revision, service.GetSessionState().Revision)
}

func TestAuthAdapterUnauthorizedInvalidatesPrincipal(t *testing.T) {
	service := authHTTPService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/login" {
			writeLogin(w, uuid.NewString())
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":"session_invalid","error":"session invalid"}`))
	}, nil)
	_, err := service.Login("user", "password")
	require.NoError(t, err)
	principal := NewPrincipal(service, nil)
	require.NotEmpty(t, principal.GetCurrentUserID())
	require.ErrorIs(t, principal.RequireAuthenticated(), models.ErrUnauthorized)
	require.False(t, service.IsAuthenticated())
	require.Empty(t, principal.GetCurrentUserID())
}

func TestAuthAdapterShutdownCancelsRequest(t *testing.T) {
	entered := make(chan struct{})
	lifecycle := operations.NewLifecycle(time.Minute)
	service := authHTTPService(t, func(w http.ResponseWriter, r *http.Request) { close(entered); <-r.Context().Done() }, lifecycle)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = lifecycle.Shutdown(ctx)
	})
	done := make(chan error, 1)
	go func() { _, err := service.NeedsInitialSetup(); done <- err }()
	awaitRequest(t, entered)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, lifecycle.Shutdown(ctx))
	require.ErrorIs(t, awaitAuthCall(t, done), context.Canceled)
}
