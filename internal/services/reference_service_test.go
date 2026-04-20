package services

import (
	"errors"
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
	// Получение списка всех типов документов из справочника
	t.Run("успех (авторизован)", func(t *testing.T) {
		svc, repo, _, _ := setupReferenceService(t, "clerk")
		mockValues := []models.DocumentType{
			{ID: uuid.New(), Name: "Приказ"},
			{ID: uuid.New(), Name: "Письмо"},
		}
		repo.On("GetAllDocumentTypes").Return(mockValues, nil).Once()

		result, err := svc.GetDocumentTypes()
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "Приказ", result[0].Name)
	})

	t.Run("ошибка базы", func(t *testing.T) {
		svc, repo, _, _ := setupReferenceService(t, "clerk")
		repo.On("GetAllDocumentTypes").Return(nil, errors.New("db error")).Once()

		result, err := svc.GetDocumentTypes()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
		assert.Nil(t, result)
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
	// Запись нового типа документа в справочник
	t.Run("запрещено (не админ)", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, "clerk")
		result, err := svc.CreateDocumentType("Test")
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
		assert.Nil(t, result)
	})

	t.Run("разрешено пользователю с системным правом references", func(t *testing.T) {
		svc, repo, _, _ := setupReferenceService(t, models.SystemPermissionReferences)
		repo.On("CreateDocumentType", "Test").Return(&models.DocumentType{ID: uuid.New(), Name: "Test"}, nil).Once()

		result, err := svc.CreateDocumentType("Test")
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("администратору без references запрещено", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, models.SystemPermissionAdmin)

		result, err := svc.CreateDocumentType("Test")
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
		assert.Nil(t, result)
	})
}

func TestReferenceService_UpdateDocumentType(t *testing.T) {
	// Переименование существующего типа документа
	idStr := uuid.New().String()

	t.Run("невалидный ID", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, models.SystemPermissionReferences)
		err := svc.UpdateDocumentType("invalid-uuid", "Новое имя")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid ID")
	})

	t.Run("запрещено (не админ)", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, "clerk")
		err := svc.UpdateDocumentType(idStr, "Test")
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
	})

	t.Run("разрешено пользователю с системным правом references", func(t *testing.T) {
		svc, repo, _, _ := setupReferenceService(t, models.SystemPermissionReferences)
		repo.On("UpdateDocumentType", mock.AnythingOfType("uuid.UUID"), "Test").Return(nil).Once()

		err := svc.UpdateDocumentType(idStr, "Test")
		require.NoError(t, err)
	})
}

func TestReferenceService_DeleteDocumentType(t *testing.T) {
	// Физическое удаление типа документа из справочника
	idStr := uuid.New().String()

	t.Run("невалидный ID", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, models.SystemPermissionReferences)
		err := svc.DeleteDocumentType("invalid-uuid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid ID")
	})

	t.Run("запрещено (не админ)", func(t *testing.T) {
		svc, _, _, _ := setupReferenceService(t, "clerk")
		err := svc.DeleteDocumentType(idStr)
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
	})

	t.Run("разрешено пользователю с системным правом references", func(t *testing.T) {
		svc, repo, _, _ := setupReferenceService(t, models.SystemPermissionReferences)
		repo.On("DeleteDocumentType", mock.AnythingOfType("uuid.UUID")).Return(nil).Once()

		err := svc.DeleteDocumentType(idStr)
		require.NoError(t, err)
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
		assert.Contains(t, err.Error(), "invalid ID")
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
		assert.Contains(t, err.Error(), "invalid ID")
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
