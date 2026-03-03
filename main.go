package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"os"

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
	journalRepo := repository.NewJournalRepository(db)

	// Создание сервисов
	authService := services.NewAuthService(db, userRepo)
	settingsService := services.NewSettingsService(db, settingsRepo, authService)
	userService := services.NewUserService(userRepo, authService)
	nomenclatureService := services.NewNomenclatureService(nomenclatureRepo, authService)
	referenceService := services.NewReferenceService(referenceRepo, authService)
	journalService := services.NewJournalService(journalRepo, authService)

	incomingDocService := services.NewIncomingDocumentService(incomingDocRepo, nomenclatureRepo, referenceRepo, departmentRepo, authService, journalService)
	outgoingDocService := services.NewOutgoingDocumentService(outgoingDocRepo, referenceRepo, nomenclatureRepo, departmentRepo, authService, settingsService, journalService)
	assignmentService := services.NewAssignmentService(assignmentRepo, userRepo, authService, journalService)
	dashboardService := services.NewDashboardService(dashboardRepo, authService)
	departmentService := services.NewDepartmentService(departmentRepo, authService)
	attachmentService := services.NewAttachmentService(attachmentRepo, settingsService, authService, journalService)
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
			journalService,
		},
	})

	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
