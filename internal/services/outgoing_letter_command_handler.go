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

// OutgoingLetterRegisterRequest описывает команду регистрации исходящего письма.
type OutgoingLetterRegisterRequest struct {
	NomenclatureID     string
	DocumentTypeID     string
	RecipientOrgName   string
	Addressee          string
	OutgoingDate       string
	Content            string
	PagesCount         int
	SenderSignatory    string
	SenderExecutor     string
	RegistrationNumber string
}

// OutgoingLetterUpdateRequest описывает команду обновления исходящего письма.
type OutgoingLetterUpdateRequest struct {
	ID               string
	DocumentTypeID   string
	RecipientOrgName string
	Addressee        string
	OutgoingDate     string
	Content          string
	PagesCount       int
	SenderSignatory  string
	SenderExecutor   string
}

// OutgoingLetterCommandHandler инкапсулирует write-операции по исходящим письмам.
type OutgoingLetterCommandHandler struct {
	repo    OutgoingDocStore
	refRepo ReferenceStore
	nomRepo NomenclatureStore
	auth    *AuthService
	journal *JournalService
	access  *DocumentAccessService
}

// Kind возвращает системный вид документа, поддерживаемый handler'ом.
func (h *OutgoingLetterCommandHandler) Kind() models.DocumentKind {
	return models.DocumentKindOutgoingLetter
}

// NewOutgoingLetterCommandHandler создает handler команд исходящих писем.
func NewOutgoingLetterCommandHandler(
	repo OutgoingDocStore,
	refRepo ReferenceStore,
	nomRepo NomenclatureStore,
	auth *AuthService,
	journal *JournalService,
	access *DocumentAccessService,
) *OutgoingLetterCommandHandler {
	return &OutgoingLetterCommandHandler{
		repo:    repo,
		refRepo: refRepo,
		nomRepo: nomRepo,
		auth:    auth,
		journal: journal,
		access:  access,
	}
}

// Register регистрирует исходящее письмо.
func (h *OutgoingLetterCommandHandler) Register(req OutgoingLetterRegisterRequest) (*dto.OutgoingDocument, error) {
	if err := h.access.RequireCreate(models.DocumentKindOutgoingLetter); err != nil {
		return nil, err
	}

	nomID, err := uuid.Parse(req.NomenclatureID)
	if err != nil {
		return nil, fmt.Errorf("неверный ID номенклатуры: %w", err)
	}
	docTypeID, err := uuid.Parse(req.DocumentTypeID)
	if err != nil {
		return nil, fmt.Errorf("неверный ID типа документа: %w", err)
	}

	recipientOrg, err := h.refRepo.FindOrCreateOrganization(req.RecipientOrgName)
	if err != nil {
		return nil, fmt.Errorf("ошибка организации получателя: %w", err)
	}

	nom, err := h.nomRepo.GetByID(nomID)
	if err != nil || nom == nil {
		return nil, fmt.Errorf("ошибка получения номенклатуры: %w", err)
	}

	outgoingNumber := strings.TrimSpace(req.RegistrationNumber)
	if nom.NumberingMode == NumberingModeManualOnly {
		if outgoingNumber == "" {
			return nil, models.NewBadRequest("укажите регистрационный номер вручную")
		}
	} else {
		nextNum, index, separator, numberingMode, err := h.nomRepo.GetNextNumber(nomID)
		if err != nil {
			return nil, fmt.Errorf("ошибка автонумерации: %w", err)
		}
		outgoingNumber = formatDocumentNumber(index, separator, numberingMode, nextNum)
	}

	outDate, err := time.Parse("2006-01-02", req.OutgoingDate)
	if err != nil {
		return nil, fmt.Errorf("неверный формат даты исходящего документа: %w", err)
	}

	createdByStr := h.auth.GetCurrentUserID()
	createdBy, err := uuid.Parse(createdByStr)
	if err != nil {
		return nil, ErrNotAuthenticated
	}

	res, err := h.repo.Create(models.CreateOutgoingDocRequest{
		NomenclatureID:  nomID,
		DocumentTypeID:  docTypeID,
		RecipientOrgID:  recipientOrg.ID,
		CreatedBy:       createdBy,
		OutgoingNumber:  outgoingNumber,
		OutgoingDate:    outDate,
		Content:         req.Content,
		PagesCount:      req.PagesCount,
		SenderSignatory: req.SenderSignatory,
		SenderExecutor:  req.SenderExecutor,
		Addressee:       req.Addressee,
	})
	if err == nil {
		h.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
			DocumentID: res.ID,
			UserID:     createdBy,
			Action:     "CREATE",
			Details:    fmt.Sprintf("Документ зарегистрирован. Рег. номер: %s", outgoingNumber),
		})
	}
	return dto.MapOutgoingDocument(res), err
}

// RegisterDocument реализует общий command-интерфейс по виду документа.
func (h *OutgoingLetterCommandHandler) RegisterDocument(req any) (any, error) {
	typedReq, ok := req.(OutgoingLetterRegisterRequest)
	if !ok {
		return nil, fmt.Errorf("invalid register request for kind %s", h.Kind())
	}

	return h.Register(typedReq)
}

// Update обновляет исходящее письмо.
func (h *OutgoingLetterCommandHandler) Update(req OutgoingLetterUpdateRequest) (*dto.OutgoingDocument, error) {
	uid, err := uuid.Parse(req.ID)
	if err != nil {
		return nil, fmt.Errorf("неверный ID документа: %w", err)
	}
	if err := h.access.RequireDocumentAction(uid, "update"); err != nil {
		return nil, err
	}
	docTypeID, err := uuid.Parse(req.DocumentTypeID)
	if err != nil {
		return nil, fmt.Errorf("неверный ID типа документа: %w", err)
	}

	recipientOrg, err := h.refRepo.FindOrCreateOrganization(req.RecipientOrgName)
	if err != nil {
		return nil, fmt.Errorf("ошибка организации получателя: %w", err)
	}

	outDate, err := time.Parse("2006-01-02", req.OutgoingDate)
	if err != nil {
		return nil, models.NewBadRequest("неверный формат даты исходящего документа")
	}

	res, err := h.repo.Update(models.UpdateOutgoingDocRequest{
		ID:              uid,
		DocumentTypeID:  docTypeID,
		RecipientOrgID:  recipientOrg.ID,
		OutgoingDate:    outDate,
		Content:         req.Content,
		PagesCount:      req.PagesCount,
		SenderSignatory: req.SenderSignatory,
		SenderExecutor:  req.SenderExecutor,
		Addressee:       req.Addressee,
	})
	if err == nil {
		currentUserID, _ := h.auth.GetCurrentUserUUID()
		h.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
			DocumentID: uid,
			UserID:     currentUserID,
			Action:     "UPDATE",
			Details:    "Документ отредактирован",
		})
	}
	return dto.MapOutgoingDocument(res), err
}

// UpdateDocument реализует общий command-интерфейс по виду документа.
func (h *OutgoingLetterCommandHandler) UpdateDocument(req any) (any, error) {
	typedReq, ok := req.(OutgoingLetterUpdateRequest)
	if !ok {
		return nil, fmt.Errorf("invalid update request for kind %s", h.Kind())
	}

	return h.Update(typedReq)
}
