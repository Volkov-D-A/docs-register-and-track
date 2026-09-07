package services

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/coordination"
	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
)

func setupAttachmentService(t *testing.T, role string) (
	*ServerAttachmentService, *mocks.AttachmentStore, *mocks.SettingsStore, *mocks.FileStorage, *mocks.IncomingDocStore, *mocks.OutgoingDocStore, *mocks.DepartmentStore, *mocks.AssignmentStore, *mocks.AcknowledgmentStore, *mocks.UserStore, *AuthService,
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

	settingsSvc := NewSettingsService(auth)
	settingsSvc.SetServerClient(&fakeServerSettingsClient{store: settingsRepo})
	accessSvc := NewDocumentAccessService(auth, depRepo, assignmentRepo, ackRepo, newRoleMappedDocumentAccessStore(role), &kindBackedDocumentStore{incoming: incomingRepo, outgoing: outgoingRepo})

	svc := NewServerAttachmentService(attachRepo, settingsSvc, auth, fileStorage, accessSvc, ServerAttachmentOptions{Assignments: assignmentRepo})
	return svc, attachRepo, settingsRepo, fileStorage, incomingRepo, outgoingRepo, depRepo, assignmentRepo, ackRepo, userRepo, auth
}

type storageMutationStub struct {
	ctx      context.Context
	finished bool
}

func (m *storageMutationStub) Context() context.Context { return m.ctx }
func (m *storageMutationStub) Finish() error {
	m.finished = true
	return nil
}

type storageMutationCoordinatorStub struct {
	started  bool
	mutation *storageMutationStub
}

type assignmentAttachmentLookupStub struct {
	*mocks.AttachmentStore
	items []models.Attachment
}

func (s *assignmentAttachmentLookupStub) GetByAssignmentID(uuid.UUID) ([]models.Attachment, error) {
	return s.items, nil
}

func (c *storageMutationCoordinatorStub) BeginStorageMutation(ctx context.Context) (coordination.StorageMutation, error) {
	c.started = true
	c.mutation = &storageMutationStub{ctx: ctx}
	return c.mutation, nil
}

func setupAttachmentServiceWithRoles(t *testing.T, roles []string) (
	*ServerAttachmentService, *mocks.AttachmentStore, *mocks.SettingsStore, *mocks.FileStorage, *mocks.IncomingDocStore, *mocks.OutgoingDocStore, *mocks.DepartmentStore, *mocks.AssignmentStore, *mocks.AcknowledgmentStore, *mocks.UserStore, *AuthService,
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

	settingsSvc := NewSettingsService(auth)
	settingsSvc.SetServerClient(&fakeServerSettingsClient{store: settingsRepo})
	accessSvc := NewDocumentAccessService(auth, depRepo, assignmentRepo, ackRepo, newRoleMappedDocumentAccessStore(roles...), &kindBackedDocumentStore{incoming: incomingRepo, outgoing: outgoingRepo})

	svc := NewServerAttachmentService(attachRepo, settingsSvc, auth, fileStorage, accessSvc, ServerAttachmentOptions{Assignments: assignmentRepo})
	return svc, attachRepo, settingsRepo, fileStorage, incomingRepo, outgoingRepo, depRepo, assignmentRepo, ackRepo, userRepo, auth
}

func setupAttachmentServiceNotAuth(t *testing.T) *ServerAttachmentService {
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
	settingsSvc := NewSettingsService(auth)
	settingsSvc.SetServerClient(&fakeServerSettingsClient{store: settingsRepo})
	accessSvc := NewDocumentAccessService(auth, depRepo, assignmentRepo, ackRepo, newRoleMappedDocumentAccessStore(), &kindBackedDocumentStore{incoming: incomingRepo, outgoing: outgoingRepo})
	return NewServerAttachmentService(attachRepo, settingsSvc, auth, fileStorage, accessSvc, ServerAttachmentOptions{Assignments: assignmentRepo})
}

