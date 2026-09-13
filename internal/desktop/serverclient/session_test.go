package serverclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sessionResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}
func sessionTestClient(t *testing.T) *Client {
	t.Helper()
	c, err := New("https://server.test")
	require.NoError(t, err)
	c.token = "old-token"
	c.sessionUserID = "old-user"
	c.sessionRevision = 1
	return c
}
func loginResponseBody() string {
	return `{"accessToken":"new-token","user":{"id":"new-user","login":"new-user"}}`
}

func TestBearerUnauthorizedEndsSessionAcrossAllRequestPaths(t *testing.T) {
	operations := map[string]func(*Client) error{
		"me":        func(c *Client) error { _, err := c.Me(context.Background()); return err },
		"documents": func(c *Client) error { _, err := c.ListDocumentAttachments(context.Background(), "doc"); return err },
		"upload": func(c *Client) error {
			_, err := c.UploadAttachment(context.Background(), "doc", "", "a.txt", 1, strings.NewReader("a"))
			return err
		},
		"download":        func(c *Client) error { _, _, err := c.GetAttachmentContent(context.Background(), "file"); return err },
		"update":          func(c *Client) error { return c.UpdateProfile(context.Background(), models.UpdateProfileRequest{}) },
		"change-password": func(c *Client) error { return c.ChangePassword(context.Background(), "old", "new") },
	}
	for name, operation := range operations {
		for _, body := range []string{`{"code":"session_invalid"}`, "proxy unauthorized response"} {
			t.Run(name+"/"+body, func(t *testing.T) {
				c := sessionTestClient(t)
				var calls, notifications int
				c.SetSessionEndedHandler(func(state SessionState) {
					notifications++
					assert.Equal(t, uint64(2), state.Revision)
					assert.False(t, state.Authenticated)
					assert.Empty(t, state.UserID)
					assert.Equal(t, "session_invalid", state.Reason)
					assert.Equal(t, state, c.SessionState(), "callback must run outside token mutex")
				})
				c.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
					calls++
					return sessionResponse(http.StatusUnauthorized, body), nil
				})
				require.ErrorIs(t, operation(c), models.ErrUnauthorized)
				require.ErrorIs(t, operation(c), models.ErrUnauthorized)
				assert.Equal(t, 1, calls, "no retries and no requests with the invalid token")
				assert.Equal(t, 1, notifications)
			})
		}
	}
}

func TestConcurrentUnauthorizedNotifiesOnce(t *testing.T) {
	c := sessionTestClient(t)
	const count = 8
	var entered, done sync.WaitGroup
	entered.Add(count)
	done.Add(count)
	release := make(chan struct{})
	var notifications atomic.Int32
	c.SetSessionEndedHandler(func(SessionState) { notifications.Add(1) })
	c.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		entered.Done()
		<-release
		return sessionResponse(http.StatusUnauthorized, ""), nil
	})
	errs := make(chan error, count)
	for i := 0; i < count; i++ {
		go func() { defer done.Done(); _, err := c.Me(context.Background()); errs <- err }()
	}
	entered.Wait()
	close(release)
	done.Wait()
	for i := 0; i < count; i++ {
		require.ErrorIs(t, <-errs, models.ErrUnauthorized)
	}
	assert.EqualValues(t, 1, notifications.Load())
	assert.False(t, c.SessionState().Authenticated)
}

func TestOldResponseCannotInvalidateNewLogin(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusOK} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			c := sessionTestClient(t)
			entered, release := make(chan struct{}), make(chan struct{})
			var notifications atomic.Int32
			c.SetSessionEndedHandler(func(SessionState) { notifications.Add(1) })
			c.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Path == "/api/v1/auth/login" {
					return sessionResponse(http.StatusOK, loginResponseBody()), nil
				}
				close(entered)
				<-release
				return sessionResponse(status, `{"id":"old-user"}`), nil
			})
			result := make(chan error, 1)
			go func() { _, err := c.Me(context.Background()); result <- err }()
			<-entered
			_, err := c.Login(context.Background(), "new-user", "password")
			require.NoError(t, err)
			close(release)
			require.ErrorIs(t, <-result, models.ErrUnauthorized)
			assert.Equal(t, "new-user", c.SessionState().UserID)
			assert.True(t, c.SessionState().Authenticated)
			assert.Zero(t, notifications.Load())
		})
	}
}

