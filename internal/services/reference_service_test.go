package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
)

func setupReferenceService(t *testing.T, role string) (*ReferenceService, *mocks.ReferenceStore, *mocks.UserStore, *AuthService) {
	t.Helper()
	refRepo := mocks.NewReferenceStore(t)
	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)
	auth.SetAccessStore(newRoleMappedDocumentAccessStore(role))

	if role != "" {
		password := "Passw0rd!"
		hash, _ := security.HashPassword(password)
		user := &models.User{
			ID:           uuid.New(),
			Login:        role + "_ref",
			PasswordHash: hash,
			IsActive:     true,
		}
		userRepo.On("GetByLogin", user.Login).Return(user, nil).Maybe()
		_, err := auth.Login(user.Login, password)
		require.NoError(t, err)
		userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()
	}

	svc := NewReferenceService(refRepo, auth, nil)
	return svc, refRepo, userRepo, auth
}

func setupReferenceServiceWithRoles(t *testing.T, roles []string) (*ReferenceService, *mocks.ReferenceStore, *mocks.UserStore, *AuthService) {
	t.Helper()
	refRepo := mocks.NewReferenceStore(t)
	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)
	auth.SetAccessStore(newRoleMappedDocumentAccessStore(roles...))

	password := "Passw0rd!"
	hash, _ := security.HashPassword(password)
	user := &models.User{
		ID:           uuid.New(),
		Login:        "multi_ref_" + uuid.New().String(),
		PasswordHash: hash,
		IsActive:     true,
	}
	userRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
	_, err := auth.Login(user.Login, password)
	require.NoError(t, err)
	userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	return NewReferenceService(refRepo, auth, nil), refRepo, userRepo, auth
}

// === Типы документов ===

func TestReferenceService_GetDocumentTypes(t *testing.T) {
	// Получение списка всех типов документов из кода.
	t.Run("успех (авторизован)", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, "clerk")

		result, err := svc.GetDocumentTypes()
		require.NoError(t, err)
		assert.Len(t, result, len(models.AllowedDocumentTypes()))
		assert.Equal(t, models.DocumentTypeLetter, result[0].Name)
		assert.Equal(t, models.DocumentTypeLetter, result[0].ID)
	})

	t.Run("не авторизован", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, "") // без входа
		result, err := svc.GetDocumentTypes()
		require.Error(t, err)
		assert.Equal(t, ErrNotAuthenticated, err)
		assert.Nil(t, result)
	})
}

func TestReferenceService_CreateDocumentType(t *testing.T) {
	// Типы документов заданы в коде и не редактируются через сервис.
	t.Run("запрещено", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, "clerk")
		result, err := svc.CreateDocumentType("Test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "не редактируются")
		assert.Nil(t, result)
	})
}

func TestReferenceService_UpdateDocumentType(t *testing.T) {
	// Типы документов заданы в коде и не редактируются через сервис.
	t.Run("запрещено", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, "clerk")
		err := svc.UpdateDocumentType("Письмо", "Test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "не редактируются")
	})
}

func TestReferenceService_DeleteDocumentType(t *testing.T) {
	// Типы документов заданы в коде и не редактируются через сервис.
	t.Run("запрещено", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, "clerk")
		err := svc.DeleteDocumentType("Письмо")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "не редактируются")
	})
}

// === Организации ===

func TestReferenceService_GetOrganizations(t *testing.T) {
	// Выгрузка полного списка организаций-корреспондентов
	t.Run("успех", func(t *testing.T) {
		svc, repo, _, _ := setupReferenceService(t, "clerk")
		mockValues := []models.Organization{
			{ID: uuid.New(), Name: "Орг 1"},
			{ID: uuid.New(), Name: "Орг 2"},
		}
		repo.On("GetAllOrganizations").Return(mockValues, nil).Once()

		result, err := svc.GetOrganizations()
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("не авторизован", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, "")
		result, err := svc.GetOrganizations()
		require.Error(t, err)
		assert.Equal(t, ErrNotAuthenticated, err)
		assert.Nil(t, result)
	})
}

