package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
)

func TestDashboardService_GetActivity(t *testing.T) {
	password := "CorrectPassw0rd!"
	hash, _ := security.HashPassword(password)

	makeService := func(t *testing.T, user *models.User, accessRoles ...string) (*DashboardService, *mocks.DashboardStore, *AuthService) {
		t.Helper()

		repo := mocks.NewDashboardStore(t)
		userRepo := mocks.NewUserStore(t)
		auth := NewAuthService(nil, userRepo)
		accessStore := newRoleMappedDocumentAccessStore(accessRoles...)
		auth.SetAccessStore(accessStore)
		access := NewDocumentAccessService(auth, nil, nil, nil, accessStore, nil, nil, nil)

		userRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
		_, err := auth.Login(user.Login, password)
		require.NoError(t, err)
		userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

		return NewDashboardService(repo, auth, access), repo, auth
	}

	t.Run("executor sees personal expiring assignments", func(t *testing.T) {
		user := &models.User{
			ID:                    uuid.New(),
			Login:                 "executor",
			PasswordHash:          hash,
			IsDocumentParticipant: true,
			IsActive:              true,
		}
		svc, repo, auth := makeService(t, user, "executor")
		assignments := []models.Assignment{{ID: uuid.New(), Status: "new"}}

		repo.On("GetExpiringAssignments", mock.MatchedBy(func(filter models.DashboardAssignmentFilter) bool {
			return filter.Days == 3 && assert.ElementsMatch(t, []string{user.ID.String()}, filter.AccessibleByUserIDs)
		})).Return(assignments, nil).Once()

		activity, err := svc.GetActivity()
		require.NoError(t, err)
		require.Len(t, activity.ExpiringAssignments, 1)

		require.NoError(t, auth.Logout())
	})

	t.Run("full document access keeps unfiltered dashboard scope", func(t *testing.T) {
		user := &models.User{
			ID:           uuid.New(),
			Login:        "clerk",
			PasswordHash: hash,
			IsActive:     true,
		}
		svc, repo, auth := makeService(t, user, "clerk")
		assignments := []models.Assignment{{ID: uuid.New(), Status: "in_progress"}}

		repo.On("GetExpiringAssignments", models.DashboardAssignmentFilter{Days: 7}).Return(assignments, nil).Once()

		activity, err := svc.GetActivity()
		require.NoError(t, err)
		require.Len(t, activity.ExpiringAssignments, 1)

		require.NoError(t, auth.Logout())
	})

	t.Run("active substitution extends personal assignment scope", func(t *testing.T) {
		user := &models.User{
			ID:                    uuid.New(),
			Login:                 "substitute",
			PasswordHash:          hash,
			IsDocumentParticipant: true,
			IsActive:              true,
		}
		principalID := uuid.New()
		repo := mocks.NewDashboardStore(t)
		userRepo := mocks.NewUserStore(t)
		auth := NewAuthService(nil, userRepo)
		accessStore := newRoleMappedDocumentAccessStore("executor")
		userRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
		_, err := auth.Login(user.Login, password)
		require.NoError(t, err)
		userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()
		access := NewDocumentAccessService(
			auth, nil, nil, nil, accessStore, nil, nil, nil,
			&userSubstitutionStoreStub{activePrincipals: []uuid.UUID{principalID}},
		)
		svc := NewDashboardService(repo, auth, access)

		repo.On("GetExpiringAssignments", mock.MatchedBy(func(filter models.DashboardAssignmentFilter) bool {
			return filter.Days == 3 && assert.ElementsMatch(t,
				[]string{user.ID.String(), principalID.String()}, filter.AccessibleByUserIDs)
		})).Return([]models.Assignment{}, nil).Once()

		_, err = svc.GetActivity()
		require.NoError(t, err)
	})

	t.Run("mixed user keeps personal expiring assignments scope", func(t *testing.T) {
		user := &models.User{
			ID:                    uuid.New(),
			Login:                 "mixed",
			PasswordHash:          hash,
			IsDocumentParticipant: true,
			IsActive:              true,
		}
		svc, repo, auth := makeService(t, user, "clerk", "executor")
		assignments := []models.Assignment{{ID: uuid.New(), Status: "in_progress"}}

		repo.On("GetExpiringAssignments", mock.MatchedBy(func(filter models.DashboardAssignmentFilter) bool {
			return filter.Days == 3 && assert.ElementsMatch(t, []string{user.ID.String()}, filter.AccessibleByUserIDs)
		})).Return(assignments, nil).Once()

		activity, err := svc.GetActivity()
		require.NoError(t, err)
		require.Len(t, activity.ExpiringAssignments, 1)

		require.NoError(t, auth.Logout())
	})

	t.Run("admin has no operational activity", func(t *testing.T) {
		user := &models.User{
			ID:                uuid.New(),
			Login:             "admin",
			PasswordHash:      hash,
			IsActive:          true,
			SystemPermissions: []string{models.SystemPermissionAdmin},
		}
		svc, _, auth := makeService(t, user, models.SystemPermissionAdmin)

		activity, err := svc.GetActivity()
		require.NoError(t, err)
		assert.Empty(t, activity.ExpiringAssignments)

		require.NoError(t, auth.Logout())
	})

	t.Run("partial document access is passed to repository scope", func(t *testing.T) {
		user := &models.User{ID: uuid.New(), Login: "limited", PasswordHash: hash, IsActive: true}
		repo := mocks.NewDashboardStore(t)
		userRepo := mocks.NewUserStore(t)
		auth := NewAuthService(nil, userRepo)
		accessStore := &kindActionDocumentAccessStore{allowed: map[models.DocumentKind]map[string]bool{
			models.DocumentKindIncomingLetter: {"read": true},
		}}
		userRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
		_, err := auth.Login(user.Login, password)
		require.NoError(t, err)
		userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()
		access := NewDocumentAccessService(auth, nil, nil, nil, accessStore, nil, nil, nil)
		svc := NewDashboardService(repo, auth, access)

		repo.On("GetExpiringAssignments", mock.MatchedBy(func(filter models.DashboardAssignmentFilter) bool {
			return filter.Days == 7 &&
				assert.Equal(t, []string{string(models.DocumentKindIncomingLetter)}, filter.AllowedDocumentKinds) &&
				assert.Equal(t, []string{user.ID.String()}, filter.AccessibleByUserIDs)
		})).Return([]models.Assignment{}, nil).Once()

		_, err = svc.GetActivity()
		require.NoError(t, err)
	})

	t.Run("not authenticated", func(t *testing.T) {
		repo := mocks.NewDashboardStore(t)
		userRepo := mocks.NewUserStore(t)
		auth := NewAuthService(nil, userRepo)
		svc := NewDashboardService(repo, auth, nil)

		activity, err := svc.GetActivity()
		require.ErrorIs(t, err, ErrNotAuthenticated)
		require.Nil(t, activity)
	})
}
