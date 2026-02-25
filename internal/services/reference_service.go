package services

import (
	"fmt"

	"github.com/google/uuid"

	"docflow/internal/dto"
	"docflow/internal/models"
)

type ReferenceService struct {
	repo ReferenceStore
	auth *AuthService
}

func NewReferenceService(repo ReferenceStore, auth *AuthService) *ReferenceService {
	return &ReferenceService{repo: repo, auth: auth}
}

// === Типы документов ===

func (s *ReferenceService) GetDocumentTypes() ([]dto.DocumentType, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	res, err := s.repo.GetAllDocumentTypes()
	return dto.MapDocumentTypes(res), err
}

func (s *ReferenceService) CreateDocumentType(name string) (*dto.DocumentType, error) {
	if !s.auth.HasRole("admin") {
		return nil, models.ErrForbidden
	}
	res, err := s.repo.CreateDocumentType(name)
	return dto.MapDocumentType(res), err
}

func (s *ReferenceService) UpdateDocumentType(id string, name string) error {
	if !s.auth.HasRole("admin") {
		return models.ErrForbidden
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}
	return s.repo.UpdateDocumentType(uid, name)
}

func (s *ReferenceService) DeleteDocumentType(id string) error {
	if !s.auth.HasRole("admin") {
		return models.ErrForbidden
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}
	return s.repo.DeleteDocumentType(uid)
}

// === Организации ===

func (s *ReferenceService) GetOrganizations() ([]dto.Organization, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	res, err := s.repo.GetAllOrganizations()
	return dto.MapOrganizations(res), err
}

func (s *ReferenceService) SearchOrganizations(query string) ([]dto.Organization, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	res, err := s.repo.SearchOrganizations(query)
	return dto.MapOrganizations(res), err
}

func (s *ReferenceService) FindOrCreateOrganization(name string) (*dto.Organization, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	res, err := s.repo.FindOrCreateOrganization(name)
	return dto.MapOrganization(res), err
}

func (s *ReferenceService) UpdateOrganization(id string, name string) error {
	if !s.auth.HasRole("admin") {
		return models.ErrForbidden
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
