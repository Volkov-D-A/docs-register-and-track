package logging

import (
	"bytes"
	"context"
	"io"
	"log"
	"log/slog"
	"os"
	"sync"

	"github.com/Volkov-D-A/docs-register-and-track/internal/shared/logger"
)

// userIDProvider supplies desktop session context to local log records.
var userIDProvider struct {
	sync.RWMutex
	get func() string
}

// SetUserIDProvider updates the desktop user context without racing log delivery.
func SetUserIDProvider(get func() string) {
	userIDProvider.Lock()
	defer userIDProvider.Unlock()
	userIDProvider.get = get
}
func currentUserIDProvider() func() string {
	userIDProvider.RLock()
	defer userIDProvider.RUnlock()
	return userIDProvider.get
}

// technicalContextHandler — обертка над slog.Handler, которая динамически добавляет
// минимальный технический контекст во все логи.
type technicalContextHandler struct {
	slog.Handler
}

func (h *technicalContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if get := currentUserIDProvider(); get != nil {
		userID := get()
		if userID != "" {
			r.AddAttrs(slog.String("app_user_id", userID))
		}
	}
	return h.Handler.Handle(ctx, r)
}

func (h *technicalContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &technicalContextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *technicalContextHandler) WithGroup(name string) slog.Handler {
	return &technicalContextHandler{Handler: h.Handler.WithGroup(name)}
}

func Init(client TelemetryClient) (*slog.Logger, func()) {
	writer := NewServerAsyncWriter(client)
	handler := &technicalContextHandler{Handler: slog.NewJSONHandler(io.MultiWriter(os.Stdout, writer), logger.CLEFHandlerOptions())}
	result, closeLogger := logger.Install(handler, func() { _ = writer.Close() })
	log.SetOutput(&stdLogFilter{})
	log.SetFlags(0)
	return result, closeLogger
}

// stdLogFilter реализует io.Writer для перехвата вывода стандартного log.
// Отбрасывает известные info-сообщения go-webview2, остальное передаёт в slog.Error.
type stdLogFilter struct{}

// Список подстрок, которые нужно молча игнорировать.
var ignoredLogMessages = [][]byte{
	[]byte("[WebView2] Environment created successfully"),
}

func (f *stdLogFilter) Write(p []byte) (int, error) {
	for _, ignore := range ignoredLogMessages {
		if bytes.Contains(p, ignore) {
			return len(p), nil // тихо отбрасываем
		}
	}
	// Всё остальное — пишем как ошибку в slog.
	msg := string(bytes.TrimRight(p, "\n\r"))
	slog.Error(msg, "source", "std_log")
	return len(p), nil
}
