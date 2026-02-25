package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"docflow/internal/dto"
	"docflow/internal/models"
)

type IncomingDocumentService struct {
	repo    IncomingDocStore
	nomRepo NomenclatureStore
	refRepo ReferenceStore
	depRepo DepartmentStore
	auth    *AuthService
}

func NewIncomingDocumentService(
	repo IncomingDocStore,
	nomRepo NomenclatureStore,
	refRepo ReferenceStore,
	depRepo DepartmentStore,
	auth *AuthService,
) *IncomingDocumentService {
	return &IncomingDocumentService{
		repo:    repo,
		nomRepo: nomRepo,
		refRepo: refRepo,
		depRepo: depRepo,
		auth:    auth,
	}
}

// GetList — список входящих документов с фильтрацией
func (s *IncomingDocumentService) GetList(filter models.DocumentFilter) (*dto.PagedResult[dto.IncomingDocument], error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
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
			deptID, s.depRepo, filter.NomenclatureIDs, filter.NomenclatureID,
		)
		if err != nil {
			return nil, err
		}
		if empty {
			return &dto.PagedResult[dto.IncomingDocument]{
				Items:      []dto.IncomingDocument{},
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
	return &dto.PagedResult[dto.IncomingDocument]{
		Items:      dto.MapIncomingDocuments(res.Items),
		TotalCount: res.TotalCount,
		Page:       res.Page,
		PageSize:   res.PageSize,
	}, nil
}

// GetByID — получить документ по ID
func (s *IncomingDocumentService) GetByID(id string) (*dto.IncomingDocument, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid ID: %w", err)
	}
	res, err := s.repo.GetByID(uid)
	return dto.MapIncomingDocument(res), err
}

// Register — регистрация нового входящего документа
func (s *IncomingDocumentService) Register(
	nomenclatureID, documentTypeID string,
	senderOrgName, recipientOrgName string,
	incomingDate, outgoingDateSender string,
	outgoingNumberSender string,
	intermediateNumber, intermediateDateStr string,
	subject, content string, pagesCount int,
	senderSignatory, senderExecutor, addressee string,
	resolution string,
) (*dto.IncomingDocument, error) {
	if !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		return nil, models.NewForbidden("недостаточно прав для регистрации документов")
	}

	nomID, err := uuid.Parse(nomenclatureID)
	if err != nil {
		return nil, fmt.Errorf("неверный ID номенклатуры: %w", err)
	}
	docTypeID, err := uuid.Parse(documentTypeID)
	if err != nil {
		return nil, fmt.Errorf("неверный ID типа документа: %w", err)
	}

	senderOrg, err := s.refRepo.FindOrCreateOrganization(senderOrgName)
	if err != nil {
		return nil, fmt.Errorf("ошибка организации отправителя: %w", err)
	}
	recipientOrg, err := s.refRepo.FindOrCreateOrganization(recipientOrgName)
	if err != nil {
		return nil, fmt.Errorf("ошибка организации получателя: %w", err)
	}

	nextNum, index, err := s.nomRepo.GetNextNumber(nomID)
	if err != nil {
		return nil, fmt.Errorf("ошибка автонумерации: %w", err)
	}
	incomingNumberStr := formatDocumentNumber(index, nextNum)

	incDate, err := time.Parse("2006-01-02", incomingDate)
	if err != nil {
		return nil, fmt.Errorf("неверный формат даты поступления: %w", err)
	}
	outDate, err := time.Parse("2006-01-02", outgoingDateSender)
	if err != nil {
		return nil, fmt.Errorf("неверный формат даты исходящего документа отправителя: %w", err)
	}

	var intNumPtr *string
	var intDatePtr *time.Time
	if intermediateNumber != "" {
		intNumPtr = &intermediateNumber
	}
	if intermediateDateStr != "" {
		if d, e := time.Parse("2006-01-02", intermediateDateStr); e == nil {
			intDatePtr = &d
		}
	}

	var resPtr *string
	if resolution != "" {
		resPtr = &resolution
	}

	createdByStr := s.auth.GetCurrentUserID()
	createdBy, err := uuid.Parse(createdByStr)
	if err != nil {
		return nil, ErrNotAuthenticated
	}

	res, err := s.repo.Create(
		nomID, docTypeID, senderOrg.ID, recipientOrg.ID, createdBy,
		incomingNumberStr, incDate,
		outgoingNumberSender, outDate,
		intNumPtr, intDatePtr,
		subject, content, pagesCount,
		senderSignatory, senderExecutor, addressee,
		resPtr,
	)
	return dto.MapIncomingDocument(res), err
}

// Update — редактирование входящего документа
func (s *IncomingDocumentService) Update(
	id, documentTypeID string,
	senderOrgName, recipientOrgName string,
	outgoingDateSender string,
	outgoingNumberSender string,
	intermediateNumber, intermediateDateStr string,
	subject, content string, pagesCount int,
	senderSignatory, senderExecutor, addressee string,
	resolution string,
) (*dto.IncomingDocument, error) {
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

	senderOrg, err := s.refRepo.FindOrCreateOrganization(senderOrgName)
	if err != nil {
		return nil, fmt.Errorf("ошибка организации отправителя: %w", err)
	}
	recipientOrg, err := s.refRepo.FindOrCreateOrganization(recipientOrgName)
	if err != nil {
		return nil, fmt.Errorf("ошибка организации получателя: %w", err)
	}

	outDate, err := time.Parse("2006-01-02", outgoingDateSender)
	if err != nil {
		return nil, models.NewBadRequest("неверный формат даты исходящего документа отправителя")
	}

	var intNumPtr *string
	var intDatePtr *time.Time
	if intermediateNumber != "" {
		intNumPtr = &intermediateNumber
	}
	if intermediateDateStr != "" {
		if d, e := time.Parse("2006-01-02", intermediateDateStr); e == nil {
			intDatePtr = &d
		}
	}

	var resPtr *string
	if resolution != "" {
		resPtr = &resolution
	}

	res, err := s.repo.Update(
		uid,
		docTypeID, senderOrg.ID, recipientOrg.ID,
		outgoingNumberSender, outDate,
		intNumPtr, intDatePtr,
		subject, content, pagesCount,
		senderSignatory, senderExecutor, addressee,
		resPtr,
	)
	return dto.MapIncomingDocument(res), err
}

// GetCount — количество для дашборда
func (s *IncomingDocumentService) GetCount() (int, error) {
	if !s.auth.IsAuthenticated() {
		return 0, ErrNotAuthenticated
	}
	return s.repo.GetCount()
}

// Delete — удаление документа
func (s *IncomingDocumentService) Delete(id string) error {
	if !s.auth.HasRole("admin") {
		return models.ErrForbidden
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}
	return s.repo.Delete(uid)
}
