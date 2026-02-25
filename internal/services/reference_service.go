package services

import (
	"fmt"

	"github.com/google/uuid"

	"docflow/internal/dto"
	"docflow/internal/models"
)

// ReferenceService предоставляет бизнес-логику для работы со справочниками (типы документов, организации).
type ReferenceService struct {
	repo ReferenceStore
	auth *AuthService
}

// NewReferenceService создает новый экземпляр ReferenceService.
func NewReferenceService(repo ReferenceStore, auth *AuthService) *ReferenceService {
	return &ReferenceService{repo: repo, auth: auth}
}

// === Типы документов ===

// GetDocumentTypes возвращает список всех типов документов.
func (s *ReferenceService) GetDocumentTypes() ([]dto.DocumentType, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	res, err := s.repo.GetAllDocumentTypes()
	return dto.MapDocumentTypes(res), err
}

// CreateDocumentType создает новый тип документа (только для администраторов).
func (s *ReferenceService) CreateDocumentType(name string) (*dto.DocumentType, error) {
	if !s.auth.HasRole("admin") {
		return nil, models.ErrForbidden
	}
	res, err := s.repo.CreateDocumentType(name)
	return dto.MapDocumentType(res), err
}

// UpdateDocumentType обновляет название типа документа (только для администраторов).
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

// DeleteDocumentType удаляет тип документа по его ID (только для администраторов).
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

// GetOrganizations возвращает список всех организаций-корреспондентов.
func (s *ReferenceService) GetOrganizations() ([]dto.Organization, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	res, err := s.repo.GetAllOrganizations()
	return dto.MapOrganizations(res), err
}

// SearchOrganizations выполняет поиск организаций по названию.
func (s *ReferenceService) SearchOrganizations(query string) ([]dto.Organization, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	res, err := s.repo.SearchOrganizations(query)
	return dto.MapOrganizations(res), err
}

// FindOrCreateOrganization ищет организацию по названию, и создает новую, если она не найдена.
func (s *ReferenceService) FindOrCreateOrganization(name string) (*dto.Organization, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	res, err := s.repo.FindOrCreateOrganization(name)
	return dto.MapOrganization(res), err
}

// UpdateOrganization обновляет название организации (только для администраторов).
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

// DeleteOrganization удаляет организацию по её ID (только для администраторов).
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
