package services

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
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
		userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()
	}

	svc := NewNomenclatureService(nomRepo, auth, nil)
	return svc, nomRepo, auth
}

func setupNomenclatureServiceWithRoles(t *testing.T, roles []string) (*NomenclatureService, *mocks.NomenclatureStore, *AuthService) {
	t.Helper()
	nomRepo := mocks.NewNomenclatureStore(t)
	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)

	password := "Passw0rd!"
	hash, _ := security.HashPassword(password)
	user := &models.User{
		ID:           uuid.New(),
		Login:        "multi_nom_" + uuid.New().String(),
		PasswordHash: hash,
		IsActive:     true,
		Roles:        roles,
	}
	userRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
	_, err := auth.Login(user.Login, password)
	require.NoError(t, err)
	userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	return NewNomenclatureService(nomRepo, auth, nil), nomRepo, auth
}

func TestNomenclatureService_GetAll(t *testing.T) {
	// Получение полного списка дел номенклатуры с учетом прав доступа
	t.Run("успех", func(t *testing.T) {
		svc, repo, _ := setupNomenclatureService(t, "clerk")
		mockValues := []models.Nomenclature{
			{ID: uuid.New(), Name: "Приказы", Index: "01-01", KindCode: "incoming_letter", Separator: "/", NumberingMode: "index_and_number"},
		}
		repo.On("GetAll", 2024, "incoming_letter").Return(mockValues, nil).Once()

		result, err := svc.GetAll(2024, "incoming_letter")
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "Приказы", result[0].Name)
	})

	t.Run("не авторизован", func(t *testing.T) {
		svc, _, _ := setupNomenclatureService(t, "")
		result, err := svc.GetAll(2024, "incoming_letter")
		require.Error(t, err)
		assert.Equal(t, ErrNotAuthenticated, err)
		assert.Nil(t, result)
	})
}

func TestNomenclatureService_GetActiveForKind(t *testing.T) {
	// Получение списка только активных дел для определенного направления
	t.Run("успех", func(t *testing.T) {
		svc, repo, _ := setupNomenclatureService(t, "clerk")
		mockValues := []models.Nomenclature{
			{ID: uuid.New(), Name: "Активные приказы", Index: "01-02", KindCode: "incoming_letter", Separator: "/", NumberingMode: "index_and_number", IsActive: true},
		}
		currentYear := time.Now().Year()
		repo.On("GetActiveByKind", "incoming_letter", currentYear).Return(mockValues, nil).Once()

		result, err := svc.GetActiveForKind("incoming_letter")
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "Активные приказы", result[0].Name)
	})

	t.Run("не авторизован", func(t *testing.T) {
		svc, _, _ := setupNomenclatureService(t, "")
		result, err := svc.GetActiveForKind("incoming_letter")
		require.Error(t, err)
		assert.Equal(t, ErrNotAuthenticated, err)
		assert.Nil(t, result)
	})
}

func TestNomenclatureService_Create(t *testing.T) {
	// Создание нового дела в номенклатуре дел
	t.Run("успех (админ)", func(t *testing.T) {
		svc, repo, _ := setupNomenclatureService(t, "admin")
		name := "Новое дело"
		expected := &models.Nomenclature{ID: uuid.New(), Name: name}
		repo.On("Create", name, "01-03", 2024, "incoming_letter", "/", "index_and_number").Return(expected, nil).Once()

		result, err := svc.Create(name, "01-03", 2024, "incoming_letter", "/", "index_and_number")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, name, result.Name)
	})

	t.Run("успех (делопроизводитель)", func(t *testing.T) {
		svc, repo, _ := setupNomenclatureService(t, "clerk")
		name := "Новое дело clerk"
		expected := &models.Nomenclature{ID: uuid.New(), Name: name}
		repo.On("Create", name, "01-03", 2024, "incoming_letter", "/", "index_and_number").Return(expected, nil).Once()

		result, err := svc.Create(name, "01-03", 2024, "incoming_letter", "/", "index_and_number")
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("запрещено (исполнитель)", func(t *testing.T) {
		svc, _, _ := setupNomenclatureService(t, "executor")
		result, err := svc.Create("Test", "idx", 2024, "incoming_letter", "/", "index_and_number")
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
		assert.Nil(t, result)
	})

	t.Run("разрешено мульти-ролевому пользователю с ролью admin", func(t *testing.T) {
		svc, repo, _ := setupNomenclatureServiceWithRoles(t, []string{"admin", "executor"})
		repo.On("Create", "Test", "idx", 2024, "incoming_letter", "/", "index_and_number").Return(&models.Nomenclature{
			ID:   uuid.New(),
			Name: "Test",
		}, nil).Once()

		result, err := svc.Create("Test", "idx", 2024, "incoming_letter", "/", "index_and_number")
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("ошибка базы", func(t *testing.T) {
		svc, repo, _ := setupNomenclatureService(t, "admin")
		repo.On("Create", "Test", "idx", 2024, "incoming_letter", "/", "index_and_number").Return(nil, errors.New("db create error")).Once()

		result, err := svc.Create("Test", "idx", 2024, "incoming_letter", "/", "index_and_number")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db create error")
		assert.Nil(t, result)
	})
}

func TestNomenclatureService_Update(t *testing.T) {
	// Обновление атрибутов дела в номенклатуре
	idStr := uuid.New().String()

	t.Run("успех (clerk)", func(t *testing.T) {
		svc, repo, _ := setupNomenclatureService(t, "clerk")
		name := "Обновлено"
		expected := &models.Nomenclature{ID: uuid.MustParse(idStr), Name: name}
		repo.On("Update", mock.AnythingOfType("uuid.UUID"), name, "01-04", 2024, "outgoing_letter", "-", "number_only", true).Return(expected, nil).Once()

		result, err := svc.Update(idStr, name, "01-04", 2024, "outgoing_letter", "-", "number_only", true)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, name, result.Name)
	})

	t.Run("невалидный ID", func(t *testing.T) {
		svc, _, _ := setupNomenclatureService(t, "admin")
		result, err := svc.Update("invalid-uuid", "Тест", "idx", 2024, "incoming_letter", "/", "index_and_number", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid ID")
		assert.Nil(t, result)
	})

	t.Run("запрещено (исполнитель)", func(t *testing.T) {
		svc, _, _ := setupNomenclatureService(t, "executor")
		result, err := svc.Update(idStr, "Тест", "idx", 2024, "incoming_letter", "/", "index_and_number", true)
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
		assert.Nil(t, result)
	})

	t.Run("разрешено мульти-ролевому пользователю с ролью admin", func(t *testing.T) {
		svc, repo, _ := setupNomenclatureServiceWithRoles(t, []string{"admin", "executor"})
		repo.On("Update", mock.AnythingOfType("uuid.UUID"), "Тест", "idx", 2024, "incoming_letter", "/", "index_and_number", true).Return(&models.Nomenclature{
			ID:   uuid.MustParse(idStr),
			Name: "Тест",
		}, nil).Once()

		result, err := svc.Update(idStr, "Тест", "idx", 2024, "incoming_letter", "/", "index_and_number", true)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestNomenclatureService_Delete(t *testing.T) {
	// Удаление дела из номенклатуры (если оно больше не используется)
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

	t.Run("разрешено пользователю с ролью admin", func(t *testing.T) {
		svc, repo, _ := setupNomenclatureServiceWithRoles(t, []string{"admin", "clerk"})
		repo.On("Delete", mock.AnythingOfType("uuid.UUID")).Return(nil).Once()

		err := svc.Delete(idStr)
		require.NoError(t, err)
	})
}
