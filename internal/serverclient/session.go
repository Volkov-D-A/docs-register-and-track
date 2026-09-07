package serverclient

import (
	"context"
	"io"
	"net/http"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// SessionState is safe to send to the renderer. It never contains credentials.
// Revision increases on login and invalidation, including logout without a token.
type SessionState struct {
	Revision      uint64 `json:"revision"`
	Authenticated bool   `json:"authenticated"`
	UserID        string `json:"userId"`
	Reason        string `json:"reason"`
}

type sessionRequestKey struct{}
type requestSession struct {
	revision uint64
	token    string
}

func (c *Client) SessionState() SessionState {
	c.tokenMu.RLock()
	defer c.tokenMu.RUnlock()
	return c.sessionStateLocked()
}

func (c *Client) sessionStateLocked() SessionState {
	return SessionState{Revision: c.sessionRevision, Authenticated: c.token != "", UserID: c.sessionUserID, Reason: c.sessionReason}
}

// SetSessionEndedHandler is configured by the desktop composition root, not Wails.
func (c *Client) SetSessionEndedHandler(handler func(SessionState)) {
	c.tokenMu.Lock()
	c.onSessionEnded = handler
	c.tokenMu.Unlock()
}

func sessionForRequest(req *http.Request) requestSession {
	session, _ := req.Context().Value(sessionRequestKey{}).(requestSession)
	return session
}

func (c *Client) endSession(session requestSession, reason string) {
	c.tokenMu.Lock()
	if c.token == "" || c.token != session.token || c.sessionRevision != session.revision {
		c.tokenMu.Unlock()
		return
	}
	c.token = ""
	c.sessionUserID = ""
	c.sessionRevision++
	c.sessionReason = reason
	state, handler := c.sessionStateLocked(), c.onSessionEnded
	cancel := c.sessionCancel
	c.sessionContext, c.sessionCancel = nil, nil
	c.tokenMu.Unlock()
	if cancel != nil {
		cancel()
	}
	// Never call application code while holding the token lock. Consumers compare
	// revisions because callbacks from concurrent transitions can arrive out of order.
	if handler != nil {
		handler(state)
	}
}

// doAuthenticated is the only execution path for bearer requests, including
// attachment streaming. Login and Basic-auth migration requests use plain HTTP.
func (c *Client) doAuthenticated(req *http.Request) (*http.Response, error) {
	session := sessionForRequest(req)
	c.tokenMu.RLock()
	current := c.token != "" && c.token == session.token && c.sessionRevision == session.revision
	sessionContext := c.sessionContext
	c.tokenMu.RUnlock()
	if !current {
		return nil, models.ErrUnauthorized
	}
	ctx, cancel := context.WithCancel(req.Context())
	stop := func() bool { return false }
	if sessionContext != nil {
		stop = context.AfterFunc(sessionContext, cancel)
	}
	cleanup := func() { stop(); cancel() }
	resp, err := c.http.Do(req.WithContext(ctx))
	if err != nil {
		cleanup()
		if !c.matchesSession(session) {
			return nil, models.ErrUnauthorized
		}
		return nil, err
	}
	resp.Body = &sessionBody{ReadCloser: resp.Body, client: c, session: session, cleanup: cleanup}
	if resp.StatusCode == http.StatusUnauthorized {
		c.endSession(session, "session_invalid")
		resp.Body.Close()
		return nil, models.ErrUnauthorized
	}
	c.tokenMu.RLock()
	current = c.token == session.token && c.sessionRevision == session.revision
	c.tokenMu.RUnlock()
	if !current {
		resp.Body.Close()
		return nil, models.ErrUnauthorized
	}
	return resp, nil
}

func withRequestSession(ctx context.Context, revision uint64, token string) context.Context {
	return context.WithValue(ctx, sessionRequestKey{}, requestSession{revision: revision, token: token})
}

func (c *Client) matchesSession(session requestSession) bool {
	c.tokenMu.RLock()
	defer c.tokenMu.RUnlock()
	return c.token != "" && c.token == session.token && c.sessionRevision == session.revision
}

// Check both sides of a blocking read: data from an ended session must not be
// decoded or committed to a downloaded file, even if headers arrived earlier.
type sessionBody struct {
	io.ReadCloser
	client  *Client
	session requestSession
	cleanup func()
}

func (b *sessionBody) Read(p []byte) (int, error) {
	if !b.client.matchesSession(b.session) {
		return 0, models.ErrUnauthorized
	}
	n, err := b.ReadCloser.Read(p)
	if !b.client.matchesSession(b.session) {
		return 0, models.ErrUnauthorized
	}
	return n, err
}
func (b *sessionBody) Close() error {
	defer b.cleanup()
	return b.ReadCloser.Close()
}
