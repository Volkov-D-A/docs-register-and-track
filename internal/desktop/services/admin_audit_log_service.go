package services

import (
	"context"
	"errors"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/serverclient"
	"time"
)

// AdminAuditLogService exposes server-owned journal reads through HTTP.
type AdminAuditLogService struct{ server serverclient.AdminAuditClient }

func NewAdminAuditLogService(client serverclient.AdminAuditClient) *AdminAuditLogService {
	return &AdminAuditLogService{server: client}
}

var errAdminAuditLogServiceClientNotConfigured = errors.New("docflow-server admin audit log client is not configured")

func (s *AdminAuditLogService) GetAll(page, pageSize int) (*dto.AdminAuditLogPage, error) {
	if s.server == nil {
		return nil, errAdminAuditLogServiceClientNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.server.GetAdminAuditLog(ctx, page, pageSize)
}
