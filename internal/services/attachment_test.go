package services

import (
	"context"
	"encoding/base64"
	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
	"os"
	"path/filepath"
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
	accessSvc := NewDocumentAccessService(auth, depRepo, assignmentRepo, ackRepo, newRoleMappedDocumentAccessStore(role), nil, incomingRepo, outgoingRepo)

	svc := NewAttachmentService(attachRepo, settingsSvc, auth, fileStorage, accessSvc)
	return svc, attachRepo, settingsRepo, fileStorage, incomingRepo, outgoingRepo, depRepo, assignmentRepo, ackRepo, userRepo, auth
}

type atomicAttachmentStore struct {
	*mocks.AttachmentStore
	effects []models.OutboxEvent
}

func (s *atomicAttachmentStore) OutboxEnabled() bool { return true }
func (s *atomicAttachmentStore) CreateWithOutbox(attachment *models.Attachment, effects []models.OutboxEvent) error {
	s.effects = append([]models.OutboxEvent(nil), effects...)
	return s.AttachmentStore.Create(attachment)
}
func (s *atomicAttachmentStore) MarkDeletingWithOutbox(attachment models.Attachment) error {
	return s.AttachmentStore.MarkDeleting(attachment.ID)
}
func (s *atomicAttachmentStore) MarkDeletingWithEffects(attachment models.Attachment, effects []models.OutboxEvent) error {
	s.effects = append([]models.OutboxEvent(nil), effects...)
	return s.AttachmentStore.MarkDeleting(attachment.ID)
}
func (s *atomicAttachmentStore) MarkDeletingMultipleWithOutbox(attachments []models.Attachment, effects []models.OutboxEvent) error {
	s.effects = append([]models.OutboxEvent(nil), effects...)
	ids := make([]uuid.UUID, 0, len(attachments))
	for _, attachment := range attachments {
		ids = append(ids, attachment.ID)
	}
	return s.AttachmentStore.MarkDeletingMultiple(ids)
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
	accessSvc := NewDocumentAccessService(auth, depRepo, assignmentRepo, ackRepo, newRoleMappedDocumentAccessStore(roles...), nil, incomingRepo, outgoingRepo)

	svc := NewAttachmentService(attachRepo, settingsSvc, auth, fileStorage, accessSvc)
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
	accessSvc := NewDocumentAccessService(auth, depRepo, assignmentRepo, ackRepo, newRoleMappedDocumentAccessStore(), nil, incomingRepo, outgoingRepo)
	return NewAttachmentService(attachRepo, settingsSvc, auth, fileStorage, accessSvc)
}

func TestSafeDownloadFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{name: "keeps simple filename", filename: "report.pdf", want: "report.pdf"},
		{name: "trims spaces", filename: "  report.pdf  ", want: "report.pdf"},
		{name: "drops parent directories", filename: "../secret/report.pdf", want: "report.pdf"},
		{name: "empty fallback", filename: "   ", want: "attachment"},
		{name: "dot fallback", filename: ".", want: "attachment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, safeDownloadFilename(tt.filename))
		})
	}
}

/*func TestAttachmentService_Upload(t *testing.T) {
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
		requireAppError(t, err, "VALIDATION_ERROR", 400, "не удалось прочитать содержимое файла")
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
		requireAppError(t, err, "VALIDATION_ERROR", 400, "размер файла превышает")
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
		requireAppError(t, err, "VALIDATION_ERROR", 400, "тип файла")
		assert.Nil(t, result)
	})
}*/

func TestAttachmentServiceUploadPathStreamsSelectedFile(t *testing.T) {
	docID := uuid.New()
	path := filepath.Join(t.TempDir(), "test.txt")
	require.NoError(t, os.WriteFile(path, []byte("Hello, world!"), 0600))
	svc, repo, settingsRepo, storage, incomingRepo, _, _, _, _, _, _ := setupAttachmentService(t, "clerk")
	atomicRepo := &atomicAttachmentStore{AttachmentStore: repo}
	svc.repo = atomicRepo
	incomingRepo.On("GetByID", docID).Return(&models.IncomingDocument{ID: docID, NomenclatureID: uuid.New()}, nil).Maybe()
	settingsRepo.On("Get", "max_file_size_mb").Return(&models.SystemSetting{Key: "max_file_size_mb", Value: "10"}, nil).Once()
	settingsRepo.On("Get", "allowed_file_types").Return(&models.SystemSetting{Key: "allowed_file_types", Value: ".txt"}, nil).Once()
	storage.On("UploadFile", mock.Anything, mock.AnythingOfType("string"), mock.Anything, int64(13), "text/plain; charset=utf-8").Return(nil).Once()
	repo.On("Create", mock.AnythingOfType("*models.Attachment")).Return(nil).Once()

	attachment, err := svc.uploadPath(docID.String(), path)
	require.NoError(t, err)
	assert.Equal(t, "test.txt", attachment.Filename)
	require.Len(t, atomicRepo.effects, 1)
	assert.Equal(t, models.OutboxEventJournal, atomicRepo.effects[0].EventType)
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
		requireAppError(t, err, "VALIDATION_ERROR", 400, "неверный ID документа")
		assert.Nil(t, result)
	})
}

