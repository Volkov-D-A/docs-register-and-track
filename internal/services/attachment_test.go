package services

import (
	"encoding/base64"
	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupAttachmentService(t *testing.T, role string) (
	*AttachmentService, *mocks.AttachmentStore, *mocks.SettingsStore, *mocks.FileStorage, *mocks.IncomingDocStore, *mocks.OutgoingDocStore, *mocks.DepartmentStore, *mocks.AssignmentStore, *mocks.AcknowledgmentStore, *mocks.UserStore, *AuthService,
) {
	t.Helper()
	attachRepo := mocks.NewAttachmentStore(t)
	settingsRepo := mocks.NewSettingsStore(t)
	fileStorage := mocks.NewFileStorage(t)
	incomingRepo := mocks.NewIncomingDocStore(t)
	outgoingRepo := mocks.NewOutgoingDocStore(t)
	incomingRepo.On("GetByID", mock.Anything).Return(func(id uuid.UUID) *models.IncomingDocument {
		return &models.IncomingDocument{ID: id, NomenclatureID: uuid.New()}
	}, nil).Maybe()
	outgoingRepo.On("GetByID", mock.Anything).Return(func(id uuid.UUID) *models.OutgoingDocument {
		return &models.OutgoingDocument{ID: id, NomenclatureID: uuid.New()}
	}, nil).Maybe()
	depRepo := mocks.NewDepartmentStore(t)
	assignmentRepo := mocks.NewAssignmentStore(t)
	ackRepo := mocks.NewAcknowledgmentStore(t)
	assignmentRepo.On("HasDocumentAccess", mock.Anything, mock.Anything).Return(true, nil).Maybe()
	ackRepo.On("HasDocumentAccess", mock.Anything, mock.Anything).Return(true, nil).Maybe()
	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)
	auth.SetAccessStore(newRoleMappedDocumentAccessStore(role))

	password := "Passw0rd!"
	hash, _ := security.HashPassword(password)
	user := &models.User{
		ID:                    uuid.New(),
		Login:                 role + "_att",
		PasswordHash:          hash,
		FullName:              "Test User",
		IsDocumentParticipant: role != "admin",
		IsActive:              true,
	}
	userRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
	_, err := auth.Login(user.Login, password)
	require.NoError(t, err)
	userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	settingsSvc := NewSettingsService(nil, settingsRepo, auth, nil)
	journalRepo := mocks.NewJournalStore(t)
	journalRepo.On("Create", mock.Anything, mock.Anything).Return(uuid.Nil, nil).Maybe()
	accessSvc := NewDocumentAccessService(auth, depRepo, assignmentRepo, ackRepo, newRoleMappedDocumentAccessStore(role), nil, incomingRepo, outgoingRepo)
	journalSvc := NewJournalService(journalRepo, auth, accessSvc)

	svc := NewAttachmentService(attachRepo, settingsSvc, auth, journalSvc, nil, fileStorage, accessSvc)
	return svc, attachRepo, settingsRepo, fileStorage, incomingRepo, outgoingRepo, depRepo, assignmentRepo, ackRepo, userRepo, auth
}

