package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

const desktopLogFlushInterval = 250 * time.Millisecond

// ServerAsyncWriter batches desktop slog records and submits them to the
// authenticated docflow-server client. Delivery is best effort and never
// blocks application logging on a full queue.
type ServerAsyncWriter struct {
	client serverclient.TelemetryClient
	events chan models.TechnicalLogEvent
	done   chan struct{}
	once   sync.Once
	wg     sync.WaitGroup
	mu     sync.RWMutex
	closed bool
}

func NewServerAsyncWriter(client serverclient.TelemetryClient) *ServerAsyncWriter {
	w := &ServerAsyncWriter{client: client, events: make(chan models.TechnicalLogEvent, 1000), done: make(chan struct{})}
	w.wg.Add(1)
	go w.start()
	return w
}

func (w *ServerAsyncWriter) Write(payload []byte) (int, error) {
	event, ok := parseTechnicalLogEvent(payload)
	if !ok || w.client == nil {
		return len(payload), nil
	}
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.closed {
		return len(payload), nil
	}
	select {
	case w.events <- event:
	default:
	}
	return len(payload), nil
}

func parseTechnicalLogEvent(payload []byte) (models.TechnicalLogEvent, bool) {
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return models.TechnicalLogEvent{}, false
	}
	message, _ := raw["@m"].(string)
	if strings.TrimSpace(message) == "" || len(message) > maxTechnicalLogMessage {
		return models.TechnicalLogEvent{}, false
	}
	event := models.TechnicalLogEvent{Message: message, Attributes: make(map[string]string)}
	if level, ok := raw["@l"].(string); ok {
		event.Level = level
	}
	if timestamp, ok := raw["@t"].(string); ok {
		event.Timestamp, _ = time.Parse(time.RFC3339Nano, timestamp)
	}
	for key, value := range raw {
		if strings.HasPrefix(key, "@") || sensitiveTechnicalLogKey(key) || key == "app_user_id" {
			continue
		}
		addTechnicalLogAttribute(event.Attributes, key, value, 0)
	}
	return event, true
}

func addTechnicalLogAttribute(attributes map[string]string, key string, value any, depth int) {
	if len(attributes) >= maxTechnicalLogAttributes || len(key) > maxTechnicalLogKey || sensitiveTechnicalLogKey(key) || key == "app_user_id" {
		return
	}
	if group, ok := value.(map[string]any); ok {
		if depth >= 4 {
			return
		}
		for childKey, childValue := range group {
			if sensitiveTechnicalLogKey(childKey) {
				continue
			}
			addTechnicalLogAttribute(attributes, key+"."+childKey, childValue, depth+1)
		}
		return
	}
	if value == nil {
		attributes[key] = "<nil>"
		return
	}
	switch value.(type) {
	case []any:
		// Structured collections may hide credentials under unnamed values.
		return
	}
	text := fmt.Sprint(value)
	if len(text) <= maxTechnicalLogValue {
		attributes[key] = text
	}
}

const (
	maxTechnicalLogMessage    = 4096
	maxTechnicalLogAttributes = 32
	maxTechnicalLogKey        = 64
	maxTechnicalLogValue      = 2048
)

func sensitiveTechnicalLogKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	if key == "app_user_id" || key == "app_user_name" {
		return true
	}
	for _, fragment := range []string{"password", "passwd", "secret", "token", "authorization", "cookie", "credential"} {
		if strings.Contains(key, fragment) {
			return true
		}
	}
	return false
}

func (w *ServerAsyncWriter) start() {
	defer w.wg.Done()
	ticker := time.NewTicker(desktopLogFlushInterval)
	defer ticker.Stop()
	pending := make([]models.TechnicalLogEvent, 0, 50)
	flush := func() {
		if len(pending) == 0 {
			return
		}
		w.send(pending)
		pending = pending[:0]
	}
	for {
		select {
		case event := <-w.events:
			pending = append(pending, event)
			if len(pending) == 50 {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-w.done:
			for len(w.events) > 0 {
				pending = append(pending, <-w.events)
				if len(pending) == 50 {
					flush()
				}
			}
			flush()
			return
		}
	}
}

func (w *ServerAsyncWriter) send(events []models.TechnicalLogEvent) {
	batch := append([]models.TechnicalLogEvent(nil), events...)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = w.client.SendTechnicalLogs(ctx, batch)
}

func (w *ServerAsyncWriter) Close() error {
	w.once.Do(func() {
		w.mu.Lock()
		w.closed = true
		close(w.done)
		w.mu.Unlock()
		w.wg.Wait()
	})
	return nil
}
