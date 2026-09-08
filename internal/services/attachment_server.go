package services

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/coordination"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/operations"
)

// ServerAttachmentService owns protected attachment operations for one HTTP request.
type ServerAttachmentService struct {
	repo             AttachmentStore
	settingsService  AttachmentSettings
	authService      AttachmentPrincipal
	fileStorage      FileStorage
	access           *DocumentAccessService
	lifecycle        *operations.Lifecycle
	metrics          *observability.Registry
	storageMutations coordination.StorageMutationCoordinator
	assignments      AssignmentStore
	substitutions    UserSubstitutionStore
}

// ServerAttachmentOptions contains optional request dependencies.
type ServerAttachmentOptions struct {
	Assignments      AssignmentStore
	Substitutions    UserSubstitutionStore
	Metrics          *observability.Registry
	Lifecycle        *operations.Lifecycle
	StorageMutations coordination.StorageMutationCoordinator
}

// NewServerAttachmentService requires the repository, settings, principal, storage
// and document access service. It panics on missing required dependencies,
// which indicate a composition error rather than a request error.
func NewServerAttachmentService(repo AttachmentStore, settings AttachmentSettings, principal AttachmentPrincipal, storage FileStorage, access *DocumentAccessService, options ServerAttachmentOptions) *ServerAttachmentService {
	if attachmentDependencyMissing(repo) || attachmentDependencyMissing(settings) || attachmentDependencyMissing(principal) || attachmentDependencyMissing(storage) || access == nil {
		panic("server attachment service: missing required dependency")
	}
	s := &ServerAttachmentService{
		repo:             repo,
		settingsService:  settings,
		authService:      principal,
		fileStorage:      storage,
		access:           access,
		assignments:      options.Assignments,
		substitutions:    options.Substitutions,
		metrics:          options.Metrics,
		lifecycle:        options.Lifecycle,
		storageMutations: options.StorageMutations,
	}
	if s.storageMutations == nil {
		s.storageMutations, _ = repo.(coordination.StorageMutationCoordinator)
	}
	return s
}

type AttachmentSettings interface {
	GetMaxFileSize() (int64, error)
	GetAllowedFileTypes() ([]string, error)
	IsAssignmentCompletionAttachmentsEnabled() bool
}

type AttachmentPrincipal interface {
	GetCurrentUser() (*dto.User, error)
	GetCurrentUserUUID() (uuid.UUID, error)
	RequireSystemPermission(string) error
}

type attachmentStoragePathStore interface {
	GetAllStoragePaths() ([]string, error)
}

type objectNameLister interface {
	ListObjectNames(ctx context.Context) ([]string, error)
}

type assignmentAttachmentCreator interface {
	CreateForAssignmentWithOutbox(*models.Attachment, bool, []models.OutboxEvent) error
}

type assignmentAttachmentStore interface {
	GetByAssignmentID(uuid.UUID) ([]models.Attachment, error)
}

func (s *ServerAttachmentService) ReconcileStorage() (*models.AttachmentStorageReconciliation, error) {
	if err := s.authService.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return nil, err
	}
	repo, ok := s.repo.(attachmentStoragePathStore)
	if !ok {
		return nil, fmt.Errorf("attachment storage reconciliation is not supported")
	}
	storage, ok := s.fileStorage.(objectNameLister)
	if !ok {
		return nil, fmt.Errorf("object storage reconciliation is not supported")
	}
	ctx, release := s.lifecycle.OperationContext()
	defer release()
	databasePaths, err := repo.GetAllStoragePaths()
	if err != nil {
		return nil, err
	}
	objectPaths, err := storage.ListObjectNames(ctx)
	if err != nil {
		return nil, err
	}
	result := reconcileAttachmentStorage(databasePaths, objectPaths)
	if s.metrics != nil {
		s.metrics.SetGauge("attachments.reconciliation.missing", float64(len(result.MissingObjects)))
		s.metrics.SetGauge("attachments.reconciliation.orphan", float64(len(result.OrphanObjects)))
	}
	return result, nil
}

