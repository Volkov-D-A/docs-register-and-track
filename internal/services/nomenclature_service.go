package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"docflow/internal/dto"
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
func (s *NomenclatureService) GetAll(year int, direction string) ([]dto.Nomenclature, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	res, err := s.repo.GetAll(year, direction)
	return dto.MapNomenclatures(res), err
}

// GetActiveForDirection — активные дела для выбора при регистрации
func (s *NomenclatureService) GetActiveForDirection(direction string) ([]dto.Nomenclature, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	year := time.Now().Year()
	res, err := s.repo.GetActiveByDirection(direction, year)
	return dto.MapNomenclatures(res), err
}

// Create — создать дело номенклатуры (только admin/clerk)
func (s *NomenclatureService) Create(name, index string, year int, direction string) (*dto.Nomenclature, error) {
	if !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		return nil, fmt.Errorf("недостаточно прав")
	}
	res, err := s.repo.Create(name, index, year, direction)
	return dto.MapNomenclature(res), err
}

// Update — обновить дело номенклатуры
func (s *NomenclatureService) Update(id string, name, index string, year int, direction string, isActive bool) (*dto.Nomenclature, error) {
	if !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		return nil, fmt.Errorf("недостаточно прав")
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid ID: %w", err)
	}
	res, err := s.repo.Update(uid, name, index, year, direction, isActive)
	return dto.MapNomenclature(res), err
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
