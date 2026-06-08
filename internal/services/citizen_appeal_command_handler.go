package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

const (
	AppealTypeSuggestion  = "предложение"
	AppealTypeApplication = "заявление"
	AppealTypeComplaint   = "жалоба"
)

var allowedAppealTypes = map[string]struct{}{
	AppealTypeSuggestion:  {},
	AppealTypeApplication: {},
	AppealTypeComplaint:   {},
}

// CitizenAppealRegisterRequest описывает команду регистрации обращения граждан.
type CitizenAppealRegisterRequest struct {
	NomenclatureID       string                              `json:"nomenclatureId"`
	IdempotencyKey       string                              `json:"idempotencyKey"`
	RegistrationDate     string                              `json:"registrationDate"`
	AppealDate           string                              `json:"appealDate"`
	ApplicantFullName    string                              `json:"applicantFullName"`
	RegistrationAddress  string                              `json:"registrationAddress"`
	AppealType           string                              `json:"appealType"`
	ApplicantCategory    string                              `json:"applicantCategory"`
	AppealPagesCount     int                                 `json:"appealPagesCount"`
	AttachmentPagesCount int                                 `json:"attachmentPagesCount"`
	HasEnvelope          bool                                `json:"hasEnvelope"`
	ReceivedFromPOS      bool                                `json:"receivedFromPos"`
	Content              string                              `json:"content"`
	RegistrationNumber   string                              `json:"registrationNumber"`
	AdminNumberOverride  *AdminNumberOverrideRequest         `json:"adminNumberOverride"`
	Correspondents       []CitizenAppealCorrespondentRequest `json:"correspondents"`
	Resolutions          []CitizenAppealResolutionRequest    `json:"resolutions"`
}

// CitizenAppealUpdateRequest описывает команду обновления обращения граждан.
type CitizenAppealUpdateRequest struct {
	ID                   string                              `json:"id"`
	RegistrationNumber   string                              `json:"registrationNumber"`
	RegistrationDate     string                              `json:"registrationDate"`
	AppealDate           string                              `json:"appealDate"`
	ApplicantFullName    string                              `json:"applicantFullName"`
	RegistrationAddress  string                              `json:"registrationAddress"`
	AppealType           string                              `json:"appealType"`
	ApplicantCategory    string                              `json:"applicantCategory"`
	AppealPagesCount     int                                 `json:"appealPagesCount"`
	AttachmentPagesCount int                                 `json:"attachmentPagesCount"`
	HasEnvelope          bool                                `json:"hasEnvelope"`
	ReceivedFromPOS      bool                                `json:"receivedFromPos"`
	Content              string                              `json:"content"`
	Correspondents       []CitizenAppealCorrespondentRequest `json:"correspondents"`
	Resolutions          []CitizenAppealResolutionRequest    `json:"resolutions"`
}

// CitizenAppealCorrespondentRequest описывает один набор внешних регистрационных реквизитов.
type CitizenAppealCorrespondentRequest struct {
	RegistrationNumber string `json:"registrationNumber"`
	RegistrationDate   string `json:"registrationDate"`
	CorrespondentName  string `json:"correspondentName"`
}

// CitizenAppealResolutionRequest описывает один набор резолюции.
type CitizenAppealResolutionRequest struct {
	Resolution          string `json:"resolution"`
	ResolutionAuthor    string `json:"resolutionAuthor"`
	ResolutionExecutors string `json:"resolutionExecutors"`
}

// CitizenAppealCommandHandler инкапсулирует write-операции по обращениям граждан.
type CitizenAppealCommandHandler struct {
	repo    CitizenAppealDocStore
	nomRepo NomenclatureStore
	refRepo ReferenceStore
	auth    *AuthService
	journal *JournalService
	access  *DocumentAccessService
}

