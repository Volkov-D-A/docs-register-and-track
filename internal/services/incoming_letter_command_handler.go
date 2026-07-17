package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// IncomingLetterRegisterRequest описывает команду регистрации входящего письма.
type IncomingLetterRegisterRequest struct {
	NomenclatureID      string                               `json:"nomenclatureId"`
	IdempotencyKey      string                               `json:"idempotencyKey"`
	DocumentTypeID      string                               `json:"documentTypeId"`
	IncomingDate        string                               `json:"incomingDate"`
	Correspondents      []IncomingLetterCorrespondentRequest `json:"correspondents"`
	Content             string                               `json:"content"`
	PagesCount          int                                  `json:"pagesCount"`
	SenderSignatory     string                               `json:"senderSignatory"`
	Resolution          string                               `json:"resolution"`
	ResolutionAuthor    string                               `json:"resolutionAuthor"`
	ResolutionExecutors string                               `json:"resolutionExecutors"`
	RegistrationNumber  string                               `json:"registrationNumber"`
	AdminNumberOverride *AdminNumberOverrideRequest          `json:"adminNumberOverride"`
}

// IncomingLetterUpdateRequest описывает команду обновления входящего письма.
type IncomingLetterUpdateRequest struct {
	ID                  string                               `json:"id"`
	DocumentTypeID      string                               `json:"documentTypeId"`
	Correspondents      []IncomingLetterCorrespondentRequest `json:"correspondents"`
	Content             string                               `json:"content"`
	PagesCount          int                                  `json:"pagesCount"`
	SenderSignatory     string                               `json:"senderSignatory"`
	Resolution          string                               `json:"resolution"`
	ResolutionAuthor    string                               `json:"resolutionAuthor"`
	ResolutionExecutors string                               `json:"resolutionExecutors"`
}

// IncomingLetterCorrespondentRequest описывает один набор реквизитов корреспондента.
type IncomingLetterCorrespondentRequest struct {
	RegistrationNumber string `json:"registrationNumber"`
	RegistrationDate   string `json:"registrationDate"`
	CorrespondentName  string `json:"correspondentName"`
}

// IncomingLetterCommandHandler инкапсулирует write-операции по входящим письмам.
type IncomingLetterCommandHandler struct {
	repo    IncomingDocStore
	nomRepo NomenclatureStore
	refRepo ReferenceStore
	auth    *AuthService
	journal *JournalService
	access  *DocumentAccessService
}
type incomingDocumentOutboxStore interface {
	UpdateWithOutbox(models.UpdateIncomingDocRequest, []models.OutboxEvent) (*models.IncomingDocument, error)
}
type incomingDocumentJournalStore interface {
	CreateWithJournal(models.CreateIncomingDocRequest, string, string) (*models.IncomingDocument, error)
}

// Kind возвращает системный вид документа, поддерживаемый handler'ом.
func (h *IncomingLetterCommandHandler) Kind() models.DocumentKind {
	return models.DocumentKindIncomingLetter
}

// NewIncomingLetterCommandHandler создает handler команд входящих писем.
func NewIncomingLetterCommandHandler(
	repo IncomingDocStore,
	nomRepo NomenclatureStore,
	refRepo ReferenceStore,
	auth *AuthService,
	journal *JournalService,
	access *DocumentAccessService,
) *IncomingLetterCommandHandler {
	return &IncomingLetterCommandHandler{
		repo:    repo,
		nomRepo: nomRepo,
		refRepo: refRepo,
		auth:    auth,
		journal: journal,
		access:  access,
	}
}

