package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// OutgoingDocumentService предоставляет бизнес-логику для работы с исходящими документами.
type OutgoingDocumentService struct {
	repo           OutgoingDocStore
	refRepo        ReferenceStore
	nomRepo        NomenclatureStore
	depRepo        DepartmentStore
	assignmentRepo AssignmentStore
	auth           *AuthService
	journal        *JournalService
}

// NewOutgoingDocumentService создает новый экземпляр OutgoingDocumentService.
func NewOutgoingDocumentService(
	repo OutgoingDocStore,
	refRepo ReferenceStore,
	nomRepo NomenclatureStore,
	depRepo DepartmentStore,
	assignmentRepo AssignmentStore,
	auth *AuthService,
	journal *JournalService,
) *OutgoingDocumentService {
	return &OutgoingDocumentService{
		repo:           repo,
		refRepo:        refRepo,
		nomRepo:        nomRepo,
		depRepo:        depRepo,
		assignmentRepo: assignmentRepo,
		auth:           auth,
		journal:        journal,
	}
}

// Register регистрирует новый исходящий документ в системе.
func (s *OutgoingDocumentService) Register(
	nomenclatureID, documentTypeID string,
	recipientOrgName, addressee string,
	outgoingDate string,
	content string, pagesCount int,
	senderSignatory, senderExecutor string,
) (*dto.OutgoingDocument, error) {
	if err := requireClerkDocumentRole(s.auth); err != nil {
		return nil, err
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

	res, err := s.repo.Create(models.CreateOutgoingDocRequest{
		NomenclatureID:  nomID,
		DocumentTypeID:  docTypeID,
		RecipientOrgID:  recipientOrg.ID,
		CreatedBy:       createdBy,
		OutgoingNumber:  outgoingNumber,
		OutgoingDate:    outDate,
		Content:         content,
		PagesCount:      pagesCount,
		SenderSignatory: senderSignatory,
		SenderExecutor:  senderExecutor,
		Addressee:       addressee,
	})
	if err == nil {
		s.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
			DocumentID:   res.ID,
			DocumentType: "outgoing",
			UserID:       createdBy,
			Action:       "CREATE",
			Details:      fmt.Sprintf("Документ зарегистрирован. Рег. номер: %s", outgoingNumber),
		})
	}
	return dto.MapOutgoingDocument(res), err
}

// Update обновляет данные существующего исходящего документа.
func (s *OutgoingDocumentService) Update(
	id, documentTypeID string,
	recipientOrgName, addressee string,
	outgoingDate string,
	content string, pagesCount int,
	senderSignatory, senderExecutor string,
) (*dto.OutgoingDocument, error) {
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

	recipientOrg, err := s.refRepo.FindOrCreateOrganization(recipientOrgName)
	if err != nil {
		return nil, fmt.Errorf("ошибка организации получателя: %w", err)
	}

	outDate, err := time.Parse("2006-01-02", outgoingDate)
	if err != nil {
		return nil, models.NewBadRequest("неверный формат даты исходящего документа")
	}

	res, err := s.repo.Update(models.UpdateOutgoingDocRequest{
		ID:              uid,
		DocumentTypeID:  docTypeID,
		RecipientOrgID:  recipientOrg.ID,
		OutgoingDate:    outDate,
		Content:         content,
		PagesCount:      pagesCount,
		SenderSignatory: senderSignatory,
		SenderExecutor:  senderExecutor,
		Addressee:       addressee,
	})
	if err == nil {
		currentUserID, _ := s.auth.GetCurrentUserUUID()
		s.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
			DocumentID:   uid,
			DocumentType: "outgoing",
			UserID:       currentUserID,
			Action:       "UPDATE",
			Details:      "Документ отредактирован",
		})
	}
	return dto.MapOutgoingDocument(res), err
}

// GetList возвращает список исходящих документов с фильтрацией и ограничением видимости для исполнителей.
func (s *OutgoingDocumentService) GetList(filter models.OutgoingDocumentFilter) (*dto.PagedResult[dto.OutgoingDocument], error) {
	if err := requireDocumentDomainReadRole(s.auth); err != nil {
		return nil, err
	}
	// Defaults
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}

	if s.auth.HasActiveRole("executor") {
		allowedIDs, err := getExecutorAllowedNomenclatureIDs(s.auth, s.depRepo)
		if err != nil {
			return nil, err
		}
		filter.AllowedNomenclatureIDs = allowedIDs
		filter.AccessibleByUserID = s.auth.GetCurrentUserID()
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

// GetByID возвращает исходящий документ по его ID.
func (s *OutgoingDocumentService) GetByID(id string) (*dto.OutgoingDocument, error) {
	if err := requireDocumentDomainReadRole(s.auth); err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid ID: %w", err)
	}
	res, err := s.repo.GetByID(uid)
	if err != nil || res == nil {
		return dto.MapOutgoingDocument(res), err
	}
	if err := requireExecutorDocumentAccess(s.auth, s.depRepo, s.assignmentRepo, res.ID, "outgoing", res.NomenclatureID); err != nil {
		return nil, err
	}
	return dto.MapOutgoingDocument(res), nil
}

// Delete удаляет исходящий документ по его ID (доступно только делопроизводителям).
func (s *OutgoingDocumentService) Delete(id string) error {
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
			DocumentType: "outgoing",
			UserID:       currentUserID,
			Action:       "DELETE",
			Details:      "Документ удален",
		})
	}
	return err
}

// GetCount возвращает общее количество исходящих документов (для дашборда).
func (s *OutgoingDocumentService) GetCount() (int, error) {
	if err := requireDocumentDomainReadRole(s.auth); err != nil {
		return 0, err
	}
	return s.repo.GetCount()
}
