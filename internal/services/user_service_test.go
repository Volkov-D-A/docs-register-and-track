package services

import (
	"context"
	"errors"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type adminAuditLogStoreMock struct {
	mock.Mock
}

// testUserClient preserves the old repository-focused assertions while the
// production UserService now delegates these commands to docflow-server.
type testUserClient struct {
	repo UserStore
	auth *AuthService
}

func (c *testUserClient) ListUsers(context.Context) ([]dto.User, error) {
	if err := c.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return nil, err
	}
	users, err := c.repo.GetAll()
	return dto.MapUsers(users), err
}

func (c *testUserClient) ListSubstitutionCandidates(context.Context) ([]dto.User, error) {
	if err := c.auth.RequireAuthenticated(); err != nil {
		return nil, err
	}
	users, err := c.repo.GetActiveUsers()
	return dto.MapUsers(users), err
}

func (c *testUserClient) CreateUser(_ context.Context, req models.CreateUserRequest) (*dto.User, error) {
	if err := c.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return nil, err
	}
	temporaryPassword := ""
	if req.Password == "" {
		password, err := security.GenerateTemporaryPassword()
		if err != nil {
			return nil, err
		}
		req.Password, temporaryPassword = password, password
	}
	req.PasswordChangeRequired = true
	user, err := c.repo.Create(req)
	if err != nil {
		return nil, err
	}
	result := dto.MapUser(user)
	result.TemporaryPassword = temporaryPassword
	return result, nil
}

func (c *testUserClient) UpdateUser(_ context.Context, req models.UpdateUserRequest) (*dto.User, error) {
	if err := c.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return nil, err
	}
	user, err := c.repo.Update(req)
	return dto.MapUser(user), activeAdministratorInvariantConflict(err)
}

func (c *testUserClient) ResetUserPassword(_ context.Context, id string) (string, error) {
	if err := c.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return "", err
	}
	uid, err := parseUUID(id)
	if err != nil {
		return "", err
	}
	user, err := c.repo.GetByID(uid)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", models.NewNotFound("пользователь не найден")
	}
	password, err := security.GenerateTemporaryPassword()
	if err != nil {
		return "", err
	}
	return password, c.repo.ResetPassword(uid, password)
}

func setTestUserClient(service *UserService, repo UserStore, auth *AuthService) {
	service.SetServerClient(&testUserClient{repo: repo, auth: auth})
}

func (m *adminAuditLogStoreMock) Create(req models.CreateAdminAuditLogRequest) (uuid.UUID, error) {
	args := m.Called(req)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *adminAuditLogStoreMock) GetAll(limit, offset int) ([]models.AdminAuditLog, int, error) {
	args := m.Called(limit, offset)

	var logs []models.AdminAuditLog
	if got := args.Get(0); got != nil {
		logs = got.([]models.AdminAuditLog)
	}

	return logs, args.Int(1), args.Error(2)
}

func TestUserService_GetAllUsers(t *testing.T) {
	// Получение списка всех заведенных пользователей в системе
	mockRepo := mocks.NewUserStore(t)
	authRepo := mocks.NewUserStore(t)
	authService := NewAuthService(nil, authRepo)
	userService := NewUserService(mockRepo, authService)
	setTestUserClient(userService, mockRepo, authService)

	login := "testuser"
	password := "CorrectPassw0rd!"
	hash, _ := security.HashPassword(password)

	// Set up auth service with admin user
	adminUser := &models.User{
		ID:           uuid.New(),
		Login:        login,
		PasswordHash: hash,
		IsActive:     true,
	}

	regularUser := &models.User{
		ID:           uuid.New(),
		Login:        "regular",
		PasswordHash: hash,
		IsActive:     true,
	}

	t.Run("success with admin role", func(t *testing.T) {
		authService.SetAccessStore(newRoleMappedDocumentAccessStore("admin"))
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
		authService.SetAccessStore(newRoleMappedDocumentAccessStore())
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

	t.Run("success with admin system permission for multi-subject user", func(t *testing.T) {
		svc, repo, _ := setupUserServiceWithRoles(t, []string{"admin", "clerk"})

		usersList := []models.User{*regularUser}
		repo.On("GetAll").Return(usersList, nil).Once()

		users, err := svc.GetAllUsers()
		require.NoError(t, err)
		assert.Len(t, users, 1)
	})
}

func setupUserService(t *testing.T, role string) (*UserService, *mocks.UserStore) {
	t.Helper()
	mockRepo := mocks.NewUserStore(t)
	authRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, authRepo)
	auth.SetAccessStore(newRoleMappedDocumentAccessStore(role))

	password := "Passw0rd!"
	hash, _ := security.HashPassword(password)
	user := &models.User{
		ID:           uuid.New(),
		Login:        role + "_usr",
		PasswordHash: hash,
		IsActive:     true,
	}
	authRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
	auth.Login(user.Login, password)
	authRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	service := NewUserService(mockRepo, auth)
	setTestUserClient(service, mockRepo, auth)
	return service, mockRepo
}

func setupUserServiceWithRoles(t *testing.T, roles []string) (*UserService, *mocks.UserStore, *AuthService) {
	t.Helper()
	mockRepo := mocks.NewUserStore(t)
	authRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, authRepo)
	auth.SetAccessStore(newRoleMappedDocumentAccessStore(roles...))

	password := "Passw0rd!"
	hash, _ := security.HashPassword(password)
	user := &models.User{
		ID:           uuid.New(),
		Login:        "multi_usr_" + uuid.New().String(),
		PasswordHash: hash,
		IsActive:     true,
	}
	authRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
	_, err := auth.Login(user.Login, password)
	require.NoError(t, err)
	authRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	service := NewUserService(mockRepo, auth)
	setTestUserClient(service, mockRepo, auth)
	return service, mockRepo, auth
}