func TestAttachmentService_Download(t *testing.T) {
	// Скачивание (выгрузка) содержимого файла вложения
	attID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc, repo, settingsRepo, fileStorage, _, _, _, _, _, _, _ := setupAttachmentService(t, "clerk")
		att := &models.Attachment{ID: attID, Filename: "file.pdf", StoragePath: "minio/path", DocumentID: uuid.New()}
		content := []byte("pdf-content")

		repo.On("GetByID", attID).Return(att, nil).Once()
		settingsRepo.On("Get", "max_file_size_mb").Return(&models.SystemSetting{Key: "max_file_size_mb", Value: "10"}, nil).Once()
		fileStorage.On("DownloadFile", mock.Anything, att.StoragePath, int64(10*1024*1024)).Return(content, nil).Once()

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
		requireAppError(t, err, "VALIDATION_ERROR", 400, "неверный ID файла")
		assert.Nil(t, result)
	})
}

func TestWriteDownloadFileWithoutOverwrite(t *testing.T) {
	downloadDir := t.TempDir()
	originalPath := filepath.Join(downloadDir, "report.pdf")
	require.NoError(t, os.WriteFile(originalPath, []byte("original"), 0644))

	firstPath, err := writeDownloadFileWithoutOverwrite(downloadDir, "../report.pdf", []byte("first"))
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(downloadDir, "report (1).pdf"), firstPath)

	secondPath, err := writeDownloadFileWithoutOverwrite(downloadDir, "report.pdf", []byte("second"))
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(downloadDir, "report (2).pdf"), secondPath)

	originalContent, err := os.ReadFile(originalPath)
	require.NoError(t, err)
	assert.Equal(t, "original", string(originalContent))

	firstContent, err := os.ReadFile(firstPath)
	require.NoError(t, err)
	assert.Equal(t, "first", string(firstContent))

	secondContent, err := os.ReadFile(secondPath)
	require.NoError(t, err)
	assert.Equal(t, "second", string(secondContent))
}

func TestAttachmentService_Delete(t *testing.T) {
	// Удаление файла вложения
	attID := uuid.New()

	t.Run("success clerk", func(t *testing.T) {
		svc, repo, _, _, _, _, _, _, _, _, _ := setupAttachmentService(t, "clerk")
		atomicRepo := &atomicAttachmentStore{AttachmentStore: repo}
		svc.repo = atomicRepo
		att := &models.Attachment{
			ID:          attID,
			DocumentID:  uuid.New(),
			StoragePath: "minio/path",
		}
		repo.On("GetByID", attID).Return(att, nil).Once()
		repo.On("MarkDeleting", attID).Return(nil).Once()
		err := svc.Delete(attID.String())
		require.NoError(t, err)
		require.Len(t, atomicRepo.effects, 1)
		assert.Equal(t, models.OutboxEventJournal, atomicRepo.effects[0].EventType)
	})

	t.Run("executor can delete with upload access", func(t *testing.T) {
		svc, repo, _, _, _, _, _, _, _, _, _ := setupAttachmentService(t, "executor")
		atomicRepo := &atomicAttachmentStore{AttachmentStore: repo}
		svc.repo = atomicRepo
		att := &models.Attachment{
			ID:          attID,
			DocumentID:  uuid.New(),
			StoragePath: "minio/path",
		}
		repo.On("GetByID", attID).Return(att, nil).Once()
		repo.On("MarkDeleting", attID).Return(nil).Once()
		err := svc.Delete(attID.String())
		require.NoError(t, err)
	})

	t.Run("queues deletion intent without synchronous storage finalization", func(t *testing.T) {
		svc, repo, _, _, _, _, _, _, _, _, _ := setupAttachmentService(t, "clerk")
		atomicRepo := &atomicAttachmentStore{AttachmentStore: repo}
		svc.repo = atomicRepo
		att := &models.Attachment{ID: attID, DocumentID: uuid.New(), StoragePath: "minio/path"}
		repo.On("GetByID", attID).Return(att, nil).Once()
		repo.On("MarkDeleting", attID).Return(nil).Once()

		err := svc.Delete(attID.String())
		require.NoError(t, err)
		assert.Len(t, atomicRepo.effects, 1)
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
		requireAppError(t, err, "FORBIDDEN", 403, "папке загрузок")
	})

	t.Run("outside downloads", func(t *testing.T) {
		svc, _, _, _, _, _, _, _, _, _, _ := setupAttachmentService(t, "executor")
		err := svc.validatePathInDownloads("C:\\Windows\\System32\\cmd.exe")
		require.Error(t, err)
		requireAppError(t, err, "FORBIDDEN", 403, "папке загрузок")
	})
}

