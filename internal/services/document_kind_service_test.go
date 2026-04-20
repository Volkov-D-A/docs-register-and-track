package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
)

func setupDocumentKindService(t *testing.T, role string) (*DocumentKindService, *AuthService) {
	t.Helper()

	userRepo := mocks.NewUserStore(t)
	depRepo := mocks.NewDepartmentStore(t)
	assignmentRepo := mocks.NewAssignmentStore(t)
	ackRepo := mocks.NewAcknowledgmentStore(t)
	auth := NewAuthService(nil, userRepo)

	password := "Passw0rd!"
	hash, _ := security.HashPassword(password)
	user := &models.User{
		ID:           uuid.New(),
		Login:        role + "_kind",
		PasswordHash: hash,
		IsActive:     true,
	}

	userRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
	_, err := auth.Login(user.Login, password)
	require.NoError(t, err)
	userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	access := NewDocumentAccessService(auth, depRepo, assignmentRepo, ackRepo, newRoleMappedDocumentAccessStore(role), nil, nil, nil)
	return NewDocumentKindService(access), auth
}

func TestDocumentKindService_GetAll(t *testing.T) {
	t.Parallel()

	service, _ := setupDocumentKindService(t, "executor")

	items, err := service.GetAll()
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "incoming_letter", items[0].Code)
	assert.Equal(t, "outgoing_letter", items[1].Code)
}

func TestDocumentKindService_GetAvailableForRegistration(t *testing.T) {
	t.Parallel()

	t.Run("clerk gets kinds", func(t *testing.T) {
		service, auth := setupDocumentKindService(t, "clerk")
		defer auth.Logout()

		items, err := service.GetAvailableForRegistration()
		require.NoError(t, err)
		require.Len(t, items, 2)
	})

	t.Run("executor gets none", func(t *testing.T) {
		service, auth := setupDocumentKindService(t, "executor")
		defer auth.Logout()

		items, err := service.GetAvailableForRegistration()
		require.NoError(t, err)
		assert.Empty(t, items)
	})
}
