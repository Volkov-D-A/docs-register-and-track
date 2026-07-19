package services

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// ReferenceService предоставляет бизнес-логику для работы со справочниками (типы документов, организации).
type ReferenceService struct {
	repo ReferenceStore
	auth *AuthService
}
type referenceOutboxStore interface {
	UpdateOrganizationWithOutbox(uuid.UUID, string, []models.OutboxEvent) error
	DeleteOrganizationWithOutbox(uuid.UUID, []models.OutboxEvent) error
	MergeOrganizationsWithOutbox(uuid.UUID, uuid.UUID, []models.OutboxEvent) error
	UpdateResolutionExecutorWithOutbox(uuid.UUID, string, []models.OutboxEvent) error
	DeleteResolutionExecutorWithOutbox(uuid.UUID, []models.OutboxEvent) error
}

var errReferenceOutboxStoreRequired = fmt.Errorf("reference store must support atomic outbox operations")

func (s *ReferenceService) auditEffect(key, action, details string) (models.OutboxEvent, error) {
	userID, userName := s.auth.GetCurrentAuditInfo()
	return NewAdminAuditOutboxEvent(key, models.CreateAdminAuditLogRequest{UserID: userID, UserName: userName, Action: action, Details: details})
}

// NewReferenceService создает новый экземпляр ReferenceService.
func NewReferenceService(repo ReferenceStore, auth *AuthService) *ReferenceService {
	return &ReferenceService{repo: repo, auth: auth}
}

func (s *ReferenceService) requireReferenceManagement() error {
	return s.auth.RequireSystemPermission(models.SystemPermissionReferences)
}

// === Типы документов ===

// GetDocumentTypes возвращает список всех типов документов.
func (s *ReferenceService) GetDocumentTypes() ([]dto.DocumentType, error) {
	if err := s.auth.RequireAuthenticated(); err != nil {
		return nil, err
	}

	items := make([]dto.DocumentType, 0, len(models.AllowedDocumentTypes()))
	for _, name := range models.AllowedDocumentTypes() {
		items = append(items, dto.DocumentType{
			ID:   name,
			Name: name,
		})
	}
	return items, nil
}

// CreateDocumentType создает новый тип документа для пользователей с доступом к справочникам.
func (s *ReferenceService) CreateDocumentType(name string) (*dto.DocumentType, error) {
	return nil, models.NewBadRequest("типы документов заданы в коде и не редактируются")
}

// UpdateDocumentType обновляет название типа документа для пользователей с доступом к справочникам.
func (s *ReferenceService) UpdateDocumentType(id string, name string) error {
	return models.NewBadRequest("типы документов заданы в коде и не редактируются")
}

// DeleteDocumentType удаляет тип документа по его ID для пользователей с доступом к справочникам.
func (s *ReferenceService) DeleteDocumentType(id string) error {
	return models.NewBadRequest("типы документов заданы в коде и не редактируются")
}

// === Организации ===

// GetOrganizations возвращает список всех организаций-корреспондентов.
func (s *ReferenceService) GetOrganizations() ([]dto.Organization, error) {
	if err := s.auth.RequireAuthenticated(); err != nil {
		return nil, err
	}
	res, err := s.repo.GetAllOrganizations()
	return dto.MapOrganizations(res), err
}

// SearchOrganizations выполняет поиск организаций по названию.
func (s *ReferenceService) SearchOrganizations(query string) ([]dto.Organization, error) {
	if err := s.auth.RequireAuthenticated(); err != nil {
		return nil, err
	}
	res, err := s.repo.SearchOrganizations(query)
	return dto.MapOrganizations(res), err
}

// FindOrCreateOrganization ищет организацию по названию, и создает новую, если она не найдена.
func (s *ReferenceService) FindOrCreateOrganization(name string) (*dto.Organization, error) {
	if err := s.auth.RequireAuthenticated(); err != nil {
		return nil, err
	}
	res, err := s.repo.FindOrCreateOrganization(name)
	return dto.MapOrganization(res), err
}

// UpdateOrganization обновляет название организации для пользователей с доступом к справочникам.
func (s *ReferenceService) UpdateOrganization(id string, name string) error {
	if err := s.requireReferenceManagement(); err != nil {
		return err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return models.NewBadRequestWrapped("неверный ID записи справочника", err)
	}
	details := fmt.Sprintf("Обновлена организация «%s»", name)
	store, ok := s.repo.(referenceOutboxStore)
	if !ok {
		return errReferenceOutboxStoreRequired
	}
	event, buildErr := s.auditEffect("organization:"+uid.String()+":update:"+uuid.NewString(), "ORG_UPDATE", details)
	if buildErr != nil {
		return buildErr
	}
	err = store.UpdateOrganizationWithOutbox(uid, name, []models.OutboxEvent{event})
	if err != nil {
		return err
	}

	return nil
}

