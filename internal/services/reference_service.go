package services

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
)

// ReferenceService предоставляет бизнес-логику для работы со справочниками (типы документов, организации).
type ReferenceService struct {
	repo         ReferenceStore
	auth         *AuthService
	auditService *AdminAuditLogService
}

// NewReferenceService создает новый экземпляр ReferenceService.
func NewReferenceService(repo ReferenceStore, auth *AuthService, auditService *AdminAuditLogService) *ReferenceService {
	return &ReferenceService{repo: repo, auth: auth, auditService: auditService}
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
	if err := s.auth.RequireRole("admin"); err != nil {
		return nil, err
	}
	res, err := s.repo.CreateDocumentType(name)
	if err != nil {
		return nil, err
	}

	userID, userName := s.auth.GetCurrentAuditInfo()
	s.auditService.LogAction(userID, userName, "DOCTYPE_CREATE", fmt.Sprintf("Создан тип документа «%s»", name))

	return dto.MapDocumentType(res), nil
}

// UpdateDocumentType обновляет название типа документа (только для администраторов).
func (s *ReferenceService) UpdateDocumentType(id string, name string) error {
	if err := s.auth.RequireRole("admin"); err != nil {
		return err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}
	if err := s.repo.UpdateDocumentType(uid, name); err != nil {
		return err
	}

	userID, userName := s.auth.GetCurrentAuditInfo()
	s.auditService.LogAction(userID, userName, "DOCTYPE_UPDATE", fmt.Sprintf("Обновлен тип документа «%s»", name))
	return nil
}

// DeleteDocumentType удаляет тип документа по его ID (только для администраторов).
func (s *ReferenceService) DeleteDocumentType(id string) error {
	if err := s.auth.RequireRole("admin"); err != nil {
		return err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}
	if err := s.repo.DeleteDocumentType(uid); err != nil {
		return err
	}

	userID, userName := s.auth.GetCurrentAuditInfo()
	s.auditService.LogAction(userID, userName, "DOCTYPE_DELETE", fmt.Sprintf("Удален тип документа (ID: %s)", id))
	return nil
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
	if err := s.auth.RequireRole("admin"); err != nil {
		return err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}
	if err := s.repo.UpdateOrganization(uid, name); err != nil {
		return err
	}

	userID, userName := s.auth.GetCurrentAuditInfo()
	s.auditService.LogAction(userID, userName, "ORG_UPDATE", fmt.Sprintf("Обновлена организация «%s»", name))
	return nil
}

// DeleteOrganization удаляет организацию по её ID (только для администраторов).
func (s *ReferenceService) DeleteOrganization(id string) error {
	if err := s.auth.RequireRole("admin"); err != nil {
		return err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}
	if err := s.repo.DeleteOrganization(uid); err != nil {
		return err
	}

	userID, userName := s.auth.GetCurrentAuditInfo()
	s.auditService.LogAction(userID, userName, "ORG_DELETE", fmt.Sprintf("Удалена организация (ID: %s)", id))
	return nil
}