func setupAttachmentServiceWithRoles(t *testing.T, roles []string) (
	*AttachmentService, *mocks.AttachmentStore, *mocks.SettingsStore, *mocks.FileStorage, *mocks.IncomingDocStore, *mocks.OutgoingDocStore, *mocks.DepartmentStore, *mocks.AssignmentStore, *mocks.AcknowledgmentStore, *mocks.UserStore, *AuthService,
) {
	t.Helper()
	attachRepo := mocks.NewAttachmentStore(t)
	settingsRepo := mocks.NewSettingsStore(t)
	fileStorage := mocks.NewFileStorage(t)
	incomingRepo := mocks.NewIncomingDocStore(t)
	outgoingRepo := mocks.NewOutgoingDocStore(t)
	incomingRepo.On("GetByID", mock.Anything).Return(func(id uuid.UUID) *models.IncomingDocument {
		return &models.IncomingDocument{ID: id, NomenclatureID: uuid.New()}
	}, nil).Maybe()
	outgoingRepo.On("GetByID", mock.Anything).Return(func(id uuid.UUID) *models.OutgoingDocument {
		return &models.OutgoingDocument{ID: id, NomenclatureID: uuid.New()}
	}, nil).Maybe()
	depRepo := mocks.NewDepartmentStore(t)
	assignmentRepo := mocks.NewAssignmentStore(t)
	ackRepo := mocks.NewAcknowledgmentStore(t)
	assignmentRepo.On("HasDocumentAccess", mock.Anything, mock.Anything).Return(true, nil).Maybe()
	ackRepo.On("HasDocumentAccess", mock.Anything, mock.Anything).Return(true, nil).Maybe()
	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)
	auth.SetAccessStore(newRoleMappedDocumentAccessStore(roles...))

	password := "Passw0rd!"
	hash, _ := security.HashPassword(password)
	user := &models.User{
		ID:                    uuid.New(),
		Login:                 "multi_att_" + uuid.New().String(),
		PasswordHash:          hash,
		FullName:              "Test User",
		IsDocumentParticipant: true,
		IsActive:              true,
	}
	userRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
	_, err := auth.Login(user.Login, password)
	require.NoError(t, err)
	userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	settingsSvc := NewSettingsService(nil, settingsRepo, auth, nil)
	journalRepo := mocks.NewJournalStore(t)
	journalRepo.On("Create", mock.Anything, mock.Anything).Return(uuid.Nil, nil).Maybe()
	accessSvc := NewDocumentAccessService(auth, depRepo, assignmentRepo, ackRepo, newRoleMappedDocumentAccessStore(roles...), nil, incomingRepo, outgoingRepo)
	journalSvc := NewJournalService(journalRepo, auth, accessSvc)

	svc := NewAttachmentService(attachRepo, settingsSvc, auth, journalSvc, nil, fileStorage, accessSvc)
	return svc, attachRepo, settingsRepo, fileStorage, incomingRepo, outgoingRepo, depRepo, assignmentRepo, ackRepo, userRepo, auth
}

func setupAttachmentServiceNotAuth(t *testing.T) *AttachmentService {
	t.Helper()
	attachRepo := mocks.NewAttachmentStore(t)
	settingsRepo := mocks.NewSettingsStore(t)
	fileStorage := mocks.NewFileStorage(t)
	incomingRepo := mocks.NewIncomingDocStore(t)
	outgoingRepo := mocks.NewOutgoingDocStore(t)
	incomingRepo.On("GetByID", mock.Anything).Return(func(id uuid.UUID) *models.IncomingDocument {
		return &models.IncomingDocument{ID: id, NomenclatureID: uuid.New()}
	}, nil).Maybe()
	outgoingRepo.On("GetByID", mock.Anything).Return(func(id uuid.UUID) *models.OutgoingDocument {
		return &models.OutgoingDocument{ID: id, NomenclatureID: uuid.New()}
	}, nil).Maybe()
	depRepo := mocks.NewDepartmentStore(t)
	assignmentRepo := mocks.NewAssignmentStore(t)
	ackRepo := mocks.NewAcknowledgmentStore(t)
	assignmentRepo.On("HasDocumentAccess", mock.Anything, mock.Anything).Return(true, nil).Maybe()
	ackRepo.On("HasDocumentAccess", mock.Anything, mock.Anything).Return(true, nil).Maybe()
	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)
	settingsSvc := NewSettingsService(nil, settingsRepo, auth, nil)
	journalRepo := mocks.NewJournalStore(t)
	journalRepo.On("Create", mock.Anything, mock.Anything).Return(uuid.Nil, nil).Maybe()
	accessSvc := NewDocumentAccessService(auth, depRepo, assignmentRepo, ackRepo, newRoleMappedDocumentAccessStore(), nil, incomingRepo, outgoingRepo)
	journalSvc := NewJournalService(journalRepo, auth, accessSvc)
	return NewAttachmentService(attachRepo, settingsSvc, auth, journalSvc, nil, fileStorage, accessSvc)
}

