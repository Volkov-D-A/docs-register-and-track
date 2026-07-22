package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"time"

	wailslogger "github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/logger"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/outbox"
	"github.com/Volkov-D-A/docs-register-and-track/internal/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/services"
	"github.com/Volkov-D-A/docs-register-and-track/internal/startupdiag"
	"github.com/Volkov-D-A/docs-register-and-track/internal/storage"
)

// WailsOptionsParams contains process-level dependencies that main owns.
type WailsOptionsParams struct {
	ConfigPath         string
	Assets             fs.FS
	ReleaseNotesSource []byte
	CloseLogger        func()
}

// NewWailsOptions builds the desktop application graph and returns Wails options.
func NewWailsOptions(cfg *config.Config, params WailsOptionsParams) (*options.App, *startupdiag.Failure) {
	db, err := database.Connect(cfg.Database)
	if err != nil {
		return nil, &startupdiag.Failure{
			Component:  "PostgreSQL",
			ConfigPath: params.ConfigPath,
			Summary:    "Не удалось подключиться к базе данных.",
			NextStep:   "Проверьте host/port/dbname/user/sslmode в config.json, расшифровку пароля и доступность PostgreSQL из рабочего места.",
			Err:        err,
		}
	}
	created := false
	defer func() {
		if !created {
			db.Close()
		}
	}()
	metrics := observability.NewRegistry(256)
	db.SetMetrics(metrics)

	userRepo := repository.NewUserRepository(db)
	userSubstitutionRepo := repository.NewUserSubstitutionRepository(db)
	nomenclatureRepo := repository.NewNomenclatureRepository(db)
	referenceRepo := repository.NewReferenceRepository(db)
	documentAccessRepo := repository.NewDocumentAccessRepository(db)
	documentRepo := repository.NewDocumentRepository(db)
	incomingDocRepo := repository.NewIncomingDocumentRepository(db)
	outgoingDocRepo := repository.NewOutgoingDocumentRepository(db)
	citizenAppealRepo := repository.NewCitizenAppealRepository(db)
	administrativeOrderRepo := repository.NewAdministrativeOrderRepository(db)
	assignmentRepo := repository.NewAssignmentRepository(db)
	departmentRepo := repository.NewDepartmentRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	attachmentRepo := repository.NewAttachmentRepository(db)
	linkRepo := repository.NewLinkRepository(db)
	acknowledgmentRepo := repository.NewAcknowledgmentRepository(db)
	dashboardRepo := repository.NewDashboardRepository(db)
	statisticsRepo := repository.NewStatisticsRepository(db)
	journalRepo := repository.NewJournalRepository(db)
	adminAuditLogRepo := repository.NewAdminAuditLogRepository(db)
	userEventRepo := repository.NewUserEventRepository(db)
	outboxRepo := repository.NewOutboxRepository(db)
	acknowledgmentRepo.SetOutbox(outboxRepo)
	attachmentRepo.SetOutbox(outboxRepo)
	assignmentRepo.SetOutbox(outboxRepo)
	linkRepo.SetOutbox(outboxRepo)
	nomenclatureRepo.SetOutbox(outboxRepo)
	departmentRepo.SetOutbox(outboxRepo)
	userSubstitutionRepo.SetOutbox(outboxRepo)
	referenceRepo.SetOutbox(outboxRepo)
	userRepo.SetOutbox(outboxRepo)
	settingsRepo.SetOutbox(outboxRepo)
	outgoingDocRepo.SetOutbox(outboxRepo)
	incomingDocRepo.SetOutbox(outboxRepo)
	citizenAppealRepo.SetOutbox(outboxRepo)
	administrativeOrderRepo.SetOutbox(outboxRepo)

	operationLifecycle := services.NewOperationLifecycle(5 * time.Minute)

	authService := services.NewAuthService(db, userRepo)
	authService.SetOperationMetrics(metrics)
	authService.SetAccessStore(documentAccessRepo)
	authService.SetSettingsStore(settingsRepo)

	logger.GetAppUserID = func() string {
		return authService.GetCurrentUserID()
	}

	adminAuditLogService := services.NewAdminAuditLogService(adminAuditLogRepo, authService)
	outboxAdminService := services.NewOutboxAdminService(outboxRepo, authService)
	settingsService := services.NewSettingsService(db, settingsRepo, authService, adminAuditLogService)
	userService := services.NewUserService(userRepo, authService)
	userSubstitutionService := services.NewUserSubstitutionService(userSubstitutionRepo, userRepo, authService)
	nomenclatureService := services.NewNomenclatureService(nomenclatureRepo, authService)
	referenceService := services.NewReferenceService(referenceRepo, authService)
	documentAccessService := services.NewDocumentAccessService(authService, departmentRepo, assignmentRepo, acknowledgmentRepo, documentAccessRepo, documentRepo, incomingDocRepo, outgoingDocRepo, userSubstitutionRepo)
	documentAccessAdminService := services.NewDocumentAccessAdminService(authService, documentAccessRepo, userRepo)
	documentKindService := services.NewDocumentKindService(documentAccessService)
	journalService := services.NewJournalService(journalRepo, authService, documentAccessService)
	journalService.SetOperationLifecycle(operationLifecycle)
	documentKindQueryRegistry := services.NewDocumentKindQueryRegistry(
		services.NewIncomingLetterQueryHandler(incomingDocRepo),
		services.NewOutgoingLetterQueryHandler(outgoingDocRepo),
		services.NewCitizenAppealQueryHandler(citizenAppealRepo),
		services.NewAdministrativeOrderQueryHandler(administrativeOrderRepo),
	)
	documentQueryService := services.NewDocumentQueryService(documentKindQueryRegistry, documentAccessService)
	documentQueryService.SetOperationMetrics(metrics)
	documentKindCommandRegistry := services.NewDocumentKindCommandRegistry(
		services.NewIncomingLetterCommandHandler(incomingDocRepo, nomenclatureRepo, referenceRepo, authService, journalService, documentAccessService),
		services.NewOutgoingLetterCommandHandler(outgoingDocRepo, referenceRepo, nomenclatureRepo, authService, journalService, documentAccessService),
		services.NewCitizenAppealCommandHandler(citizenAppealRepo, nomenclatureRepo, referenceRepo, authService, journalService, documentAccessService),
		services.NewAdministrativeOrderCommandHandler(administrativeOrderRepo, nomenclatureRepo, authService, journalService, documentAccessService),
	)
	documentRegistrationService := services.NewDocumentRegistrationService(documentKindCommandRegistry)
	documentRegistrationService.SetOperationLifecycle(operationLifecycle)
	documentRegistrationService.SetOperationMetrics(metrics)
	userEventService := services.NewUserEventService(userEventRepo, authService)
	administrativeOrderService := services.NewAdministrativeOrderService(administrativeOrderRepo, authService, documentAccessService)
	assignmentService := services.NewAssignmentService(assignmentRepo, userRepo, authService, documentAccessService, userEventService)
	assignmentService.SetSubstitutionStore(userSubstitutionRepo)
	departmentService := services.NewDepartmentService(departmentRepo, authService)

	minioService, err := storage.NewMinioService(cfg.Minio)
	if err != nil {
		return nil, &startupdiag.Failure{
			Component:  "MinIO",
			ConfigPath: params.ConfigPath,
			Summary:    "Не удалось подключиться к объектному хранилищу.",
			NextStep:   "Проверьте endpoint/useSSL/bucket/accessKeyId в config.json, расшифровку secretAccessKey и доступность MinIO из рабочего места.",
			Err:        err,
		}
	}
	attachmentService := services.NewAttachmentService(attachmentRepo, settingsService, authService, minioService, documentAccessService)
	attachmentService.SetOperationLifecycle(operationLifecycle)
	attachmentService.SetOperationMetrics(metrics)
	outboxWorker := outbox.NewWorker(outboxRepo, userEventRepo, journalRepo, adminAuditLogRepo, attachmentRepo, minioService)
	outboxWorker.SetMetrics(metrics)
	backgroundServices := newBackgroundLifecycle(
		db,
		outboxWorker,
		attachmentService.ProcessPendingDeletions,
	)
	services.ConfigureSchemaLifecycle(authService, settingsService, backgroundServices)

	dashboardService := services.NewDashboardService(dashboardRepo, authService, documentAccessService)
	dashboardService.SetOperationMetrics(metrics)
	statisticsService := services.NewStatisticsService(statisticsRepo, authService, minioService)
	statisticsService.SetOperationLifecycle(operationLifecycle)
	statisticsService.SetOperationMetrics(metrics)
	linkService := services.NewLinkService(linkRepo, incomingDocRepo, outgoingDocRepo, citizenAppealRepo, administrativeOrderRepo, documentAccessService, authService)
	linkService.SetOperationLifecycle(operationLifecycle)
	linkService.SetOperationMetrics(metrics)
	acknowledgmentService := services.NewAcknowledgmentService(acknowledgmentRepo, userRepo, authService, documentAccessService, userEventService)
	acknowledgmentService.SetSubstitutionStore(userSubstitutionRepo)
	systemService := services.NewSystemService(db)
	releaseNoteService, err := services.NewReleaseNoteService(params.ReleaseNotesSource)
	if err != nil {
		return nil, &startupdiag.Failure{
			Component:  "release notes",
			ConfigPath: params.ConfigPath,
			Summary:    "Не удалось загрузить встроенные release notes.",
			NextStep:   "Проверьте, что сборка выполнена через release workflow и generated release assets актуальны.",
			Err:        err,
		}
	}
	themeService, err := services.NewThemeService()
	if err != nil {
		return nil, &startupdiag.Failure{
			Component:  "local theme state",
			ConfigPath: params.ConfigPath,
			Summary:    "Не удалось инициализировать локальное состояние темы.",
			NextStep:   "Проверьте доступность пользовательского config directory и права записи для профиля пользователя.",
			Err:        err,
		}
	}

	wailsOptions := &options.App{
		Title:  "Система регистрации документов",
		Width:  1280,
		Height: 1000,
		AssetServer: &assetserver.Options{
			Assets: params.Assets,
		},
		Logger:         logger.NewWailsAdapter(),
		LogLevel:       wailslogger.ERROR,
		ErrorFormatter: formatBackendError,
		OnStartup: func(ctx context.Context) {
			go observability.LogPeriodically(ctx, metrics, slog.Default(), time.Minute)
			attachmentService.Startup(ctx)
			backgroundServices.SetApplicationContext(ctx)
			backgroundServices.ReconcileSchema()
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnShutdown: func(ctx context.Context) {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := backgroundServices.Stop(shutdownCtx); err != nil {
				slog.Warn("shutdown continued before background services stopped", "error", err)
			}
			if err := operationLifecycle.Shutdown(shutdownCtx); err != nil {
				slog.Warn("shutdown continued before all backend operations finished", "error", err)
			}
			metrics.LogSnapshot(slog.Default())
			db.Close()
			if params.CloseLogger != nil {
				params.CloseLogger()
			}
		},
		Bind: []interface{}{
			authService,
			userService,
			userSubstitutionService,
			nomenclatureService,
			referenceService,
			documentAccessAdminService,
			documentKindService,
			documentQueryService,
			documentRegistrationService,
			administrativeOrderService,
			assignmentService,
			dashboardService,
			statisticsService,
			departmentService,
			settingsService,
			attachmentService,
			linkService,
			acknowledgmentService,
			systemService,
			releaseNoteService,
			themeService,
			journalService,
			adminAuditLogService,
			userEventService,
			outboxAdminService,
		},
	}
	created = true
	return wailsOptions, nil
}

func formatBackendError(err error) any {
	// Wails v2.13 wraps a formatted error value in JavaScript's Error constructor.
	// Return a JSON string so the frontend can recover the structured error code
	// from Error.message (rather than receiving "[object Object]").
	format := func(code, message string, status int) string {
		payload, marshalErr := json.Marshal(map[string]any{
			"code":    code,
			"message": message,
			"status":  status,
		})
		if marshalErr != nil {
			return `{"code":"INTERNAL_ERROR","message":"произошла внутренняя ошибка","status":500}`
		}
		return string(payload)
	}

	if appErr, ok := models.AsAppError(err); ok {
		if appErr.StatusCode() >= 500 {
			attrs := []any{"type", "backend_binding", "code", appErr.SafeKind(), "status", appErr.StatusCode(), "error", appErr.Error()}
			if appErr.Internal != nil {
				attrs = append(attrs, "internal", appErr.Internal.Error())
			}
			slog.Error("Backend binding failed", attrs...)
		}
		return format(appErr.SafeKind(), appErr.SafeMessage(), appErr.StatusCode())
	}
	slog.Error("Backend binding failed", "type", "backend_binding", "error_type", fmt.Sprintf("%T", err), "error", err.Error())
	return format("INTERNAL_ERROR", "произошла внутренняя ошибка", 500)
}
