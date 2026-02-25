package services

import (
	"docflow/internal/dto"
	"docflow/internal/models"
	"fmt"

	"github.com/google/uuid"
)

// DepartmentService предоставляет бизнес-логику для работы с подразделениями.
type DepartmentService struct {
	repo DepartmentStore
	auth *AuthService
}

// NewDepartmentService создает новый экземпляр DepartmentService.
func NewDepartmentService(repo DepartmentStore, auth *AuthService) *DepartmentService {
	return &DepartmentService{
		repo: repo,
		auth: auth,
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
	if !s.auth.HasRole("admin") {
		return nil, models.ErrForbidden
	}
	res, err := s.repo.Create(name, nomenclatureIDs)
	return dto.MapDepartment(res), err
}

// UpdateDepartment обновляет данные существующего подразделения.
func (s *DepartmentService) UpdateDepartment(id, name string, nomenclatureIDs []string) (*dto.Department, error) {
	if !s.auth.HasRole("admin") {
		return nil, models.ErrForbidden
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid department ID: %w", err)
	}
	res, err := s.repo.Update(uid, name, nomenclatureIDs)
	return dto.MapDepartment(res), err
}

// DeleteDepartment удаляет подразделение по его ID.
func (s *DepartmentService) DeleteDepartment(id string) error {
	if !s.auth.HasRole("admin") {
		return models.ErrForbidden
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid department ID: %w", err)
	}
	return s.repo.Delete(uid)
}
