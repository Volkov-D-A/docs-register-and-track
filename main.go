package main

import (
	"context"
	"embed"
	"fmt"
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
		log.Printf("Warning: Could not load config: %v. Using defaults.", err)
		cfg = &config.Config{
			Database: config.DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "docflow",
				Password: "docflow_password",
				DBName:   "docflow",
				SSLMode:  "disable",
			},
		}
	}

	// Подключение к БД
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Применение миграций
	if err := db.RunMigrations("internal/database/migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	fmt.Println("Database migrations applied successfully")

	// Создание репозиториев
	userRepo := repository.NewUserRepository(db)
	nomenclatureRepo := repository.NewNomenclatureRepository(db)
	referenceRepo := repository.NewReferenceRepository(db)
	incomingDocRepo := repository.NewIncomingDocumentRepository(db)
	outgoingDocRepo := repository.NewOutgoingDocumentRepository(db)
	assignmentRepo := repository.NewAssignmentRepository(db)

	// Создание сервисов
	authService := services.NewAuthService(userRepo)
	userService := services.NewUserService(userRepo, authService)
	nomenclatureService := services.NewNomenclatureService(nomenclatureRepo, authService)
	referenceService := services.NewReferenceService(referenceRepo, authService)
	incomingDocService := services.NewIncomingDocumentService(incomingDocRepo, nomenclatureRepo, referenceRepo, authService)
	outgoingDocService := services.NewOutgoingDocumentService(outgoingDocRepo, referenceRepo, nomenclatureRepo, authService)
	assignmentService := services.NewAssignmentService(assignmentRepo, userRepo, authService)
	dashboardService := services.NewDashboardService(db, authService)

	// Запуск приложения Wails
	err = wails.Run(&options.App{
		Title:  "Документооборот",
		Width:  1280,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnStartup: func(ctx context.Context) {
			authService.SetContext(ctx)
			userService.SetContext(ctx)
			nomenclatureService.SetContext(ctx)
			referenceService.SetContext(ctx)
			incomingDocService.SetContext(ctx)
			outgoingDocService.SetContext(ctx)
			assignmentService.SetContext(ctx)
			dashboardService.SetContext(ctx)
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
		},
	})

	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
