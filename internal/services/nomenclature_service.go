package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// NomenclatureService предоставляет бизнес-логику для работы с номенклатурой дел.
type NomenclatureService struct {
	repo         NomenclatureStore
	auth         *AuthService
	auditService *AdminAuditLogService
}

type nomenclatureOutboxStore interface {
	CreateWithOutbox(string, string, int, string, string, string, int, []models.OutboxEvent) (*models.Nomenclature, error)
	UpdateWithOutbox(uuid.UUID, string, string, int, string, string, string, bool, []models.OutboxEvent) (*models.Nomenclature, error)
	DeleteWithOutbox(uuid.UUID, []models.OutboxEvent) error
}

var errNomenclatureOutboxStoreRequired = fmt.Errorf("nomenclature store must support atomic outbox operations")

// NewNomenclatureService создает новый экземпляр NomenclatureService.
func NewNomenclatureService(repo NomenclatureStore, auth *AuthService, auditService *AdminAuditLogService) *NomenclatureService {
	return &NomenclatureService{repo: repo, auth: auth, auditService: auditService}
}

// GetAll возвращает все дела номенклатуры за указанный год и по указанному виду документа.
func (s *NomenclatureService) GetAll(year int, kindCode string) ([]dto.Nomenclature, error) {
	if err := s.auth.RequireAuthenticated(); err != nil {
		return nil, err
	}
	res, err := s.repo.GetAll(year, kindCode)
	return dto.MapNomenclatures(res), err
}

// GetActiveForKind возвращает активные дела для выбора при регистрации документов.
func (s *NomenclatureService) GetActiveForKind(kindCode string) ([]dto.Nomenclature, error) {
	if err := s.auth.RequireAuthenticated(); err != nil {
		return nil, err
	}
	year := time.Now().Year()
	res, err := s.repo.GetActiveByKind(kindCode, year)
	return dto.MapNomenclatures(res), err
}

// Create создает новое дело номенклатуры (доступно только администраторам и делопроизводителям).
func (s *NomenclatureService) Create(name, index string, year int, kindCode, separator, numberingMode string, startNumber int) (*dto.Nomenclature, error) {
	if err := s.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return nil, err
	}
	if startNumber < 1 {
		startNumber = 1
	}

	userID, userName := s.auth.GetCurrentAuditInfo()
	var res *models.Nomenclature
	var err error
	store, ok := s.repo.(nomenclatureOutboxStore)
	if !ok {
		return nil, errNomenclatureOutboxStoreRequired
	}
	event, buildErr := NewAdminAuditOutboxEvent("nomenclature:"+uuid.NewString()+":create", models.CreateAdminAuditLogRequest{UserID: userID, UserName: userName, Action: "NOMENCLATURE_CREATE", Details: fmt.Sprintf("Создано дело «%s» (%s), вид: %s, год: %d, стартовый номер: %d", name, index, kindCode, year, startNumber)})
	if buildErr != nil {
		return nil, buildErr
	}
	res, err = store.CreateWithOutbox(name, index, year, kindCode, separator, numberingMode, startNumber, []models.OutboxEvent{event})
	if err != nil {
		return nil, err
	}

	return dto.MapNomenclature(res), nil
}

// Update обновляет существующее дело номенклатуры.
func (s *NomenclatureService) Update(id string, name, index string, year int, kindCode, separator, numberingMode string, isActive bool) (*dto.Nomenclature, error) {
	if err := s.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, models.NewBadRequestWrapped("неверный ID номенклатуры", err)
	}
	userID, userName := s.auth.GetCurrentAuditInfo()
	var res *models.Nomenclature
	store, ok := s.repo.(nomenclatureOutboxStore)
	if !ok {
		return nil, errNomenclatureOutboxStoreRequired
	}
	event, buildErr := NewAdminAuditOutboxEvent("nomenclature:"+uid.String()+":update:"+uuid.NewString(), models.CreateAdminAuditLogRequest{UserID: userID, UserName: userName, Action: "NOMENCLATURE_UPDATE", Details: fmt.Sprintf("Обновлено дело «%s» (%s)", name, index)})
	if buildErr != nil {
		return nil, buildErr
	}
	res, err = store.UpdateWithOutbox(uid, name, index, year, kindCode, separator, numberingMode, isActive, []models.OutboxEvent{event})
	if err != nil {
		return nil, err
	}

	return dto.MapNomenclature(res), nil
}

// Delete удаляет дело номенклатуры по его ID (доступно только администраторам).
func (s *NomenclatureService) Delete(id string) error {
	if err := s.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return models.NewBadRequestWrapped("неверный ID номенклатуры", err)
	}
	userID, userName := s.auth.GetCurrentAuditInfo()
	store, ok := s.repo.(nomenclatureOutboxStore)
	if !ok {
		return errNomenclatureOutboxStoreRequired
	}
	event, buildErr := NewAdminAuditOutboxEvent("nomenclature:"+uid.String()+":delete", models.CreateAdminAuditLogRequest{UserID: userID, UserName: userName, Action: "NOMENCLATURE_DELETE", Details: fmt.Sprintf("Удалено дело (ID: %s)", id)})
	if buildErr != nil {
		return buildErr
	}
	err = store.DeleteWithOutbox(uid, []models.OutboxEvent{event})
	if err != nil {
		return err
	}
	return nil
}
