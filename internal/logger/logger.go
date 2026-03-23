package logger

import (
	"bytes"
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
)

// GetAppUser — глобальная функция для получения текущего пользователя приложения.
// Должна быть инициализирована из authService.
var GetAppUser func() string

// appUserHandler — обертка над slog.Handler, которая динамически добавляет поле app_user во все логи.
type appUserHandler struct {
	slog.Handler
}

func (h *appUserHandler) Handle(ctx context.Context, r slog.Record) error {
	if GetAppUser != nil {
		user := GetAppUser()
		if user != "" {
			r.AddAttrs(slog.String("app_user", user))
		}
	}
	return h.Handler.Handle(ctx, r)
}

func (h *appUserHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &appUserHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *appUserHandler) WithGroup(name string) slog.Handler {
	return &appUserHandler{Handler: h.Handler.WithGroup(name)}
}

// Init инициализирует стандартный логгер slog.
// Возвращает логгер и функцию для корректного закрытия (flush) ресурсов при выходе.
func Init(cfg config.SeqConfig) (*slog.Logger, func()) {
	var handler slog.Handler
	var closer func()

	// Настройки форматирования ключей для CLEF (Compact Log Event Format)
	opts := &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey:
				a.Key = "@t"
			case slog.LevelKey:
				a.Key = "@l"
			case slog.MessageKey:
				a.Key = "@m"
			}
			return a
		},
	}

	if cfg.Enabled && cfg.URL != "" {
		w := NewSeqAsyncWriter(cfg.URL)
		handler = &appUserHandler{Handler: slog.NewJSONHandler(w, opts)}
		closer = func() {
			_ = w.Close()
		}
	} else {
		// Обычный вывод в консоль, если Seq выключен (для fallback)
		handler = &appUserHandler{Handler: slog.NewJSONHandler(os.Stdout, opts)}
		closer = func() {}
	}

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
