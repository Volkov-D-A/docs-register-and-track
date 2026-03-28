package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// IncomingDocumentService предоставляет бизнес-логику для работы с входящими документами.
type IncomingDocumentService struct {
	repo    IncomingDocStore
	nomRepo NomenclatureStore
	refRepo ReferenceStore
	depRepo DepartmentStore
	auth    *AuthService
	journal *JournalService
}

// NewIncomingDocumentService создает новый экземпляр IncomingDocumentService.
func NewIncomingDocumentService(
	repo IncomingDocStore,
	nomRepo NomenclatureStore,
	refRepo ReferenceStore,
	depRepo DepartmentStore,
	auth *AuthService,
	journal *JournalService,
) *IncomingDocumentService {
	return &IncomingDocumentService{
		repo:    repo,
		nomRepo: nomRepo,
		refRepo: refRepo,
		depRepo: depRepo,
		auth:    auth,
		journal: journal,
	}
}

// GetList возвращает список входящих документов с учетом фильтров и прав доступа (для исполнителя видимость ограничена).
func (s *IncomingDocumentService) GetList(filter models.DocumentFilter) (*dto.PagedResult[dto.IncomingDocument], error) {
	if err := requireDocumentDomainReadRole(s.auth); err != nil {
		return nil, err
	}

	filteredIDs, empty, err := applyExecutorNomenclatureFilter(
		s.auth, s.depRepo, filter.NomenclatureIDs, filter.NomenclatureID,
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

// GetByID возвращает входящий документ по его ID.
func (s *IncomingDocumentService) GetByID(id string) (*dto.IncomingDocument, error) {
	if err := requireDocumentDomainReadRole(s.auth); err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid ID: %w", err)
	}
	res, err := s.repo.GetByID(uid)
	if err != nil || res == nil {
		return dto.MapIncomingDocument(res), err
	}
	if err := requireExecutorNomenclatureAccess(s.auth, s.depRepo, res.NomenclatureID); err != nil {
		return nil, err
	}
	return dto.MapIncomingDocument(res), nil
}

// Register регистрирует новый входящий документ в системе.
// Доступно только делопроизводителям.
func (s *IncomingDocumentService) Register(
	nomenclatureID, documentTypeID string,
	senderOrgName string,
	incomingDate, outgoingDateSender string,
	outgoingNumberSender string,
	intermediateNumber, intermediateDateStr string,
	content string, pagesCount int,
	senderSignatory string,
	resolution, resolutionAuthor, resolutionExecutors string,
) (*dto.IncomingDocument, error) {
	if err := requireClerkDocumentRole(s.auth); err != nil {
		return nil, err
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

	// Авто-добавление исполнителей резолюции в справочник
	if resolutionExecutors != "" {
		for _, name := range strings.Split(resolutionExecutors, "; ") {
			name = strings.TrimSpace(name)
			if name != "" {
				s.refRepo.FindOrCreateResolutionExecutor(name)
			}
		}
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
	var resAuthorPtr *string
	if resolutionAuthor != "" {
		resAuthorPtr = &resolutionAuthor
	}
	var resExecutorsPtr *string
	if resolutionExecutors != "" {
		resExecutorsPtr = &resolutionExecutors
	}

	createdByStr := s.auth.GetCurrentUserID()
	createdBy, err := uuid.Parse(createdByStr)
	if err != nil {
		return nil, ErrNotAuthenticated
	}

	res, err := s.repo.Create(models.CreateIncomingDocRequest{
		NomenclatureID:       nomID,
		DocumentTypeID:       docTypeID,
		SenderOrgID:          senderOrg.ID,
		CreatedBy:            createdBy,
		IncomingNumber:       incomingNumberStr,
		IncomingDate:         incDate,
		OutgoingNumberSender: outgoingNumberSender,
		OutgoingDateSender:   outDate,
		IntermediateNumber:   intNumPtr,
		IntermediateDate:     intDatePtr,
		Content:              content,
		PagesCount:           pagesCount,
		SenderSignatory:      senderSignatory,
		Resolution:           resPtr,
		ResolutionAuthor:     resAuthorPtr,
		ResolutionExecutors:  resExecutorsPtr,
	})
	if err == nil {
		s.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
			DocumentID:   res.ID,
			DocumentType: "incoming",
			UserID:       createdBy,
			Action:       "CREATE",
			Details:      fmt.Sprintf("Документ зарегистрирован. Рег. номер: %s", incomingNumberStr),
		})
	}
	return dto.MapIncomingDocument(res), err
}

// Update обновляет данные существующего входящего документа.
// Доступно только делопроизводителям.
func (s *IncomingDocumentService) Update(
	id, documentTypeID string,
	senderOrgName string,
	outgoingDateSender string,
	outgoingNumberSender string,
	intermediateNumber, intermediateDateStr string,
	content string, pagesCount int,
	senderSignatory string,
	resolution, resolutionAuthor, resolutionExecutors string,
) (*dto.IncomingDocument, error) {
	if err := requireClerkDocumentRole(s.auth); err != nil {
		return nil, err
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

	// Авто-добавление исполнителей резолюции в справочник
	if resolutionExecutors != "" {
		for _, name := range strings.Split(resolutionExecutors, "; ") {
			name = strings.TrimSpace(name)
			if name != "" {
				s.refRepo.FindOrCreateResolutionExecutor(name)
			}
		}
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
	var resAuthorPtr *string
	if resolutionAuthor != "" {
		resAuthorPtr = &resolutionAuthor
	}
	var resExecutorsPtr *string
	if resolutionExecutors != "" {
		resExecutorsPtr = &resolutionExecutors
	}

	res, err := s.repo.Update(models.UpdateIncomingDocRequest{
		ID:                   uid,
		DocumentTypeID:       docTypeID,
		SenderOrgID:          senderOrg.ID,
		OutgoingNumberSender: outgoingNumberSender,
		OutgoingDateSender:   outDate,
		IntermediateNumber:   intNumPtr,
		IntermediateDate:     intDatePtr,
		Content:              content,
		PagesCount:           pagesCount,
		SenderSignatory:      senderSignatory,
		Resolution:           resPtr,
		ResolutionAuthor:     resAuthorPtr,
		ResolutionExecutors:  resExecutorsPtr,
	})
	if err == nil {
		currentUserID, _ := s.auth.GetCurrentUserUUID()
		s.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
			DocumentID:   uid,
			DocumentType: "incoming",
			UserID:       currentUserID,
			Action:       "UPDATE",
			Details:      "Документ отредактирован",
		})
	}
	return dto.MapIncomingDocument(res), err
}

// GetCount возвращает общее количество входящих документов (например, для дашборда).
func (s *IncomingDocumentService) GetCount() (int, error) {
	if err := requireDocumentDomainReadRole(s.auth); err != nil {
		return 0, err
	}
	return s.repo.GetCount()
}

// Delete удаляет входящий документ по его ID.
// Доступно только делопроизводителям.
func (s *IncomingDocumentService) Delete(id string) error {
	if err := requireClerkDocumentRole(s.auth); err != nil {
		return err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}

	err = s.repo.Delete(uid)
	if err == nil {
		currentUserID, _ := s.auth.GetCurrentUserUUID()
		s.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
			DocumentID:   uid,
			DocumentType: "incoming",
			UserID:       currentUserID,
			Action:       "DELETE",
			Details:      "Документ удален",
		})
	}
	return err
}
