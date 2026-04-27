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

// IncomingLetterRegisterRequest описывает команду регистрации входящего письма.
type IncomingLetterRegisterRequest struct {
	NomenclatureID      string
	DocumentTypeID      string
	IncomingDate        string
	Correspondents      []IncomingLetterCorrespondentRequest
	Content             string
	PagesCount          int
	SenderSignatory     string
	Resolution          string
	ResolutionAuthor    string
	ResolutionExecutors string
	RegistrationNumber  string
}

// IncomingLetterUpdateRequest описывает команду обновления входящего письма.
type IncomingLetterUpdateRequest struct {
	ID                  string
	DocumentTypeID      string
	Correspondents      []IncomingLetterCorrespondentRequest
	Content             string
	PagesCount          int
	SenderSignatory     string
	Resolution          string
	ResolutionAuthor    string
	ResolutionExecutors string
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
	if err := h.access.RequireCreate(models.DocumentKindIncomingLetter); err != nil {
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

	if req.ResolutionExecutors != "" {
		for _, name := range strings.Split(req.ResolutionExecutors, "; ") {
			name = strings.TrimSpace(name)
			if name != "" {
				h.refRepo.FindOrCreateResolutionExecutor(name)
			}
		}
	}

	nom, err := h.nomRepo.GetByID(nomID)
	if err != nil || nom == nil {
		return nil, fmt.Errorf("ошибка получения номенклатуры: %w", err)
	}

	incomingNumberStr := strings.TrimSpace(req.RegistrationNumber)
	if nom.NumberingMode == NumberingModeManualOnly {
		if incomingNumberStr == "" {
			return nil, models.NewBadRequest("укажите регистрационный номер вручную")
		}
	} else {
		nextNum, index, separator, numberingMode, err := h.nomRepo.GetNextNumber(nomID)
		if err != nil {
			return nil, fmt.Errorf("ошибка автонумерации: %w", err)
		}
		incomingNumberStr = formatDocumentNumber(index, separator, numberingMode, nextNum)
	}

	incDate, err := time.Parse("2006-01-02", req.IncomingDate)
	if err != nil {
		return nil, fmt.Errorf("неверный формат даты поступления: %w", err)
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

	createdByStr := h.auth.GetCurrentUserID()
	createdBy, err := uuid.Parse(createdByStr)
	if err != nil {
		return nil, ErrNotAuthenticated
	}

	res, err := h.repo.Create(models.CreateIncomingDocRequest{
		NomenclatureID:      nomID,
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
	})
	if err == nil {
		h.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
			DocumentID: res.ID,
			UserID:     createdBy,
			Action:     "CREATE",
			Details:    fmt.Sprintf("Документ зарегистрирован. Рег. номер: %s", incomingNumberStr),
		})
	}
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

// Update обновляет входящее письмо.
func (h *IncomingLetterCommandHandler) Update(req IncomingLetterUpdateRequest) (*dto.IncomingDocument, error) {
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

	res, err := h.repo.Update(models.UpdateIncomingDocRequest{
		ID:                  uid,
		DocumentTypeID:      docTypeID,
		Correspondents:      correspondents,
		Content:             req.Content,
		PagesCount:          req.PagesCount,
		SenderSignatory:     req.SenderSignatory,
		Resolution:          resPtr,
		ResolutionAuthor:    resAuthorPtr,
		ResolutionExecutors: resExecutorsPtr,
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
