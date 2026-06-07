package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func setupUserSubstitutionService(t *testing.T, currentUser *models.User) (*UserSubstitutionService, *userSubstitutionStoreStub, *mocks.UserStore, *AuthService) {
	t.Helper()
	store := &userSubstitutionStoreStub{}
	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)
	auth.SetAccessStore(newRoleMappedDocumentAccessStore("admin"))
	if currentUser != nil {
		auth.currentUserID = currentUser.ID
		userRepo.On("GetByID", currentUser.ID).Return(currentUser, nil).Maybe()
	}
	return NewUserSubstitutionService(store, userRepo, auth, nil), store, userRepo, auth
}

func TestUserSubstitutionService_UpdateMySubstitution(t *testing.T) {
	t.Run("allows active substitute from same department without document participant flag", func(t *testing.T) {
		departmentID := uuid.New()
		principal := &models.User{
			ID:                    uuid.New(),
			IsActive:              true,
			IsDocumentParticipant: true,
			DepartmentID:          &departmentID,
			Department:            &models.Department{ID: departmentID},
		}
		substitute := &models.User{
			ID:           uuid.New(),
			IsActive:     true,
			DepartmentID: &departmentID,
			Department:   &models.Department{ID: departmentID},
		}
		svc, store, userRepo, _ := setupUserSubstitutionService(t, principal)
		userRepo.On("GetByID", substitute.ID).Return(substitute, nil).Once()

		result, err := svc.UpdateMySubstitution(models.UpdateUserSubstitutionRequest{
			SubstituteUserID: substitute.ID.String(),
			StartsAt:         "2026-06-01",
			EndsAt:           "2026-06-10",
			IsActive:         true,
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, substitute.ID.String(), result.SubstituteUserID)
		require.Len(t, store.replaceCalls, 1)
		assert.Equal(t, principal.ID, store.replaceCalls[0].principalUserID)
		require.NotNil(t, store.replaceCalls[0].substituteUserID)
		assert.Equal(t, substitute.ID, *store.replaceCalls[0].substituteUserID)
		require.NotNil(t, store.replaceCalls[0].startsAt)
		require.NotNil(t, store.replaceCalls[0].endsAt)
	})

	t.Run("rejects substitute from another department", func(t *testing.T) {
		principalDepartmentID := uuid.New()
		otherDepartmentID := uuid.New()
		principal := &models.User{
			ID:                    uuid.New(),
			IsActive:              true,
			IsDocumentParticipant: true,
			DepartmentID:          &principalDepartmentID,
			Department:            &models.Department{ID: principalDepartmentID},
		}
		substitute := &models.User{
			ID:           uuid.New(),
			IsActive:     true,
			DepartmentID: &otherDepartmentID,
			Department:   &models.Department{ID: otherDepartmentID},
		}
		svc, store, userRepo, _ := setupUserSubstitutionService(t, principal)
		userRepo.On("GetByID", substitute.ID).Return(substitute, nil).Once()

		result, err := svc.UpdateMySubstitution(models.UpdateUserSubstitutionRequest{
			SubstituteUserID: substitute.ID.String(),
			IsActive:         true,
		})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Empty(t, store.replaceCalls)
		assert.Contains(t, err.Error(), "замещающий должен быть из подразделения")
	})

	t.Run("clears substitution even when principal is not document participant", func(t *testing.T) {
		principal := &models.User{ID: uuid.New(), IsActive: true, IsDocumentParticipant: false}
		svc, store, _, _ := setupUserSubstitutionService(t, principal)

		result, err := svc.UpdateMySubstitution(models.UpdateUserSubstitutionRequest{})

		require.NoError(t, err)
		assert.Nil(t, result)
		require.Len(t, store.replaceCalls, 1)
		assert.Nil(t, store.replaceCalls[0].substituteUserID)
	})
}

func TestUserService_GetSubstitutionCandidates(t *testing.T) {
	svc, repo := setupUserService(t, "executor")
	expected := []models.User{{ID: uuid.New(), FullName: "Active user", IsActive: true}}
	repo.On("GetActiveUsers").Return(expected, nil).Once()

	result, err := svc.GetSubstitutionCandidates()

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "Active user", result[0].FullName)
}
