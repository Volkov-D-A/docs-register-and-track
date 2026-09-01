package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"

	"github.com/google/uuid"
)

// AdminAuditLogService предоставляет бизнес-логику для журнала действий администраторов.
type AdminAuditLogService struct {
	repo   AdminAuditLogStore
	auth   SystemPermissionPrincipal
	server serverclient.AdminAuditClient
}

type SystemPermissionPrincipal interface {
	RequireSystemPermission(string) error
}

// NewAdminAuditLogService создает новый экземпляр AdminAuditLogService.
func NewAdminAuditLogService(repo AdminAuditLogStore, auth SystemPermissionPrincipal) *AdminAuditLogService {
	return &AdminAuditLogService{
		repo: repo,
		auth: auth,
	}
}

func NewAdminAuditLogServiceWithClient(client serverclient.AdminAuditClient) *AdminAuditLogService {
	return &AdminAuditLogService{server: client}
}

// GetAll возвращает записи журнала с пагинацией (только для администраторов).
func (s *AdminAuditLogService) GetAll(page, pageSize int) (*dto.AdminAuditLogPage, error) {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return s.server.GetAdminAuditLog(ctx, page, pageSize)
	}
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
	if s == nil || s.repo == nil {
		return
	}
	req := models.CreateAdminAuditLogRequest{
		UserID:   userID,
		UserName: userName,
		Action:   action,
		Details:  details,
	}
	if _, err := s.repo.Create(req); err != nil {
		slog.Error("failed to write administrative audit", "action", action, "error", err)
	}
}
