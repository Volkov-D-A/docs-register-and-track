package serverclient

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// LiveEvent contains no tokens and can safely cross the Wails bridge.
type LiveEvent struct {
	Topic     string                  `json:"topic"`
	Revision  uint64                  `json:"revision"`
	Operation *models.BackupOperation `json:"operation,omitempty"`
}

// ConfigureEvents is called by the composition root before login. Cancelling ctx
// stops both ordinary session streams and independent restore observation.
func (c *Client) ConfigureEvents(ctx context.Context, handler func(LiveEvent)) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	c.eventsContext, c.onEvent = ctx, handler
}

func (c *Client) startSessionEvents() {
	c.tokenMu.RLock()
	parent, sessionCtx, handler := c.eventsContext, c.sessionContext, c.onEvent
	session := requestSession{revision: c.sessionRevision, token: c.token}
	c.tokenMu.RUnlock()
	if parent == nil || sessionCtx == nil || handler == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithCancel(parent)
		defer cancel()
		stop := context.AfterFunc(sessionCtx, cancel)
		defer stop()
		c.reconnectEvents(ctx, "/api/v1/events", session.token, func(topic string, _ []byte) bool {
			if !c.matchesSession(session) {
				return false
			}
			if topic != "heartbeat" {
				handler(LiveEvent{Topic: topic, Revision: session.revision})
			}
			return true
		}, func(status int) bool {
			if status == http.StatusUnauthorized {
				c.endSession(session, "session_invalid")
				return false
			}
			return c.matchesSession(session)
		})
	}()
}

func (c *Client) observeOperation(started models.BackupOperationStarted) {
	c.tokenMu.RLock()
	parent, handler := c.eventsContext, c.onEvent
	c.tokenMu.RUnlock()
	if parent == nil || handler == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithDeadline(parent, started.ExpiresAt)
		defer cancel()
		c.reconnectEvents(ctx, "/api/v1/admin/backups/operations/"+url.PathEscape(started.Job.ID)+"/events", started.StatusToken, func(topic string, data []byte) bool {
			if topic != "operation" {
				return true
			}
			var op models.BackupOperation
			if json.Unmarshal(data, &op) != nil || op.ID != started.Job.ID {
				return true
			}
			handler(LiveEvent{Topic: "operation", Operation: &op})
			switch op.State {
			case "completed", "failed", "cancelled", "interrupted", "rolled_back", "rollback_failed", "recovery_required":
				return false
			}
			return true
		}, func(status int) bool { return status != http.StatusForbidden && status != http.StatusUnauthorized })
	}()
}

func (c *Client) reconnectEvents(ctx context.Context, path, token string, receive func(string, []byte) bool, retry func(int) bool) {
	client := *c.http
	// Stream lifetime is controlled by contexts, not the ordinary request timeout.
	client.Timeout = 0
	delay := time.Second
	for ctx.Err() == nil {
		connectionCtx, stopConnection := context.WithCancel(ctx)
		idle := time.AfterFunc(45*time.Second, stopConnection)
		req, err := http.NewRequestWithContext(connectionCtx, http.MethodGet, c.baseURL+path, nil)
		if err != nil {
			idle.Stop()
			stopConnection()
			return
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "text/event-stream")
		resp, err := client.Do(req)
		if err == nil {
			if resp.StatusCode == http.StatusOK && strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
				delay = time.Second
				err = readEvents(resp.Body, func(topic string, data []byte) bool {
					idle.Reset(45 * time.Second)
					return receive(topic, data)
				})
				resp.Body.Close()
				idle.Stop()
				stopConnection()
				if errors.Is(err, errStopEvents) {
					return
				}
			} else {
				resp.Body.Close()
				idle.Stop()
				stopConnection()
				if !retry(resp.StatusCode) {
					return
				}
			}
		}
		idle.Stop()
		stopConnection()
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		if delay < 30*time.Second {
			delay *= 2
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
		}
	}
}

var errStopEvents = errors.New("event observation complete")

func readEvents(body io.Reader, receive func(string, []byte) bool) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	name := ""
	var data []byte
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if name != "" && !receive(name, data) {
				return errStopEvents
			}
			name, data = "", nil
		} else if strings.HasPrefix(line, "event:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			if len(data) > 0 {
				data = append(data, '\n')
			}
			data = append(data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " ")...)
			if len(data) > 1<<20 {
				return errors.New("event exceeds size limit")
			}
		}
	}
	return scanner.Err()
}
