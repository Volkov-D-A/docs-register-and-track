package logging

import (
	"log/slog"
	"os"

	"github.com/Volkov-D-A/docs-register-and-track/internal/logger"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/config"
)

func Init(cfg config.SeqConfig) (*slog.Logger, func()) {
	var handler slog.Handler
	var closer func()

	if cfg.Enabled && cfg.URL != "" {
		w := NewSeqAsyncWriter(cfg.URL)
		handler = slog.NewJSONHandler(w, logger.CLEFHandlerOptions())
		closer = func() {
			_ = w.Close()
		}
	} else {
		// Обычный вывод в консоль, если Seq выключен (для fallback)
		handler = slog.NewJSONHandler(os.Stdout, logger.CLEFHandlerOptions())
		closer = func() {}
	}

	return logger.Install(handler, closer)
}
