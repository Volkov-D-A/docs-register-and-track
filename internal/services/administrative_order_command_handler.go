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

// AdministrativeOrderRegisterRequest описывает команду регистрации приказа.
type AdministrativeOrderRegisterRequest struct {
	NomenclatureID          string   `json:"nomenclatureId"`
	IdempotencyKey          string   `json:"idempotencyKey"`
	OrderDate               string   `json:"orderDate"`
	Title                   string   `json:"title"`
	ExecutionController     string   `json:"executionController"`
	ExecutionDeadline       string   `json:"executionDeadline"`
	IsActive                bool     `json:"isActive"`
	CancelledAt             string   `json:"cancelledAt"`
	AcknowledgmentFullNames []string `json:"acknowledgmentFullNames"`
	RegistrationNumber      string   `json:"registrationNumber"`
}

// AdministrativeOrderUpdateRequest описывает команду обновления приказа.
type AdministrativeOrderUpdateRequest struct {
	ID                      string   `json:"id"`
	OrderDate               string   `json:"orderDate"`
	Title                   string   `json:"title"`
	ExecutionController     string   `json:"executionController"`
	ExecutionDeadline       string   `json:"executionDeadline"`
	IsActive                bool     `json:"isActive"`
	CancelledAt             string   `json:"cancelledAt"`
	AcknowledgmentFullNames []string `json:"acknowledgmentFullNames"`
}

// AdministrativeOrderCommandHandler инкапсулирует write-операции по приказам.
type AdministrativeOrderCommandHandler struct {
	repo    AdministrativeOrderDocStore
	nomRepo NomenclatureStore
	auth    *AuthService
	journal *JournalService
	access  *DocumentAccessService
}

// NewAdministrativeOrderCommandHandler создает handler команд приказов.
func NewAdministrativeOrderCommandHandler(
	repo AdministrativeOrderDocStore,
	nomRepo NomenclatureStore,
	auth *AuthService,
	journal *JournalService,
	access *DocumentAccessService,
) *AdministrativeOrderCommandHandler {
	return &AdministrativeOrderCommandHandler{
		repo:    repo,
		nomRepo: nomRepo,
		auth:    auth,
		journal: journal,
		access:  access,
	}
}

// Kind возвращает системный вид документа.
func (h *AdministrativeOrderCommandHandler) Kind() models.DocumentKind {
	return models.DocumentKindAdministrativeOrder
}

// Register регистрирует приказ.
func (h *AdministrativeOrderCommandHandler) Register(req AdministrativeOrderRegisterRequest) (*dto.AdministrativeOrderDocument, error) {
	if err := h.access.RequireCreate(models.DocumentKindAdministrativeOrder); err != nil {
		return nil, err
	}

	nomID, err := uuid.Parse(req.NomenclatureID)
	if err != nil {
		return nil, models.NewBadRequest("неверный ID номенклатуры")
	}
	idempotencyKey, err := uuid.Parse(req.IdempotencyKey)
	if err != nil || idempotencyKey == uuid.Nil {
		return nil, models.NewBadRequest("неверный ключ идемпотентности")
	}

	orderDate, err := time.Parse("2006-01-02", req.OrderDate)
	if err != nil {
		return nil, models.NewBadRequest("неверный формат даты приказа")
	}
	deadline, err := parseOptionalDate(req.ExecutionDeadline, "неверный формат срока выполнения")
	if err != nil {
		return nil, err
	}
	executionController := strings.TrimSpace(req.ExecutionController)
	if executionController == "" {
		return nil, models.NewBadRequest("укажите контроль за выполнением")
	}
	cancelledAt, err := parseOptionalDateTime(req.CancelledAt, "неверный формат даты отмены")
	if err != nil {
		return nil, err
	}
	if err := validateOrderActivity(req.IsActive, cancelledAt); err != nil {
		return nil, err
	}

	orderNumber := strings.TrimSpace(req.RegistrationNumber)
	createdBy, err := h.auth.GetCurrentUserUUID()
	if err != nil {
		return nil, ErrNotAuthenticated
	}

	res, err := h.repo.Create(models.CreateAdministrativeOrderDocRequest{
		NomenclatureID:          nomID,
		IdempotencyKey:          idempotencyKey,
		CreatedBy:               createdBy,
		OrderNumber:             orderNumber,
		OrderDate:               orderDate,
		Title:                   strings.TrimSpace(req.Title),
		ExecutionController:     executionController,
		ExecutionDeadline:       deadline,
		IsActive:                req.IsActive,
		CancelledAt:             cancelledAt,
		AcknowledgmentFullNames: normalizeFullNames(req.AcknowledgmentFullNames),
	})
	if err == nil {
		h.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
			DocumentID: res.ID,
			UserID:     createdBy,
			Action:     "CREATE",
			Details:    fmt.Sprintf("Приказ зарегистрирован. Рег. номер: %s", orderNumber),
		})
	}
	return dto.MapAdministrativeOrderDocument(res), err
}