func TestSlowLogoutCannotEraseNewLogin(t *testing.T) {
	c := sessionTestClient(t)
	entered, release := make(chan struct{}), make(chan struct{})
	c.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/api/v1/auth/login" {
			return sessionResponse(http.StatusOK, loginResponseBody()), nil
		}
		assert.Equal(t, "Bearer old-token", req.Header.Get("Authorization"))
		close(entered)
		<-release
		return sessionResponse(http.StatusUnauthorized, ""), nil
	})
	done := make(chan error, 1)
	go func() { done <- c.Logout(context.Background()) }()
	<-entered
	assert.False(t, c.SessionState().Authenticated)
	_, err := c.Login(context.Background(), "new-user", "password")
	require.NoError(t, err)
	close(release)
	<-done
	assert.True(t, c.SessionState().Authenticated)
	assert.Equal(t, "new-user", c.SessionState().UserID)
}

func TestLogoutDiscardsPendingLogin(t *testing.T) {
	c := sessionTestClient(t)
	c.token = ""
	entered, release := make(chan struct{}), make(chan struct{})
	c.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		close(entered)
		<-release
		return sessionResponse(http.StatusOK, loginResponseBody()), nil
	})
	done := make(chan error, 1)
	go func() { _, err := c.Login(context.Background(), "user", "pass"); done <- err }()
	<-entered
	require.NoError(t, c.Logout(context.Background()))
	close(release)
	require.ErrorIs(t, <-done, models.ErrUnauthorized)
	assert.False(t, c.SessionState().Authenticated)
}

func TestNonSessionFailuresKeepSession(t *testing.T) {
	for _, status := range []int{http.StatusForbidden, http.StatusServiceUnavailable, http.StatusInternalServerError, 0} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			c := sessionTestClient(t)
			c.SetSessionEndedHandler(func(SessionState) { t.Error("unexpected session invalidation") })
			c.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
				if status == 0 {
					return nil, errors.New("network unavailable")
				}
				return sessionResponse(status, `{}`), nil
			})
			_, err := c.Me(context.Background())
			require.Error(t, err)
			assert.True(t, c.SessionState().Authenticated)
		})
	}
	c := sessionTestClient(t)
	c.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return sessionResponse(http.StatusUnauthorized, `{"code":"invalid_credentials"}`), nil
	})
	_, err := c.Login(context.Background(), "wrong", "wrong")
	require.Error(t, err)
	_, err = c.Apply(context.Background(), "admin", "wrong")
	require.Error(t, err)
	assert.True(t, c.SessionState().Authenticated)
}

func TestPasswordChangeTransportFailureEndsSession(t *testing.T) {
	c := sessionTestClient(t)
	var notifications int
	c.SetSessionEndedHandler(func(SessionState) { notifications++ })
	c.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("connection lost after commit") })
	require.Error(t, c.ChangePassword(context.Background(), "old", "new"))
	assert.False(t, c.SessionState().Authenticated)
	assert.Equal(t, 1, notifications)
}

func TestSessionEndCancelsInFlightRequest(t *testing.T) {
	c := sessionTestClient(t)
	c.sessionContext, c.sessionCancel = context.WithCancel(context.Background())
	entered := make(chan struct{})
	c.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/api/v1/auth/me" {
			close(entered)
			<-req.Context().Done()
			return nil, req.Context().Err()
		}
		return sessionResponse(http.StatusUnauthorized, ""), nil
	})
	done := make(chan error, 1)
	go func() { _, err := c.Me(context.Background()); done <- err }()
	<-entered
	_, err := c.ListDocumentAttachments(context.Background(), "doc")
	require.ErrorIs(t, err, models.ErrUnauthorized)
	require.ErrorIs(t, <-done, models.ErrUnauthorized)
}

func TestSessionEndRejectsAnAlreadyOpenedDownload(t *testing.T) {
	c := sessionTestClient(t)
	c.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/api/v1/auth/me" {
			return sessionResponse(http.StatusUnauthorized, ""), nil
		}
		resp := sessionResponse(http.StatusOK, "secret")
		resp.Header.Set("Content-Disposition", `attachment; filename="a.txt"`)
		resp.ContentLength = 6
		return resp, nil
	})
	_, body, err := c.GetAttachmentContent(context.Background(), "file")
	require.NoError(t, err)
	defer body.Close()
	_, err = c.Me(context.Background())
	require.ErrorIs(t, err, models.ErrUnauthorized)
	data, err := io.ReadAll(body)
	require.ErrorIs(t, err, models.ErrUnauthorized)
	assert.Empty(t, data)
}
