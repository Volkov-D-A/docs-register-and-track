package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2"
	wailslogger "github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/logger"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/services"
	"github.com/Volkov-D-A/docs-register-and-track/internal/startupdiag"
	"github.com/Volkov-D-A/docs-register-and-track/internal/storage"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed internal/releaseassets/current_release.yaml
var releaseNotesSource []byte

func main() {
	// CLI-утилита: шифрование пароля для config.json
	encryptFlag := flag.String("encrypt-password", "", "Зашифровать пароль для config.json и вывести результат")
	flag.Parse()

	if *encryptFlag != "" {
		encrypted, err := config.EncryptPassword(*encryptFlag)
		if err != nil {
			failStartup(startupdiag.Failure{
				Component: "config encryption",
				Summary:   "Не удалось зашифровать пароль для config.json.",
				NextStep:  "Проверьте значение ENCRYPTION_KEY и повторите команду --encrypt-password.",
				Err:       err,
			})
		}
		// Wails собирается как GUI-приложение (без консоли), поэтому
		// записываем результат в файл рядом с исполняемым файлом.
		outputFile := "encrypted_password.txt"
		content := fmt.Sprintf("Зашифрованный пароль для поля \"password\" в config.json:\n\n%s\n", encrypted)
		if err := os.WriteFile(outputFile, []byte(content), 0600); err != nil {
			failStartup(startupdiag.Failure{
				Component: "config encryption",
				Summary:   "Не удалось записать encrypted_password.txt.",
				NextStep:  "Проверьте права записи в текущий каталог или запустите команду из доступного рабочего каталога.",
				Err:       err,
			})
		}
		log.Printf("Результат записан в файл: %s", outputFile)
		return
	}

	// Загрузка конфигурации
	configPath := config.GetDefaultConfigPath()
	cfg, err := config.Load(configPath)
	if err != nil {
		failStartup(startupdiag.Failure{
			Component:  "configuration",
			ConfigPath: configPath,
			Summary:    "Не удалось загрузить config.json.",
			NextStep:   "Проверьте DOCFLOW_CONFIG_PATH, наличие файла, права чтения, JSON-синтаксис и ENCRYPTION_KEY для ENC:-значений.",
			Err:        err,
		})
	}

	// Инициализация логгера
	_, closeLogger := logger.Init(cfg.Seq)
	defer closeLogger()

	// Подключение к БД
	db, err := database.Connect(cfg.Database)
	if err != nil {
		failStartup(startupdiag.Failure{
			Component:  "PostgreSQL",
			ConfigPath: configPath,
			Summary:    "Не удалось подключиться к базе данных.",
			NextStep:   "Проверьте host/port/dbname/user/sslmode в config.json, расшифровку пароля и доступность PostgreSQL из рабочего места.",
			Err:        err,
		})
	}

	// Создание репозиториев
	userRepo := repository.NewUserRepository(db)
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

	// Создание сервисов
	authService := services.NewAuthService(db, userRepo)
	authService.SetAccessStore(documentAccessRepo)

	// Подключаем к техническим логам только идентификатор пользователя.
	logger.GetAppUserID = func() string {
		uid, _ := authService.GetCurrentAuditInfo()
		if uid == uuid.Nil {
			return ""
		}
		return uid.String()
	}

	adminAuditLogService := services.NewAdminAuditLogService(adminAuditLogRepo, authService)
	authService.SetAdminAuditLogService(adminAuditLogService)
	settingsService := services.NewSettingsService(db, settingsRepo, authService, adminAuditLogService)
	userService := services.NewUserService(userRepo, authService, adminAuditLogService)
	nomenclatureService := services.NewNomenclatureService(nomenclatureRepo, authService, adminAuditLogService)
	referenceService := services.NewReferenceService(referenceRepo, authService, adminAuditLogService)
	documentAccessService := services.NewDocumentAccessService(authService, departmentRepo, assignmentRepo, acknowledgmentRepo, documentAccessRepo, documentRepo, incomingDocRepo, outgoingDocRepo)
	documentAccessAdminService := services.NewDocumentAccessAdminService(authService, documentAccessRepo, userRepo)
	documentKindService := services.NewDocumentKindService(documentAccessService)
	journalService := services.NewJournalService(journalRepo, authService, documentAccessService)
	documentKindQueryRegistry := services.NewDocumentKindQueryRegistry(
		services.NewIncomingLetterQueryHandler(incomingDocRepo),
		services.NewOutgoingLetterQueryHandler(outgoingDocRepo),
		services.NewCitizenAppealQueryHandler(citizenAppealRepo),
		services.NewAdministrativeOrderQueryHandler(administrativeOrderRepo),
	)
	documentQueryService := services.NewDocumentQueryService(documentKindQueryRegistry, documentAccessService)
	documentKindCommandRegistry := services.NewDocumentKindCommandRegistry(
		services.NewIncomingLetterCommandHandler(incomingDocRepo, nomenclatureRepo, referenceRepo, authService, journalService, documentAccessService),
		services.NewOutgoingLetterCommandHandler(outgoingDocRepo, referenceRepo, nomenclatureRepo, authService, journalService, documentAccessService),
		services.NewCitizenAppealCommandHandler(citizenAppealRepo, nomenclatureRepo, referenceRepo, authService, journalService, documentAccessService),
		services.NewAdministrativeOrderCommandHandler(administrativeOrderRepo, nomenclatureRepo, authService, journalService, documentAccessService),
	)
	documentRegistrationService := services.NewDocumentRegistrationService(documentKindCommandRegistry)
	administrativeOrderService := services.NewAdministrativeOrderService(administrativeOrderRepo, authService, documentAccessService, journalService)
	assignmentService := services.NewAssignmentService(assignmentRepo, userRepo, authService, journalService, documentAccessService)
	departmentService := services.NewDepartmentService(departmentRepo, authService, adminAuditLogService)

	minioService, err := storage.NewMinioService(cfg.Minio)
	if err != nil {
		failStartup(startupdiag.Failure{
			Component:  "MinIO",
			ConfigPath: configPath,
			Summary:    "Не удалось подключиться к объектному хранилищу.",
			NextStep:   "Проверьте endpoint/useSSL/bucket/accessKeyId в config.json, расшифровку secretAccessKey и доступность MinIO из рабочего места.",
			Err:        err,
		})
	}

	dashboardService := services.NewDashboardService(dashboardRepo, authService, documentAccessService)
	statisticsService := services.NewStatisticsService(statisticsRepo, authService, minioService)
	attachmentService := services.NewAttachmentService(attachmentRepo, settingsService, authService, journalService, adminAuditLogService, minioService, documentAccessService)
	linkService := services.NewLinkService(linkRepo, incomingDocRepo, outgoingDocRepo, citizenAppealRepo, administrativeOrderRepo, documentAccessService, authService, journalService)
	acknowledgmentService := services.NewAcknowledgmentService(acknowledgmentRepo, userRepo, authService, journalService, documentAccessService)
	systemService := services.NewSystemService(db)
	releaseNoteService, err := services.NewReleaseNoteService(releaseNotesSource)
	if err != nil {
		failStartup(startupdiag.Failure{
			Component:  "release notes",
			ConfigPath: configPath,
			Summary:    "Не удалось загрузить встроенные release notes.",
			NextStep:   "Проверьте, что сборка выполнена через release workflow и generated release assets актуальны.",
			Err:        err,
		})
	}
	themeService, err := services.NewThemeService()
	if err != nil {
		failStartup(startupdiag.Failure{
			Component:  "local theme state",
			ConfigPath: configPath,
			Summary:    "Не удалось инициализировать локальное состояние темы.",
			NextStep:   "Проверьте доступность пользовательского config directory и права записи для профиля пользователя.",
			Err:        err,
		})
	}

	// Запуск приложения Wails
	err = wails.Run(&options.App{
		Title:  "Система регистрации документов",
		Width:  1280,
		Height: 1000,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Logger:   logger.NewWailsAdapter(),
		LogLevel: wailslogger.ERROR, // Пишем системные ошибки Wails
		ErrorFormatter: func(err error) any {
			// Перехватываем все ошибки, которые методы сервисов (Bindings) возвращают во фронтенд
			if appErr, ok := models.AsAppError(err); ok {
				if appErr.StatusCode() >= 500 {
					slog.Error("Backend binding failed", "type", "backend_binding", "code", appErr.SafeKind(), "status", appErr.StatusCode())
				}
				return map[string]any{
					"code":    appErr.SafeKind(),
					"message": appErr.SafeMessage(),
					"status":  appErr.StatusCode(),
				}
			}
			slog.Error("Backend binding failed", "type", "backend_binding", "error_type", fmt.Sprintf("%T", err))
			return map[string]any{
				"code":    "INTERNAL_ERROR",
				"message": "произошла внутренняя ошибка",
				"status":  500,
			}
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnShutdown: func(ctx context.Context) {
			db.Close()
			closeLogger()
		},

		Bind: []interface{}{
			authService,
			userService,
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
		},
	})

	if err != nil {
		failStartup(startupdiag.Failure{
			Component:  "Wails",
			ConfigPath: configPath,
			Summary:    "Не удалось запустить desktop UI.",
			NextStep:   "Проверьте WebView2/runtime окружение, технический лог и повторите smoke на целевой ОС.",
			Err:        err,
		})
	}
}

func failStartup(failure startupdiag.Failure) {
	startupdiag.Log(slog.Default(), failure)
	startupdiag.Write(os.Stderr, failure)
	os.Exit(1)
}