func TestReferenceService_SearchOrganizations(t *testing.T) {
	// Поиск внешних организаций по частичному совпадению названия (поиск для поля ввода)
	t.Run("успех", func(t *testing.T) {
		svc, repo, _, _ := setupReferenceService(t, "clerk")
		mockValues := []models.Organization{
			{ID: uuid.New(), Name: "Поиск Орг"},
		}
		repo.On("SearchOrganizations", "Поиск").Return(mockValues, nil).Once()

		result, err := svc.SearchOrganizations("Поиск")
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "Поиск Орг", result[0].Name)
	})

	t.Run("не авторизован", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, "")
		result, err := svc.SearchOrganizations("Test")
		require.Error(t, err)
		assert.Equal(t, ErrNotAuthenticated, err)
		assert.Nil(t, result)
	})
}

func TestReferenceService_FindOrCreateOrganization(t *testing.T) {
	// Поиск организации по точному названию или её автоматическое добавление в справочник
	t.Run("успех", func(t *testing.T) {
		svc, repo, _, _ := setupReferenceService(t, "clerk")
		name := "Новая Орг"
		expected := &models.Organization{ID: uuid.New(), Name: name}
		repo.On("FindOrCreateOrganization", name).Return(expected, nil).Once()

		result, err := svc.FindOrCreateOrganization(name)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, name, result.Name)
	})

	t.Run("не авторизован", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, "")
		result, err := svc.FindOrCreateOrganization("Test")
		require.Error(t, err)
		assert.Equal(t, ErrNotAuthenticated, err)
		assert.Nil(t, result)
	})
}

func TestReferenceService_UpdateOrganization(t *testing.T) {
	// Изменение официального названия организации-корреспондента
	idStr := uuid.New().String()

	t.Run("невалидный ID", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, models.SystemPermissionReferences)
		err := svc.UpdateOrganization("invalid-uuid", "Тест")
		require.Error(t, err)
		requireAppError(t, err, "VALIDATION_ERROR", 400, "неверный ID записи справочника")
	})

	t.Run("запрещено (не админ)", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, "clerk")
		err := svc.UpdateOrganization(idStr, "Тест")
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
	})

	t.Run("разрешено пользователю с системным правом references", func(t *testing.T) {
		svc, repo, _, _ := setupReferenceService(t, models.SystemPermissionReferences)
		repo.On("UpdateOrganization", mock.AnythingOfType("uuid.UUID"), "Тест").Return(nil).Once()

		err := svc.UpdateOrganization(idStr, "Тест")
		require.NoError(t, err)
	})
}

func TestReferenceService_DeleteOrganization(t *testing.T) {
	// Удаление организации из справочника контрагентов
	idStr := uuid.New().String()

	t.Run("невалидный ID", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, models.SystemPermissionReferences)
		err := svc.DeleteOrganization("invalid-uuid")
		require.Error(t, err)
		requireAppError(t, err, "VALIDATION_ERROR", 400, "неверный ID записи справочника")
	})

	t.Run("запрещено (не админ)", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, "clerk")
		err := svc.DeleteOrganization(idStr)
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
	})

	t.Run("разрешено пользователю с системным правом references", func(t *testing.T) {
		svc, repo, _, _ := setupReferenceService(t, models.SystemPermissionReferences)
		repo.On("DeleteOrganization", mock.AnythingOfType("uuid.UUID")).Return(nil).Once()

		err := svc.DeleteOrganization(idStr)
		require.NoError(t, err)
	})
}

func TestReferenceService_MergeOrganizations(t *testing.T) {
	sourceID := uuid.New()
	targetID := uuid.New()

	t.Run("невалидный ID исходной организации", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, models.SystemPermissionReferences)
		err := svc.MergeOrganizations("invalid-uuid", targetID.String())
		require.Error(t, err)
		requireAppError(t, err, "VALIDATION_ERROR", 400, "неверный ID исходной организации")
	})

	t.Run("невалидный ID целевой организации", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, models.SystemPermissionReferences)
		err := svc.MergeOrganizations(sourceID.String(), "invalid-uuid")
		require.Error(t, err)
		requireAppError(t, err, "VALIDATION_ERROR", 400, "неверный ID целевой организации")
	})

	t.Run("запрещено (не админ)", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, "clerk")
		err := svc.MergeOrganizations(sourceID.String(), targetID.String())
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
	})

	t.Run("нельзя объединить саму с собой", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, models.SystemPermissionReferences)
		err := svc.MergeOrganizations(sourceID.String(), sourceID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "саму с собой")
	})

	t.Run("разрешено пользователю с системным правом references", func(t *testing.T) {
		svc, repo, _, _ := setupReferenceService(t, models.SystemPermissionReferences)
		repo.On("MergeOrganizations", sourceID, targetID).Return(nil).Once()

		err := svc.MergeOrganizations(sourceID.String(), targetID.String())
		require.NoError(t, err)
	})
}

