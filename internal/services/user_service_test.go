package services

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserService_GetAllUsers(t *testing.T) {
	// Получение списка всех заведенных пользователей в системе
	mockRepo := mocks.NewUserStore(t)
	authRepo := mocks.NewUserStore(t)
	authService := NewAuthService(nil, authRepo)
	userService := NewUserService(mockRepo, authService, nil)

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
		authRepo.On("GetByID", adminUser.ID).Return(adminUser, nil).Maybe()

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
		authRepo.On("GetByID", regularUser.ID).Return(regularUser, nil).Maybe()

		users, err := userService.GetAllUsers()
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
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

func setupUserService(t *testing.T, role string) (*UserService, *mocks.UserStore) {
	t.Helper()
	mockRepo := mocks.NewUserStore(t)
	authRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, authRepo)

	password := "Passw0rd!"
	hash, _ := security.HashPassword(password)
	user := &models.User{
		ID:           uuid.New(),
		Login:        role + "_usr",
		PasswordHash: hash,
		IsActive:     true,
		Roles:        []string{role},
	}
	authRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
	auth.Login(user.Login, password)
	authRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	return NewUserService(mockRepo, auth, nil), mockRepo
}

func TestUserService_CreateUser(t *testing.T) {
	// Создание новой карточки пользователя системы
	t.Run("success admin", func(t *testing.T) {
		svc, repo := setupUserService(t, "admin")
		req := models.CreateUserRequest{Login: "newuser", Password: "Pass1234!", FullName: "New User", Roles: []string{"executor"}}
		repo.On("Create", req).Return(&models.User{ID: uuid.New(), Login: "newuser"}, nil).Once()
		result, err := svc.CreateUser(req)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("forbidden clerk", func(t *testing.T) {
		svc, _ := setupUserService(t, "clerk")
		result, err := svc.CreateUser(models.CreateUserRequest{})
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
		assert.Nil(t, result)
	})
}

func TestUserService_UpdateUser(t *testing.T) {
	// Обновление профиля пользователя
	t.Run("success admin", func(t *testing.T) {
		svc, repo := setupUserService(t, "admin")
		uid := uuid.New()
		req := models.UpdateUserRequest{ID: uid.String(), FullName: "Updated", Roles: []string{"executor"}, IsActive: true}
		repo.On("Update", req).Return(&models.User{ID: uid, FullName: "Updated"}, nil).Once()
		result, err := svc.UpdateUser(req)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("forbidden executor", func(t *testing.T) {
		svc, _ := setupUserService(t, "executor")
		result, err := svc.UpdateUser(models.UpdateUserRequest{})
		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestUserService_ResetPassword(t *testing.T) {
	// Принудительный сброс и установка нового пароля администратором для другого пользователя
	uid := uuid.New()

	t.Run("success admin", func(t *testing.T) {
		svc, repo := setupUserService(t, "admin")
		repo.On("ResetPassword", uid, "NewPass123!").Return(nil).Once()
		err := svc.ResetPassword(uid.String(), "NewPass123!")
		require.NoError(t, err)
	})

	t.Run("forbidden executor", func(t *testing.T) {
		svc, _ := setupUserService(t, "executor")
		err := svc.ResetPassword(uid.String(), "NewPass123!")
		require.Error(t, err)
	})

	t.Run("invalid ID", func(t *testing.T) {
		svc, _ := setupUserService(t, "admin")
		err := svc.ResetPassword("not-uuid", "NewPass123!")
		require.Error(t, err)
	})
}

func TestUserService_GetExecutors(t *testing.T) {
	// Получение списка всех активных исполнителей для выдачи поручений
	t.Run("success", func(t *testing.T) {
		svc, repo := setupUserService(t, "executor")
		repo.On("GetExecutors").Return([]models.User{{ID: uuid.New()}}, nil).Once()
		result, err := svc.GetExecutors()
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})
}
