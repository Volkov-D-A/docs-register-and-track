package services

import (
	"docflow/internal/mocks"
	"docflow/internal/models"
	"docflow/internal/security"
	"encoding/base64"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupAttachmentService(t *testing.T, role string) (
	*AttachmentService, *mocks.AttachmentStore, *mocks.SettingsStore, *AuthService,
) {
	t.Helper()
	attachRepo := mocks.NewAttachmentStore(t)
	settingsRepo := mocks.NewSettingsStore(t)
	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)

	password := "Passw0rd!"
	hash, _ := security.HashPassword(password)
	user := &models.User{
		ID:           uuid.New(),
		Login:        role + "_att",
		PasswordHash: hash,
		FullName:     "Test User",
		IsActive:     true,
		Roles:        []string{role},
	}
	userRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
	_, err := auth.Login(user.Login, password)
	require.NoError(t, err)

	settingsSvc := NewSettingsService(nil, settingsRepo, auth)
	journalRepo := mocks.NewJournalStore(t)
	journalRepo.On("Create", mock.Anything, mock.Anything).Return(uuid.Nil, nil).Maybe()
	journalSvc := NewJournalService(journalRepo, auth)

	svc := NewAttachmentService(attachRepo, settingsSvc, auth, journalSvc)
	return svc, attachRepo, settingsRepo, auth
}

func setupAttachmentServiceNotAuth(t *testing.T) *AttachmentService {
	t.Helper()
	attachRepo := mocks.NewAttachmentStore(t)
	settingsRepo := mocks.NewSettingsStore(t)
	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)
	settingsSvc := NewSettingsService(nil, settingsRepo, auth)
	journalRepo := mocks.NewJournalStore(t)
	journalRepo.On("Create", mock.Anything, mock.Anything).Return(uuid.Nil, nil).Maybe()
	journalSvc := NewJournalService(journalRepo, auth)

	return NewAttachmentService(attachRepo, settingsSvc, auth, journalSvc)
}

func TestAttachmentService_Upload(t *testing.T) {
	// Загрузка нового файла вложения к документу (проверка размера, типа и сохранение)
	docID := uuid.New()
	content := []byte("Hello, world!")
	b64 := base64.StdEncoding.EncodeToString(content)

	t.Run("success", func(t *testing.T) {
		svc, repo, settingsRepo, _ := setupAttachmentService(t, "clerk")

		settingsRepo.On("Get", "max_file_size_mb").Return(
			&models.SystemSetting{Key: "max_file_size_mb", Value: "10"}, nil,
		).Once()
		settingsRepo.On("Get", "allowed_file_types").Return(
			&models.SystemSetting{Key: "allowed_file_types", Value: ".pdf,.doc,.txt"}, nil,
		).Once()
		repo.On("Create", mock.AnythingOfType("*models.Attachment")).Return(nil).Once()

		result, err := svc.Upload(docID.String(), "incoming", "test.txt", b64)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "test.txt", result.Filename)
	})

	t.Run("not authenticated", func(t *testing.T) {
		svc := setupAttachmentServiceNotAuth(t)
		result, err := svc.Upload(docID.String(), "incoming", "test.txt", b64)
		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("invalid base64", func(t *testing.T) {
		svc, _, _, _ := setupAttachmentService(t, "clerk")

		result, err := svc.Upload(docID.String(), "incoming", "test.txt", "!!!not-base64!!!")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode")
		assert.Nil(t, result)
	})

	t.Run("file too large", func(t *testing.T) {
		svc, _, settingsRepo, _ := setupAttachmentService(t, "clerk")

		// Max 1 byte
		settingsRepo.On("Get", "max_file_size_mb").Return(
			&models.SystemSetting{Key: "max_file_size_mb", Value: "0"}, nil,
		).Once()

		result, err := svc.Upload(docID.String(), "incoming", "test.txt", b64)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum")
		assert.Nil(t, result)
	})

	t.Run("forbidden extension", func(t *testing.T) {
		svc, _, settingsRepo, _ := setupAttachmentService(t, "clerk")

		settingsRepo.On("Get", "max_file_size_mb").Return(
			&models.SystemSetting{Key: "max_file_size_mb", Value: "10"}, nil,
		).Once()
		settingsRepo.On("Get", "allowed_file_types").Return(
			&models.SystemSetting{Key: "allowed_file_types", Value: ".pdf,.doc"}, nil,
		).Once()

		result, err := svc.Upload(docID.String(), "incoming", "virus.exe", b64)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not allowed")
		assert.Nil(t, result)
	})
}

func TestAttachmentService_GetList(t *testing.T) {
	// Получение списка всех вложений для заданного документа
	docID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc, repo, _, _ := setupAttachmentService(t, "executor")
		attachments := []models.Attachment{{ID: uuid.New(), DocumentID: docID, Filename: "f.pdf"}}
		repo.On("GetByDocumentID", docID).Return(attachments, nil).Once()

		result, err := svc.GetList(docID.String())
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("invalid document ID", func(t *testing.T) {
		svc, _, _, _ := setupAttachmentService(t, "executor")
		result, err := svc.GetList("not-a-uuid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid document ID")
		assert.Nil(t, result)
	})
}

func TestAttachmentService_Download(t *testing.T) {
	// Скачивание (выгрузка) содержимого файла вложения
	attID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc, repo, _, _ := setupAttachmentService(t, "executor")
		att := &models.Attachment{ID: attID, Filename: "file.pdf"}
		content := []byte("pdf-content")

		repo.On("GetByID", attID).Return(att, nil).Once()
		repo.On("GetContent", attID).Return(content, nil).Once()

		result, err := svc.Download(attID.String())
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "file.pdf", result.Filename)
		assert.Equal(t, base64.StdEncoding.EncodeToString(content), result.Content)
	})

	t.Run("invalid ID", func(t *testing.T) {
		svc, _, _, _ := setupAttachmentService(t, "executor")
		result, err := svc.Download("not-a-uuid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid attachment ID")
		assert.Nil(t, result)
	})
}

func TestAttachmentService_Delete(t *testing.T) {
	// Удаление файла вложения
	attID := uuid.New()

	t.Run("success clerk", func(t *testing.T) {
		svc, repo, _, _ := setupAttachmentService(t, "clerk")
		repo.On("GetByID", attID).Return(&models.Attachment{
			ID:           attID,
			DocumentID:   uuid.New(),
			DocumentType: "incoming",
		}, nil).Once()
		repo.On("Delete", attID).Return(nil).Once()
		err := svc.Delete(attID.String())
		require.NoError(t, err)
	})

	t.Run("forbidden executor", func(t *testing.T) {
		svc, _, _, _ := setupAttachmentService(t, "executor")
		err := svc.Delete(attID.String())
		require.Error(t, err)
	})
}

func TestAttachmentService_ValidatePathInDownloads(t *testing.T) {
	// Проверка пути доступа к файлу для защиты от уязвимости Path Traversal
	t.Run("valid path", func(t *testing.T) {
		svc, _, _, _ := setupAttachmentService(t, "executor")
		downloadDir, _ := svc.getDownloadDir()
		err := svc.validatePathInDownloads(downloadDir + "/test.pdf")
		require.NoError(t, err)
	})

	t.Run("path traversal attack", func(t *testing.T) {
		svc, _, _, _ := setupAttachmentService(t, "executor")
		err := svc.validatePathInDownloads("C:\\Windows\\System32\\..\\..\\test.pdf")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("outside downloads", func(t *testing.T) {
		svc, _, _, _ := setupAttachmentService(t, "executor")
		err := svc.validatePathInDownloads("C:\\Windows\\System32\\cmd.exe")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})
}
