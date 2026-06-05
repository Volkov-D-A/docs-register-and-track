package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"sync"

	"github.com/wailsapp/wails/v2"

	"github.com/Volkov-D-A/docs-register-and-track/internal/app"
	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/logger"
	"github.com/Volkov-D-A/docs-register-and-track/internal/startupdiag"
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

	_, closeLogger := logger.Init(cfg.Seq)
	var closeLoggerOnce sync.Once
	closeLoggerSafely := func() {
		closeLoggerOnce.Do(closeLogger)
	}
	defer closeLoggerSafely()

	wailsOptions, failure := app.NewWailsOptions(cfg, app.WailsOptionsParams{
		ConfigPath:         configPath,
		Assets:             assets,
		ReleaseNotesSource: releaseNotesSource,
		CloseLogger:        closeLoggerSafely,
	})
	if failure != nil {
		failStartup(*failure)
	}

	if err := wails.Run(wailsOptions); err != nil {
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