// === Исполнители резолюции ===

func TestReferenceService_GetResolutionExecutors(t *testing.T) {
	t.Run("успех", func(t *testing.T) {
		svc, repo, _, _ := setupReferenceService(t, "clerk")
		mockValues := []models.ResolutionExecutor{
			{ID: uuid.New(), Name: "Исполнитель 1"},
			{ID: uuid.New(), Name: "Исполнитель 2"},
		}
		repo.On("GetAllResolutionExecutors").Return(mockValues, nil).Once()

		result, err := svc.GetResolutionExecutors()

		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, "Исполнитель 1", result[0].Name)
	})

	t.Run("не авторизован", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, "")

		result, err := svc.GetResolutionExecutors()

		require.ErrorIs(t, err, ErrNotAuthenticated)
		assert.Nil(t, result)
	})
}

func TestReferenceService_SearchResolutionExecutors(t *testing.T) {
	t.Run("успех", func(t *testing.T) {
		svc, repo, _, _ := setupReferenceService(t, "clerk")
		mockValues := []models.ResolutionExecutor{{ID: uuid.New(), Name: "Исполнитель"}}
		repo.On("SearchResolutionExecutors", "Исп").Return(mockValues, nil).Once()

		result, err := svc.SearchResolutionExecutors("Исп")

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "Исполнитель", result[0].Name)
	})

	t.Run("не авторизован", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, "")

		result, err := svc.SearchResolutionExecutors("Исп")

		require.ErrorIs(t, err, ErrNotAuthenticated)
		assert.Nil(t, result)
	})
}

func TestReferenceService_FindOrCreateResolutionExecutor(t *testing.T) {
	t.Run("успех", func(t *testing.T) {
		svc, repo, _, _ := setupReferenceService(t, "clerk")
		name := "Новый исполнитель"
		expected := &models.ResolutionExecutor{ID: uuid.New(), Name: name}
		repo.On("FindOrCreateResolutionExecutor", name).Return(expected, nil).Once()

		result, err := svc.FindOrCreateResolutionExecutor(name)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, name, result.Name)
	})

	t.Run("не авторизован", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, "")

		result, err := svc.FindOrCreateResolutionExecutor("Исполнитель")

		require.ErrorIs(t, err, ErrNotAuthenticated)
		assert.Nil(t, result)
	})
}

func TestReferenceService_UpdateResolutionExecutor(t *testing.T) {
	idStr := uuid.New().String()

	t.Run("невалидный ID", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, models.SystemPermissionReferences)

		err := svc.UpdateResolutionExecutor("invalid-uuid", "Тест")

		require.Error(t, err)
		requireAppError(t, err, "VALIDATION_ERROR", 400, "неверный ID записи справочника")
	})

	t.Run("запрещено", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, "clerk")

		err := svc.UpdateResolutionExecutor(idStr, "Тест")

		require.ErrorIs(t, err, models.ErrForbidden)
	})

	t.Run("разрешено пользователю с системным правом references", func(t *testing.T) {
		svc, repo, _, _ := setupReferenceService(t, models.SystemPermissionReferences)
		repo.On("UpdateResolutionExecutor", mock.AnythingOfType("uuid.UUID"), "Тест").Return(nil).Once()

		err := svc.UpdateResolutionExecutor(idStr, "Тест")

		require.NoError(t, err)
	})
}

func TestReferenceService_DeleteResolutionExecutor(t *testing.T) {
	idStr := uuid.New().String()

	t.Run("невалидный ID", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, models.SystemPermissionReferences)

		err := svc.DeleteResolutionExecutor("invalid-uuid")

		require.Error(t, err)
		requireAppError(t, err, "VALIDATION_ERROR", 400, "неверный ID записи справочника")
	})

	t.Run("запрещено", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, "clerk")

		err := svc.DeleteResolutionExecutor(idStr)

		require.ErrorIs(t, err, models.ErrForbidden)
	})

	t.Run("разрешено пользователю с системным правом references", func(t *testing.T) {
		svc, repo, _, _ := setupReferenceService(t, models.SystemPermissionReferences)
		repo.On("DeleteResolutionExecutor", mock.AnythingOfType("uuid.UUID")).Return(nil).Once()

		err := svc.DeleteResolutionExecutor(idStr)

		require.NoError(t, err)
	})
}
