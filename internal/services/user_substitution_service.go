package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// UserSubstitutionService управляет замещающими исполнителями пользователя.
type UserSubstitutionService struct {
	repo         UserSubstitutionStore
	userRepo     UserStore
	auth         *AuthService
	auditService *AdminAuditLogService
}

// NewUserSubstitutionService создает сервис замещений.
func NewUserSubstitutionService(repo UserSubstitutionStore, userRepo UserStore, auth *AuthService, auditService *AdminAuditLogService) *UserSubstitutionService {
	return &UserSubstitutionService{repo: repo, userRepo: userRepo, auth: auth, auditService: auditService}
}

func parseOptionalSubstitutionDate(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, models.NewBadRequestWrapped("неверный формат даты замещения", err)
	}
	return &t, nil
}

func (s *UserSubstitutionService) validateSubstitution(principal *models.User, req models.UpdateUserSubstitutionRequest) (*uuid.UUID, *time.Time, *time.Time, error) {
	startsAt, err := parseOptionalSubstitutionDate(req.StartsAt)
	if err != nil {
		return nil, nil, nil, err
	}
	endsAt, err := parseOptionalSubstitutionDate(req.EndsAt)
	if err != nil {
		return nil, nil, nil, err
	}
	if startsAt != nil && endsAt != nil && startsAt.After(*endsAt) {
		return nil, nil, nil, models.NewBadRequest("дата начала замещения не может быть позже даты окончания")
	}
	if req.SubstituteUserID == "" {
		return nil, startsAt, endsAt, nil
	}

	substituteID, err := parseUUID(req.SubstituteUserID)
	if err != nil {
		return nil, nil, nil, err
	}
	if principal == nil {
		return nil, nil, nil, models.NewNotFound("пользователь не найден")
	}
	if substituteID == principal.ID {
		return nil, nil, nil, models.NewBadRequest("пользователь не может замещать самого себя")
	}

	substitute, err := s.userRepo.GetByID(substituteID)
	if err != nil {
		return nil, nil, nil, err
	}
	if substitute == nil {
		return nil, nil, nil, models.NewNotFound("замещающий пользователь не найден")
	}
	if !substitute.IsActive {
		return nil, nil, nil, models.NewBadRequest("замещающий должен быть активным пользователем")
	}
	if principal.DepartmentID == nil {
		return nil, nil, nil, models.NewBadRequest("для замещаемого должно быть указано подразделение")
	}
	if substitute.DepartmentID == nil || *substitute.DepartmentID != *principal.DepartmentID {
		return nil, nil, nil, models.NewBadRequest("замещающий должен быть из подразделения замещаемого")
	}

	return &substituteID, startsAt, endsAt, nil
}

func (s *UserSubstitutionService) saveForPrincipal(principalID uuid.UUID, req models.UpdateUserSubstitutionRequest, requireAdmin bool) (*dto.UserSubstitution, error) {
	if requireAdmin {
		if err := s.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
			return nil, err
		}
	} else if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}

	principal, err := s.userRepo.GetByID(principalID)
	if err != nil {
		return nil, err
	}
	if principal == nil {
		return nil, models.NewNotFound("пользователь не найден")
	}
	if req.SubstituteUserID == "" {
		res, err := s.repo.ReplaceForPrincipal(principalID, nil, nil, nil, false, nil)
		if err != nil {
			return nil, err
		}
		return dto.MapUserSubstitution(res), nil
	}
	if !principal.IsDocumentParticipant {
		return nil, models.NewBadRequest("замещение доступно только участникам документооборота")
	}

	substituteID, startsAt, endsAt, err := s.validateSubstitution(principal, req)
	if err != nil {
		return nil, err
	}

	actorID, actorName := s.auth.GetCurrentAuditInfo()
	var createdBy *uuid.UUID
	if actorID != uuid.Nil {
		createdBy = &actorID
	}
	res, err := s.repo.ReplaceForPrincipal(principalID, substituteID, startsAt, endsAt, req.IsActive, createdBy)
	if err != nil {
		return nil, err
	}

	if requireAdmin && s.auditService != nil {
		details := fmt.Sprintf("Обновлено замещение пользователя «%s»", principal.FullName)
		s.auditService.LogAction(actorID, actorName, "USER_SUBSTITUTION_UPDATE", details)
	}
	return dto.MapUserSubstitution(res), nil
}

// GetMySubstitution возвращает настройку замещения текущего пользователя.
func (s *UserSubstitutionService) GetMySubstitution() (*dto.UserSubstitution, error) {
	userID, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return nil, err
	}
	res, err := s.repo.GetByPrincipalID(userID)
	return dto.MapUserSubstitution(res), err
}

// UpdateMySubstitution обновляет замещающего текущего пользователя.
func (s *UserSubstitutionService) UpdateMySubstitution(req models.UpdateUserSubstitutionRequest) (*dto.UserSubstitution, error) {
	userID, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return nil, err
	}
	return s.saveForPrincipal(userID, req, false)
}

// GetUserSubstitution возвращает настройку замещения выбранного пользователя. Доступно администратору.
func (s *UserSubstitutionService) GetUserSubstitution(userID string) (*dto.UserSubstitution, error) {
	if err := s.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return nil, err
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}
	res, err := s.repo.GetByPrincipalID(uid)
	return dto.MapUserSubstitution(res), err
}

// UpdateUserSubstitution обновляет замещение выбранного пользователя. Доступно администратору.
func (s *UserSubstitutionService) UpdateUserSubstitution(req models.UpdateUserSubstitutionRequest) (*dto.UserSubstitution, error) {
	principalID, err := parseUUID(req.PrincipalUserID)
	if err != nil {
		return nil, err
	}
	return s.saveForPrincipal(principalID, req, true)
}
