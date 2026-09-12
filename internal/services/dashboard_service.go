package services

import (
	"context"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/operations"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"
	serverservices "github.com/Volkov-D-A/docs-register-and-track/internal/server/services"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

// DashboardService предоставляет данные текущей активности для дашборда.
type DashboardService struct {
	repo    ports.DashboardStore
	auth    ports.DocumentAccessPrincipal
	access  *serverservices.DocumentAccessService
	metrics *observability.Registry
	server  serverclient.DashboardClient
}

func (s *DashboardService) SetOperationMetrics(metrics *observability.Registry) {
	s.metrics = metrics
}

// NewDashboardService создает новый экземпляр DashboardService.
func NewDashboardService(repo ports.DashboardStore, auth ports.DocumentAccessPrincipal, access *serverservices.DocumentAccessService) *DashboardService {
	return &DashboardService{repo: repo, auth: auth, access: access}
}

func NewDashboardServiceWithClient(client serverclient.DashboardClient) *DashboardService {
	return &DashboardService{server: client}
}

// GetActivity возвращает оперативные данные для главного экрана.
func (s *DashboardService) GetActivity() (*dto.DashboardActivity, error) {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return s.server.GetDashboardActivity(ctx)
	}
	return operations.Measure(s.metrics, "dashboard.get_activity", func() (*dto.DashboardActivity, error) {
		if err := s.auth.RequireAuthenticated(); err != nil {
			return nil, err
		}

		user, err := s.auth.GetCurrentUser()
		if err != nil {
			return nil, err
		}

		activity := &dto.DashboardActivity{
			ExpiringAssignments: []dto.Assignment{},
		}

		if s.access == nil {
			return activity, nil
		}

		readableKinds, err := s.access.GetDocumentKindsWithAction("read")
		if err != nil {
			return nil, err
		}
		if len(readableKinds) == 0 && !user.IsDocumentParticipant {
			return activity, nil
		}

		filter := models.DashboardAssignmentFilter{Days: 7}
		if user.IsDocumentParticipant {
			filter.Days = 3
			subjectIDs, err := s.access.GetCurrentUserAndSubstitutionSubjectIDs()
			if err != nil {
				return nil, err
			}
			filter.AccessibleByUserIDs = serverservices.UUIDStrings(subjectIDs)
		} else if len(readableKinds) < len(models.AllDocumentKindSpecs()) {
			filter.AllowedDocumentKinds = serverservices.DocumentKindCodes(readableKinds)
			subjectIDs, err := s.access.GetCurrentUserAndSubstitutionSubjectIDs()
			if err != nil {
				return nil, err
			}
			filter.AccessibleByUserIDs = serverservices.UUIDStrings(subjectIDs)
		}

		assignments, err := s.repo.GetExpiringAssignments(filter)
		if err != nil {
			return nil, err
		}
		if assignments != nil {
			activity.ExpiringAssignments = dto.MapAssignments(assignments)
		}

		return activity, nil
	})
}
