package services

import (
	"context"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
)

// SystemService предоставляет системные методы для фронтенда (проверка БД и др.).
type SystemService struct {
	ctx context.Context
	db  *database.DB
}

// NewSystemService создает новый экземпляр SystemService.
func NewSystemService(db *database.DB) *SystemService {
	return &SystemService{
		db: db,
	}
}

// Startup вызывается Wails при старте приложения
func (s *SystemService) Startup(ctx context.Context) {
	s.ctx = ctx
}

