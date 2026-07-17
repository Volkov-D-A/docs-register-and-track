package services

import (
	"fmt"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"

	"github.com/google/uuid"
)

// DepartmentService предоставляет бизнес-логику для работы с подразделениями.
type DepartmentService struct {
	repo         DepartmentStore
	auth         *AuthService
	auditService *AdminAuditLogService
}
type departmentOutboxStore interface {
	CreateWithOutbox(string, []string, []models.OutboxEvent) (*models.Department, error)
	UpdateWithOutbox(uuid.UUID, string, []string, []models.OutboxEvent) (*models.Department, error)
	DeleteWithOutbox(uuid.UUID, []models.OutboxEvent) error
}

var errDepartmentOutboxStoreRequired = fmt.Errorf("department store must support atomic outbox operations")

// NewDepartmentService создает новый экземпляр DepartmentService.
func NewDepartmentService(repo DepartmentStore, auth *AuthService, auditService *AdminAuditLogService) *DepartmentService {
	return &DepartmentService{
		repo:         repo,
		auth:         auth,
		auditService: auditService,
	}
}

// GetAllDepartments возвращает список всех подразделений.
func (s *DepartmentService) GetAllDepartments() ([]dto.Department, error) {
	if err := s.auth.RequireAuthenticated(); err != nil {
		return nil, err
	}
	res, err := s.repo.GetAll()
	return dto.MapDepartments(res), err
}

// CreateDepartment создает новое подразделение.
func (s *DepartmentService) CreateDepartment(name string, nomenclatureIDs []string) (*dto.Department, error) {
	if err := s.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return nil, err
	}
	userID, userName := s.auth.GetCurrentAuditInfo()
	var res *models.Department
	var err error
	store, ok := s.repo.(departmentOutboxStore)
	if !ok {
		return nil, errDepartmentOutboxStoreRequired
	}
	event, buildErr := NewAdminAuditOutboxEvent("department:"+uuid.NewString()+":create", models.CreateAdminAuditLogRequest{UserID: userID, UserName: userName, Action: "DEPT_CREATE", Details: fmt.Sprintf("Создано подразделение «%s»", name)})
	if buildErr != nil {
		return nil, buildErr
	}
	res, err = store.CreateWithOutbox(name, nomenclatureIDs, []models.OutboxEvent{event})
	if err != nil {
		return nil, err
	}

	return dto.MapDepartment(res), nil
}

// UpdateDepartment обновляет данные существующего подразделения.
func (s *DepartmentService) UpdateDepartment(id, name string, nomenclatureIDs []string) (*dto.Department, error) {
	if err := s.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, models.NewBadRequestWrapped("неверный ID отдела", err)
	}
	userID, userName := s.auth.GetCurrentAuditInfo()
	var res *models.Department
	store, ok := s.repo.(departmentOutboxStore)
	if !ok {
		return nil, errDepartmentOutboxStoreRequired
	}
	event, buildErr := NewAdminAuditOutboxEvent("department:"+uid.String()+":update:"+uuid.NewString(), models.CreateAdminAuditLogRequest{UserID: userID, UserName: userName, Action: "DEPT_UPDATE", Details: fmt.Sprintf("Обновлено подразделение «%s»", name)})
	if buildErr != nil {
		return nil, buildErr
	}
	res, err = store.UpdateWithOutbox(uid, name, nomenclatureIDs, []models.OutboxEvent{event})
	if err != nil {
		return nil, err
	}

	return dto.MapDepartment(res), nil
}

// DeleteDepartment удаляет подразделение по его ID.
func (s *DepartmentService) DeleteDepartment(id string) error {
	if err := s.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return models.NewBadRequestWrapped("неверный ID отдела", err)
	}
	userID, userName := s.auth.GetCurrentAuditInfo()
	store, ok := s.repo.(departmentOutboxStore)
	if !ok {
		return errDepartmentOutboxStoreRequired
	}
	event, buildErr := NewAdminAuditOutboxEvent("department:"+uid.String()+":delete", models.CreateAdminAuditLogRequest{UserID: userID, UserName: userName, Action: "DEPT_DELETE", Details: fmt.Sprintf("Удалено подразделение (ID: %s)", id)})
	if buildErr != nil {
		return buildErr
	}
	err = store.DeleteWithOutbox(uid, []models.OutboxEvent{event})
	if err != nil {
		return err
	}

	return nil
}
