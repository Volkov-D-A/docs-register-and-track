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
	"github.com/Volkov-D-A/docs-register-and-track/internal/logger"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/releaseassets"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
	"github.com/Volkov-D-A/docs-register-and-track/internal/services"
	"github.com/Volkov-D-A/docs-register-and-track/internal/startupdiag"
)

// WailsOptionsParams contains process-level dependencies that main owns.
type WailsOptionsParams struct {
	ConfigPath         string
	Assets             fs.FS
	ReleaseNotesSource []byte
	CloseLogger        func()
	ServerClient       *serverclient.Client
}

func NewDesktopServerClient(cfg *config.Config) (*serverclient.Client, error) {
	serverURL := cfg.Server.URL
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}
	return serverclient.NewWithOptions(serverURL, serverclient.Options{AllowInsecureHTTP: cfg.Server.AllowInsecureHTTP})
}

// NewBindingsWailsOptions returns the public service types needed by the Wails
// bindings generator without constructing runtime infrastructure dependencies.
func NewBindingsWailsOptions() *options.App {
	return &options.App{
		Bind: []interface{}{
			&services.AuthService{},
			&services.UserService{},
			&services.UserSubstitutionService{},
			&services.NomenclatureService{},
			&services.ReferenceService{},
			&services.DocumentAccessAdminService{},
			&services.DocumentKindService{},
			&services.DocumentQueryService{},
			&services.DocumentRegistrationService{},
			&services.AdministrativeOrderService{},
			&services.AssignmentService{},
			&services.DashboardService{},
			&services.StatisticsService{},
			&services.DepartmentService{},
			&services.SettingsService{},
			&services.AttachmentService{},
			&services.LinkService{},
			&services.AcknowledgmentService{},
			&services.SystemService{},
			&services.ReleaseNoteService{},
			&services.ThemeService{},
			&services.JournalService{},
			&services.AdminAuditLogService{},
			&services.UserEventService{},
			&services.OutboxAdminService{},
		},
	}
}

type wailsOptionsDependencies struct {
	newThemeService func() (*services.ThemeService, error)
}

// NewWailsOptions builds the desktop application graph and returns Wails options.
func NewWailsOptions(cfg *config.Config, params WailsOptionsParams) (*options.App, *startupdiag.Failure) {
	return newWailsOptionsWithDependencies(cfg, params, wailsOptionsDependencies{
		newThemeService: services.NewThemeService,
	})
}

func newWailsOptionsWithDependencies(
	cfg *config.Config,
	params WailsOptionsParams,
	dependencies wailsOptionsDependencies,
) (*options.App, *startupdiag.Failure) {
	metrics := observability.NewRegistry(256)

	operationLifecycle := services.NewOperationLifecycle(5 * time.Minute)

	authService := services.NewAuthService(nil, nil)
	authService.SetOperationMetrics(metrics)

	logger.GetAppUserID = func() string {
		return authService.GetCurrentUserID()
	}

	settingsService := services.NewSettingsService(authService)
	serverClient := params.ServerClient
	if serverClient == nil {
		var err error
		serverClient, err = NewDesktopServerClient(cfg)
		if err != nil {
			return nil, &startupdiag.Failure{
				Component:  "docflow-server",
				ConfigPath: params.ConfigPath,
				Summary:    "Некорректный адрес серверного API.",
				NextStep:   "Проверьте server.url в desktop config.json.",
				Err:        err,
			}
		}
	}
	adminAuditLogService := services.NewAdminAuditLogServiceWithClient(serverClient)
	outboxAdminService := services.NewOutboxAdminServiceWithClient(serverClient)
	settingsService.SetMigrationClient(serverClient)
	settingsService.SetServerClient(serverClient)
	authService.SetServerAuth(serverClient)
	userService := services.NewUserService(nil, authService)
	userService.SetServerClient(serverClient)
	userSubstitutionService := services.NewUserSubstitutionService(nil, nil, authService)
	userSubstitutionService.SetServerClient(serverClient)
	nomenclatureService := services.NewNomenclatureService()
	nomenclatureService.SetServerClient(serverClient)
	referenceService := services.NewReferenceService(authService)
	referenceService.SetServerClient(serverClient)
	documentAccessAdminService := services.NewDocumentAccessAdminService(authService, nil, nil)
	documentAccessAdminService.SetServerClient(serverClient)
	documentKindService := services.NewDocumentKindService(nil)
	documentKindService.SetServerClient(serverClient)
	journalService := services.NewJournalServiceWithClient(serverClient)
	documentQueryService := services.NewDocumentQueryService()
	documentQueryService.SetServerClient(serverClient)
	documentQueryService.SetOperationMetrics(metrics)
	documentRegistrationService := services.NewDocumentRegistrationServiceWithClient(serverClient)
	documentRegistrationService.SetOperationLifecycle(operationLifecycle)
	documentRegistrationService.SetOperationMetrics(metrics)
	userEventService := services.NewUserEventServiceWithClient(serverClient)
	administrativeOrderService := services.NewAdministrativeOrderServiceWithClient(serverClient)
	assignmentService := services.NewAssignmentServiceWithClient(serverClient)
	departmentService := services.NewDepartmentService()
	departmentService.SetServerClient(serverClient)

	attachmentService := services.NewAttachmentServiceWithClient(serverClient)
	attachmentService.SetOperationLifecycle(operationLifecycle)
	attachmentService.SetOperationMetrics(metrics)
	backgroundServices := newBackgroundLifecycle(
		newServerMigrationStatusReader(serverClient),
		nil,
		nil,
	)
	services.ConfigureSchemaLifecycle(authService, settingsService, backgroundServices)

	dashboardService := services.NewDashboardServiceWithClient(serverClient)
	statisticsService := services.NewStatisticsServiceWithClient(serverClient)
	linkService := services.NewLinkServiceWithClient(serverClient)
	acknowledgmentService := services.NewAcknowledgmentServiceWithClient(serverClient)
	clientVersion, err := releaseassets.CurrentVersion()
	if err != nil {
		return nil, &startupdiag.Failure{
			Component:  "release version",
			ConfigPath: params.ConfigPath,
			Summary:    "Не удалось определить версию приложения.",
			NextStep:   "Пересоберите приложение через release workflow.",
			Err:        err,
		}
	}
	systemService := services.NewSystemServiceWithClient(serverClient, clientVersion)
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
	themeService, err := dependencies.newThemeService()
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
			systemService.Startup(ctx)
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
