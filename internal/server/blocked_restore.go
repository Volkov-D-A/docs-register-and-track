package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/backup"
	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// A failed replacement never bootstraps migrations or opens business routes.
// Only previously issued status capabilities work on the usual server port.
func newBlockedApp(cfg *config.Config) (*App, error) {
	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}
	s := &backup.Service{Directory: cfg.Backup.Directory}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/admin/backups/operations/{id}", func(w http.ResponseWriter, r *http.Request) {
		token, _ := bearerToken(r.Header.Get("Authorization"))
		op, err := s.OperationStatus(r.PathValue("id"), token)
		if err != nil {
			writeAPIError(w, 403, "forbidden", models.ErrForbidden)
			return
		}
		op.State = "recovery_required"
		op.CanCancel = false
		op.Error = "Восстановление прервано. Требуется проверка локального журнала и полный сброс целевой БД и файлового хранилища перед новым восстановлением."
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, 200, op)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeAPIError(w, 503, "maintenance", errors.New("recovery required"))
	})
	address := cfg.Server.ListenAddress
	if address == "" {
		address = ":8080"
	}
	return &App{cfg: cfg, backups: s, http: &http.Server{Addr: address, Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 << 10}}, nil
}

func (a *App) runBlocked(ctx context.Context) error {
	done := make(chan error, 1)
	go func() { done <- a.http.ListenAndServe() }()
	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		stop, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return a.http.Shutdown(stop)
	}
}