// Register регистрирует входящее письмо.
func (h *IncomingLetterCommandHandler) Register(req IncomingLetterRegisterRequest) (*dto.IncomingDocument, error) {
	adminOverride, err := buildAdminNumberOverride(req.AdminNumberOverride)
	if err != nil {
		return nil, err
	}
	if adminOverride != nil {
		if err := h.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
			return nil, err
		}
	} else {
		if err := h.access.RequireCreate(models.DocumentKindIncomingLetter); err != nil {
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

	if req.ResolutionExecutors != "" {
		for _, name := range strings.Split(req.ResolutionExecutors, "; ") {
			name = strings.TrimSpace(name)
			if name != "" {
				h.refRepo.FindOrCreateResolutionExecutor(name)
			}
		}
	}

	incomingNumberStr := strings.TrimSpace(req.RegistrationNumber)
	incDate, err := time.Parse("2006-01-02", req.IncomingDate)
	if err != nil {
		return nil, models.NewBadRequest("неверный формат даты поступления")
	}
	correspondents, err := h.buildCorrespondents(req.Correspondents)
	if err != nil {
		return nil, err
	}

	var resPtr *string
	if req.Resolution != "" {
		resPtr = &req.Resolution
	}
	var resAuthorPtr *string
	if req.ResolutionAuthor != "" {
		resAuthorPtr = &req.ResolutionAuthor
	}
	var resExecutorsPtr *string
	if req.ResolutionExecutors != "" {
		resExecutorsPtr = &req.ResolutionExecutors
	}

	createdBy, err := h.auth.GetCurrentUserUUID()
	if err != nil {
		return nil, err
	}

	createReq := models.CreateIncomingDocRequest{
		NomenclatureID:      nomID,
		IdempotencyKey:      idempotencyKey,
		AdminNumberOverride: adminOverride,
		DocumentTypeID:      docTypeID,
		CreatedBy:           createdBy,
		IncomingNumber:      incomingNumberStr,
		IncomingDate:        incDate,
		Correspondents:      correspondents,
		Content:             req.Content,
		PagesCount:          req.PagesCount,
		SenderSignatory:     req.SenderSignatory,
		Resolution:          resPtr,
		ResolutionAuthor:    resAuthorPtr,
		ResolutionExecutors: resExecutorsPtr,
	}
	store, ok := h.repo.(incomingDocumentJournalStore)
	if !ok {
		return nil, fmt.Errorf("incoming document store must support atomic journal operations")
	}
	res, err := store.CreateWithJournal(createReq, "CREATE", "Документ зарегистрирован. Рег. номер: %s")
	return dto.MapIncomingDocument(res), err
}

// RegisterDocument реализует общий command-интерфейс по виду документа.
func (h *IncomingLetterCommandHandler) RegisterDocument(req any) (any, error) {
	typedReq, ok := req.(IncomingLetterRegisterRequest)
	if !ok {
		return nil, fmt.Errorf("invalid register request for kind %s", h.Kind())
	}

	return h.Register(typedReq)
}

// CreateAdminDraft создает черновик входящего письма с административно заданным номером.
func (h *IncomingLetterCommandHandler) CreateAdminDraft(req AdminDraftCreateRequest) (any, error) {
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
	org, err := h.refRepo.FindOrCreateOrganization(adminDraftPlaceholder)
	if err != nil {
		return nil, fmt.Errorf("ошибка организации корреспондента: %w", err)
	}

	createReq := models.CreateIncomingDocRequest{
		NomenclatureID:      nomID,
		IdempotencyKey:      uuid.New(),
		AdminNumberOverride: adminOverride,
		DocumentTypeID:      models.DocumentTypeLetter,
		CreatedBy:           createdBy,
		IncomingDate:        registrationDate,
		Correspondents: []models.DocumentCorrespondentRegistration{{
			RegistrationNumber: adminDraftPlaceholder,
			RegistrationDate:   registrationDate,
			CorrespondentOrgID: org.ID,
			Position:           1,
		}},
		Content:         adminDraftPlaceholder,
		PagesCount:      1,
		SenderSignatory: adminDraftPlaceholder,
	}
	store, ok := h.repo.(incomingDocumentJournalStore)
	if !ok {
		return nil, fmt.Errorf("incoming document store must support atomic journal operations")
	}
	res, err := store.CreateWithJournal(createReq, "ADMIN_DRAFT_CREATE", "Создан административный черновик. Рег. номер: %s")
	return dto.MapIncomingDocument(res), err
}

// Update обновляет входящее письмо.
func (h *IncomingLetterCommandHandler) Update(req IncomingLetterUpdateRequest) (*dto.IncomingDocument, error) {
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

	if req.ResolutionExecutors != "" {
		for _, name := range strings.Split(req.ResolutionExecutors, "; ") {
			name = strings.TrimSpace(name)
			if name != "" {
				h.refRepo.FindOrCreateResolutionExecutor(name)
			}
		}
	}

	correspondents, err := h.buildCorrespondents(req.Correspondents)
	if err != nil {
		return nil, err
	}

	var resPtr *string
	if req.Resolution != "" {
		resPtr = &req.Resolution
	}
	var resAuthorPtr *string
	if req.ResolutionAuthor != "" {
		resAuthorPtr = &req.ResolutionAuthor
	}
	var resExecutorsPtr *string
	if req.ResolutionExecutors != "" {
		resExecutorsPtr = &req.ResolutionExecutors
	}

	updateReq := models.UpdateIncomingDocRequest{
		ID:                  uid,
		DocumentTypeID:      docTypeID,
		Correspondents:      correspondents,
		Content:             req.Content,
		PagesCount:          req.PagesCount,
		SenderSignatory:     req.SenderSignatory,
		Resolution:          resPtr,
		ResolutionAuthor:    resAuthorPtr,
		ResolutionExecutors: resExecutorsPtr,
	}
	store, ok := h.repo.(incomingDocumentOutboxStore)
	if !ok {
		return nil, fmt.Errorf("incoming document store must support atomic outbox operations")
	}
	currentUserID, _ := h.auth.GetCurrentUserUUID()
	event, buildErr := NewJournalOutboxEvent("incoming:"+uid.String()+":update:"+uuid.NewString(), models.CreateJournalEntryRequest{DocumentID: uid, UserID: currentUserID, Action: "UPDATE", Details: "Документ отредактирован"})
	if buildErr != nil {
		return nil, buildErr
	}
	res, err := store.UpdateWithOutbox(updateReq, []models.OutboxEvent{event})
	return dto.MapIncomingDocument(res), err
}

func (h *IncomingLetterCommandHandler) buildCorrespondents(reqs []IncomingLetterCorrespondentRequest) ([]models.DocumentCorrespondentRegistration, error) {
	if len(reqs) == 0 {
		return nil, models.NewBadRequest("укажите реквизиты корреспондента")
	}

	result := make([]models.DocumentCorrespondentRegistration, 0, len(reqs))
	for i, req := range reqs {
		number := strings.TrimSpace(req.RegistrationNumber)
		dateStr := strings.TrimSpace(req.RegistrationDate)
		name := strings.TrimSpace(req.CorrespondentName)

		if number == "" && dateStr == "" && name == "" {
			continue
		}
		if number == "" {
			return nil, models.NewBadRequest("укажите регистрационный номер корреспондента")
		}
		if dateStr == "" {
			return nil, models.NewBadRequest("укажите дату регистрации корреспондента")
		}
		if name == "" {
			return nil, models.NewBadRequest("укажите корреспондента")
		}

		regDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, models.NewBadRequest("неверный формат даты регистрации корреспондента")
		}

		org, err := h.refRepo.FindOrCreateOrganization(name)
		if err != nil {
			return nil, fmt.Errorf("ошибка корреспондента: %w", err)
		}

		result = append(result, models.DocumentCorrespondentRegistration{
			RegistrationNumber: number,
			RegistrationDate:   regDate,
			CorrespondentOrgID: org.ID,
			CorrespondentName:  org.Name,
			Position:           i + 1,
		})
	}

	if len(result) == 0 {
		return nil, models.NewBadRequest("укажите реквизиты корреспондента")
	}

	return result, nil
}

// UpdateDocument реализует общий command-интерфейс по виду документа.
func (h *IncomingLetterCommandHandler) UpdateDocument(req any) (any, error) {
	typedReq, ok := req.(IncomingLetterUpdateRequest)
	if !ok {
		return nil, fmt.Errorf("invalid update request for kind %s", h.Kind())
	}

	return h.Update(typedReq)
}
