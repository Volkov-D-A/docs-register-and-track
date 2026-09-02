package main

import (
	"embed"
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
	if runBindingsGenerator() {
		return
	}

	configPath := config.GetDefaultConfigPath()
	cfg, err := config.Load(configPath)
	if err != nil {
		failStartup(startupdiag.Failure{
			Component:  "configuration",
			ConfigPath: configPath,
			Summary:    "Не удалось загрузить config.json.",
			NextStep:   "Проверьте DOCFLOW_CONFIG_PATH, наличие файла, права чтения и JSON-синтаксис.",
			Err:        err,
		})
	}

	serverClient, err := app.NewDesktopServerClient(cfg)
	if err != nil {
		failStartup(startupdiag.Failure{
			Component:  "docflow-server",
			ConfigPath: configPath,
			Summary:    "Некорректный адрес серверного API.",
			NextStep:   "Проверьте server.url в desktop config.json.",
			Err:        err,
		})
	}

	_, closeLogger := logger.InitDesktop(serverClient)
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
		ServerClient:       serverClient,
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
