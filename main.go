package main

import (
	"context"
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"docflow/internal/config"
	"docflow/internal/database"
	"docflow/internal/repository"
	"docflow/internal/services"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Загрузка конфигурации
	cfg, err := config.Load(config.GetDefaultConfigPath())
	if err != nil {
		log.Printf("Warning: Could not load config: %v.", err)
	}

	// Подключение к БД
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Printf("Warning: Failed to establish database connection pool: %v", err)
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

	// Создание сервисов
	authService := services.NewAuthService(db, userRepo)
	settingsService := services.NewSettingsService(db, settingsRepo, authService)
	userService := services.NewUserService(userRepo, authService)
	nomenclatureService := services.NewNomenclatureService(nomenclatureRepo, authService)
	referenceService := services.NewReferenceService(referenceRepo, authService)
	incomingDocService := services.NewIncomingDocumentService(incomingDocRepo, nomenclatureRepo, referenceRepo, departmentRepo, authService)
	outgoingDocService := services.NewOutgoingDocumentService(outgoingDocRepo, referenceRepo, nomenclatureRepo, departmentRepo, authService, settingsService)
	assignmentService := services.NewAssignmentService(assignmentRepo, userRepo, authService)
	dashboardService := services.NewDashboardService(dashboardRepo, authService)
	departmentService := services.NewDepartmentService(departmentRepo, authService)
	attachmentService := services.NewAttachmentService(attachmentRepo, settingsService, authService)
	linkService := services.NewLinkService(linkRepo, incomingDocRepo, outgoingDocRepo, authService)
	acknowledgmentService := services.NewAcknowledgmentService(acknowledgmentRepo, userRepo, authService)
	systemService := services.NewSystemService(db)

	// Запуск приложения Wails
	err = wails.Run(&options.App{
		Title:  "Система регистрации документов",
		Width:  1280,
		Height: 1000,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnShutdown: func(ctx context.Context) {
			log.Println("Gracefully shutting down database connection...")
			db.Close()
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
		},
	})

	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