func reconcileAttachmentStorage(databasePaths, objectPaths []string) *models.AttachmentStorageReconciliation {
	databaseSet := make(map[string]struct{}, len(databasePaths))
	for _, path := range databasePaths {
		databaseSet[path] = struct{}{}
	}
	objectSet := make(map[string]struct{}, len(objectPaths))
	for _, path := range objectPaths {
		objectSet[path] = struct{}{}
	}
	result := &models.AttachmentStorageReconciliation{MissingObjects: make([]string, 0), OrphanObjects: make([]string, 0)}
	for path := range databaseSet {
		if _, ok := objectSet[path]; !ok {
			result.MissingObjects = append(result.MissingObjects, path)
		}
	}
	for path := range objectSet {
		if _, ok := databaseSet[path]; !ok {
			result.OrphanObjects = append(result.OrphanObjects, path)
		}
	}
	sort.Strings(result.MissingObjects)
	sort.Strings(result.OrphanObjects)
	return result
}

func (s *ServerAttachmentService) requireAssignmentUploadAccess(assignmentID uuid.UUID) (*models.Assignment, bool, error) {
	if s.assignments == nil {
		return nil, false, fmt.Errorf("assignment store is not configured")
	}
	assignment, err := s.assignments.GetByID(assignmentID)
	if err != nil {
		return nil, false, err
	}
	if assignment == nil {
		return nil, false, models.NewNotFound("поручение не найдено")
	}
	if assignment.SeriesID != nil && !assignment.IsSeriesCurrent {
		return nil, false, models.ErrForbidden
	}
	currentUserID, err := s.authService.GetCurrentUserUUID()
	if err != nil {
		return nil, false, err
	}
	canAct := assignment.ExecutorID == currentUserID
	if !canAct && s.substitutions != nil {
		canAct, err = s.substitutions.IsActiveSubstitute(currentUserID, assignment.ExecutorID)
		if err != nil {
			return nil, false, err
		}
	}
	canUploadDirectly := s.access.RequireDocumentAction(assignment.DocumentID, "upload") == nil
	if !canUploadDirectly && (!canAct || !s.settingsService.IsAssignmentCompletionAttachmentsEnabled()) {
		return nil, false, models.ErrForbidden
	}
	requireInProgress := !canUploadDirectly
	if requireInProgress && assignment.Status != "in_progress" {
		return nil, false, models.NewConflict("файлы исполнения можно добавлять только для поручения в работе")
	}
	return assignment, requireInProgress, nil
}

func (s *ServerAttachmentService) UploadAssignmentContent(assignmentIDStr, filename string, size int64, content io.Reader) (*dto.Attachment, error) {
	assignmentID, err := uuid.Parse(assignmentIDStr)
	if err != nil {
		return nil, models.NewBadRequestWrapped("неверный ID поручения", err)
	}
	assignment, _, err := s.requireAssignmentUploadAccess(assignmentID)
	if err != nil {
		return nil, err
	}
	return s.UploadContent(assignment.DocumentID.String(), &assignmentID, filename, size, content)
}

func (s *ServerAttachmentService) MaxUploadSize() int64 {
	maxSize, _ := s.settingsService.GetMaxFileSize()
	return maxSize
}