// DeleteOrganization удаляет организацию по её ID для пользователей с доступом к справочникам.
func (s *ReferenceService) DeleteOrganization(id string) error {
	if err := s.requireReferenceManagement(); err != nil {
		return err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return models.NewBadRequestWrapped("неверный ID записи справочника", err)
	}
	details := fmt.Sprintf("Удалена организация (ID: %s)", id)
	store, ok := s.repo.(referenceOutboxStore)
	if !ok {
		return errReferenceOutboxStoreRequired
	}
	event, buildErr := s.auditEffect("organization:"+uid.String()+":delete", "ORG_DELETE", details)
	if buildErr != nil {
		return buildErr
	}
	err = store.DeleteOrganizationWithOutbox(uid, []models.OutboxEvent{event})
	if err != nil {
		return err
	}

	return nil
}

// MergeOrganizations объединяет две записи справочника организаций.
func (s *ReferenceService) MergeOrganizations(sourceID string, targetID string) error {
	if err := s.requireReferenceManagement(); err != nil {
		return err
	}
	sourceUID, err := uuid.Parse(sourceID)
	if err != nil {
		return models.NewBadRequestWrapped("неверный ID исходной организации", err)
	}
	targetUID, err := uuid.Parse(targetID)
	if err != nil {
		return models.NewBadRequestWrapped("неверный ID целевой организации", err)
	}
	if sourceUID == targetUID {
		return models.NewBadRequest("нельзя объединить организацию саму с собой")
	}
	details := fmt.Sprintf("Объединены организации: %s -> %s", sourceID, targetID)
	store, ok := s.repo.(referenceOutboxStore)
	if !ok {
		return errReferenceOutboxStoreRequired
	}
	event, buildErr := s.auditEffect("organization:"+sourceUID.String()+":merge:"+targetUID.String(), "ORG_MERGE", details)
	if buildErr != nil {
		return buildErr
	}
	err = store.MergeOrganizationsWithOutbox(sourceUID, targetUID, []models.OutboxEvent{event})
	if err != nil {
		return err
	}

	return nil
}

// === Исполнители резолюции ===

// GetResolutionExecutors возвращает список всех исполнителей резолюции.
func (s *ReferenceService) GetResolutionExecutors() ([]dto.ResolutionExecutor, error) {
	if err := s.auth.RequireAuthenticated(); err != nil {
		return nil, err
	}
	res, err := s.repo.GetAllResolutionExecutors()
	return dto.MapResolutionExecutors(res), err
}

// SearchResolutionExecutors выполняет поиск исполнителей резолюции по имени.
func (s *ReferenceService) SearchResolutionExecutors(query string) ([]dto.ResolutionExecutor, error) {
	if err := s.auth.RequireAuthenticated(); err != nil {
		return nil, err
	}
	res, err := s.repo.SearchResolutionExecutors(query)
	return dto.MapResolutionExecutors(res), err
}

// FindOrCreateResolutionExecutor ищет исполнителя по имени, и создает нового, если он не найден.
func (s *ReferenceService) FindOrCreateResolutionExecutor(name string) (*dto.ResolutionExecutor, error) {
	if err := s.auth.RequireAuthenticated(); err != nil {
		return nil, err
	}
	res, err := s.repo.FindOrCreateResolutionExecutor(name)
	return dto.MapResolutionExecutor(res), err
}

// UpdateResolutionExecutor обновляет имя исполнителя резолюции для пользователей с доступом к справочникам.
func (s *ReferenceService) UpdateResolutionExecutor(id string, name string) error {
	if err := s.requireReferenceManagement(); err != nil {
		return err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return models.NewBadRequestWrapped("неверный ID записи справочника", err)
	}
	details := fmt.Sprintf("Обновлен исполнитель резолюции «%s»", name)
	store, ok := s.repo.(referenceOutboxStore)
	if !ok {
		return errReferenceOutboxStoreRequired
	}
	event, buildErr := s.auditEffect("resolution-executor:"+uid.String()+":update:"+uuid.NewString(), "RESEXEC_UPDATE", details)
	if buildErr != nil {
		return buildErr
	}
	err = store.UpdateResolutionExecutorWithOutbox(uid, name, []models.OutboxEvent{event})
	if err != nil {
		return err
	}

	return nil
}

// DeleteResolutionExecutor удаляет исполнителя резолюции по его ID для пользователей с доступом к справочникам.
func (s *ReferenceService) DeleteResolutionExecutor(id string) error {
	if err := s.requireReferenceManagement(); err != nil {
		return err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return models.NewBadRequestWrapped("неверный ID записи справочника", err)
	}
	details := fmt.Sprintf("Удален исполнитель резолюции (ID: %s)", id)
	store, ok := s.repo.(referenceOutboxStore)
	if !ok {
		return errReferenceOutboxStoreRequired
	}
	event, buildErr := s.auditEffect("resolution-executor:"+uid.String()+":delete", "RESEXEC_DELETE", details)
	if buildErr != nil {
		return buildErr
	}
	err = store.DeleteResolutionExecutorWithOutbox(uid, []models.OutboxEvent{event})
	if err != nil {
		return err
	}

	return nil
}