// RegisterDocument реализует общий command-интерфейс по виду документа.
func (h *AdministrativeOrderCommandHandler) RegisterDocument(req any) (any, error) {
	typedReq, ok := req.(AdministrativeOrderRegisterRequest)
	if !ok {
		return nil, fmt.Errorf("invalid register request for kind %s", h.Kind())
	}
	return h.Register(typedReq)
}

// Update обновляет приказ.
func (h *AdministrativeOrderCommandHandler) Update(req AdministrativeOrderUpdateRequest) (*dto.AdministrativeOrderDocument, error) {
	uid, err := uuid.Parse(req.ID)
	if err != nil {
		return nil, fmt.Errorf("неверный ID документа: %w", err)
	}
	if err := h.access.RequireDocumentAction(uid, "update"); err != nil {
		return nil, err
	}

	orderDate, err := time.Parse("2006-01-02", req.OrderDate)
	if err != nil {
		return nil, models.NewBadRequest("неверный формат даты приказа")
	}
	deadline, err := parseOptionalDate(req.ExecutionDeadline, "неверный формат срока выполнения")
	if err != nil {
		return nil, err
	}
	executionController := strings.TrimSpace(req.ExecutionController)
	if executionController == "" {
		return nil, models.NewBadRequest("укажите контроль за выполнением")
	}
	cancelledAt, err := parseOptionalDateTime(req.CancelledAt, "неверный формат даты отмены")
	if err != nil {
		return nil, err
	}
	if err := validateOrderActivity(req.IsActive, cancelledAt); err != nil {
		return nil, err
	}

	res, err := h.repo.Update(models.UpdateAdministrativeOrderDocRequest{
		ID:                      uid,
		OrderDate:               orderDate,
		Title:                   strings.TrimSpace(req.Title),
		ExecutionController:     executionController,
		ExecutionDeadline:       deadline,
		IsActive:                req.IsActive,
		CancelledAt:             cancelledAt,
		AcknowledgmentFullNames: normalizeFullNames(req.AcknowledgmentFullNames),
	})
	if err == nil {
		currentUserID, _ := h.auth.GetCurrentUserUUID()
		h.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
			DocumentID: uid,
			UserID:     currentUserID,
			Action:     "UPDATE",
			Details:    "Приказ отредактирован",
		})
	}
	return dto.MapAdministrativeOrderDocument(res), err
}

// UpdateDocument реализует общий command-интерфейс по виду документа.
func (h *AdministrativeOrderCommandHandler) UpdateDocument(req any) (any, error) {
	typedReq, ok := req.(AdministrativeOrderUpdateRequest)
	if !ok {
		return nil, fmt.Errorf("invalid update request for kind %s", h.Kind())
	}
	return h.Update(typedReq)
}

func parseOptionalDate(value string, message string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, models.NewBadRequest(message)
	}
	return &parsed, nil
}

func parseOptionalDateTime(value string, message string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return &parsed, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, models.NewBadRequest(message)
	}
	return &parsed, nil
}

func validateOrderActivity(isActive bool, cancelledAt *time.Time) error {
	if isActive && cancelledAt != nil {
		return models.NewBadRequest("для действующего приказа дата отмены должна быть пустой")
	}
	if !isActive && cancelledAt == nil {
		return models.NewBadRequest("для недействующего приказа укажите дату отмены")
	}
	return nil
}

func normalizeFullNames(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}