func (s *ServerAttachmentService) UploadContent(documentIDStr string, assignmentID *uuid.UUID, filename string, size int64, content io.Reader) (*dto.Attachment, error) {
	ctx, release := s.lifecycle.OperationContext()
	defer release()

	currentUser, err := s.authService.GetCurrentUser()
	if err != nil {
		return nil, models.ErrUnauthorized
	}

	documentID, err := uuid.Parse(documentIDStr)
	if err != nil {
		return nil, models.NewBadRequestWrapped("неверный ID документа", err)
	}
	if _, err := s.access.RequireExists(documentID); err != nil {
		return nil, err
	}

	canUploadDirectly := s.access.RequireDocumentAction(documentID, "upload") == nil
	if assignmentID == nil && !canUploadDirectly {
		if !s.settingsService.IsAssignmentCompletionAttachmentsEnabled() {
			return nil, models.NewForbidden("загрузка файлов при завершении поручения отключена в настройках")
		}
		hasAssignmentAccess, accessErr := s.access.HasAssignmentAccess(documentID)
		if accessErr != nil {
			return nil, accessErr
		}
		if !hasAssignmentAccess {
			return nil, models.ErrForbidden
		}
	}

	filename = safeDownloadFilename(filename)
	if filename == "attachment" || len(filename) > 255 || size < 0 || content == nil {
		return nil, models.NewBadRequest("файл для загрузки указан некорректно")
	}

	// Проверка размера до чтения содержимого.
	maxSize, _ := s.settingsService.GetMaxFileSize() // returns bytes
	if size > maxSize {
		return nil, models.NewBadRequest(fmt.Sprintf("размер файла превышает максимально допустимый (%d МБ)", maxSize/(1024*1024)))
	}

	// 3. Проверка типа файла
	allowedTypes, _ := s.settingsService.GetAllowedFileTypes()
	ext := strings.ToLower(filepath.Ext(filename))
	allowed := false
	for _, t := range allowedTypes {
		if t == ext {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, models.NewBadRequest(fmt.Sprintf("тип файла %q не разрешен", ext))
	}

	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	objectName := uuid.New().String() + ext
	var mutation coordination.StorageMutation
	if s.storageMutations != nil {
		mutation, err = s.storageMutations.BeginStorageMutation(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to coordinate storage upload: %w", err)
		}
		ctx = mutation.Context()
		defer func() {
			if finishErr := mutation.Finish(); finishErr != nil {
				slog.Warn("failed to finish storage upload coordination", "error", finishErr, "object", objectName)
			}
		}()
	}
	if err := s.fileStorage.UploadFile(ctx, objectName, io.LimitReader(content, size), size, contentType); err != nil {
		return nil, fmt.Errorf("failed to upload file to storage: %v", err)
	}

	// 4. Сохранение в БД
	userID, err := uuid.Parse(currentUser.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid current user ID: %w", err)
	}

	attachment := &models.Attachment{
		DocumentID:   documentID,
		AssignmentID: assignmentID,
		Filename:     filename,
		FileSize:     size,
		ContentType:  contentType,
		StoragePath:  objectName,
		UploadedBy:   userID,
	}

	event, buildErr := NewJournalOutboxEvent("attachment:"+objectName+":upload:journal", models.CreateJournalEntryRequest{DocumentID: documentID, UserID: userID, Action: "FILE_UPLOAD", Details: fmt.Sprintf("Добавлен файл: %s", filename)})
	if buildErr != nil {
		return nil, buildErr
	}
	if assignmentID != nil {
		// Revalidate after the potentially slow storage upload. The repository also
		// checks current-iteration and status predicates in the INSERT transaction.
		latest, latestRequireInProgress, accessErr := s.requireAssignmentUploadAccess(*assignmentID)
		if accessErr != nil {
			_ = s.fileStorage.DeleteFile(ctx, objectName)
			return nil, accessErr
		}
		if latest.DocumentID != documentID {
			_ = s.fileStorage.DeleteFile(ctx, objectName)
			return nil, models.NewConflict("поручение было изменено; повторите загрузку")
		}
		creator, ok := s.repo.(assignmentAttachmentCreator)
		if !ok {
			_ = s.fileStorage.DeleteFile(ctx, objectName)
			return nil, fmt.Errorf("assignment attachment creation is not supported")
		}
		err = creator.CreateForAssignmentWithOutbox(attachment, latestRequireInProgress, []models.OutboxEvent{event})
	} else {
		err = s.repo.CreateWithOutbox(attachment, []models.OutboxEvent{event})
	}
	if err != nil {
		// Попытка откатить (удалить) файл из хранилища, если сохранение в БД не удалось
		_ = s.fileStorage.DeleteFile(ctx, objectName)
		return nil, err
	}

	attachment.UploadedByName = currentUser.FullName
	if s.metrics != nil {
		s.metrics.AddCounter("attachments.upload.bytes", float64(attachment.FileSize))
	}

	return dto.MapAttachment(attachment), nil
}

func (s *ServerAttachmentService) GetAssignmentFiles(assignmentIDStr string) ([]dto.Attachment, error) {
	assignmentID, err := uuid.Parse(assignmentIDStr)
	if err != nil {
		return nil, models.NewBadRequestWrapped("неверный ID поручения", err)
	}
	if s.assignments == nil {
		return nil, fmt.Errorf("assignment store is not configured")
	}
	assignment, err := s.assignments.GetByID(assignmentID)
	if err != nil {
		return nil, err
	}
	if assignment == nil {
		return nil, models.NewNotFound("поручение не найдено")
	}
	if err = s.access.RequireDocumentAction(assignment.DocumentID, "assign"); err != nil {
		return nil, err
	}
	repo, ok := s.repo.(assignmentAttachmentStore)
	if !ok {
		return nil, fmt.Errorf("assignment attachment lookup is not supported")
	}
	items, err := repo.GetByAssignmentID(assignmentID)
	if err != nil {
		return nil, err
	}
	return dto.MapAttachments(items), nil
}

func (s *ServerAttachmentService) GetList(documentIDStr string) ([]dto.Attachment, error) {
	return operations.Measure(s.metrics, "attachments.get_list", func() ([]dto.Attachment, error) {
		documentID, err := uuid.Parse(documentIDStr)
		if err != nil {
			return nil, models.NewBadRequestWrapped("неверный ID документа", err)
		}
		if err := s.access.RequireReadAnyType(documentID); err != nil {
			return nil, err
		}
		res, err := s.repo.GetByDocumentID(documentID)
		attachments := dto.MapAttachments(res)
		if err == nil && s.metrics != nil {
			s.metrics.AddCounter("attachments.list.items", float64(len(attachments)))
		}
		return attachments, err
	})
}

func (s *ServerAttachmentService) Delete(idStr string) error {
	_, release := s.lifecycle.OperationContext()
	defer release()

	// Проверка прав доступа
	id, err := uuid.Parse(idStr)
	if err != nil {
		return models.NewBadRequestWrapped("неверный ID файла", err)
	}

	// Получение вложения для журналирования
	attachment, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if attachment == nil {
		return nil
	}
	if err := s.access.RequireDocumentAction(attachment.DocumentID, "upload"); err != nil {
		return err
	}

	// First commit the deletion intent. From this point the attachment is hidden
	// from reads, so a later database failure cannot leave a visible broken link.
	currentUserID, _ := s.authService.GetCurrentUserUUID()
	event, buildErr := NewJournalOutboxEvent("attachment:"+attachment.ID.String()+":delete:journal", models.CreateJournalEntryRequest{DocumentID: attachment.DocumentID, UserID: currentUserID, Action: "FILE_DELETE", Details: fmt.Sprintf("Удален файл: %s", attachment.Filename)})
	if buildErr != nil {
		return buildErr
	}
	return s.repo.MarkDeletingWithEffects(*attachment, []models.OutboxEvent{event})
}

func (s *ServerAttachmentService) AuthorizeDownload(idStr string) (*models.Attachment, error) {
	if err := s.access.RequireDomainRead(); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, models.NewBadRequestWrapped("неверный ID файла", err)
	}
	attachment, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if attachment == nil {
		return nil, models.NewNotFound("вложение не найдено")
	}
	if err := s.access.RequireReadAnyType(attachment.DocumentID); err != nil {
		return nil, err
	}
	return attachment, nil
}

