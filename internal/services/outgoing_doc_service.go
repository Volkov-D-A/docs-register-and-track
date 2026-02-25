package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"docflow/internal/dto"
	"docflow/internal/models"
)

type OutgoingDocumentService struct {
	repo            OutgoingDocStore
	refRepo         ReferenceStore
	nomRepo         NomenclatureStore
	depRepo         DepartmentStore
	auth            *AuthService
	settingsService *SettingsService
}

func NewOutgoingDocumentService(
	repo OutgoingDocStore,
	refRepo ReferenceStore,
	nomRepo NomenclatureStore,
	depRepo DepartmentStore,
	auth *AuthService,
	settingsService *SettingsService,
) *OutgoingDocumentService {
	return &OutgoingDocumentService{
		repo:            repo,
		refRepo:         refRepo,
		nomRepo:         nomRepo,
		depRepo:         depRepo,
		auth:            auth,
		settingsService: settingsService,
	}
}

// Register — регистрация нового исходящего документа
func (s *OutgoingDocumentService) Register(
	nomenclatureID, documentTypeID string,
	recipientOrgName, addressee string,
	outgoingDate string,
	subject, content string, pagesCount int,
	senderSignatory, senderExecutor string,
) (*dto.OutgoingDocument, error) {
	if !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		return nil, models.NewForbidden("недостаточно прав для регистрации документов")
	}

	// Парсинг UUID
	nomID, err := uuid.Parse(nomenclatureID)
	if err != nil {
		return nil, fmt.Errorf("неверный ID номенклатуры: %w", err)
	}
	docTypeID, err := uuid.Parse(documentTypeID)
	if err != nil {
		return nil, fmt.Errorf("неверный ID типа документа: %w", err)
	}

	// Автосоздание организации получателя
	recipientOrg, err := s.refRepo.FindOrCreateOrganization(recipientOrgName)
	if err != nil {
		return nil, fmt.Errorf("ошибка организации получателя: %w", err)
	}

	// Организация-отправитель (наша организация) — пока хардкодим или берем из настроек
	// В данном случае, sender_org_id в таблице outgoing_documents — это организация ОТ КОТОРОЙ уходит документ.
	// Обычно это "Наша Организация".
	// Для упрощения, пока создадим/найдем организацию "Наша Организация" или возьмем первую попавшуюся.
	// TODO: Вынести ID "Нашей Организации" в конфиг или настройки.
	orgName := s.settingsService.GetOrganizationName()
	senderOrg, err := s.refRepo.FindOrCreateOrganization(orgName)
	if err != nil {
		return nil, fmt.Errorf("ошибка вашей организации: %w", err)
	}

	// Получение рег. номера из номенклатуры
	nextNum, index, err := s.nomRepo.GetNextNumber(nomID)
	if err != nil {
		return nil, fmt.Errorf("ошибка автонумерации: %w", err)
	}
	outgoingNumber := formatDocumentNumber(index, nextNum)

	// Парсинг даты
	outDate, err := time.Parse("2006-01-02", outgoingDate)
	if err != nil {
		return nil, fmt.Errorf("неверный формат даты исходящего документа: %w", err)
	}

	// ID текущего пользователя
	createdByStr := s.auth.GetCurrentUserID()
	createdBy, err := uuid.Parse(createdByStr)
	if err != nil {
		return nil, ErrNotAuthenticated
	}

	res, err := s.repo.Create(
		nomID, docTypeID, senderOrg.ID, recipientOrg.ID, createdBy,
		outgoingNumber, outDate,
		subject, content, pagesCount,
		senderSignatory, senderExecutor, addressee,
	)
	return dto.MapOutgoingDocument(res), err
}

// Update — редактирование исходящего документа
func (s *OutgoingDocumentService) Update(
	id, documentTypeID string,
	recipientOrgName, addressee string,
	outgoingDate string,
	subject, content string, pagesCount int,
	senderSignatory, senderExecutor string,
) (*dto.OutgoingDocument, error) {
	if !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		return nil, models.NewForbidden("недостаточно прав для редактирования документов")
	}

	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("неверный ID документа: %w", err)
	}
	docTypeID, err := uuid.Parse(documentTypeID)
	if err != nil {
		return nil, fmt.Errorf("неверный ID типа документа: %w", err)
	}

	recipientOrg, err := s.refRepo.FindOrCreateOrganization(recipientOrgName)
	if err != nil {
		return nil, fmt.Errorf("ошибка организации получателя: %w", err)
	}

	senderOrg, err := s.refRepo.FindOrCreateOrganization(s.settingsService.GetOrganizationName())
	if err != nil {
		return nil, fmt.Errorf("ошибка вашей организации: %w", err)
	}

	outDate, err := time.Parse("2006-01-02", outgoingDate)
	if err != nil {
		return nil, models.NewBadRequest("неверный формат даты исходящего документа")
	}

	res, err := s.repo.Update(
		uid,
		docTypeID, senderOrg.ID, recipientOrg.ID,
		outDate,
		subject, content, pagesCount,
		senderSignatory, senderExecutor, addressee,
	)
	return dto.MapOutgoingDocument(res), err
}

// GetList — получение списка с фильтрацией
func (s *OutgoingDocumentService) GetList(filter models.OutgoingDocumentFilter) (*dto.PagedResult[dto.OutgoingDocument], error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	// Defaults
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}

	user, err := s.auth.GetCurrentUser()
	if err != nil {
		return nil, err
	}

	// Если пользователь — исполнитель, ограничиваем видимость по номенклатурам подразделения
	if s.auth.HasRole("executor") && !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		var deptID *uuid.UUID
		if user.Department != nil {
			if parsed, err := uuid.Parse(user.Department.ID); err == nil {
				deptID = &parsed
			}
		}
		filteredIDs, empty, err := filterNomenclaturesByDepartment(
			deptID, s.depRepo, filter.NomenclatureIDs, "",
		)
		if err != nil {
			return nil, err
		}
		if empty {
			return &dto.PagedResult[dto.OutgoingDocument]{
				Items:      []dto.OutgoingDocument{},
				TotalCount: 0,
				Page:       filter.Page,
				PageSize:   filter.PageSize,
			}, nil
		}
		if filteredIDs != nil {
			filter.NomenclatureIDs = filteredIDs
		}
	}

	res, err := s.repo.GetList(filter)
	if err != nil {
		return nil, err
	}
	return &dto.PagedResult[dto.OutgoingDocument]{
		Items:      dto.MapOutgoingDocuments(res.Items),
		TotalCount: res.TotalCount,
		Page:       res.Page,
		PageSize:   res.PageSize,
	}, nil
}

// GetByID — получение документа по ID
func (s *OutgoingDocumentService) GetByID(id string) (*dto.OutgoingDocument, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid ID: %w", err)
	}
	res, err := s.repo.GetByID(uid)
	return dto.MapOutgoingDocument(res), err
}

// Delete — удаление документа
func (s *OutgoingDocumentService) Delete(id string) error {
	if !s.auth.HasRole("admin") {
		return models.ErrForbidden
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}
	return s.repo.Delete(uid)
}

// GetCount — количество для дашборда
func (s *OutgoingDocumentService) GetCount() (int, error) {
	if !s.auth.IsAuthenticated() {
		return 0, ErrNotAuthenticated
	}
	return s.repo.GetCount()
}
