package services

import (
	"context"
	"docflow/internal/models"
	"docflow/internal/repository"
	"fmt"

	"github.com/google/uuid"
)

type DepartmentService struct {
	ctx  context.Context
	repo *repository.DepartmentRepository
	auth *AuthService
}

func NewDepartmentService(repo *repository.DepartmentRepository, auth *AuthService) *DepartmentService {
	return &DepartmentService{
		repo: repo,
		auth: auth,
	}
}

func (s *DepartmentService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

func (s *DepartmentService) GetAllDepartments() ([]models.Department, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	return s.repo.GetAll()
}

func (s *DepartmentService) CreateDepartment(name string) (*models.Department, error) {
	if !s.auth.HasRole("admin") {
		return nil, fmt.Errorf("недостаточно прав")
	}
	return s.repo.Create(name)
}

func (s *DepartmentService) UpdateDepartment(id, name string) (*models.Department, error) {
	if !s.auth.HasRole("admin") {
		return nil, fmt.Errorf("недостаточно прав")
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid department ID: %w", err)
	}
	return s.repo.Update(uid, name)
}

func (s *DepartmentService) DeleteDepartment(id string) error {
	if !s.auth.HasRole("admin") {
		return fmt.Errorf("недостаточно прав")
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid department ID: %w", err)
	}
	return s.repo.Delete(uid)
}
