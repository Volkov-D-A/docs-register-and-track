package services

import (
	"docflow/internal/dto"
	"docflow/internal/models"

	"github.com/google/uuid"
)

// AdminAuditLogService предоставляет бизнес-логику для журнала действий администраторов.
type AdminAuditLogService struct {
	repo AdminAuditLogStore
	auth *AuthService
}

// NewAdminAuditLogService создает новый экземпляр AdminAuditLogService.
func NewAdminAuditLogService(repo AdminAuditLogStore, auth *AuthService) *AdminAuditLogService {
	return &AdminAuditLogService{
		repo: repo,
		auth: auth,
	}
}

// GetAll возвращает записи журнала с пагинацией (только для администраторов).
func (s *AdminAuditLogService) GetAll(page, pageSize int) (*dto.AdminAuditLogPage, error) {
	if !s.auth.HasRole("admin") {
		return nil, models.NewForbidden("Недостаточно прав для просмотра журнала действий")
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

// LogAction — внутренний метод для логирования действий администраторов.
// Вызывается из других сервисов. Безопасен для вызова с nil receiver.
func (s *AdminAuditLogService) LogAction(userID uuid.UUID, userName, action, details string) {
	if s == nil {
		return
	}
	_, _ = s.repo.Create(models.CreateAdminAuditLogRequest{
		UserID:   userID,
		UserName: userName,
		Action:   action,
		Details:  details,
	})
}