func TestAttachmentService_BulkDeleteOlderThan(t *testing.T) {
	t.Run("queues deletion and audit through atomic store", func(t *testing.T) {
		svc, repo, _, _, _, _, _, _, _, _, _ := setupAttachmentServiceWithRoles(t, []string{"admin"})
		atomicRepo := &atomicAttachmentStore{AttachmentStore: repo}
		svc.repo = atomicRepo
		attachment := models.Attachment{ID: uuid.New(), StoragePath: "old.pdf"}
		repo.On("GetOlderThan", mock.AnythingOfType("time.Time")).Return([]models.Attachment{attachment}, nil).Once()
		repo.On("MarkDeletingMultiple", []uuid.UUID{attachment.ID}).Return(nil).Once()

		count, err := svc.BulkDeleteOlderThan("2024-01-01T00:00:00Z")
		require.NoError(t, err)
		assert.Equal(t, 1, count)
		require.Len(t, atomicRepo.effects, 1)
		assert.Equal(t, models.OutboxEventAudit, atomicRepo.effects[0].EventType)
	})

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

	t.Run("invalid date", func(t *testing.T) {
		svc, _, _, _, _, _, _, _, _, _, _ := setupAttachmentServiceWithRoles(t, []string{"admin"})

		count, err := svc.BulkDeleteOlderThan("not-a-date")
		require.Error(t, err)
		requireAppError(t, err, "VALIDATION_ERROR", 400, "неверный формат даты")
		assert.Equal(t, 0, count)
	})

	t.Run("repo fetch error", func(t *testing.T) {
		svc, repo, _, _, _, _, _, _, _, _, _ := setupAttachmentServiceWithRoles(t, []string{"admin"})
		repo.On("GetOlderThan", mock.AnythingOfType("time.Time")).Return(nil, assert.AnError).Once()

		count, err := svc.BulkDeleteOlderThan("2024-01-01T00:00:00Z")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch old attachments")
		assert.Equal(t, 0, count)
	})

	t.Run("queues all records for worker delivery", func(t *testing.T) {
		svc, repo, _, _, _, _, _, _, _, _, _ := setupAttachmentServiceWithRoles(t, []string{"admin"})
		atomicRepo := &atomicAttachmentStore{AttachmentStore: repo}
		svc.repo = atomicRepo
		firstID := uuid.New()
		secondID := uuid.New()
		attachments := []models.Attachment{
			{ID: firstID, StoragePath: "ok.pdf"},
			{ID: secondID, StoragePath: "missing.pdf"},
		}
		repo.On("GetOlderThan", mock.AnythingOfType("time.Time")).Return(attachments, nil).Once()
		repo.On("MarkDeletingMultiple", []uuid.UUID{firstID, secondID}).Return(nil).Once()

		count, err := svc.BulkDeleteOlderThan("2024-01-01T00:00:00Z")
		require.NoError(t, err)
		assert.Equal(t, 2, count)
		assert.Len(t, atomicRepo.effects, 1)
	})

	t.Run("queues record without synchronous storage finalization", func(t *testing.T) {
		svc, repo, _, _, _, _, _, _, _, _, _ := setupAttachmentServiceWithRoles(t, []string{"admin"})
		atomicRepo := &atomicAttachmentStore{AttachmentStore: repo}
		svc.repo = atomicRepo
		attachmentID := uuid.New()
		attachments := []models.Attachment{{ID: attachmentID, StoragePath: "ok.pdf"}}
		repo.On("GetOlderThan", mock.AnythingOfType("time.Time")).Return(attachments, nil).Once()
		repo.On("MarkDeletingMultiple", []uuid.UUID{attachmentID}).Return(nil).Once()

		count, err := svc.BulkDeleteOlderThan("2024-01-01T00:00:00Z")
		require.NoError(t, err)
		assert.Equal(t, 1, count)
		assert.Len(t, atomicRepo.effects, 1)
	})
}

func TestAttachmentService_ProcessPendingDeletions(t *testing.T) {
	svc, repo, _, fileStorage, _, _, _, _, _, _, _ := setupAttachmentService(t, "clerk")
	attachment := models.Attachment{ID: uuid.New(), StoragePath: "retry.pdf"}
	repo.On("GetPendingDeletion").Return([]models.Attachment{attachment}, nil).Once()
	fileStorage.On("DeleteFile", mock.Anything, attachment.StoragePath).Return(nil).Once()
	repo.On("DeleteMarked", attachment.ID).Return(nil).Once()

	err := svc.ProcessPendingDeletions(context.Background())
	require.NoError(t, err)
}

func TestReconcileAttachmentStorage(t *testing.T) {
	result := reconcileAttachmentStorage(
		[]string{"objects/present.pdf", "objects/missing.pdf"},
		[]string{"objects/present.pdf", "objects/orphan.pdf"},
	)
	require.Equal(t, []string{"objects/missing.pdf"}, result.MissingObjects)
	require.Equal(t, []string{"objects/orphan.pdf"}, result.OrphanObjects)
}
