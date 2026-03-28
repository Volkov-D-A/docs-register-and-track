package services

import (
	"errors"
	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---------- helpers ----------

// loginUser — вспомогательная функция: логинит пользователя через мок и возвращает AuthService.
func loginUser(t *testing.T, mockRepo *mocks.UserStore, user *models.User, password string) *AuthService {
	t.Helper()
	authService := NewAuthService(nil, mockRepo)
	mockRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
	_, err := authService.Login(user.Login, password)
	require.NoError(t, err)
	mockRepo.On("GetByID", user.ID).Return(user, nil).Maybe()
	return authService
}

func newTestUser() (*models.User, string) {
	password := "CorrectPassw0rd!"
	hash, _ := security.HashPassword(password)
	return &models.User{
		ID:           uuid.New(),
		Login:        "testuser",
		PasswordHash: hash,
		FullName:     "Test User",
		IsActive:     true,
		Roles:        []string{"executor"},
	}, password
}

// ---------- TestAuthService_Login ----------

func TestAuthService_Login(t *testing.T) {
	// Аутентификация пользователя (вход в систему) и валидация пароля
	mockRepo := mocks.NewUserStore(t)
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

// ---------- TestAuthService_Logout ----------

func TestAuthService_Logout(t *testing.T) {
	// Выход пользователя из системы (сброс текущего пользователя в сессии/контексте)
	user, password := newTestUser()

	t.Run("logout resets currentUser", func(t *testing.T) {
		mockRepo := mocks.NewUserStore(t)
		authService := loginUser(t, mockRepo, user, password)
		require.True(t, authService.IsAuthenticated())

		err := authService.Logout()
		require.NoError(t, err)
		assert.False(t, authService.IsAuthenticated())
		assert.Empty(t, authService.GetCurrentUserID())
	})

	t.Run("logout without login", func(t *testing.T) {
		mockRepo := mocks.NewUserStore(t)
		authService := NewAuthService(nil, mockRepo)

		err := authService.Logout()
		require.NoError(t, err)
		assert.False(t, authService.IsAuthenticated())
	})
}

// ---------- TestAuthService_GetCurrentUser ----------

func TestAuthService_GetCurrentUser(t *testing.T) {
	// Получение данных текущего авторизованного пользователя
	user, password := newTestUser()

	t.Run("success", func(t *testing.T) {
		mockRepo := mocks.NewUserStore(t)
		authService := loginUser(t, mockRepo, user, password)

		dto, err := authService.GetCurrentUser()
		require.NoError(t, err)
		require.NotNil(t, dto)
		assert.Equal(t, user.ID.String(), dto.ID)
		assert.Equal(t, user.Login, dto.Login)
	})

	t.Run("not authenticated", func(t *testing.T) {
		mockRepo := mocks.NewUserStore(t)
		authService := NewAuthService(nil, mockRepo)

		dto, err := authService.GetCurrentUser()
		require.Error(t, err)
		assert.Equal(t, ErrNotAuthenticated, err)
		assert.Nil(t, dto)
	})
}

// ---------- TestAuthService_ChangePassword ----------

func TestAuthService_ChangePassword(t *testing.T) {
	// Смена пароля пользователя (с проверкой старого пароля и сложности нового)
	user, password := newTestUser()
	newPassword := "NewPassw0rd!"

	t.Run("success", func(t *testing.T) {
		mockRepo := mocks.NewUserStore(t)
		authService := loginUser(t, mockRepo, user, password)

		mockRepo.On("UpdatePassword", user.ID, mock.AnythingOfType("string")).Return(nil).Once()

		err := authService.ChangePassword(password, newPassword)
		require.NoError(t, err)
	})

	t.Run("wrong old password", func(t *testing.T) {
		mockRepo := mocks.NewUserStore(t)
		authService := loginUser(t, mockRepo, user, password)

		err := authService.ChangePassword("WrongOldPass1!", newPassword)
		require.Error(t, err)
		assert.Equal(t, ErrWrongPassword, err)
	})

	t.Run("weak new password", func(t *testing.T) {
		mockRepo := mocks.NewUserStore(t)
		authService := loginUser(t, mockRepo, user, password)

		err := authService.ChangePassword(password, "weak")
		require.Error(t, err)
	})

	t.Run("not authenticated", func(t *testing.T) {
		mockRepo := mocks.NewUserStore(t)
		authService := NewAuthService(nil, mockRepo)

		err := authService.ChangePassword(password, newPassword)
		require.Error(t, err)
		assert.Equal(t, ErrNotAuthenticated, err)
	})
}

// ---------- TestAuthService_UpdateProfile ----------

func TestAuthService_UpdateProfile(t *testing.T) {
	// Обновление профиля пользователя (например, смена ФИО или логина)
	user, password := newTestUser()
	req := models.UpdateProfileRequest{Login: "newlogin", FullName: "New Name"}

	t.Run("success", func(t *testing.T) {
		mockRepo := mocks.NewUserStore(t)
		authService := NewAuthService(nil, mockRepo)

		// Inline login to avoid .Maybe() from loginUser
		mockRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
		authService.Login(user.Login, password)

		updatedUser := &models.User{
			ID:           user.ID,
			Login:        req.Login,
			FullName:     req.FullName,
			PasswordHash: user.PasswordHash,
			IsActive:     true,
			Roles:        user.Roles,
		}

		mockRepo.On("UpdateProfile", user.ID, req).Return(nil).Once()
		mockRepo.On("GetByID", user.ID).Return(updatedUser, nil).Once()

		err := authService.UpdateProfile(req)
		require.NoError(t, err)

		// Проверяем что currentUser обновился
		dto, err := authService.GetCurrentUser()
		require.NoError(t, err)
		assert.Equal(t, req.Login, dto.Login)
		assert.Equal(t, req.FullName, dto.FullName)
	})

	t.Run("not authenticated", func(t *testing.T) {
		mockRepo := mocks.NewUserStore(t)
		authService := NewAuthService(nil, mockRepo)

		err := authService.UpdateProfile(req)
		require.Error(t, err)
		assert.Equal(t, ErrNotAuthenticated, err)
	})

	t.Run("repo error on UpdateProfile", func(t *testing.T) {
		mockRepo := mocks.NewUserStore(t)
		authService := loginUser(t, mockRepo, user, password)

		repoErr := errors.New("db connection error")
		mockRepo.On("UpdateProfile", user.ID, req).Return(repoErr).Once()

		err := authService.UpdateProfile(req)
		require.Error(t, err)
		assert.Equal(t, repoErr, err)
	})
}

// ---------- TestAuthService_IsAuthenticated ----------

func TestAuthService_IsAuthenticated(t *testing.T) {
	// Проверка статуса авторизации (авторизован ли кто-то в данный момент)
	user, password := newTestUser()

	t.Run("after login", func(t *testing.T) {
		mockRepo := mocks.NewUserStore(t)
		authService := loginUser(t, mockRepo, user, password)
		assert.True(t, authService.IsAuthenticated())
	})

	t.Run("after logout", func(t *testing.T) {
		mockRepo := mocks.NewUserStore(t)
		authService := loginUser(t, mockRepo, user, password)
		authService.Logout()
		assert.False(t, authService.IsAuthenticated())
	})

	t.Run("without login", func(t *testing.T) {
		mockRepo := mocks.NewUserStore(t)
		authService := NewAuthService(nil, mockRepo)
		assert.False(t, authService.IsAuthenticated())
	})
}

// ---------- TestAuthService_HasRole ----------

func TestAuthService_HasRole(t *testing.T) {
	// Проверка наличия необходимой роли у текущего пользователя
	t.Run("has role", func(t *testing.T) {
		user := &models.User{
			ID:           uuid.New(),
			Login:        "admin",
			PasswordHash: func() string { h, _ := security.HashPassword("Passw0rd!"); return h }(),
			IsActive:     true,
			Roles:        []string{"admin", "clerk"},
		}
		mockRepo := mocks.NewUserStore(t)
		authService := loginUser(t, mockRepo, user, "Passw0rd!")
		assert.True(t, authService.HasRole("admin"))
		assert.True(t, authService.HasRole("clerk"))
	})

	t.Run("no role", func(t *testing.T) {
		user, password := newTestUser() // roles = ["executor"]
		mockRepo := mocks.NewUserStore(t)
		authService := loginUser(t, mockRepo, user, password)
		assert.False(t, authService.HasRole("admin"))
	})

	t.Run("not authenticated", func(t *testing.T) {
		mockRepo := mocks.NewUserStore(t)
		authService := NewAuthService(nil, mockRepo)
		assert.False(t, authService.HasRole("admin"))
	})
}

func TestAuthService_ActiveRole(t *testing.T) {
	t.Run("login sets default active role by priority", func(t *testing.T) {
		user := &models.User{
			ID:           uuid.New(),
			Login:        "multi_role_user",
			PasswordHash: func() string { h, _ := security.HashPassword("Passw0rd!"); return h }(),
			IsActive:     true,
			Roles:        []string{"executor", "clerk"},
		}
		mockRepo := mocks.NewUserStore(t)
		authService := loginUser(t, mockRepo, user, "Passw0rd!")

		assert.Equal(t, "clerk", authService.GetActiveRole())
		assert.True(t, authService.HasActiveRole("clerk"))
		assert.False(t, authService.HasActiveRole("executor"))
	})

	t.Run("set active role switches to allowed role", func(t *testing.T) {
		user := &models.User{
			ID:           uuid.New(),
			Login:        "switch_role_user",
			PasswordHash: func() string { h, _ := security.HashPassword("Passw0rd!"); return h }(),
			IsActive:     true,
			Roles:        []string{"admin", "clerk"},
		}
		mockRepo := mocks.NewUserStore(t)
		authService := loginUser(t, mockRepo, user, "Passw0rd!")

		err := authService.SetActiveRole("clerk")
		require.NoError(t, err)
		assert.Equal(t, "clerk", authService.GetActiveRole())
		assert.NoError(t, authService.RequireActiveRole("clerk"))
		assert.ErrorIs(t, authService.RequireActiveRole("admin"), models.ErrForbidden)
	})

	t.Run("set active role rejects missing role", func(t *testing.T) {
		user, password := newTestUser()
		mockRepo := mocks.NewUserStore(t)
		authService := loginUser(t, mockRepo, user, password)

		err := authService.SetActiveRole("admin")
		assert.ErrorIs(t, err, models.ErrForbidden)
		assert.Equal(t, "executor", authService.GetActiveRole())
	})

	t.Run("logout clears active role", func(t *testing.T) {
		user, password := newTestUser()
		mockRepo := mocks.NewUserStore(t)
		authService := loginUser(t, mockRepo, user, password)

		require.NoError(t, authService.Logout())
		assert.Empty(t, authService.GetActiveRole())
	})
}

// ---------- TestAuthService_NeedsInitialSetup ----------

func TestAuthService_NeedsInitialSetup(t *testing.T) {
	// Проверка необходимости первоначальной настройки системы (если пользователей еще нет)
	t.Run("no users - needs setup", func(t *testing.T) {
		mockRepo := mocks.NewUserStore(t)
		authService := NewAuthService(nil, mockRepo)
		mockRepo.On("CountUsers").Return(0, nil).Once()

		needs, err := authService.NeedsInitialSetup()
		require.NoError(t, err)
		assert.True(t, needs)
	})

	t.Run("users exist - no setup needed", func(t *testing.T) {
		mockRepo := mocks.NewUserStore(t)
		authService := NewAuthService(nil, mockRepo)
		mockRepo.On("CountUsers").Return(3, nil).Once()

		needs, err := authService.NeedsInitialSetup()
		require.NoError(t, err)
		assert.False(t, needs)
	})

	t.Run("generic repo error", func(t *testing.T) {
		mockRepo := mocks.NewUserStore(t)
		authService := NewAuthService(nil, mockRepo)
		mockRepo.On("CountUsers").Return(0, errors.New("connection refused")).Once()

		_, err := authService.NeedsInitialSetup()
		require.Error(t, err)
	})
}

// ---------- TestAuthService_InitialSetup ----------

func TestAuthService_InitialSetup(t *testing.T) {
	// Первоначальная настройка системы (создание главного администратора)
	goodPassword := "Admin1Pass!"

	t.Run("success", func(t *testing.T) {
		mockRepo := mocks.NewUserStore(t)
		authService := NewAuthService(nil, mockRepo)

		// Первый вызов CountUsers — таблица существует, 0 пользователей
		mockRepo.On("CountUsers").Return(0, nil).Twice()
		mockRepo.On("Create", mock.MatchedBy(func(req models.CreateUserRequest) bool {
			return req.Login == "admin" && req.FullName == "Администратор" &&
				len(req.Roles) == 1 && req.Roles[0] == "admin"
		})).Return(&models.User{ID: uuid.New()}, nil).Once()

		err := authService.InitialSetup(goodPassword)
		require.NoError(t, err)
	})

	t.Run("already setup", func(t *testing.T) {
		mockRepo := mocks.NewUserStore(t)
		authService := NewAuthService(nil, mockRepo)

		// Первый вызов — 0, второй — 1 (кто-то уже создал)
		mockRepo.On("CountUsers").Return(0, nil).Once()
		mockRepo.On("CountUsers").Return(1, nil).Once()

		err := authService.InitialSetup(goodPassword)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "уже выполнена")
	})

	t.Run("weak password", func(t *testing.T) {
		mockRepo := mocks.NewUserStore(t)
		authService := NewAuthService(nil, mockRepo)

		mockRepo.On("CountUsers").Return(0, nil).Twice()

		err := authService.InitialSetup("weak")
		require.Error(t, err)
	})
}
