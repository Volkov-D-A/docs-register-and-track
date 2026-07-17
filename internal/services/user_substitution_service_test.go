package services

import (
	"testing"
	"time"

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

type atomicUserSubstitutionStore struct {
	*userSubstitutionStoreStub
	effects []models.OutboxEvent
}

func (s *atomicUserSubstitutionStore) ReplaceForPrincipalWithOutbox(
	principalUserID uuid.UUID,
	substituteUserID *uuid.UUID,
	startsAt, endsAt *time.Time,
	isActive bool,
	createdBy *uuid.UUID,
	effects []models.OutboxEvent,
) (*models.UserSubstitution, error) {
	s.effects = append([]models.OutboxEvent(nil), effects...)
	return s.userSubstitutionStoreStub.ReplaceForPrincipal(principalUserID, substituteUserID, startsAt, endsAt, isActive, createdBy)
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

func TestUserSubstitutionServiceUpdateUserSubstitutionPassesAuditEffectToAtomicStore(t *testing.T) {
	departmentID := uuid.New()
	admin := &models.User{ID: uuid.New(), FullName: "Администратор", IsActive: true}
	principal := &models.User{ID: uuid.New(), FullName: "Основной пользователь", IsActive: true, IsDocumentParticipant: true, DepartmentID: &departmentID}
	substitute := &models.User{ID: uuid.New(), IsActive: true, DepartmentID: &departmentID}
	svc, store, userRepo, _ := setupUserSubstitutionService(t, admin)
	atomicStore := &atomicUserSubstitutionStore{userSubstitutionStoreStub: store}
	svc.repo = atomicStore
	userRepo.On("GetByID", principal.ID).Return(principal, nil).Once()
	userRepo.On("GetByID", substitute.ID).Return(substitute, nil).Once()

	result, err := svc.UpdateUserSubstitution(models.UpdateUserSubstitutionRequest{
		PrincipalUserID:  principal.ID.String(),
		SubstituteUserID: substitute.ID.String(),
		IsActive:         true,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, atomicStore.effects, 1)
	assert.Equal(t, models.OutboxEventAudit, atomicStore.effects[0].EventType)
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
