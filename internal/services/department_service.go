package services

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"fmt"

	"github.com/google/uuid"
)

// DepartmentService предоставляет бизнес-логику для работы с подразделениями.
type DepartmentService struct {
	repo         DepartmentStore
	auth         *AuthService
	auditService *AdminAuditLogService
}

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
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	res, err := s.repo.GetAll()
	return dto.MapDepartments(res), err
}

// CreateDepartment создает новое подразделение.
func (s *DepartmentService) CreateDepartment(name string, nomenclatureIDs []string) (*dto.Department, error) {
	if err := s.auth.RequireRole("admin"); err != nil {
		return nil, err
	}
	res, err := s.repo.Create(name, nomenclatureIDs)
	if err != nil {
		return nil, err
	}

	userID, userName := s.auth.GetCurrentAuditInfo()
	s.auditService.LogAction(userID, userName, "DEPT_CREATE", fmt.Sprintf("Создано подразделение «%s»", name))

	return dto.MapDepartment(res), nil
}

// UpdateDepartment обновляет данные существующего подразделения.
func (s *DepartmentService) UpdateDepartment(id, name string, nomenclatureIDs []string) (*dto.Department, error) {
	if err := s.auth.RequireRole("admin"); err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid department ID: %w", err)
	}
	res, err := s.repo.Update(uid, name, nomenclatureIDs)
	if err != nil {
		return nil, err
	}

	userID, userName := s.auth.GetCurrentAuditInfo()
	s.auditService.LogAction(userID, userName, "DEPT_UPDATE", fmt.Sprintf("Обновлено подразделение «%s»", name))

	return dto.MapDepartment(res), nil
}

// DeleteDepartment удаляет подразделение по его ID.
func (s *DepartmentService) DeleteDepartment(id string) error {
	if err := s.auth.RequireRole("admin"); err != nil {
		return err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid department ID: %w", err)
	}
	if err := s.repo.Delete(uid); err != nil {
		return err
	}

	userID, userName := s.auth.GetCurrentAuditInfo()
	s.auditService.LogAction(userID, userName, "DEPT_DELETE", fmt.Sprintf("Удалено подразделение (ID: %s)", id))
	return nil
}

