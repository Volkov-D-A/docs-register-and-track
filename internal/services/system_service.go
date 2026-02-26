package services

import (
	"context"

	"docflow/internal/database"
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

// CheckDBConnection проверяет доступность базы данных.
// Возвращает true, если база данных доступна, иначе false.
func (s *SystemService) CheckDBConnection() bool {
	if s.db == nil || s.db.DB == nil {
		return false
	}
	err := s.db.Ping()
	return err == nil
}

// ReconnectDB пытается переподключиться к базе данных.
// Возвращает true в случае успеха, иначе false.
func (s *SystemService) ReconnectDB() bool {
	if s.db == nil || s.db.DB == nil {
		return false
	}
	// Метод Ping() в database/sql автоматически пытается установить соединение,
	// если оно было разорвано или не было установлено.
	err := s.db.Ping()
	return err == nil
}
