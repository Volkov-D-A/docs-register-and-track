package services

import (
	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// DashboardService предоставляет данные текущей активности для дашборда.
type DashboardService struct {
	repo   DashboardStore
	auth   *AuthService
	access *DocumentAccessService
}

// NewDashboardService создает новый экземпляр DashboardService.
func NewDashboardService(repo DashboardStore, auth *AuthService, access *DocumentAccessService) *DashboardService {
	return &DashboardService{repo: repo, auth: auth, access: access}
}

func (s *DashboardService) determineDashboardProfile(user *dto.User) string {
	if user == nil {
		return "executor"
	}

	hasSystemPermission := func(expected string) bool {
		for _, permission := range user.SystemPermissions {
			if permission == expected {
				return true
			}
		}
		return false
	}

	canCreate := false
	canRead := false
	if s.access != nil {
		canCreate, _ = s.access.HasAnyDocumentAction("create")
		canRead, _ = s.access.HasAnyDocumentAction("read")
	}

	hasClerkFlow := canCreate || canRead
	hasExecutorFlow := user.IsDocumentParticipant

	switch {
	case hasSystemPermission(models.SystemPermissionAdmin) && !hasClerkFlow && !hasExecutorFlow:
		return "admin"
	case hasClerkFlow && hasExecutorFlow:
		return "mixed"
	case hasClerkFlow:
		return "clerk"
	case hasExecutorFlow:
		return "executor"
	case hasSystemPermission(models.SystemPermissionAdmin):
		return "admin"
	default:
		return "executor"
	}
}

// GetActivity возвращает оперативные данные для главного экрана.
func (s *DashboardService) GetActivity() (*dto.DashboardActivity, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}

	user, err := s.auth.GetCurrentUser()
	if err != nil {
		return nil, err
	}

	activity := &dto.DashboardActivity{
		ExpiringAssignments: []dto.Assignment{},
	}

	switch s.determineDashboardProfile(user) {
	case "admin":
		return activity, nil
	case "clerk":
		assignments, err := s.repo.GetExpiringAssignments(nil, 7)
		if err != nil {
			return nil, err
		}
		if assignments != nil {
			activity.ExpiringAssignments = dto.MapAssignments(assignments)
		}
	default:
		userID, err := uuid.Parse(user.ID)
		if err != nil {
			return nil, err
		}
		assignments, err := s.repo.GetExpiringAssignments(&userID, 3)
		if err != nil {
			return nil, err
		}
		if assignments != nil {
			activity.ExpiringAssignments = dto.MapAssignments(assignments)
		}
	}

	return activity, nil
}
