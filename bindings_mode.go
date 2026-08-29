//go:build bindings

package main

import (
	"github.com/wailsapp/wails/v2"

	"github.com/Volkov-D-A/docs-register-and-track/internal/app"
	"github.com/Volkov-D-A/docs-register-and-track/internal/startupdiag"
)

func runBindingsGenerator() bool {
	if err := wails.Run(app.NewBindingsWailsOptions()); err != nil {
		failStartup(startupdiag.Failure{
			Component: "Wails bindings",
			Summary:   "Не удалось сгенерировать Wails bindings.",
			NextStep:  "Проверьте Go toolchain, версию Wails и публичные сигнатуры сервисов.",
			Err:       err,
		})
	}
	return true
}