// NewCitizenAppealCommandHandler создает handler команд обращений граждан.
func NewCitizenAppealCommandHandler(
	repo CitizenAppealDocStore,
	nomRepo NomenclatureStore,
	refRepo ReferenceStore,
	auth *AuthService,
	journal *JournalService,
	access *DocumentAccessService,
) *CitizenAppealCommandHandler {
	return &CitizenAppealCommandHandler{
		repo:    repo,
		nomRepo: nomRepo,
		refRepo: refRepo,
		auth:    auth,
		journal: journal,
		access:  access,
	}
}

// Kind возвращает системный вид документа, поддерживаемый handler'ом.
func (h *CitizenAppealCommandHandler) Kind() models.DocumentKind {
	return models.DocumentKindCitizenAppeal
}

// Register регистрирует обращения граждан.
func (h *CitizenAppealCommandHandler) Register(req CitizenAppealRegisterRequest) (*dto.CitizenAppealDocument, error) {
	adminOverride, err := buildAdminNumberOverride(req.AdminNumberOverride)
	if err != nil {
		return nil, err
	}
	if adminOverride != nil {
		if err := h.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
			return nil, err
		}
	} else {
		if err := h.access.RequireCreate(models.DocumentKindCitizenAppeal); err != nil {
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

	registrationNumber := strings.TrimSpace(req.RegistrationNumber)
	registrationDate, err := parseCommandDate(req.RegistrationDate, "даты регистрации")
	if err != nil {
		return nil, err
	}
	appealDate, err := parseCommandDate(req.AppealDate, "даты обращения")
	if err != nil {
		return nil, err
	}
	appealType, err := normalizeAppealType(req.AppealType)
	if err != nil {
		return nil, err
	}
	if err := validateCitizenAppealRequiredFields(req.ApplicantFullName, req.RegistrationAddress, req.ApplicantCategory, req.Content); err != nil {
		return nil, err
	}
	if err := validateCitizenAppealPages(req.AppealPagesCount, req.AttachmentPagesCount); err != nil {
		return nil, err
	}

	correspondents, err := h.buildCorrespondents(req.Correspondents)
	if err != nil {
		return nil, err
	}
	resolutions, err := h.buildResolutions(req.Resolutions)
	if err != nil {
		return nil, err
	}

	createdBy, err := h.auth.GetCurrentUserUUID()
	if err != nil {
		return nil, ErrNotAuthenticated
	}

	res, err := h.repo.Create(models.CreateCitizenAppealDocRequest{
		NomenclatureID:       nomID,
		IdempotencyKey:       idempotencyKey,
		AdminNumberOverride:  adminOverride,
		CreatedBy:            createdBy,
		RegistrationNumber:   registrationNumber,
		RegistrationDate:     registrationDate,
		AppealDate:           appealDate,
		Content:              strings.TrimSpace(req.Content),
		ApplicantFullName:    strings.TrimSpace(req.ApplicantFullName),
		RegistrationAddress:  strings.TrimSpace(req.RegistrationAddress),
		AppealType:           appealType,
		ApplicantCategory:    strings.TrimSpace(req.ApplicantCategory),
		AppealPagesCount:     req.AppealPagesCount,
		AttachmentPagesCount: req.AttachmentPagesCount,
		HasEnvelope:          req.HasEnvelope,
		ReceivedFromPOS:      req.ReceivedFromPOS,
		Correspondents:       correspondents,
		Resolutions:          resolutions,
	})
	if err == nil {
		h.journal.LogAction(nil, models.CreateJournalEntryRequest{
			DocumentID: res.ID,
			UserID:     createdBy,
			Action:     "CREATE",
			Details:    fmt.Sprintf("Обращение зарегистрировано. Номер: %s", registrationNumber),
		})
	}
	return dto.MapCitizenAppealDocument(res), err
}

// RegisterDocument реализует общий command-интерфейс по виду документа.
func (h *CitizenAppealCommandHandler) RegisterDocument(req any) (any, error) {
	typedReq, ok := req.(CitizenAppealRegisterRequest)
	if !ok {
		return nil, fmt.Errorf("invalid register request for kind %s", h.Kind())
	}

	return h.Register(typedReq)
}

// CreateAdminDraft создает черновик обращения граждан с административно заданным номером.
func (h *CitizenAppealCommandHandler) CreateAdminDraft(req AdminDraftCreateRequest) (any, error) {
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

	res, err := h.repo.Create(models.CreateCitizenAppealDocRequest{
		NomenclatureID:       nomID,
		IdempotencyKey:       uuid.New(),
		AdminNumberOverride:  adminOverride,
		CreatedBy:            createdBy,
		RegistrationDate:     registrationDate,
		AppealDate:           registrationDate,
		Content:              adminDraftPlaceholder,
		ApplicantFullName:    adminDraftPlaceholder,
		RegistrationAddress:  adminDraftPlaceholder,
		AppealType:           AppealTypeApplication,
		ApplicantCategory:    adminDraftPlaceholder,
		AppealPagesCount:     1,
		AttachmentPagesCount: 0,
	})
	if err == nil {
		h.journal.LogAction(nil, models.CreateJournalEntryRequest{
			DocumentID: res.ID,
			UserID:     createdBy,
			Action:     "ADMIN_DRAFT_CREATE",
			Details:    fmt.Sprintf("Создан административный черновик. Рег. номер: %s", res.RegistrationNumber),
		})
	}
	return dto.MapCitizenAppealDocument(res), err
}

// Update обновляет обращения граждан.
func (h *CitizenAppealCommandHandler) Update(req CitizenAppealUpdateRequest) (*dto.CitizenAppealDocument, error) {
	uid, err := uuid.Parse(req.ID)
	if err != nil {
		return nil, models.NewBadRequestWrapped("неверный ID документа", err)
	}
	if err := h.access.RequireDocumentAction(uid, "update"); err != nil {
		return nil, err
	}

	registrationNumber := strings.TrimSpace(req.RegistrationNumber)
	if registrationNumber == "" {
		return nil, models.NewBadRequest("укажите номер документа")
	}
	registrationDate, err := parseCommandDate(req.RegistrationDate, "даты регистрации")
	if err != nil {
		return nil, err
	}
	appealDate, err := parseCommandDate(req.AppealDate, "даты обращения")
	if err != nil {
		return nil, err
	}
	appealType, err := normalizeAppealType(req.AppealType)
	if err != nil {
		return nil, err
	}
	if err := validateCitizenAppealRequiredFields(req.ApplicantFullName, req.RegistrationAddress, req.ApplicantCategory, req.Content); err != nil {
		return nil, err
	}
	if err := validateCitizenAppealPages(req.AppealPagesCount, req.AttachmentPagesCount); err != nil {
		return nil, err
	}

	correspondents, err := h.buildCorrespondents(req.Correspondents)
	if err != nil {
		return nil, err
	}
	resolutions, err := h.buildResolutions(req.Resolutions)
	if err != nil {
		return nil, err
	}

	res, err := h.repo.Update(models.UpdateCitizenAppealDocRequest{
		ID:                   uid,
		RegistrationNumber:   registrationNumber,
		RegistrationDate:     registrationDate,
		AppealDate:           appealDate,
		Content:              strings.TrimSpace(req.Content),
		ApplicantFullName:    strings.TrimSpace(req.ApplicantFullName),
		RegistrationAddress:  strings.TrimSpace(req.RegistrationAddress),
		AppealType:           appealType,
		ApplicantCategory:    strings.TrimSpace(req.ApplicantCategory),
		AppealPagesCount:     req.AppealPagesCount,
		AttachmentPagesCount: req.AttachmentPagesCount,
		HasEnvelope:          req.HasEnvelope,
		ReceivedFromPOS:      req.ReceivedFromPOS,
		Correspondents:       correspondents,
		Resolutions:          resolutions,
	})
	if err == nil {
		currentUserID, _ := h.auth.GetCurrentUserUUID()
		h.journal.LogAction(nil, models.CreateJournalEntryRequest{
			DocumentID: uid,
			UserID:     currentUserID,
			Action:     "UPDATE",
			Details:    "Обращение отредактировано",
		})
	}
	return dto.MapCitizenAppealDocument(res), err
}

// UpdateDocument реализует общий command-интерфейс по виду документа.
func (h *CitizenAppealCommandHandler) UpdateDocument(req any) (any, error) {
	typedReq, ok := req.(CitizenAppealUpdateRequest)
	if !ok {
		return nil, fmt.Errorf("invalid update request for kind %s", h.Kind())
	}

	return h.Update(typedReq)
}

func (h *CitizenAppealCommandHandler) buildCorrespondents(reqs []CitizenAppealCorrespondentRequest) ([]models.DocumentCorrespondentRegistration, error) {
	if len(reqs) == 0 {
		return nil, nil
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

	return result, nil
}

func (h *CitizenAppealCommandHandler) buildResolutions(reqs []CitizenAppealResolutionRequest) ([]models.DocumentResolution, error) {
	if len(reqs) == 0 {
		return nil, nil
	}

	result := make([]models.DocumentResolution, 0, len(reqs))
	for i, req := range reqs {
		resolution := strings.TrimSpace(req.Resolution)
		author := strings.TrimSpace(req.ResolutionAuthor)
		executors := strings.TrimSpace(req.ResolutionExecutors)

		if resolution == "" && author == "" && executors == "" {
			continue
		}
		if resolution == "" {
			return nil, models.NewBadRequest("укажите текст резолюции")
		}
		if err := h.ensureResolutionExecutors(executors); err != nil {
			return nil, err
		}

		result = append(result, models.DocumentResolution{
			Resolution:          stringPtr(resolution),
			ResolutionAuthor:    optionalStringPtr(author),
			ResolutionExecutors: optionalStringPtr(executors),
			Position:            i + 1,
		})
	}

	return result, nil
}

func (h *CitizenAppealCommandHandler) ensureResolutionExecutors(executors string) error {
	if executors == "" {
		return nil
	}
	for _, name := range strings.Split(executors, "; ") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, err := h.refRepo.FindOrCreateResolutionExecutor(name); err != nil {
			return fmt.Errorf("ошибка исполнителя резолюции: %w", err)
		}
	}
	return nil
}

func parseCommandDate(value, fieldName string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, models.NewBadRequestWrapped(fmt.Sprintf("неверный формат %s", fieldName), err)
	}
	return parsed, nil
}

func normalizeAppealType(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if _, ok := allowedAppealTypes[normalized]; !ok {
		return "", models.NewBadRequest("неверный вид обращения")
	}
	return normalized, nil
}

func validateCitizenAppealRequiredFields(applicantFullName, registrationAddress, applicantCategory, content string) error {
	if strings.TrimSpace(applicantFullName) == "" {
		return models.NewBadRequest("укажите ФИО обратившегося")
	}
	if strings.TrimSpace(registrationAddress) == "" {
		return models.NewBadRequest("укажите адрес регистрации")
	}
	if strings.TrimSpace(applicantCategory) == "" {
		return models.NewBadRequest("укажите категорию обратившегося")
	}
	if strings.TrimSpace(content) == "" {
		return models.NewBadRequest("укажите краткое содержание обращения")
	}
	return nil
}

func validateCitizenAppealPages(appealPagesCount, attachmentPagesCount int) error {
	if appealPagesCount < 1 {
		return models.NewBadRequest("укажите количество листов обращения")
	}
	if attachmentPagesCount < 0 {
		return models.NewBadRequest("количество листов приложения не может быть отрицательным")
	}
	return nil
}

func stringPtr(value string) *string {
	return &value
}

func optionalStringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
