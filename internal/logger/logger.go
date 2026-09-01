package logger

import (
	"bytes"
	"context"
	"io"
	"log"
	"log/slog"
	"os"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

// GetAppUserID — глобальная функция для получения ID текущего пользователя приложения.
// Должна быть инициализирована из authService.
var GetAppUserID func() string

// technicalContextHandler — обертка над slog.Handler, которая динамически добавляет
// минимальный технический контекст во все логи.
type technicalContextHandler struct {
	slog.Handler
}

func (h *technicalContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if GetAppUserID != nil {
		userID := GetAppUserID()
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

// Init инициализирует стандартный логгер slog.
// Возвращает логгер и функцию для корректного закрытия (flush) ресурсов при выходе.
func Init(cfg config.SeqConfig) (*slog.Logger, func()) {
	var handler slog.Handler
	var closer func()

	if cfg.Enabled && cfg.URL != "" {
		w := NewSeqAsyncWriter(cfg.URL)
		handler = &technicalContextHandler{Handler: slog.NewJSONHandler(w, clefHandlerOptions())}
		closer = func() {
			_ = w.Close()
		}
	} else {
		// Обычный вывод в консоль, если Seq выключен (для fallback)
		handler = &technicalContextHandler{Handler: slog.NewJSONHandler(os.Stdout, clefHandlerOptions())}
		closer = func() {}
	}

	return install(handler, closer)
}

func clefHandlerOptions() *slog.HandlerOptions {
	return &slog.HandlerOptions{ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
		switch a.Key {
		case slog.TimeKey:
			a.Key = "@t"
		case slog.LevelKey:
			a.Key = "@l"
		case slog.MessageKey:
			a.Key = "@m"
		}
		return a
	}}
}

func install(handler slog.Handler, closer func()) (*slog.Logger, func()) {
	logger := slog.New(handler)

	// Добавляем глобальные атрибуты ко всем логам по умолчанию
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		logger = logger.With("hostname", hostname)
	}
	slog.SetDefault(logger)

	// Перехватываем стандартный пакет log (используется go-webview2).
	// Фильтруем известные info-сообщения; остальное пишем в slog.Error.
	log.SetOutput(&stdLogFilter{})
	log.SetFlags(0) // убираем timestamp, чтобы не мешал сравнению

	return logger, closer
}

// InitDesktop keeps a local JSON fallback and forwards authenticated log
// batches through docflow-server. Desktop never receives a Seq endpoint.
func InitDesktop(client serverclient.TelemetryClient) (*slog.Logger, func()) {
	writer := NewServerAsyncWriter(client)
	handler := &technicalContextHandler{Handler: slog.NewJSONHandler(io.MultiWriter(os.Stdout, writer), clefHandlerOptions())}
	return install(handler, func() { _ = writer.Close() })
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