func (s *ServerAttachmentService) StreamAttachment(ctx context.Context, attachment *models.Attachment, writer io.Writer) error {
	if attachment == nil {
		return models.NewNotFound("вложение не найдено")
	}
	maxSize, _ := s.settingsService.GetMaxFileSize()
	if err := s.fileStorage.DownloadFileToWriter(ctx, attachment.StoragePath, writer, maxSize); err != nil {
		return err
	}
	return nil
}

func (s *ServerAttachmentService) BulkDeleteOlderThan(dateStr string) (int, error) {
	_, release := s.lifecycle.OperationContext()
	defer release()

	// Проверка прав доступа
	if err := s.authService.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return 0, err
	}

	date, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return 0, models.NewBadRequestWrapped("неверный формат даты", err)
	}

	attachments, err := s.repo.GetOlderThan(date)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch old attachments: %v", err)
	}

	if len(attachments) == 0 {
		return 0, nil
	}
	currentUserID, _ := s.authService.GetCurrentUserUUID()
	var currentUserName string
	if u, err := s.authService.GetCurrentUser(); err == nil {
		currentUserName = u.FullName
	}
	details := fmt.Sprintf("Массовое удаление файлов: поставлено в очередь %d, загруженных до %s", len(attachments), date.Format("02.01.2006"))
	event, buildErr := NewAdminAuditOutboxEvent("attachments:bulk-delete:"+date.UTC().Format(time.RFC3339Nano), models.CreateAdminAuditLogRequest{UserID: currentUserID, UserName: currentUserName, Action: "FILES_BULK_DELETE", Details: details})
	if buildErr != nil {
		return 0, buildErr
	}
	if err := s.repo.MarkDeletingMultipleWithOutbox(attachments, []models.OutboxEvent{event}); err != nil {
		return 0, fmt.Errorf("failed to queue attachment deletion: %w", err)
	}
	return len(attachments), nil
}
