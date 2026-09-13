package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestDashboardService_GetActivity(t *testing.T) {

	makeService := func(t *testing.T, user *models.User, accessRoles ...string) (*DashboardService, *mocks.DashboardStore, *attachmentPrincipalStub) {
		t.Helper()

		repo := mocks.NewDashboardStore(t)
		userRepo := mocks.NewUserStore(t)
		auth := newAttachmentPrincipalStub(userRepo)
		accessStore := newRoleMappedDocumentAccessStore(accessRoles...)
		auth.SetAccessStore(accessStore)
		access := NewDocumentAccessService(auth, nil, nil, nil, accessStore, nil)

		auth.currentUserID = user.ID
		userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

		return NewDashboardService(repo, auth, access, nil), repo, auth
	}

	t.Run("executor sees personal expiring assignments", func(t *testing.T) {
		user := &models.User{
			ID:    uuid.New(),
			Login: "executor",

			IsDocumentParticipant: true,
			IsActive:              true,
		}
		svc, repo, _ := makeService(t, user, "executor")
		assignments := []models.Assignment{{ID: uuid.New(), Status: "new"}}

		repo.On("GetExpiringAssignments", mock.MatchedBy(func(filter models.DashboardAssignmentFilter) bool {
			return filter.Days == 3 && assert.ElementsMatch(t, []string{user.ID.String()}, filter.AccessibleByUserIDs)
		})).Return(assignments, nil).Once()

		activity, err := svc.GetActivity()
		require.NoError(t, err)
		require.Len(t, activity.ExpiringAssignments, 1)
	})

	t.Run("full document access keeps unfiltered dashboard scope", func(t *testing.T) {
		user := &models.User{
			ID:    uuid.New(),
			Login: "clerk",

			IsActive: true,
		}
		svc, repo, _ := makeService(t, user, "clerk")
		assignments := []models.Assignment{{ID: uuid.New(), Status: "in_progress"}}

		repo.On("GetExpiringAssignments", models.DashboardAssignmentFilter{Days: 7}).Return(assignments, nil).Once()

		activity, err := svc.GetActivity()
		require.NoError(t, err)
		require.Len(t, activity.ExpiringAssignments, 1)
	})

	t.Run("active substitution extends personal assignment scope", func(t *testing.T) {
		user := &models.User{
			ID:    uuid.New(),
			Login: "substitute",

			IsDocumentParticipant: true,
			IsActive:              true,
		}
		principalID := uuid.New()
		repo := mocks.NewDashboardStore(t)
		userRepo := mocks.NewUserStore(t)
		auth := newAttachmentPrincipalStub(userRepo)
		accessStore := newRoleMappedDocumentAccessStore("executor")
		auth.currentUserID = user.ID
		userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()
		access := NewDocumentAccessService(
			auth, nil, nil, nil, accessStore, nil,
			&userSubstitutionStoreStub{activePrincipals: []uuid.UUID{principalID}},
		)
		svc := NewDashboardService(repo, auth, access, nil)

		repo.On("GetExpiringAssignments", mock.MatchedBy(func(filter models.DashboardAssignmentFilter) bool {
			return filter.Days == 3 && assert.ElementsMatch(t,
				[]string{user.ID.String(), principalID.String()}, filter.AccessibleByUserIDs)
		})).Return([]models.Assignment{}, nil).Once()

		_, err := svc.GetActivity()
		require.NoError(t, err)
	})

	t.Run("mixed user keeps personal expiring assignments scope", func(t *testing.T) {
		user := &models.User{
			ID:    uuid.New(),
			Login: "mixed",

			IsDocumentParticipant: true,
			IsActive:              true,
		}
		svc, repo, _ := makeService(t, user, "clerk", "executor")
		assignments := []models.Assignment{{ID: uuid.New(), Status: "in_progress"}}

		repo.On("GetExpiringAssignments", mock.MatchedBy(func(filter models.DashboardAssignmentFilter) bool {
			return filter.Days == 3 && assert.ElementsMatch(t, []string{user.ID.String()}, filter.AccessibleByUserIDs)
		})).Return(assignments, nil).Once()

		activity, err := svc.GetActivity()
		require.NoError(t, err)
		require.Len(t, activity.ExpiringAssignments, 1)
	})

	t.Run("admin has no operational activity", func(t *testing.T) {
		user := &models.User{
			ID:    uuid.New(),
			Login: "admin",

			IsActive:          true,
			SystemPermissions: []string{models.SystemPermissionAdmin},
		}
		svc, _, _ := makeService(t, user, models.SystemPermissionAdmin)

		activity, err := svc.GetActivity()
		require.NoError(t, err)
		assert.Empty(t, activity.ExpiringAssignments)
	})

	t.Run("partial document access is passed to repository scope", func(t *testing.T) {
		user := &models.User{ID: uuid.New(), Login: "limited", IsActive: true}
		repo := mocks.NewDashboardStore(t)
		userRepo := mocks.NewUserStore(t)
		auth := newAttachmentPrincipalStub(userRepo)
		accessStore := &kindActionDocumentAccessStore{allowed: map[models.DocumentKind]map[string]bool{
			models.DocumentKindIncomingLetter: {"read": true},
		}}
		auth.currentUserID = user.ID
		userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()
		access := NewDocumentAccessService(auth, nil, nil, nil, accessStore, nil)
		svc := NewDashboardService(repo, auth, access, nil)

		repo.On("GetExpiringAssignments", mock.MatchedBy(func(filter models.DashboardAssignmentFilter) bool {
			return filter.Days == 7 &&
				assert.Equal(t, []string{string(models.DocumentKindIncomingLetter)}, filter.AllowedDocumentKinds) &&
				assert.Equal(t, []string{user.ID.String()}, filter.AccessibleByUserIDs)
		})).Return([]models.Assignment{}, nil).Once()

		_, err := svc.GetActivity()
		require.NoError(t, err)
	})

	t.Run("not authenticated", func(t *testing.T) {
		repo := mocks.NewDashboardStore(t)
		userRepo := mocks.NewUserStore(t)
		auth := newAttachmentPrincipalStub(userRepo)
		svc := NewDashboardService(repo, auth, nil, nil)

		activity, err := svc.GetActivity()
		require.ErrorIs(t, err, models.ErrUnauthorized)
		require.Nil(t, activity)
	})
}