func TestServerAttachmentUploadStreamsContent(t *testing.T) {
	docID := uuid.New()
	svc, repo, settingsRepo, storage, incomingRepo, _, _, _, _, _, _ := setupAttachmentService(t, "clerk")
	coordinator := &storageMutationCoordinatorStub{}
	svc.storageMutations = coordinator
	incomingRepo.On("GetByID", docID).Return(&models.IncomingDocument{ID: docID, NomenclatureID: uuid.New()}, nil).Maybe()
	settingsRepo.On("Get", "max_file_size_mb").Return(&models.SystemSetting{Key: "max_file_size_mb", Value: "10"}, nil).Once()
	settingsRepo.On("Get", "allowed_file_types").Return(&models.SystemSetting{Key: "allowed_file_types", Value: ".txt"}, nil).Once()
	storage.On("UploadFile", mock.Anything, mock.AnythingOfType("string"), mock.Anything, int64(13), "text/plain; charset=utf-8").
		Run(func(mock.Arguments) {
			assert.True(t, coordinator.started, "mutation must be registered before MinIO upload")
		}).
		Return(nil).Once()
	repo.On("CreateWithOutbox", mock.AnythingOfType("*models.Attachment"), mock.MatchedBy(func(effects []models.OutboxEvent) bool {
		return len(effects) == 1 && effects[0].EventType == models.OutboxEventJournal
	})).Return(nil).Once()

	attachment, err := svc.UploadContent(docID.String(), nil, "test.txt", 13, strings.NewReader("Hello, world!"))
	require.NoError(t, err)
	assert.Equal(t, "test.txt", attachment.Filename)
	require.NotNil(t, coordinator.mutation)
	assert.True(t, coordinator.mutation.finished)
}

func TestAttachmentServiceAssignmentUploadRevalidatesAfterStorageUpload(t *testing.T) {
	documentID, assignmentID, seriesID := uuid.New(), uuid.New(), uuid.New()
	svc, attachmentRepo, settingsRepo, storage, _, _, _, assignmentRepo, _, _, auth := setupAttachmentService(t, "")
	currentUserID, err := auth.GetCurrentUserUUID()
	require.NoError(t, err)

	inProgress := &models.Assignment{ID: assignmentID, DocumentID: documentID, ExecutorID: currentUserID, Status: "in_progress", SeriesID: &seriesID, IsSeriesCurrent: true}
	finished := *inProgress
	finished.Status = "finished"
	assignmentRepo.On("GetByID", assignmentID).Return(inProgress, nil).Once()
	assignmentRepo.On("GetByID", assignmentID).Return(&finished, nil).Once()
	settingsRepo.On("Get", "assignment_completion_attachments_enabled").Return(&models.SystemSetting{Key: "assignment_completion_attachments_enabled", Value: "true"}, nil).Twice()
	settingsRepo.On("Get", "max_file_size_mb").Return(&models.SystemSetting{Key: "max_file_size_mb", Value: "10"}, nil).Once()
	settingsRepo.On("Get", "allowed_file_types").Return(&models.SystemSetting{Key: "allowed_file_types", Value: ".txt"}, nil).Once()
	storage.On("UploadFile", mock.Anything, mock.AnythingOfType("string"), mock.Anything, int64(14), "text/plain; charset=utf-8").Return(nil).Once()
	storage.On("DeleteFile", mock.Anything, mock.AnythingOfType("string")).Return(nil).Once()

	attachment, err := svc.UploadAssignmentContent(assignmentID.String(), "result.txt", 14, strings.NewReader("completed work"))
	require.Nil(t, attachment)
	requireAppError(t, err, "CONFLICT", 409, "только для поручения в работе")
	attachmentRepo.AssertNotCalled(t, "CreateForAssignmentWithOutbox", mock.Anything, mock.Anything, mock.Anything)
}

func TestServerAttachmentUploadAssignmentContentPersistsEffects(t *testing.T) {
	documentID, assignmentID, seriesID := uuid.New(), uuid.New(), uuid.New()
	svc, attachmentRepo, settingsRepo, storage, _, _, _, assignmentRepo, _, _, auth := setupAttachmentService(t, "")
	currentUserID, err := auth.GetCurrentUserUUID()
	require.NoError(t, err)
	assignment := &models.Assignment{ID: assignmentID, DocumentID: documentID, ExecutorID: currentUserID, Status: "in_progress", SeriesID: &seriesID, IsSeriesCurrent: true}
	assignmentRepo.On("GetByID", assignmentID).Return(assignment, nil).Times(2)
	settingsRepo.On("Get", "assignment_completion_attachments_enabled").Return(&models.SystemSetting{Key: "assignment_completion_attachments_enabled", Value: "true"}, nil).Times(2)
	settingsRepo.On("Get", "max_file_size_mb").Return(&models.SystemSetting{Key: "max_file_size_mb", Value: "10"}, nil).Once()
	settingsRepo.On("Get", "allowed_file_types").Return(&models.SystemSetting{Key: "allowed_file_types", Value: ".txt"}, nil).Once()
	storage.On("UploadFile", mock.Anything, mock.AnythingOfType("string"), mock.Anything, int64(14), "text/plain; charset=utf-8").Return(nil).Once()
	attachmentRepo.On("CreateForAssignmentWithOutbox", mock.MatchedBy(func(a *models.Attachment) bool {
		return a.AssignmentID != nil && *a.AssignmentID == assignmentID && a.DocumentID == documentID
	}), true, mock.Anything).Return(nil).Once()

	items, err := svc.UploadAssignmentContent(assignmentID.String(), "result.txt", 14, strings.NewReader("completed work"))
	require.NoError(t, err)
	require.NotNil(t, items)
	assert.Equal(t, "result.txt", items.Filename)
}

