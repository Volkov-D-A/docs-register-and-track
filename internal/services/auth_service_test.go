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

func TestAuthService_Login(t *testing.T) {
	mockRepo := mocks.NewUserStore(t)
	// Мы передаем nil для БД, так как не тестируем InitialSetup, которому нужна БД в данном сценарии работы.
	// Для Login нужен только UserStore.
	authService := NewAuthService(nil, mockRepo)

	login := "testuser"
	password := "CorrectPassw0rd!"
	hash, _ := security.HashPassword(password)
	userID := uuid.New()

	activeUser := &models.User{
		ID:           userID,
		Login:        login,
		PasswordHash: hash,
		IsActive:     true,
	}

	inactiveUser := &models.User{
		ID:           userID,
		Login:        login,
		PasswordHash: hash,
		IsActive:     false,
	}

	t.Run("success", func(t *testing.T) {
		mockRepo.On("GetByLogin", login).Return(activeUser, nil).Once()

		userDTO, err := authService.Login(login, password)

		require.NoError(t, err)
		require.NotNil(t, userDTO)
		assert.Equal(t, userID.String(), userDTO.ID)
		assert.Equal(t, login, userDTO.Login)
		assert.True(t, authService.IsAuthenticated())
		assert.Equal(t, userID.String(), authService.GetCurrentUserID())

		// Сброс состояния
		authService.Logout()
		mockRepo.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		mockRepo.On("GetByLogin", "unknown").Return(nil, nil).Once()

		userDTO, err := authService.Login("unknown", password)

		require.Error(t, err)
		assert.Equal(t, ErrInvalidCredentials, err)
		assert.Nil(t, userDTO)
		assert.False(t, authService.IsAuthenticated())
		mockRepo.AssertExpectations(t)
	})

	t.Run("wrong password", func(t *testing.T) {
		mockRepo.On("GetByLogin", login).Return(activeUser, nil).Once()

		userDTO, err := authService.Login(login, "WrongPass1!")

		require.Error(t, err)
		assert.Equal(t, ErrInvalidCredentials, err)
		assert.Nil(t, userDTO)
		assert.False(t, authService.IsAuthenticated())
		mockRepo.AssertExpectations(t)
	})

	t.Run("user inactive", func(t *testing.T) {
		mockRepo.On("GetByLogin", login).Return(inactiveUser, nil).Once()

		userDTO, err := authService.Login(login, password)

		require.Error(t, err)
		assert.Equal(t, ErrUserNotActive, err)
		assert.Nil(t, userDTO)
		assert.False(t, authService.IsAuthenticated())
		mockRepo.AssertExpectations(t)
	})
}
