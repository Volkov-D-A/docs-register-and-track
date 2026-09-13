package services

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"
)

// AdminAuditLogService предоставляет бизнес-логику для журнала действий администраторов.
type AdminAuditLogService struct {
	repo ports.AdminAuditLogStore
	auth ports.SystemPermissionPrincipal
}

// NewAdminAuditLogService создает новый экземпляр AdminAuditLogService.
func NewAdminAuditLogService(repo ports.AdminAuditLogStore, auth ports.SystemPermissionPrincipal) *AdminAuditLogService {
	return &AdminAuditLogService{
		repo: repo,
		auth: auth,
	}
}

// GetAll возвращает записи журнала с пагинацией (только для администраторов).
func (s *AdminAuditLogService) GetAll(page, pageSize int) (*dto.AdminAuditLogPage, error) {
	if err := s.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return nil, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	offset := (page - 1) * pageSize
	entries, total, err := s.repo.GetAll(pageSize, offset)
	if err != nil {
		return nil, err
	}

	return &dto.AdminAuditLogPage{
		Items: dto.MapAdminAuditLogs(entries),
		Total: total,
		Page:  page,
	}, nil
}
