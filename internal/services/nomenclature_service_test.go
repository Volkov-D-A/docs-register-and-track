package services

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"docflow/internal/mocks"
	"docflow/internal/models"
	"docflow/internal/security"
)

func setupNomenclatureService(t *testing.T, role string) (*NomenclatureService, *mocks.NomenclatureStore, *AuthService) {
	t.Helper()
	nomRepo := mocks.NewNomenclatureStore(t)
	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)

	if role != "" {
		password := "Passw0rd!"
		hash, _ := security.HashPassword(password)
		user := &models.User{
			ID:           uuid.New(),
			Login:        role + "_nom",
			PasswordHash: hash,
			IsActive:     true,
			Roles:        []string{role},
		}
		userRepo.On("GetByLogin", user.Login).Return(user, nil).Maybe()
		_, err := auth.Login(user.Login, password)
		require.NoError(t, err)
	}

	svc := NewNomenclatureService(nomRepo, auth)
	return svc, nomRepo, auth
}

func TestNomenclatureService_GetAll(t *testing.T) {
	t.Run("успех", func(t *testing.T) {
		svc, repo, _ := setupNomenclatureService(t, "clerk")
		mockValues := []models.Nomenclature{
			{ID: uuid.New(), Name: "Приказы", Index: "01-01", Direction: "incoming"},
		}
		repo.On("GetAll", 2024, "incoming").Return(mockValues, nil).Once()

		result, err := svc.GetAll(2024, "incoming")
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "Приказы", result[0].Name)
	})

	t.Run("не авторизован", func(t *testing.T) {
		svc, _, _ := setupNomenclatureService(t, "")
		result, err := svc.GetAll(2024, "incoming")
		require.Error(t, err)
		assert.Equal(t, ErrNotAuthenticated, err)
		assert.Nil(t, result)
	})
}

func TestNomenclatureService_GetActiveForDirection(t *testing.T) {
	t.Run("успех", func(t *testing.T) {
		svc, repo, _ := setupNomenclatureService(t, "clerk")
		mockValues := []models.Nomenclature{
			{ID: uuid.New(), Name: "Активные приказы", Index: "01-02", Direction: "incoming", IsActive: true},
		}
		currentYear := time.Now().Year()
		repo.On("GetActiveByDirection", "incoming", currentYear).Return(mockValues, nil).Once()

		result, err := svc.GetActiveForDirection("incoming")
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "Активные приказы", result[0].Name)
	})

	t.Run("не авторизован", func(t *testing.T) {
		svc, _, _ := setupNomenclatureService(t, "")
		result, err := svc.GetActiveForDirection("incoming")
		require.Error(t, err)
		assert.Equal(t, ErrNotAuthenticated, err)
		assert.Nil(t, result)
	})
}

func TestNomenclatureService_Create(t *testing.T) {
	t.Run("успех (админ)", func(t *testing.T) {
		svc, repo, _ := setupNomenclatureService(t, "admin")
		name := "Новое дело"
		expected := &models.Nomenclature{ID: uuid.New(), Name: name}
		repo.On("Create", name, "01-03", 2024, "incoming").Return(expected, nil).Once()

		result, err := svc.Create(name, "01-03", 2024, "incoming")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, name, result.Name)
	})

	t.Run("успех (делопроизводитель)", func(t *testing.T) {
		svc, repo, _ := setupNomenclatureService(t, "clerk")
		name := "Новое дело clerk"
		expected := &models.Nomenclature{ID: uuid.New(), Name: name}
		repo.On("Create", name, "01-03", 2024, "incoming").Return(expected, nil).Once()

		result, err := svc.Create(name, "01-03", 2024, "incoming")
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("запрещено (исполнитель)", func(t *testing.T) {
		svc, _, _ := setupNomenclatureService(t, "executor")
		result, err := svc.Create("Test", "idx", 2024, "dir")
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
		assert.Nil(t, result)
	})

	t.Run("ошибка базы", func(t *testing.T) {
		svc, repo, _ := setupNomenclatureService(t, "admin")
		repo.On("Create", "Test", "idx", 2024, "dir").Return(nil, errors.New("db create error")).Once()

		result, err := svc.Create("Test", "idx", 2024, "dir")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db create error")
		assert.Nil(t, result)
	})
}

func TestNomenclatureService_Update(t *testing.T) {
	idStr := uuid.New().String()

	t.Run("успех (clerk)", func(t *testing.T) {
		svc, repo, _ := setupNomenclatureService(t, "clerk")
		name := "Обновлено"
		expected := &models.Nomenclature{ID: uuid.MustParse(idStr), Name: name}
		repo.On("Update", mock.AnythingOfType("uuid.UUID"), name, "01-04", 2024, "outgoing", true).Return(expected, nil).Once()

		result, err := svc.Update(idStr, name, "01-04", 2024, "outgoing", true)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, name, result.Name)
	})

	t.Run("невалидный ID", func(t *testing.T) {
		svc, _, _ := setupNomenclatureService(t, "admin")
		result, err := svc.Update("invalid-uuid", "Тест", "idx", 2024, "dir", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid ID")
		assert.Nil(t, result)
	})

	t.Run("запрещено (исполнитель)", func(t *testing.T) {
		svc, _, _ := setupNomenclatureService(t, "executor")
		result, err := svc.Update(idStr, "Тест", "idx", 2024, "dir", true)
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
		assert.Nil(t, result)
	})
}

func TestNomenclatureService_Delete(t *testing.T) {
	idStr := uuid.New().String()

	t.Run("успех (админ)", func(t *testing.T) {
		svc, repo, _ := setupNomenclatureService(t, "admin")
		repo.On("Delete", mock.AnythingOfType("uuid.UUID")).Return(nil).Once()

		err := svc.Delete(idStr)
		require.NoError(t, err)
	})

	t.Run("невалидный ID", func(t *testing.T) {
		svc, _, _ := setupNomenclatureService(t, "admin")
		err := svc.Delete("invalid-uuid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid ID")
	})

	t.Run("запрещено (делопроизводитель)", func(t *testing.T) { // доступно только админам
		svc, _, _ := setupNomenclatureService(t, "clerk")
		err := svc.Delete(idStr)
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
	})
}
