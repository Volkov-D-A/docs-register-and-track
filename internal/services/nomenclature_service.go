package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"docflow/internal/models"
	"docflow/internal/repository"
)

type NomenclatureService struct {
	ctx  context.Context
	repo *repository.NomenclatureRepository
	auth *AuthService
}

func NewNomenclatureService(repo *repository.NomenclatureRepository, auth *AuthService) *NomenclatureService {
	return &NomenclatureService{repo: repo, auth: auth}
}

func (s *NomenclatureService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// GetAll — получить все дела номенклатуры
func (s *NomenclatureService) GetAll(year int, direction string) ([]models.Nomenclature, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	return s.repo.GetAll(year, direction)
}

// GetActiveForDirection — активные дела для выбора при регистрации
func (s *NomenclatureService) GetActiveForDirection(direction string) ([]models.Nomenclature, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	year := time.Now().Year()
	return s.repo.GetActiveByDirection(direction, year)
}

// Create — создать дело номенклатуры (только admin/clerk)
func (s *NomenclatureService) Create(name, index string, year int, direction string) (*models.Nomenclature, error) {
	if !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		return nil, fmt.Errorf("недостаточно прав")
	}
	return s.repo.Create(name, index, year, direction)
}

// Update — обновить дело номенклатуры
func (s *NomenclatureService) Update(id string, name, index string, year int, direction string, isActive bool) (*models.Nomenclature, error) {
	if !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		return nil, fmt.Errorf("недостаточно прав")
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid ID: %w", err)
	}
	return s.repo.Update(uid, name, index, year, direction, isActive)
}

// Delete — удалить дело номенклатуры
func (s *NomenclatureService) Delete(id string) error {
	if !s.auth.HasRole("admin") {
		return fmt.Errorf("недостаточно прав")
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}
	return s.repo.Delete(uid)
}