func TestAttachmentServiceGetAssignmentFilesRequiresManagerAndReturnsIterationFiles(t *testing.T) {
	documentID, assignmentID := uuid.New(), uuid.New()
	svc, attachmentRepo, _, _, _, _, _, assignmentRepo, _, _, _ := setupAttachmentService(t, "clerk")
	svc.repo = &assignmentAttachmentLookupStub{
		AttachmentStore: attachmentRepo,
		items:           []models.Attachment{{ID: uuid.New(), DocumentID: documentID, AssignmentID: &assignmentID, Filename: "result.pdf"}},
	}
	assignmentRepo.On("GetByID", assignmentID).Return(&models.Assignment{ID: assignmentID, DocumentID: documentID}, nil).Once()

	files, err := svc.GetAssignmentFiles(assignmentID.String())
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, "result.pdf", files[0].Filename)
	assert.Equal(t, assignmentID.String(), files[0].AssignmentID)
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

func TestAttachmentService_Delete(t *testing.T) {
	// Удаление файла вложения
	attID := uuid.New()

	t.Run("success clerk", func(t *testing.T) {
		svc, repo, _, _, _, _, _, _, _, _, _ := setupAttachmentService(t, "clerk")
		att := &models.Attachment{
			ID:          attID,
			DocumentID:  uuid.New(),
			StoragePath: "minio/path",
		}
		repo.On("GetByID", attID).Return(att, nil).Once()
		repo.On("MarkDeletingWithEffects", *att, mock.MatchedBy(func(effects []models.OutboxEvent) bool {
			return len(effects) == 1 && effects[0].EventType == models.OutboxEventJournal
		})).Return(nil).Once()
		err := svc.Delete(attID.String())
		require.NoError(t, err)
	})

	t.Run("executor can delete with upload access", func(t *testing.T) {
		svc, repo, _, _, _, _, _, _, _, _, _ := setupAttachmentService(t, "executor")
		att := &models.Attachment{
			ID:          attID,
			DocumentID:  uuid.New(),
			StoragePath: "minio/path",
		}
		repo.On("GetByID", attID).Return(att, nil).Once()
		repo.On("MarkDeletingWithEffects", *att, mock.Anything).Return(nil).Once()
		err := svc.Delete(attID.String())
		require.NoError(t, err)
	})

	t.Run("queues deletion intent without synchronous storage finalization", func(t *testing.T) {
		svc, repo, _, _, _, _, _, _, _, _, _ := setupAttachmentService(t, "clerk")
		att := &models.Attachment{ID: attID, DocumentID: uuid.New(), StoragePath: "minio/path"}
		repo.On("GetByID", attID).Return(att, nil).Once()
		repo.On("MarkDeletingWithEffects", *att, mock.Anything).Return(nil).Once()

		err := svc.Delete(attID.String())
		require.NoError(t, err)
	})
}

