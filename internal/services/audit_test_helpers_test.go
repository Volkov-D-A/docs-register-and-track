package services

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
)

type captureAdminAuditLogStore struct {
	requests []models.CreateAdminAuditLogRequest
}

func (s *captureAdminAuditLogStore) Create(req models.CreateAdminAuditLogRequest) (uuid.UUID, error) {
	s.requests = append(s.requests, req)
	return uuid.New(), nil
}
func (s *captureAdminAuditLogStore) GetAll(int, int) ([]models.AdminAuditLog, int, error) {
	return nil, 0, nil
}
