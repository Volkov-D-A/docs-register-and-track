package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"docflow/internal/models"
	"docflow/internal/repository"
)

type ReferenceService struct {
	ctx  context.Context
	repo *repository.ReferenceRepository
	auth *AuthService
}

func NewReferenceService(repo *repository.ReferenceRepository, auth *AuthService) *ReferenceService {
	return &ReferenceService{repo: repo, auth: auth}
}

func (s *ReferenceService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// === Типы документов ===

func (s *ReferenceService) GetDocumentTypes() ([]models.DocumentType, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	return s.repo.GetAllDocumentTypes()
}

func (s *ReferenceService) CreateDocumentType(name string) (*models.DocumentType, error) {
	if !s.auth.HasRole("admin") {
		return nil, fmt.Errorf("недостаточно прав")
	}
	return s.repo.CreateDocumentType(name)
}

func (s *ReferenceService) UpdateDocumentType(id string, name string) error {
	if !s.auth.HasRole("admin") {
		return fmt.Errorf("недостаточно прав")
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}
	return s.repo.UpdateDocumentType(uid, name)
}

func (s *ReferenceService) DeleteDocumentType(id string) error {
	if !s.auth.HasRole("admin") {
		return fmt.Errorf("недостаточно прав")
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}
	return s.repo.DeleteDocumentType(uid)
}

// === Организации ===

func (s *ReferenceService) GetOrganizations() ([]models.Organization, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	return s.repo.GetAllOrganizations()
}

func (s *ReferenceService) SearchOrganizations(query string) ([]models.Organization, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	return s.repo.SearchOrganizations(query)
}

func (s *ReferenceService) FindOrCreateOrganization(name string) (*models.Organization, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	return s.repo.FindOrCreateOrganization(name)
}

func (s *ReferenceService) UpdateOrganization(id string, name string) error {
	if !s.auth.HasRole("admin") {
		return fmt.Errorf("недостаточно прав")
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}
	return s.repo.UpdateOrganization(uid, name)
}

func (s *ReferenceService) DeleteOrganization(id string) error {
	if !s.auth.HasRole("admin") {
		return fmt.Errorf("недостаточно прав")
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}
	return s.repo.DeleteOrganization(uid)
}
