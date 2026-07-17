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
	auth.SetAccessStore(newRoleMappedDocumentAccessStore(role))

	if role != "" {
		password := "Passw0rd!"
		hash, _ := security.HashPassword(password)
		user := &models.User{
			ID:           uuid.New(),
			Login:        role + "_nom",
			PasswordHash: hash,
			IsActive:     true,
		}
		userRepo.On("GetByLogin", user.Login).Return(user, nil).Maybe()
		_, err := auth.Login(user.Login, password)
		require.NoError(t, err)
		userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()
	}

	svc := NewNomenclatureService(&atomicNomenclatureStore{NomenclatureStore: nomRepo}, auth, nil)
	return svc, nomRepo, auth
}

func setupNomenclatureServiceWithRoles(t *testing.T, roles []string) (*NomenclatureService, *mocks.NomenclatureStore, *AuthService) {
	t.Helper()
	nomRepo := mocks.NewNomenclatureStore(t)
	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)
	auth.SetAccessStore(newRoleMappedDocumentAccessStore(roles...))

	password := "Passw0rd!"
	hash, _ := security.HashPassword(password)
	user := &models.User{
		ID:           uuid.New(),
		Login:        "multi_nom_" + uuid.New().String(),
		PasswordHash: hash,
		IsActive:     true,
	}
	userRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
	_, err := auth.Login(user.Login, password)
	require.NoError(t, err)
	userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	return NewNomenclatureService(&atomicNomenclatureStore{NomenclatureStore: nomRepo}, auth, nil), nomRepo, auth
}

type atomicNomenclatureStore struct {
	*mocks.NomenclatureStore
	effects []models.OutboxEvent
}

func (s *atomicNomenclatureStore) CreateWithOutbox(name, index string, year int, kindCode, separator, numberingMode string, startNumber int, effects []models.OutboxEvent) (*models.Nomenclature, error) {
	s.effects = append([]models.OutboxEvent(nil), effects...)
	return s.NomenclatureStore.Create(name, index, year, kindCode, separator, numberingMode, startNumber)
}

func (s *atomicNomenclatureStore) UpdateWithOutbox(id uuid.UUID, name, index string, year int, kindCode, separator, numberingMode string, isActive bool, effects []models.OutboxEvent) (*models.Nomenclature, error) {
	s.effects = append([]models.OutboxEvent(nil), effects...)
	return s.NomenclatureStore.Update(id, name, index, year, kindCode, separator, numberingMode, isActive)
}

func (s *atomicNomenclatureStore) DeleteWithOutbox(id uuid.UUID, effects []models.OutboxEvent) error {
	s.effects = append([]models.OutboxEvent(nil), effects...)
	return s.NomenclatureStore.Delete(id)
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
		expected := &models.Nomenclature{ID: uuid.New(), Name: name, NextNumber: 12}
		repo.On("Create", name, "01-03", 2024, "incoming_letter", "/", "index_and_number", 12).Return(expected, nil).Once()

		result, err := svc.Create(name, "01-03", 2024, "incoming_letter", "/", "index_and_number", 12)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, name, result.Name)
		assert.Equal(t, 12, result.NextNumber)
	})

	t.Run("запрещено (делопроизводитель)", func(t *testing.T) {
		svc, _, _ := setupNomenclatureService(t, "clerk")
		result, err := svc.Create("Новое дело clerk", "01-03", 2024, "incoming_letter", "/", "index_and_number", 1)
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
		assert.Nil(t, result)
	})

	t.Run("запрещено (исполнитель)", func(t *testing.T) {
		svc, _, _ := setupNomenclatureService(t, "executor")
		result, err := svc.Create("Test", "idx", 2024, "incoming_letter", "/", "index_and_number", 1)
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
		assert.Nil(t, result)
	})

	t.Run("разрешено мульти-ролевому пользователю с ролью admin", func(t *testing.T) {
		svc, repo, _ := setupNomenclatureServiceWithRoles(t, []string{"admin", "executor"})
		repo.On("Create", "Test", "idx", 2024, "incoming_letter", "/", "index_and_number", 1).Return(&models.Nomenclature{
			ID:   uuid.New(),
			Name: "Test",
		}, nil).Once()

		result, err := svc.Create("Test", "idx", 2024, "incoming_letter", "/", "index_and_number", 1)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("номер меньше единицы заменяется дефолтом", func(t *testing.T) {
		svc, repo, _ := setupNomenclatureService(t, "admin")
		repo.On("Create", "Test", "idx", 2024, "incoming_letter", "/", "index_and_number", 1).Return(&models.Nomenclature{
			ID:         uuid.New(),
			Name:       "Test",
			NextNumber: 1,
		}, nil).Once()

		result, err := svc.Create("Test", "idx", 2024, "incoming_letter", "/", "index_and_number", 0)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result.NextNumber)
	})

	t.Run("ошибка базы", func(t *testing.T) {
		svc, repo, _ := setupNomenclatureService(t, "admin")
		repo.On("Create", "Test", "idx", 2024, "incoming_letter", "/", "index_and_number", 1).Return(nil, errors.New("db create error")).Once()

		result, err := svc.Create("Test", "idx", 2024, "incoming_letter", "/", "index_and_number", 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db create error")
		assert.Nil(t, result)
	})
}

func TestNomenclatureServiceCreatePassesAuditEffectToAtomicStore(t *testing.T) {
	svc, repo, _ := setupNomenclatureService(t, "admin")
	repo.On("Create", "Тест", "01-01", 2026, "incoming_letter", "/", "index_and_number", 1).Return(&models.Nomenclature{ID: uuid.New(), Name: "Тест"}, nil).Once()

	_, err := svc.Create("Тест", "01-01", 2026, "incoming_letter", "/", "index_and_number", 1)
	require.NoError(t, err)
	atomicRepo := svc.repo.(*atomicNomenclatureStore)
	require.Len(t, atomicRepo.effects, 1)
	assert.Equal(t, models.OutboxEventAudit, atomicRepo.effects[0].EventType)
}

func TestNomenclatureService_Update(t *testing.T) {
	// Обновление атрибутов дела в номенклатуре
	idStr := uuid.New().String()

	t.Run("запрещено (clerk)", func(t *testing.T) {
		svc, _, _ := setupNomenclatureService(t, "clerk")
		result, err := svc.Update(idStr, "Обновлено", "01-04", 2024, "outgoing_letter", "-", "number_only", true)
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
		assert.Nil(t, result)
	})

	t.Run("невалидный ID", func(t *testing.T) {
		svc, _, _ := setupNomenclatureService(t, "admin")
		result, err := svc.Update("invalid-uuid", "Тест", "idx", 2024, "incoming_letter", "/", "index_and_number", true)
		require.Error(t, err)
		requireAppError(t, err, "VALIDATION_ERROR", 400, "неверный ID номенклатуры")
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
		requireAppError(t, err, "VALIDATION_ERROR", 400, "неверный ID номенклатуры")
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
