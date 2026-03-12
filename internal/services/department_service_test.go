package services

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"docflow/internal/mocks"
	"docflow/internal/models"
	"docflow/internal/security"
)

func setupDepartmentService(t *testing.T, role string) (*DepartmentService, *mocks.DepartmentStore, *AuthService) {
	t.Helper()
	depRepo := mocks.NewDepartmentStore(t)
	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)

	if role != "" {
		password := "Passw0rd!"
		hash, _ := security.HashPassword(password)
		user := &models.User{
			ID:           uuid.New(),
			Login:        role + "_dep",
			PasswordHash: hash,
			IsActive:     true,
			Roles:        []string{role},
		}
		userRepo.On("GetByLogin", user.Login).Return(user, nil).Maybe()
		_, err := auth.Login(user.Login, password)
		require.NoError(t, err)
	}

	svc := NewDepartmentService(depRepo, auth, nil)
	return svc, depRepo, auth
}

func TestDepartmentService_GetAllDepartments(t *testing.T) {
	// Получение списка всех подразделений организации
	t.Run("успех", func(t *testing.T) {
		svc, repo, _ := setupDepartmentService(t, "clerk")
		mockValues := []models.Department{
			{ID: uuid.New(), Name: "Отдел кадров"},
			{ID: uuid.New(), Name: "Бухгалтерия"},
		}
		repo.On("GetAll").Return(mockValues, nil).Once()

		result, err := svc.GetAllDepartments()
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "Отдел кадров", result[0].Name)
	})

	t.Run("ошибка базы", func(t *testing.T) {
		svc, repo, _ := setupDepartmentService(t, "clerk")
		repo.On("GetAll").Return(nil, errors.New("db error")).Once()

		result, err := svc.GetAllDepartments()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
		assert.Nil(t, result)
	})

	t.Run("не авторизован", func(t *testing.T) {
		svc, _, _ := setupDepartmentService(t, "")
		result, err := svc.GetAllDepartments()
		require.Error(t, err)
		assert.Equal(t, ErrNotAuthenticated, err)
		assert.Nil(t, result)
	})
}

func TestDepartmentService_CreateDepartment(t *testing.T) {
	// Создание нового подразделения с сохранением привязанных индексов номенклатуры
	t.Run("успех (админ)", func(t *testing.T) {
		svc, repo, _ := setupDepartmentService(t, "admin")
		name := "IT Отдел"
		nomIDs := []string{uuid.New().String(), uuid.New().String()}

		expected := &models.Department{ID: uuid.New(), Name: name}
		repo.On("Create", name, nomIDs).Return(expected, nil).Once()

		result, err := svc.CreateDepartment(name, nomIDs)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, name, result.Name)
	})

	t.Run("запрещено (не админ)", func(t *testing.T) {
		svc, _, _ := setupDepartmentService(t, "clerk")
		result, err := svc.CreateDepartment("Test", []string{})
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
		assert.Nil(t, result)
	})
}

func TestDepartmentService_UpdateDepartment(t *testing.T) {
	// Обновление данных существующего подразделения
	idStr := uuid.New().String()

	t.Run("успех (админ)", func(t *testing.T) {
		svc, repo, _ := setupDepartmentService(t, "admin")
		name := "Обновленный IT Отдел"
		nomIDs := []string{uuid.New().String()}

		expected := &models.Department{ID: uuid.MustParse(idStr), Name: name}
		repo.On("Update", mock.AnythingOfType("uuid.UUID"), name, nomIDs).Return(expected, nil).Once()

		result, err := svc.UpdateDepartment(idStr, name, nomIDs)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, name, result.Name)
	})

	t.Run("невалидный ID", func(t *testing.T) {
		svc, _, _ := setupDepartmentService(t, "admin")
		result, err := svc.UpdateDepartment("invalid-uuid", "Тест", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid department ID")
		assert.Nil(t, result)
	})

	t.Run("запрещено (не админ)", func(t *testing.T) {
		svc, _, _ := setupDepartmentService(t, "clerk")
		result, err := svc.UpdateDepartment(idStr, "Тест", nil)
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
		assert.Nil(t, result)
	})
}

func TestDepartmentService_DeleteDepartment(t *testing.T) {
	// Удаление подразделения из базы
	idStr := uuid.New().String()

	t.Run("успех (админ)", func(t *testing.T) {
		svc, repo, _ := setupDepartmentService(t, "admin")
		repo.On("Delete", mock.AnythingOfType("uuid.UUID")).Return(nil).Once()

		err := svc.DeleteDepartment(idStr)
		require.NoError(t, err)
	})

	t.Run("невалидный ID", func(t *testing.T) {
		svc, _, _ := setupDepartmentService(t, "admin")
		err := svc.DeleteDepartment("invalid-uuid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid department ID")
	})

	t.Run("запрещено (не админ)", func(t *testing.T) {
		svc, _, _ := setupDepartmentService(t, "clerk")
		err := svc.DeleteDepartment(idStr)
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
	})
}
