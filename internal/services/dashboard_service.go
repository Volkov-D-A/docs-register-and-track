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

type dashboardAssignmentScope string

const (
	dashboardAssignmentScopeNone     dashboardAssignmentScope = "none"
	dashboardAssignmentScopeGlobal   dashboardAssignmentScope = "global"
	dashboardAssignmentScopePersonal dashboardAssignmentScope = "personal"
)

// NewDashboardService создает новый экземпляр DashboardService.
func NewDashboardService(repo DashboardStore, auth *AuthService, access *DocumentAccessService) *DashboardService {
	return &DashboardService{repo: repo, auth: auth, access: access}
}

func (s *DashboardService) determineDashboardAssignmentScope(user *dto.User) dashboardAssignmentScope {
	if user == nil {
		return dashboardAssignmentScopePersonal
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
	switch {
	case hasSystemPermission(models.SystemPermissionAdmin) && !hasClerkFlow && !user.IsDocumentParticipant:
		return dashboardAssignmentScopeNone
	case user.IsDocumentParticipant:
		return dashboardAssignmentScopePersonal
	case hasClerkFlow:
		return dashboardAssignmentScopeGlobal
	case hasSystemPermission(models.SystemPermissionAdmin):
		return dashboardAssignmentScopeNone
	default:
		return dashboardAssignmentScopePersonal
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

	switch s.determineDashboardAssignmentScope(user) {
	case dashboardAssignmentScopeNone:
		return activity, nil
	case dashboardAssignmentScopeGlobal:
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
