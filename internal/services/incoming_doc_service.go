package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"docflow/internal/models"
	"docflow/internal/repository"
)

type IncomingDocumentService struct {
	ctx     context.Context
	repo    *repository.IncomingDocumentRepository
	nomRepo *repository.NomenclatureRepository
	refRepo *repository.ReferenceRepository
	depRepo *repository.DepartmentRepository
	auth    *AuthService
}

func NewIncomingDocumentService(
	repo *repository.IncomingDocumentRepository,
	nomRepo *repository.NomenclatureRepository,
	refRepo *repository.ReferenceRepository,
	depRepo *repository.DepartmentRepository,
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

func (s *IncomingDocumentService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// GetList — список входящих документов с фильтрацией
func (s *IncomingDocumentService) GetList(filter models.DocumentFilter) (*models.PagedResult[models.IncomingDocument], error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}

	user, err := s.auth.GetCurrentUser()
	if err != nil {
		return nil, err
	}

	// Если пользователь — исполнитель, ограничиваем видимость по номенклатурам подразделения
	if s.auth.HasRole("executor") && !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		filteredIDs, empty, err := filterNomenclaturesByDepartment(
			user.DepartmentID, s.depRepo, filter.NomenclatureIDs, filter.NomenclatureID,
		)
		if err != nil {
			return nil, err
		}
		if empty {
			return &models.PagedResult[models.IncomingDocument]{
				Items:      []models.IncomingDocument{},
				TotalCount: 0,
				Page:       filter.Page,
				PageSize:   filter.PageSize,
			}, nil
		}
		if filteredIDs != nil {
			filter.NomenclatureIDs = filteredIDs
		}
	}

	return s.repo.GetList(filter)
}

// GetByID — получить документ по ID
func (s *IncomingDocumentService) GetByID(id string) (*models.IncomingDocument, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid ID: %w", err)
	}
	return s.repo.GetByID(uid)
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
) (*models.IncomingDocument, error) {
	if !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		return nil, fmt.Errorf("недостаточно прав для регистрации документов")
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

	return s.repo.Create(
		nomID, docTypeID, senderOrg.ID, recipientOrg.ID, createdBy,
		incomingNumberStr, incDate,
		outgoingNumberSender, outDate,
		intNumPtr, intDatePtr,
		subject, content, pagesCount,
		senderSignatory, senderExecutor, addressee,
		resPtr,
	)
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
) (*models.IncomingDocument, error) {
	if !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		return nil, fmt.Errorf("недостаточно прав для редактирования документов")
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
		outDate = time.Now()
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

	return s.repo.Update(
		uid,
		docTypeID, senderOrg.ID, recipientOrg.ID,
		outgoingNumberSender, outDate,
		intNumPtr, intDatePtr,
		subject, content, pagesCount,
		senderSignatory, senderExecutor, addressee,
		resPtr,
	)
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
		return fmt.Errorf("недостаточно прав")
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}
	return s.repo.Delete(uid)
}
