package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v2"
	wailslogger "github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/logger"
	"github.com/Volkov-D-A/docs-register-and-track/internal/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/services"
	"github.com/Volkov-D-A/docs-register-and-track/internal/storage"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// CLI-утилита: шифрование пароля для config.json
	encryptFlag := flag.String("encrypt-password", "", "Зашифровать пароль для config.json и вывести результат")
	flag.Parse()

	if *encryptFlag != "" {
		encrypted, err := config.EncryptPassword(*encryptFlag)
		if err != nil {
			log.Fatalf("Ошибка шифрования: %v", err)
		}
		// Wails собирается как GUI-приложение (без консоли), поэтому
		// записываем результат в файл рядом с исполняемым файлом.
		outputFile := "encrypted_password.txt"
		content := fmt.Sprintf("Зашифрованный пароль для поля \"password\" в config.json:\n\n%s\n", encrypted)
		if err := os.WriteFile(outputFile, []byte(content), 0600); err != nil {
			log.Fatalf("Ошибка записи файла: %v", err)
		}
		log.Printf("Результат записан в файл: %s", outputFile)
		return
	}

	// Загрузка конфигурации
	cfg, err := config.Load(config.GetDefaultConfigPath())
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Инициализация логгера
	_, closeLogger := logger.Init(cfg.Seq)
	defer closeLogger()

	// Подключение к БД
	db, err := database.Connect(cfg.Database)
	if err != nil {
		slog.Error("Critical: Failed to establish database connection pool", "error", err)
		os.Exit(1)
	}

	// Создание репозиториев
	userRepo := repository.NewUserRepository(db)
	nomenclatureRepo := repository.NewNomenclatureRepository(db)
	referenceRepo := repository.NewReferenceRepository(db)
	incomingDocRepo := repository.NewIncomingDocumentRepository(db)
	outgoingDocRepo := repository.NewOutgoingDocumentRepository(db)
	assignmentRepo := repository.NewAssignmentRepository(db)
	departmentRepo := repository.NewDepartmentRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	attachmentRepo := repository.NewAttachmentRepository(db)
	linkRepo := repository.NewLinkRepository(db)
	acknowledgmentRepo := repository.NewAcknowledgmentRepository(db)
	dashboardRepo := repository.NewDashboardRepository(db)
	journalRepo := repository.NewJournalRepository(db)
	adminAuditLogRepo := repository.NewAdminAuditLogRepository(db)

	// Создание сервисов
	authService := services.NewAuthService(db, userRepo)
	
	// Подключаем информацию о пользователе к логгеру
	logger.GetAppUser = func() string {
		_, name := authService.GetCurrentAuditInfo()
		if name == "system" {
			return ""
		}
		return name
	}
	
	adminAuditLogService := services.NewAdminAuditLogService(adminAuditLogRepo, authService)
	settingsService := services.NewSettingsService(db, settingsRepo, authService, adminAuditLogService)
	userService := services.NewUserService(userRepo, authService, adminAuditLogService)
	nomenclatureService := services.NewNomenclatureService(nomenclatureRepo, authService, adminAuditLogService)
	referenceService := services.NewReferenceService(referenceRepo, authService, adminAuditLogService)
	journalService := services.NewJournalService(journalRepo, authService)

	incomingDocService := services.NewIncomingDocumentService(incomingDocRepo, nomenclatureRepo, referenceRepo, departmentRepo, authService, journalService)
	outgoingDocService := services.NewOutgoingDocumentService(outgoingDocRepo, referenceRepo, nomenclatureRepo, departmentRepo, authService, settingsService, journalService)
	assignmentService := services.NewAssignmentService(assignmentRepo, userRepo, authService, journalService)
	departmentService := services.NewDepartmentService(departmentRepo, authService, adminAuditLogService)

	minioService, err := storage.NewMinioService(cfg.Minio)
	if err != nil {
		slog.Error("Critical: Failed to establish MinIO connection", "error", err)
		os.Exit(1)
	}

	dashboardService := services.NewDashboardService(dashboardRepo, authService, minioService)
	attachmentService := services.NewAttachmentService(attachmentRepo, settingsService, authService, journalService, adminAuditLogService, minioService)
	linkService := services.NewLinkService(linkRepo, incomingDocRepo, outgoingDocRepo, authService, journalService)
	acknowledgmentService := services.NewAcknowledgmentService(acknowledgmentRepo, userRepo, authService, journalService)
	systemService := services.NewSystemService(db)

	// Запуск приложения Wails
	err = wails.Run(&options.App{
		Title:  "Система регистрации документов",
		Width:  1280,
		Height: 1000,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Logger:   logger.NewWailsAdapter(),
		LogLevel: wailslogger.ERROR, // Пишем только ошибки, чтобы не спамить стандартными логами Wails
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnShutdown: func(ctx context.Context) {
			slog.Info("Gracefully shutting down database connection...")
			db.Close()
			closeLogger()
		},

		Bind: []interface{}{
			authService,
			userService,
			nomenclatureService,
			referenceService,
			incomingDocService,
			outgoingDocService,
			assignmentService,
			dashboardService,
			departmentService,
			settingsService,
			attachmentService,
			linkService,
			acknowledgmentService,
			systemService,
			journalService,
			adminAuditLogService,
		},
	})

	if err != nil {
		slog.Error("Error starting Wails app", "error", err)
		os.Exit(1)
	}
}
