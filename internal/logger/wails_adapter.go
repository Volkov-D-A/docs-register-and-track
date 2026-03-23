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

func (a *WailsAdapter) Print(_ string)   {}
func (a *WailsAdapter) Trace(_ string)   {}
func (a *WailsAdapter) Debug(_ string)   {}
func (a *WailsAdapter) Info(_ string)    {}
func (a *WailsAdapter) Warning(_ string) {}

func (a *WailsAdapter) Error(message string) {
	slog.Error(message)
}

func (a *WailsAdapter) Fatal(message string) {
	slog.Error(message)
	os.Exit(1)
}
