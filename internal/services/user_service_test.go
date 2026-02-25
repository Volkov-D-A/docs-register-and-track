package services

import (
	"docflow/internal/mocks"
	"docflow/internal/models"
	"docflow/internal/security"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserService_GetAllUsers(t *testing.T) {
	mockRepo := mocks.NewUserStore(t)
	authRepo := mocks.NewUserStore(t)
	authService := NewAuthService(nil, authRepo)
	userService := NewUserService(mockRepo, authService)

	login := "testuser"
	password := "CorrectPassw0rd!"
	hash, _ := security.HashPassword(password)

	// Set up auth service with admin user
	adminUser := &models.User{
		ID:           uuid.New(),
		Login:        login,
		PasswordHash: hash,
		IsActive:     true,
		Roles:        []string{"admin"},
	}

	regularUser := &models.User{
		ID:           uuid.New(),
		Login:        "regular",
		PasswordHash: hash,
		IsActive:     true,
		Roles:        []string{"user"},
	}

	t.Run("success with admin role", func(t *testing.T) {
		authRepo.On("GetByLogin", login).Return(adminUser, nil).Once()
		authService.Login(login, password)

		usersList := []models.User{*regularUser}
		mockRepo.On("GetAll").Return(usersList, nil).Once()

		users, err := userService.GetAllUsers()
		require.NoError(t, err)
		assert.Len(t, users, 1)

		authService.Logout()
	})

	t.Run("failure with regular role", func(t *testing.T) {
		authRepo.On("GetByLogin", "regular").Return(regularUser, nil).Once()
		authService.Login("regular", password)

		users, err := userService.GetAllUsers()
		require.Error(t, err)
		assert.Equal(t, ErrNotAuthenticated, err)
		assert.Nil(t, users)

		authService.Logout()
	})

	t.Run("failure unauthenticated", func(t *testing.T) {
		users, err := userService.GetAllUsers()
		require.Error(t, err)
		assert.Equal(t, ErrNotAuthenticated, err)
		assert.Nil(t, users)
	})
}
