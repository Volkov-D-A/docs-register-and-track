package logger

import (
	"log/slog"
	"os"
)

// WailsAdapter перенаправляет логи из встроенного обработчика Wails в стандартный slog.
// Это позволяет автоматически отлавливать все ошибки, возвращаемые из методов сервисов.
type WailsAdapter struct{}

// NewWailsAdapter возвращает новый адаптер для Wails.
func NewWailsAdapter() *WailsAdapter {
	return &WailsAdapter{}
}

func (a *WailsAdapter) Print(message string) {
	slog.Info(message)
}

func (a *WailsAdapter) Trace(message string) {
	slog.Debug(message)
}

func (a *WailsAdapter) Debug(message string) {
	slog.Debug(message)
}

func (a *WailsAdapter) Info(message string) {
	slog.Info(message)
}

func (a *WailsAdapter) Warning(message string) {
	slog.Warn(message)
}

func (a *WailsAdapter) Error(message string) {
	slog.Error(message)
}

func (a *WailsAdapter) Fatal(message string) {
	slog.Error(message)
	os.Exit(1)
}
