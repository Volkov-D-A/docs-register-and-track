package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// OutgoingLetterRegisterRequest описывает команду регистрации исходящего письма.
type OutgoingLetterRegisterRequest struct {
	NomenclatureID      string                      `json:"nomenclatureId"`
	IdempotencyKey      string                      `json:"idempotencyKey"`
	DocumentTypeID      string                      `json:"documentTypeId"`
	RecipientOrgName    string                      `json:"recipientOrgName"`
	Addressee           string                      `json:"addressee"`
	OutgoingDate        string                      `json:"outgoingDate"`
	Content             string                      `json:"content"`
	PagesCount          int                         `json:"pagesCount"`
	SenderSignatory     string                      `json:"senderSignatory"`
	SenderExecutor      string                      `json:"senderExecutor"`
	RegistrationNumber  string                      `json:"registrationNumber"`
	AdminNumberOverride *AdminNumberOverrideRequest `json:"adminNumberOverride"`
}

// OutgoingLetterUpdateRequest описывает команду обновления исходящего письма.
type OutgoingLetterUpdateRequest struct {
	ID               string `json:"id"`
	DocumentTypeID   string `json:"documentTypeId"`
	RecipientOrgName string `json:"recipientOrgName"`
	Addressee        string `json:"addressee"`
	OutgoingDate     string `json:"outgoingDate"`
	Content          string `json:"content"`
	PagesCount       int    `json:"pagesCount"`
	SenderSignatory  string `json:"senderSignatory"`
	SenderExecutor   string `json:"senderExecutor"`
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
	adminOverride, err := buildAdminNumberOverride(req.AdminNumberOverride)
	if err != nil {
		return nil, err
	}
	if adminOverride != nil {
		if err := h.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
			return nil, err
		}
	} else {
		if err := h.access.RequireCreate(models.DocumentKindOutgoingLetter); err != nil {
			return nil, err
		}
	}

	nomID, err := uuid.Parse(req.NomenclatureID)
	if err != nil {
		return nil, models.NewBadRequest("неверный ID номенклатуры")
	}
	idempotencyKey, err := uuid.Parse(req.IdempotencyKey)
	if err != nil || idempotencyKey == uuid.Nil {
		return nil, models.NewBadRequest("неверный ключ идемпотентности")
	}
	docTypeID := models.NormalizeDocumentType(req.DocumentTypeID)
	if !models.IsAllowedDocumentType(docTypeID) {
		return nil, models.NewBadRequest("неверный тип документа")
	}

	recipientOrg, err := h.refRepo.FindOrCreateOrganization(req.RecipientOrgName)
	if err != nil {
		return nil, fmt.Errorf("ошибка организации получателя: %w", err)
	}

	outgoingNumber := strings.TrimSpace(req.RegistrationNumber)
	outDate, err := time.Parse("2006-01-02", req.OutgoingDate)
	if err != nil {
		return nil, models.NewBadRequest("неверный формат даты исходящего документа")
	}

	createdBy, err := h.auth.GetCurrentUserUUID()
	if err != nil {
		return nil, err
	}

	res, err := h.repo.Create(models.CreateOutgoingDocRequest{
		NomenclatureID:      nomID,
		IdempotencyKey:      idempotencyKey,
		AdminNumberOverride: adminOverride,
		DocumentTypeID:      docTypeID,
		RecipientOrgID:      recipientOrg.ID,
		CreatedBy:           createdBy,
		OutgoingNumber:      outgoingNumber,
		OutgoingDate:        outDate,
		Content:             req.Content,
		PagesCount:          req.PagesCount,
		SenderSignatory:     req.SenderSignatory,
		SenderExecutor:      req.SenderExecutor,
		Addressee:           req.Addressee,
	})
	if err == nil {
		h.journal.LogAction(nil, models.CreateJournalEntryRequest{
			DocumentID: res.ID,
			UserID:     createdBy,
			Action:     "CREATE",
			Details:    fmt.Sprintf("Документ зарегистрирован. Рег. номер: %s", res.OutgoingNumber),
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

// CreateAdminDraft создает черновик исходящего письма с административно заданным номером.
func (h *OutgoingLetterCommandHandler) CreateAdminDraft(req AdminDraftCreateRequest) (any, error) {
	if err := h.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return nil, err
	}
	nomID, err := uuid.Parse(req.NomenclatureID)
	if err != nil {
		return nil, models.NewBadRequest("неверный ID номенклатуры")
	}
	registrationDate, err := parseCommandDate(req.RegistrationDate, "даты регистрации")
	if err != nil {
		return nil, err
	}
	adminOverride, err := buildAdminNumberOverride(req.AdminNumberOverride)
	if err != nil {
		return nil, err
	}
	if adminOverride == nil {
		return nil, models.NewBadRequest("укажите административный номер")
	}
	createdBy, err := h.auth.GetCurrentUserUUID()
	if err != nil {
		return nil, ErrNotAuthenticated
	}
	recipientOrg, err := h.refRepo.FindOrCreateOrganization(adminDraftPlaceholder)
	if err != nil {
		return nil, fmt.Errorf("ошибка организации получателя: %w", err)
	}

	res, err := h.repo.Create(models.CreateOutgoingDocRequest{
		NomenclatureID:      nomID,
		IdempotencyKey:      uuid.New(),
		AdminNumberOverride: adminOverride,
		DocumentTypeID:      models.DocumentTypeLetter,
		RecipientOrgID:      recipientOrg.ID,
		CreatedBy:           createdBy,
		OutgoingDate:        registrationDate,
		Content:             adminDraftPlaceholder,
		PagesCount:          1,
		SenderSignatory:     adminDraftPlaceholder,
		SenderExecutor:      adminDraftPlaceholder,
		Addressee:           adminDraftPlaceholder,
	})
	if err == nil {
		h.journal.LogAction(nil, models.CreateJournalEntryRequest{
			DocumentID: res.ID,
			UserID:     createdBy,
			Action:     "ADMIN_DRAFT_CREATE",
			Details:    fmt.Sprintf("Создан административный черновик. Рег. номер: %s", res.OutgoingNumber),
		})
	}
	return dto.MapOutgoingDocument(res), err
}

// Update обновляет исходящее письмо.
func (h *OutgoingLetterCommandHandler) Update(req OutgoingLetterUpdateRequest) (*dto.OutgoingDocument, error) {
	uid, err := uuid.Parse(req.ID)
	if err != nil {
		return nil, models.NewBadRequestWrapped("неверный ID документа", err)
	}
	if err := h.access.RequireDocumentAction(uid, "update"); err != nil {
		return nil, err
	}
	docTypeID := models.NormalizeDocumentType(req.DocumentTypeID)
	if !models.IsAllowedDocumentType(docTypeID) {
		return nil, models.NewBadRequest("неверный тип документа")
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
		h.journal.LogAction(nil, models.CreateJournalEntryRequest{
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