func TestAttachmentService_BulkDeleteOlderThan(t *testing.T) {
	t.Run("queues deletion and audit through atomic store", func(t *testing.T) {
		svc, repo, _, _, _, _, _, _, _, _, _ := setupAttachmentServiceWithRoles(t, []string{"admin"})
		attachment := models.Attachment{ID: uuid.New(), StoragePath: "old.pdf"}
		repo.On("GetOlderThan", mock.AnythingOfType("time.Time")).Return([]models.Attachment{attachment}, nil).Once()
		repo.On("MarkDeletingMultipleWithOutbox", []models.Attachment{attachment}, mock.MatchedBy(func(effects []models.OutboxEvent) bool {
			return len(effects) == 1 && effects[0].EventType == models.OutboxEventAudit
		})).Return(nil).Once()

		count, err := svc.BulkDeleteOlderThan("2024-01-01T00:00:00Z")
		require.NoError(t, err)
		assert.Equal(t, 1, count)
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
		firstID := uuid.New()
		secondID := uuid.New()
		attachments := []models.Attachment{
			{ID: firstID, StoragePath: "ok.pdf"},
			{ID: secondID, StoragePath: "missing.pdf"},
		}
		repo.On("GetOlderThan", mock.AnythingOfType("time.Time")).Return(attachments, nil).Once()
		repo.On("MarkDeletingMultipleWithOutbox", attachments, mock.Anything).Return(nil).Once()

		count, err := svc.BulkDeleteOlderThan("2024-01-01T00:00:00Z")
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("queues record without synchronous storage finalization", func(t *testing.T) {
		svc, repo, _, _, _, _, _, _, _, _, _ := setupAttachmentServiceWithRoles(t, []string{"admin"})
		attachmentID := uuid.New()
		attachments := []models.Attachment{{ID: attachmentID, StoragePath: "ok.pdf"}}
		repo.On("GetOlderThan", mock.AnythingOfType("time.Time")).Return(attachments, nil).Once()
		repo.On("MarkDeletingMultipleWithOutbox", attachments, mock.Anything).Return(nil).Once()

		count, err := svc.BulkDeleteOlderThan("2024-01-01T00:00:00Z")
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})
}

func TestReconcileAttachmentStorage(t *testing.T) {
	result := reconcileAttachmentStorage(
		[]string{"objects/present.pdf", "objects/missing.pdf"},
		[]string{"objects/present.pdf", "objects/orphan.pdf"},
	)
	require.Equal(t, []string{"objects/missing.pdf"}, result.MissingObjects)
	require.Equal(t, []string{"objects/orphan.pdf"}, result.OrphanObjects)
}

func TestServerAttachmentUploadValidation(t *testing.T) {
	for _, tc := range []struct {
		name, role, filename, max, allowed, want string
		size                                     int64
	}{
		{"too large", "clerk", "test.txt", "1", "", "размер файла", 2 * 1024 * 1024},
		{"forbidden type", "clerk", "test.exe", "10", ".txt", "тип файла", 1},
		{"participant disabled", "", "test.txt", "", "", "отключена", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, settings, _, _, _, _, _, _, _, _ := setupAttachmentService(t, tc.role)
			if tc.max != "" {
				settings.On("Get", "max_file_size_mb").Return(&models.SystemSetting{Value: tc.max}, nil).Once()
			}
			if tc.allowed != "" {
				settings.On("Get", "allowed_file_types").Return(&models.SystemSetting{Value: tc.allowed}, nil).Once()
			}
			if tc.role == "" {
				settings.On("Get", "assignment_completion_attachments_enabled").Return(&models.SystemSetting{Value: "false"}, nil).Once()
			}
			item, err := svc.UploadContent(uuid.NewString(), nil, tc.filename, tc.size, strings.NewReader("x"))
			require.Nil(t, item)
			require.ErrorContains(t, err, tc.want)
		})
	}
	t.Run("unauthenticated", func(t *testing.T) {
		svc := setupAttachmentServiceNotAuth(t)
		_, err := svc.UploadContent(uuid.NewString(), nil, "test.txt", 1, strings.NewReader("x"))
		require.ErrorIs(t, err, models.ErrUnauthorized)
	})
}

func TestServerAttachmentUploadCompensatesMetadataFailure(t *testing.T) {
	svc, repo, settings, storage, _, _, _, _, _, _, _ := setupAttachmentService(t, "clerk")
	settings.On("Get", "max_file_size_mb").Return(&models.SystemSetting{Value: "10"}, nil).Once()
	settings.On("Get", "allowed_file_types").Return(&models.SystemSetting{Value: ".txt"}, nil).Once()
	var object string
	storage.On("UploadFile", mock.Anything, mock.Anything, mock.Anything, int64(1), mock.Anything).Run(func(args mock.Arguments) { object = args.String(1) }).Return(nil).Once()
	repo.On("CreateWithOutbox", mock.Anything, mock.Anything).Return(assert.AnError).Once()
	storage.On("DeleteFile", mock.Anything, mock.MatchedBy(func(name string) bool { return name == object })).Return(nil).Once()
	item, err := svc.UploadContent(uuid.NewString(), nil, "test.txt", 1, strings.NewReader("x"))
	require.Nil(t, item)
	require.ErrorIs(t, err, assert.AnError)
}

func TestServerAttachmentConfiguration(t *testing.T) {
	require.Panics(t, func() { NewServerAttachmentService(nil, nil, nil, nil, nil, ServerAttachmentOptions{}) })
	svc, repo, _, storage, _, _, _, assignments, _, _, auth := setupAttachmentService(t, "clerk")
	coordinator := &storageMutationCoordinatorStub{}
	configured := NewServerAttachmentService(repo, svc.settingsService, auth, storage, svc.access, ServerAttachmentOptions{Assignments: assignments, StorageMutations: coordinator})
	require.Same(t, assignments, configured.assignments)
	require.Same(t, coordinator, configured.storageMutations)
	var missing *mocks.AttachmentStore
	require.Panics(t, func() {
		NewServerAttachmentService(missing, svc.settingsService, auth, storage, svc.access, ServerAttachmentOptions{})
	})
}
