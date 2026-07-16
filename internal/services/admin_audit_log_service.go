package services

import (
	"log/slog"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"

	"github.com/google/uuid"
)

// AdminAuditLogService предоставляет бизнес-логику для журнала действий администраторов.
type AdminAuditLogService struct {
	repo   AdminAuditLogStore
	auth   *AuthService
	outbox *OutboxPublisher
}

func (s *AdminAuditLogService) SetOutboxPublisher(publisher *OutboxPublisher) { s.outbox = publisher }

// NewAdminAuditLogService создает новый экземпляр AdminAuditLogService.
func NewAdminAuditLogService(repo AdminAuditLogStore, auth *AuthService) *AdminAuditLogService {
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

// LogAction — внутренний метод для логирования действий администраторов.
// Вызывается из других сервисов. Безопасен для вызова с nil receiver.
func (s *AdminAuditLogService) LogAction(userID uuid.UUID, userName, action, details string) {
	if s == nil {
		return
	}
	req := models.CreateAdminAuditLogRequest{
		UserID:   userID,
		UserName: userName,
		Action:   action,
		Details:  details,
	}
	if s.outbox != nil {
		if err := s.outbox.PublishAdminAudit("admin-audit:"+uuid.NewString(), req); err != nil {
			slog.Error("failed to enqueue administrative audit", "action", action, "error", err)
		}
		return
	}
	if _, err := s.repo.Create(req); err != nil {
		slog.Error("failed to write administrative audit", "action", action, "error", err)
	}
}
