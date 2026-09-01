package services

import (
	"context"
)

// SystemService предоставляет системные методы для фронтенда (проверка БД и др.).
type SystemService struct {
	ctx context.Context
}

// NewSystemService создает новый экземпляр SystemService.
func NewSystemService() *SystemService { return &SystemService{} }

// Startup вызывается Wails при старте приложения
func (s *SystemService) Startup(ctx context.Context) {
	s.ctx = ctx
}