func TestAttachmentService_Upload(t *testing.T) {
	// Загрузка нового файла вложения к документу (проверка размера, типа и сохранение)
	docID := uuid.New()
	content := []byte("Hello, world!")
	b64 := base64.StdEncoding.EncodeToString(content)

	t.Run("success", func(t *testing.T) {
		svc, repo, settingsRepo, fileStorage, incomingRepo, _, _, _, _, _, _ := setupAttachmentService(t, "clerk")
		incomingRepo.On("GetByID", docID).Return(&models.IncomingDocument{ID: docID, NomenclatureID: uuid.New()}, nil).Maybe()

		settingsRepo.On("Get", "max_file_size_mb").Return(
			&models.SystemSetting{Key: "max_file_size_mb", Value: "10"}, nil,
		).Once()
		settingsRepo.On("Get", "allowed_file_types").Return(
			&models.SystemSetting{Key: "allowed_file_types", Value: ".pdf,.doc,.txt"}, nil,
		).Once()
		fileStorage.On("UploadFile", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8"), "text/plain; charset=utf-8").Return(nil).Once()
		repo.On("Create", mock.AnythingOfType("*models.Attachment")).Return(nil).Once()

		result, err := svc.Upload(docID.String(), "test.txt", b64)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "test.txt", result.Filename)
	})

	t.Run("success participant with assignment access", func(t *testing.T) {
		svc, repo, settingsRepo, fileStorage, incomingRepo, _, _, assignmentRepo, _, _, auth := setupAttachmentService(t, "")
		currentUserID, err := auth.GetCurrentUserUUID()
		require.NoError(t, err)

		incomingRepo.On("GetByID", docID).Return(&models.IncomingDocument{ID: docID, NomenclatureID: uuid.New()}, nil).Maybe()
		settingsRepo.On("Get", "assignment_completion_attachments_enabled").Return(
			&models.SystemSetting{Key: "assignment_completion_attachments_enabled", Value: "true"}, nil,
		).Once()
		assignmentRepo.On("HasDocumentAccess", currentUserID, docID).Return(true, nil).Maybe()

		settingsRepo.On("Get", "max_file_size_mb").Return(
			&models.SystemSetting{Key: "max_file_size_mb", Value: "10"}, nil,
		).Once()
		settingsRepo.On("Get", "allowed_file_types").Return(
			&models.SystemSetting{Key: "allowed_file_types", Value: ".pdf,.doc,.txt"}, nil,
		).Once()
		fileStorage.On("UploadFile", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8"), "text/plain; charset=utf-8").Return(nil).Once()
		repo.On("Create", mock.AnythingOfType("*models.Attachment")).Return(nil).Once()

		result, err := svc.Upload(docID.String(), "test.txt", b64)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "test.txt", result.Filename)
	})

	t.Run("participant upload disabled by settings", func(t *testing.T) {
		svc, _, settingsRepo, _, _, _, _, _, _, _, _ := setupAttachmentService(t, "")
		settingsRepo.On("Get", "assignment_completion_attachments_enabled").Return(
			&models.SystemSetting{Key: "assignment_completion_attachments_enabled", Value: "false"}, nil,
		).Once()

		result, err := svc.Upload(docID.String(), "test.txt", b64)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "отключена")
	})

	t.Run("not authenticated", func(t *testing.T) {
		svc := setupAttachmentServiceNotAuth(t)
		result, err := svc.Upload(docID.String(), "test.txt", b64)
		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("invalid base64", func(t *testing.T) {
		svc, _, _, _, incomingRepo, _, _, _, _, _, _ := setupAttachmentService(t, "clerk")
		incomingRepo.On("GetByID", docID).Return(&models.IncomingDocument{ID: docID, NomenclatureID: uuid.New()}, nil).Maybe()

		result, err := svc.Upload(docID.String(), "test.txt", "!!!not-base64!!!")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode")
		assert.Nil(t, result)
	})

	t.Run("file too large", func(t *testing.T) {
		svc, _, settingsRepo, _, incomingRepo, _, _, _, _, _, _ := setupAttachmentService(t, "clerk")
		incomingRepo.On("GetByID", docID).Return(&models.IncomingDocument{ID: docID, NomenclatureID: uuid.New()}, nil).Maybe()

		// Max 1 byte
		settingsRepo.On("Get", "max_file_size_mb").Return(
			&models.SystemSetting{Key: "max_file_size_mb", Value: "0"}, nil,
		).Once()

		result, err := svc.Upload(docID.String(), "test.txt", b64)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum")
		assert.Nil(t, result)
	})

	t.Run("forbidden extension", func(t *testing.T) {
		svc, _, settingsRepo, _, incomingRepo, _, _, _, _, _, _ := setupAttachmentService(t, "clerk")
		incomingRepo.On("GetByID", docID).Return(&models.IncomingDocument{ID: docID, NomenclatureID: uuid.New()}, nil).Maybe()

		settingsRepo.On("Get", "max_file_size_mb").Return(
			&models.SystemSetting{Key: "max_file_size_mb", Value: "10"}, nil,
		).Once()
		settingsRepo.On("Get", "allowed_file_types").Return(
			&models.SystemSetting{Key: "allowed_file_types", Value: ".pdf,.doc"}, nil,
		).Once()

		result, err := svc.Upload(docID.String(), "virus.exe", b64)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not allowed")
		assert.Nil(t, result)
	})
}

func TestAttachmentService_GetList(t *testing.T) {
	// Получение списка всех вложений для заданного документа
	docID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc, repo, _, _, _, _, _, _, _, _, _ := setupAttachmentService(t, "clerk")
		attachments := []models.Attachment{{ID: uuid.New(), DocumentID: docID, Filename: "f.pdf"}}
		repo.On("GetByDocumentID", docID).Return(attachments, nil).Once()

		result, err := svc.GetList(docID.String())
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("invalid document ID", func(t *testing.T) {
		svc, _, _, _, _, _, _, _, _, _, _ := setupAttachmentService(t, "executor")
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
		svc, repo, _, fileStorage, _, _, _, _, _, _, _ := setupAttachmentService(t, "clerk")
		att := &models.Attachment{ID: attID, Filename: "file.pdf", StoragePath: "minio/path", DocumentID: uuid.New()}
		content := []byte("pdf-content")

		repo.On("GetByID", attID).Return(att, nil).Once()
		fileStorage.On("DownloadFile", mock.Anything, att.StoragePath).Return(content, nil).Once()

		result, err := svc.Download(attID.String())
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "file.pdf", result.Filename)
		assert.Equal(t, base64.StdEncoding.EncodeToString(content), result.Content)
	})

	t.Run("invalid ID", func(t *testing.T) {
		svc, _, _, _, _, _, _, _, _, _, _ := setupAttachmentService(t, "executor")
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
		svc, repo, _, fileStorage, _, _, _, _, _, _, _ := setupAttachmentService(t, "clerk")
		att := &models.Attachment{
			ID:          attID,
			DocumentID:  uuid.New(),
			StoragePath: "minio/path",
		}
		repo.On("GetByID", attID).Return(att, nil).Once()
		fileStorage.On("DeleteFile", mock.Anything, att.StoragePath).Return(nil).Once()
		repo.On("Delete", attID).Return(nil).Once()
		err := svc.Delete(attID.String())
		require.NoError(t, err)
	})

	t.Run("executor can delete with upload access", func(t *testing.T) {
		svc, repo, _, fileStorage, _, _, _, _, _, _, _ := setupAttachmentService(t, "executor")
		att := &models.Attachment{
			ID:          attID,
			DocumentID:  uuid.New(),
			StoragePath: "minio/path",
		}
		repo.On("GetByID", attID).Return(att, nil).Once()
		fileStorage.On("DeleteFile", mock.Anything, att.StoragePath).Return(nil).Once()
		repo.On("Delete", attID).Return(nil).Once()
		err := svc.Delete(attID.String())
		require.NoError(t, err)
	})
}