func TestUserService_CreateUser(t *testing.T) {
	// Создание новой карточки пользователя системы
	t.Run("success admin", func(t *testing.T) {
		svc, repo := setupUserService(t, "admin")
		req := models.CreateUserRequest{Login: "newuser", Password: "Pass1234!", FullName: "New User"}
		repo.On("Create", mock.MatchedBy(func(actual models.CreateUserRequest) bool {
			return actual.Login == req.Login &&
				actual.Password == req.Password &&
				actual.FullName == req.FullName &&
				actual.PasswordChangeRequired
		})).Return(&models.User{ID: uuid.New(), Login: "newuser"}, nil).Once()
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

	t.Run("allowed for user with admin role", func(t *testing.T) {
		svc, repo, auth := setupUserServiceWithRoles(t, []string{"admin", "clerk"})
		req := models.CreateUserRequest{}
		repo.On("Create", mock.MatchedBy(func(actual models.CreateUserRequest) bool {
			return actual.Password != "" && actual.PasswordChangeRequired
		})).Return(&models.User{ID: uuid.New()}, nil).Once()

		result, err := svc.CreateUser(req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.TemporaryPassword)
		_ = auth
	})
}

func TestUserService_UpdateUser(t *testing.T) {
	// Обновление профиля пользователя
	t.Run("success admin", func(t *testing.T) {
		svc, repo := setupUserService(t, "admin")
		uid := uuid.New()
		req := models.UpdateUserRequest{ID: uid.String(), FullName: "Updated", IsActive: true}
		repo.On("Update", req).Return(&models.User{ID: uid, FullName: "Updated"}, nil).Once()
		result, err := svc.UpdateUser(req)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("forbidden executor", func(t *testing.T) {
		svc, _ := setupUserService(t, "executor")
		result, err := svc.UpdateUser(models.UpdateUserRequest{})
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
		assert.Nil(t, result)
	})

	t.Run("allowed for user with admin role", func(t *testing.T) {
		svc, repo, _ := setupUserServiceWithRoles(t, []string{"admin", "clerk"})
		req := models.UpdateUserRequest{}
		repo.On("Update", req).Return(&models.User{ID: uuid.New()}, nil).Once()

		result, err := svc.UpdateUser(req)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestUserService_ResetPassword(t *testing.T) {
	// Принудительный сброс и установка нового пароля администратором для другого пользователя
	uid := uuid.New()

	t.Run("success admin", func(t *testing.T) {
		svc, repo := setupUserService(t, "admin")

		targetUser := &models.User{ID: uid, FullName: "Иван Петров", Login: "ipetrov"}
		repo.On("GetByID", uid).Return(targetUser, nil).Once()
		repo.On("ResetPassword", uid, mock.MatchedBy(func(password string) bool {
			return security.ValidatePassword(password) == nil
		})).Return(nil).Once()

		temporaryPassword, err := svc.ResetPassword(uid.String())
		require.NoError(t, err)
		assert.NotEmpty(t, temporaryPassword)
	})

	t.Run("forbidden executor", func(t *testing.T) {
		svc, _ := setupUserService(t, "executor")
		_, err := svc.ResetPassword(uid.String())
		require.Error(t, err)
	})

	t.Run("invalid ID", func(t *testing.T) {
		svc, _ := setupUserService(t, "admin")
		_, err := svc.ResetPassword("not-uuid")
		require.Error(t, err)
	})

	t.Run("returns not found for absent user", func(t *testing.T) {
		svc, repo := setupUserService(t, "admin")
		repo.On("GetByID", uid).Return(nil, nil).Once()

		_, err := svc.ResetPassword(uid.String())

		requireAppError(t, err, "NOT_FOUND", 404, "пользователь не найден")
	})

	t.Run("allowed for user with admin role", func(t *testing.T) {
		svc, repo, _ := setupUserServiceWithRoles(t, []string{"admin", "clerk"})
		targetUser := &models.User{ID: uid, FullName: "User", Login: "user"}
		repo.On("GetByID", uid).Return(targetUser, nil).Once()
		repo.On("ResetPassword", uid, mock.AnythingOfType("string")).Return(nil).Once()

		temporaryPassword, err := svc.ResetPassword(uid.String())
		require.NoError(t, err)
		assert.NotEmpty(t, temporaryPassword)
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

	t.Run("rejects unauthenticated user", func(t *testing.T) {
		repo := mocks.NewUserStore(t)
		auth := NewAuthService(nil, mocks.NewUserStore(t))
		svc := NewUserService(repo, auth)

		result, err := svc.GetExecutors()

		require.ErrorIs(t, err, ErrNotAuthenticated)
		assert.Nil(t, result)
	})

	t.Run("rejects user deactivated after login", func(t *testing.T) {
		repo := mocks.NewUserStore(t)
		authRepo := mocks.NewUserStore(t)
		user := &models.User{ID: uuid.New(), IsActive: false}
		auth := NewAuthService(nil, authRepo)
		auth.currentUserID = user.ID
		authRepo.On("GetByID", user.ID).Return(user, nil).Once()
		svc := NewUserService(repo, auth)

		result, err := svc.GetExecutors()

		require.ErrorIs(t, err, models.ErrUnauthorized)
		assert.Nil(t, result)
		assert.False(t, auth.IsAuthenticated())
	})

	t.Run("propagates repository error", func(t *testing.T) {
		expectedErr := errors.New("executors failed")
		svc, repo := setupUserService(t, "executor")
		repo.On("GetExecutors").Return(nil, expectedErr).Once()

		result, err := svc.GetExecutors()

		require.ErrorIs(t, err, expectedErr)
		assert.Nil(t, result)
	})
}
