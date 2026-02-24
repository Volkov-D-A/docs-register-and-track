package services

import (
	"context"
	"docflow/internal/dto"
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

func (s *DepartmentService) GetAllDepartments() ([]dto.Department, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	res, err := s.repo.GetAll()
	return dto.MapDepartments(res), err
}

func (s *DepartmentService) CreateDepartment(name string, nomenclatureIDs []string) (*dto.Department, error) {
	if !s.auth.HasRole("admin") {
		return nil, fmt.Errorf("недостаточно прав")
	}
	res, err := s.repo.Create(name, nomenclatureIDs)
	return dto.MapDepartment(res), err
}

func (s *DepartmentService) UpdateDepartment(id, name string, nomenclatureIDs []string) (*dto.Department, error) {
	if !s.auth.HasRole("admin") {
		return nil, fmt.Errorf("недостаточно прав")
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid department ID: %w", err)
	}
	res, err := s.repo.Update(uid, name, nomenclatureIDs)
	return dto.MapDepartment(res), err
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