func TestAttachmentService_ValidatePathInDownloads(t *testing.T) {
	// Проверка пути доступа к файлу для защиты от уязвимости Path Traversal
	t.Run("valid path", func(t *testing.T) {
		svc, _, _, _, _, _, _, _, _, _, _ := setupAttachmentService(t, "executor")
		downloadDir, _ := svc.getDownloadDir()
		err := svc.validatePathInDownloads(downloadDir + "/test.pdf")
		require.NoError(t, err)
	})

	t.Run("path traversal attack", func(t *testing.T) {
		svc, _, _, _, _, _, _, _, _, _, _ := setupAttachmentService(t, "executor")
		err := svc.validatePathInDownloads("C:\\Windows\\System32\\..\\..\\test.pdf")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("outside downloads", func(t *testing.T) {
		svc, _, _, _, _, _, _, _, _, _, _ := setupAttachmentService(t, "executor")
		err := svc.validatePathInDownloads("C:\\Windows\\System32\\cmd.exe")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})
}

func TestAttachmentService_BulkDeleteOlderThan(t *testing.T) {
	t.Run("forbidden without admin role", func(t *testing.T) {
		svc, _, _, _, _, _, _, _, _, _, _ := setupAttachmentServiceWithRoles(t, []string{"clerk"})

		count, err := svc.BulkDeleteOlderThan("2024-01-01T00:00:00Z")
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
		assert.Equal(t, 0, count)
	})

	t.Run("allowed for user with admin role regardless of other roles", func(t *testing.T) {
		svc, repo, _, _, _, _, _, _, _, _, _ := setupAttachmentServiceWithRoles(t, []string{"admin", "clerk"})
		repo.On("GetOlderThan", mock.AnythingOfType("time.Time")).Return([]models.Attachment{}, nil).Once()

		count, err := svc.BulkDeleteOlderThan("2024-01-01T00:00:00Z")
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}
