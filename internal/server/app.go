package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/background"
	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/outbox"
	"github.com/Volkov-D-A/docs-register-and-track/internal/releaseassets"
	"github.com/Volkov-D-A/docs-register-and-track/internal/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/services"
	"github.com/Volkov-D-A/docs-register-and-track/internal/storage"
)

const shutdownTimeout = 30 * time.Second

type App struct {
	db        *database.DB
	cfg       *config.Config
	metrics   *observability.Registry
	lifecycle *background.Lifecycle
	http      *http.Server
	storage   serverStorage
	version   string
	closeOnce sync.Once
}

type dependencies struct {
	connectDatabase func(config.DatabaseConfig) (*database.DB, error)
	newStorage      func(config.MinioConfig) (serverStorage, error)
}

type serverStorage interface {
	outbox.FileDeleter
	services.StorageInfoProvider
	services.FileStorage
}

func New(cfg *config.Config) (*App, error) {
	return newWithDependencies(cfg, dependencies{
		connectDatabase: database.Connect,
		newStorage: func(cfg config.MinioConfig) (serverStorage, error) {
			return storage.NewMinioService(cfg)
		},
	})
}

func newWithDependencies(cfg *config.Config, deps dependencies) (*App, error) {
	if err := ValidateConfig(cfg); err != nil {
		return nil, fmt.Errorf("validate server configuration: %w", err)
	}
	version, err := releaseassets.CurrentVersion()
	if err != nil {
		return nil, fmt.Errorf("read server version: %w", err)
	}

	db, err := deps.connectDatabase(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("connect PostgreSQL: %w", err)
	}
	created := false
	defer func() {
		if !created {
			_ = db.Close()
		}
	}()

	metrics := observability.NewRegistry(256)
	db.SetMetrics(metrics)
	objectStorage, err := deps.newStorage(cfg.Minio)
	if err != nil {
		return nil, fmt.Errorf("connect MinIO: %w", err)
	}

	outboxRepo := repository.NewOutboxRepository(db)
	workerOptions, err := ResolveOutboxOptions(cfg.Outbox)
	if err != nil {
		return nil, fmt.Errorf("configure outbox worker: %w", err)
	}
	worker, err := outbox.NewWorkerWithOptions(
		outboxRepo,
		repository.NewUserEventRepository(db),
		repository.NewJournalRepository(db),
		repository.NewAdminAuditLogRepository(db),
		repository.NewAttachmentRepository(db),
		objectStorage,
		workerOptions,
	)
	if err != nil {
		return nil, fmt.Errorf("create outbox worker: %w", err)
	}
	worker.SetMetrics(metrics)
	listenAddress := strings.TrimSpace(cfg.Server.ListenAddress)
	if listenAddress == "" {
		listenAddress = ":8080"
	}

	app := &App{
		db:      db,
		cfg:     cfg,
		metrics: metrics,
		storage: objectStorage,
		version: version,
		lifecycle: background.NewLifecycle(
			db,
			&leasedWorker{db: db, worker: worker},
			(&sessionCleaner{store: repository.NewServerSessionRepository(db)}).Run,
		),
	}
	app.http = &http.Server{
		Addr:              listenAddress,
		Handler:           newManagementAPI(app).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		// Attachment uploads are streamed and may legitimately outlive ordinary
		// JSON requests; request-local contexts still cancel abandoned transfers.
		ReadTimeout: 3 * time.Minute,
		// Schema changes can legitimately take longer than ordinary requests.
		// The desktop client still supplies a bounded per-operation context.
		WriteTimeout:   3 * time.Minute,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 32 << 10,
	}
	created = true
	return app, nil
}

func (a *App) Run(ctx context.Context) error {
	if a == nil || a.lifecycle == nil {
		return fmt.Errorf("server application is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	initialized, err := a.db.IsApplicationSchemaInitialized(ctx)
	if err != nil {
		return fmt.Errorf("check database initialization: %w", err)
	}
	if !initialized {
		slog.Warn("empty database detected; applying embedded bootstrap migrations")
		if err := a.db.RunMigrations(database.DefaultMigrationsPath); err != nil {
			return fmt.Errorf("apply bootstrap migrations: %w", err)
		}
		slog.Info("bootstrap migrations applied")
	}
	a.lifecycle.SetApplicationContext(ctx)
	a.lifecycle.ReconcileSchema()
	httpErr := make(chan error, 1)
	go func() {
		slog.Info("docflow management API started", "listen_address", a.http.Addr)
		err := a.http.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			httpErr <- err
		}
	}()
	slog.Info("docflow server started", "component", "management-api-and-outbox-worker")
	var runErr error
	select {
	case <-ctx.Done():
	case err := <-httpErr:
		runErr = fmt.Errorf("management API stopped: %w", err)
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := a.http.Shutdown(stopCtx); err != nil {
		return fmt.Errorf("stop management API: %w", err)
	}
	if err := a.lifecycle.Stop(stopCtx); err != nil {
		return fmt.Errorf("stop background services: %w", err)
	}
	slog.Info("docflow server stopped")
	return runErr
}

func (a *App) Close() error {
	if a == nil || a.db == nil {
		return nil
	}
	var closeErr error
	a.closeOnce.Do(func() {
		closeErr = a.db.Close()
	})
	return closeErr
}

func HealthCheck(ctx context.Context, cfg *config.Config) error {
	if err := ValidateConfig(cfg); err != nil {
		return fmt.Errorf("validate server configuration: %w", err)
	}
	db, err := database.Connect(cfg.Database)
	if err != nil {
		return fmt.Errorf("connect PostgreSQL: %w", err)
	}
	defer db.Close()

	status, err := db.GetMigrationStatus(database.DefaultMigrationsPath)
	if err != nil {
		return fmt.Errorf("read database schema status: %w", err)
	}
	if status == nil || !status.UpToDate || !status.Compatible {
		return fmt.Errorf("database schema is not ready")
	}
	if err := storage.CheckMinio(ctx, cfg.Minio); err != nil {
		return fmt.Errorf("check MinIO: %w", err)
	}
	return nil
}
