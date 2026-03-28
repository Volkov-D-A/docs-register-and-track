package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
)

// NomenclatureService предоставляет бизнес-логику для работы с номенклатурой дел.
type NomenclatureService struct {
	repo         NomenclatureStore
	auth         *AuthService
	auditService *AdminAuditLogService
}

// NewNomenclatureService создает новый экземпляр NomenclatureService.
func NewNomenclatureService(repo NomenclatureStore, auth *AuthService, auditService *AdminAuditLogService) *NomenclatureService {
	return &NomenclatureService{repo: repo, auth: auth, auditService: auditService}
}

// GetAll возвращает все дела номенклатуры за указанный год и по указанному направлению.
func (s *NomenclatureService) GetAll(year int, direction string) ([]dto.Nomenclature, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	res, err := s.repo.GetAll(year, direction)
	return dto.MapNomenclatures(res), err
}

// GetActiveForDirection возвращает активные дела для выбора при регистрации документов.
func (s *NomenclatureService) GetActiveForDirection(direction string) ([]dto.Nomenclature, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	year := time.Now().Year()
	res, err := s.repo.GetActiveByDirection(direction, year)
	return dto.MapNomenclatures(res), err
}

// Create создает новое дело номенклатуры (доступно только администраторам и делопроизводителям).
func (s *NomenclatureService) Create(name, index string, year int, direction string) (*dto.Nomenclature, error) {
	if err := s.auth.RequireAnyActiveRole("admin", "clerk"); err != nil {
		return nil, err
	}
	res, err := s.repo.Create(name, index, year, direction)
	if err != nil {
		return nil, err
	}

	userID, userName := s.auth.GetCurrentAuditInfo()
	s.auditService.LogAction(userID, userName, "NOMENCLATURE_CREATE", fmt.Sprintf("Создано дело «%s» (%s), год: %d", name, index, year))

	return dto.MapNomenclature(res), nil
}

// Update обновляет существующее дело номенклатуры.
func (s *NomenclatureService) Update(id string, name, index string, year int, direction string, isActive bool) (*dto.Nomenclature, error) {
	if err := s.auth.RequireAnyActiveRole("admin", "clerk"); err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid ID: %w", err)
	}
	res, err := s.repo.Update(uid, name, index, year, direction, isActive)
	if err != nil {
		return nil, err
	}

	userID, userName := s.auth.GetCurrentAuditInfo()
	s.auditService.LogAction(userID, userName, "NOMENCLATURE_UPDATE", fmt.Sprintf("Обновлено дело «%s» (%s)", name, index))

	return dto.MapNomenclature(res), nil
}

// Delete удаляет дело номенклатуры по его ID (доступно только администраторам).
func (s *NomenclatureService) Delete(id string) error {
	if err := s.auth.RequireActiveRole("admin"); err != nil {
		return err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}
	if err := s.repo.Delete(uid); err != nil {
		return err
	}

	userID, userName := s.auth.GetCurrentAuditInfo()
	s.auditService.LogAction(userID, userName, "NOMENCLATURE_DELETE", fmt.Sprintf("Удалено дело (ID: %s)", id))
	return nil
}
