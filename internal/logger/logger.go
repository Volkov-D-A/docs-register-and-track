package logger

import (
	"log"
	"log/slog"
	"os"
)

func CLEFHandlerOptions() *slog.HandlerOptions {
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

func Install(handler slog.Handler, closer func()) (*slog.Logger, func()) {
	logger := slog.New(handler)

	// Добавляем глобальные атрибуты ко всем логам по умолчанию
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		logger = logger.With("hostname", hostname)
	}
	slog.SetDefault(logger)
	log.SetOutput(slog.NewLogLogger(logger.With("source", "std_log").Handler(), slog.LevelError).Writer())
	log.SetFlags(0)

	return logger, closer
}
