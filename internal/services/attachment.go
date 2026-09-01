package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/Volkov-D-A/docs-register-and-track/internal/coordination"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

// AttachmentService предоставляет бизнес-логику для работы с вложениями (файлами) документов.
type AttachmentService struct {
	repo             AttachmentStore
	settingsService  AttachmentSettings
	authService      AttachmentPrincipal
	fileStorage      FileStorage
	access           *DocumentAccessService
	lifecycle        *OperationLifecycle
	uiContext        context.Context
	metrics          *observability.Registry
	storageMutations coordination.StorageMutationCoordinator
	assignments      AssignmentStore
	substitutions    UserSubstitutionStore
	openFilesDialog  func(context.Context, wailsruntime.OpenDialogOptions) ([]string, error)
	server           serverclient.AttachmentClient
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

// NewAttachmentServiceWithClient creates the desktop adapter. Native file
// dialogs and local downloads stay in desktop, while all protected data and
// object-storage operations are performed by docflow-server.
func NewAttachmentServiceWithClient(client serverclient.AttachmentClient) *AttachmentService {
	return &AttachmentService{server: client, openFilesDialog: wailsruntime.OpenMultipleFilesDialog}
}

type attachmentStoragePathStore interface {
	GetAllStoragePaths() ([]string, error)
}

type objectNameLister interface {
	ListObjectNames(ctx context.Context) ([]string, error)
}

// NewAttachmentService создает новый экземпляр AttachmentService.
func NewAttachmentService(repo AttachmentStore, settingsService *SettingsService, authService *AuthService, fs FileStorage, access *DocumentAccessService) *AttachmentService {
	return NewServerAttachmentService(repo, settingsService, authService, fs, access)
}

// NewServerAttachmentService creates the request-scoped server implementation.
func NewServerAttachmentService(repo AttachmentStore, settingsService AttachmentSettings, authService AttachmentPrincipal, fs FileStorage, access *DocumentAccessService) *AttachmentService {
	service := &AttachmentService{
		repo:            repo,
		settingsService: settingsService,
		authService:     authService,
		fileStorage:     fs,
		access:          access,
		openFilesDialog: wailsruntime.OpenMultipleFilesDialog,
	}
	if coordinator, ok := repo.(coordination.StorageMutationCoordinator); ok {
		service.storageMutations = coordinator
	}
	return service
}

func (s *AttachmentService) SetOperationLifecycle(lifecycle *OperationLifecycle) {
	s.lifecycle = lifecycle
}

func (s *AttachmentService) SetOperationMetrics(metrics *observability.Registry) { s.metrics = metrics }

func (s *AttachmentService) SetAssignmentStore(store AssignmentStore) { s.assignments = store }
func (s *AttachmentService) SetSubstitutionStore(store UserSubstitutionStore) {
	s.substitutions = store
}

// ReconcileStorage compares database metadata and MinIO without modifying
// either side. It is intentionally available only to administrators.
func (s *AttachmentService) ReconcileStorage() (*models.AttachmentStorageReconciliation, error) {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return s.server.ReconcileAttachmentStorage(ctx)
	}
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
	ctx, release := serviceOperationContext(s.lifecycle)
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

// Startup receives the Wails context required to display the native file picker.
func (s *AttachmentService) Startup(ctx context.Context) { s.uiContext = ctx }

// Upload lets the user choose files in the native OS dialog and streams each
// selected file to MinIO. No renderer-provided path or base64 payload is trusted.
func (s *AttachmentService) Upload(documentIDStr string) ([]dto.Attachment, error) {
	return measureOperation(s.metrics, "attachments.upload", func() ([]dto.Attachment, error) {
		if s.uiContext == nil {
			return nil, fmt.Errorf("file picker is not initialized")
		}
		paths, err := s.openFilesDialog(s.uiContext, wailsruntime.OpenDialogOptions{Title: "Выберите файлы для вложения"})
		if err != nil {
			return nil, fmt.Errorf("failed to choose files: %w", err)
		}
		attachments := make([]dto.Attachment, 0, len(paths))
		for _, path := range paths {
			attachment, err := s.uploadSelectedPath(documentIDStr, "", path)
			if err != nil {
				return nil, err
			}
			attachments = append(attachments, *attachment)
		}
		return attachments, nil
	})
}

// UploadForAssignment links completion evidence to one concrete iteration.
func (s *AttachmentService) UploadForAssignment(assignmentIDStr string) ([]dto.Attachment, error) {
	return measureOperation(s.metrics, "attachments.upload.assignment", func() ([]dto.Attachment, error) {
		if s.uiContext == nil {
			return nil, fmt.Errorf("file picker is not initialized")
		}
		if s.server == nil && s.assignments == nil {
			return nil, fmt.Errorf("assignment store is not configured")
		}
		assignmentID, err := uuid.Parse(assignmentIDStr)
		if err != nil {
			return nil, models.NewBadRequestWrapped("неверный ID поручения", err)
		}
		if s.server == nil {
			if _, _, err = s.requireAssignmentUploadAccess(assignmentID); err != nil {
				return nil, err
			}
		}
		paths, err := s.openFilesDialog(s.uiContext, wailsruntime.OpenDialogOptions{Title: "Выберите файлы для отчёта об исполнении"})
		if err != nil {
			return nil, fmt.Errorf("failed to choose files: %w", err)
		}
		items := make([]dto.Attachment, 0, len(paths))
		for _, path := range paths {
			item, uploadErr := s.uploadSelectedPath("", assignmentID.String(), path)
			if uploadErr != nil {
				return nil, uploadErr
			}
			items = append(items, *item)
		}
		return items, nil
	})
}

func (s *AttachmentService) uploadSelectedPath(documentID, assignmentID, path string) (*dto.Attachment, error) {
	if s.server == nil {
		if assignmentID != "" {
			id, err := uuid.Parse(assignmentID)
			if err != nil {
				return nil, models.NewBadRequestWrapped("неверный ID поручения", err)
			}
			return s.uploadPathForAssignment(id, path)
		}
		return s.uploadPath(documentID, path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, models.NewBadRequestWrapped("не удалось открыть выбранный файл", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, models.NewBadRequest("выбранный путь не является обычным файлом")
	}
	ctx, release := serviceOperationContext(s.lifecycle)
	defer release()
	return s.server.UploadAttachment(ctx, documentID, assignmentID, filepath.Base(path), info.Size(), file)
}

func (s *AttachmentService) requireAssignmentUploadAccess(assignmentID uuid.UUID) (*models.Assignment, bool, error) {
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

func (s *AttachmentService) uploadPathForAssignment(assignmentID uuid.UUID, path string) (*dto.Attachment, error) {
	assignment, _, err := s.requireAssignmentUploadAccess(assignmentID)
	if err != nil {
		return nil, err
	}
	return s.uploadPathLinked(assignment.DocumentID.String(), path, &assignmentID)
}

// UploadAssignmentContent resolves the assignment server-side and binds the
// uploaded object to its current iteration.
func (s *AttachmentService) UploadAssignmentContent(assignmentIDStr, filename string, size int64, content io.Reader) (*dto.Attachment, error) {
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

func (s *AttachmentService) uploadPath(documentIDStr, path string) (*dto.Attachment, error) {
	return s.uploadPathLinked(documentIDStr, path, nil)
}

func (s *AttachmentService) uploadPathLinked(documentIDStr, path string, assignmentID *uuid.UUID) (*dto.Attachment, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, models.NewBadRequestWrapped("не удалось открыть выбранный файл", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, models.NewBadRequest("выбранный путь не является обычным файлом")
	}
	return s.UploadContent(documentIDStr, assignmentID, filepath.Base(path), info.Size(), file)
}

// MaxUploadSize returns the application limit used by the HTTP body guard.
func (s *AttachmentService) MaxUploadSize() int64 {
	maxSize, _ := s.settingsService.GetMaxFileSize()
	return maxSize
}

// UploadContent validates and streams one request body into object storage.
func (s *AttachmentService) UploadContent(documentIDStr string, assignmentID *uuid.UUID, filename string, size int64, content io.Reader) (*dto.Attachment, error) {
	ctx, release := serviceOperationContext(s.lifecycle)
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

type assignmentAttachmentCreator interface {
	CreateForAssignmentWithOutbox(*models.Attachment, bool, []models.OutboxEvent) error
}

type assignmentAttachmentStore interface {
	GetByAssignmentID(uuid.UUID) ([]models.Attachment, error)
}

// GetAssignmentFiles is manager-only because it is used by the protected
// history view of a recurring series.
func (s *AttachmentService) GetAssignmentFiles(assignmentIDStr string) ([]dto.Attachment, error) {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return s.server.ListAssignmentAttachments(ctx, assignmentIDStr)
	}
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

// GetList — получить вложения документа
func (s *AttachmentService) GetList(documentIDStr string) ([]dto.Attachment, error) {
	return measureOperation(s.metrics, "attachments.get_list", func() ([]dto.Attachment, error) {
		if s.server != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			return s.server.ListDocumentAttachments(ctx, documentIDStr)
		}
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

// Delete — удалить вложение
func (s *AttachmentService) Delete(idStr string) error {
	if s.server != nil {
		ctx, release := serviceOperationContext(s.lifecycle)
		defer release()
		return s.server.DeleteAttachment(ctx, idStr)
	}
	_, release := serviceOperationContext(s.lifecycle)
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

// AuthorizeDownload resolves metadata after enforcing document access. It is
// used only inside the server before response headers are written.
func (s *AttachmentService) AuthorizeDownload(idStr string) (*models.Attachment, error) {
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

// StreamAttachment writes bounded object content after AuthorizeDownload.
func (s *AttachmentService) StreamAttachment(ctx context.Context, attachment *models.Attachment, writer io.Writer) error {
	if attachment == nil {
		return models.NewNotFound("вложение не найдено")
	}
	maxSize, _ := s.settingsService.GetMaxFileSize()
	if err := s.fileStorage.DownloadFileToWriter(ctx, attachment.StoragePath, writer, maxSize); err != nil {
		return err
	}
	return nil
}

// DownloadToDisk — сохранить файл в папку «Загрузки» пользователя и вернуть полный путь
func (s *AttachmentService) DownloadToDisk(idStr string) (string, error) {
	return measureOperation(s.metrics, "attachments.download", func() (string, error) {
		ctx, release := serviceOperationContext(s.lifecycle)
		defer release()
		if s.server != nil {
			attachment, content, err := s.server.GetAttachmentContent(ctx, idStr)
			if err != nil {
				return "", err
			}
			defer content.Close()
			downloadDir, err := s.getDownloadDir()
			if err != nil {
				return "", err
			}
			return writeDownloadFileFromStorage(downloadDir, attachment.Filename, func(file *os.File) error {
				_, copyErr := io.Copy(file, content)
				return copyErr
			})
		}

		if err := s.access.RequireDomainRead(); err != nil {
			return "", err
		}

		id, err := uuid.Parse(idStr)
		if err != nil {
			return "", models.NewBadRequestWrapped("неверный ID файла", err)
		}

		// Получение метаданных
		attachment, err := s.repo.GetByID(id)
		if err != nil {
			return "", err
		}
		if attachment == nil {
			return "", nil
		}
		if err := s.access.RequireReadAnyType(attachment.DocumentID); err != nil {
			return "", err
		}

		// Получение содержимого
		// Определение пути для сохранения
		currentUser, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("failed to get current user: %v", err)
		}

		// Формирование пути к папке "Downloads"
		downloadDir := filepath.Join(currentUser.HomeDir, "Downloads")

		// Создание директории, если не существует
		if err := os.MkdirAll(downloadDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create download directory: %v", err)
		}

		maxSize, _ := s.settingsService.GetMaxFileSize()
		fullPath, err := writeDownloadFileFromStorage(downloadDir, attachment.Filename, func(file *os.File) error {
			return s.fileStorage.DownloadFileToWriter(ctx, attachment.StoragePath, file, maxSize)
		})
		if err != nil {
			return "", fmt.Errorf("failed to write file: %v", err)
		}

		if s.metrics != nil {
			s.metrics.AddCounter("attachments.download.bytes", float64(attachment.FileSize))
		}
		return fullPath, nil
	})
}

func writeDownloadFileFromStorage(downloadDir, filename string, write func(*os.File) error) (string, error) {
	cleanFilename := safeDownloadFilename(filename)
	ext := filepath.Ext(cleanFilename)
	base := strings.TrimSuffix(cleanFilename, ext)
	for i := 0; i < 1000; i++ {
		candidate := cleanFilename
		if i > 0 {
			candidate = fmt.Sprintf("%s (%d)%s", base, i, ext)
		}
		fullPath := filepath.Join(downloadDir, candidate)
		file, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if err := write(file); err != nil {
			_ = file.Close()
			_ = os.Remove(fullPath)
			return "", err
		}
		if err := file.Close(); err != nil {
			_ = os.Remove(fullPath)
			return "", err
		}
		return fullPath, nil
	}
	return "", fmt.Errorf("failed to choose unique download filename for %q", cleanFilename)
}

func safeDownloadFilename(filename string) string {
	filename = strings.ReplaceAll(filename, "\\", "/")
	cleanFilename := filepath.Base(strings.TrimSpace(filename))
	cleanFilename = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, cleanFilename)
	if cleanFilename == "" || cleanFilename == "." || cleanFilename == string(filepath.Separator) {
		return "attachment"
	}
	return cleanFilename
}

// getDownloadDir — получить путь к папке «Загрузки» текущего пользователя
func (s *AttachmentService) getDownloadDir() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %v", err)
	}
	return filepath.Join(currentUser.HomeDir, "Downloads"), nil
}

// validatePathInDownloads — проверка, что путь находится внутри папки «Загрузки»
// для предотвращения атак через произвольные пути
func (s *AttachmentService) validatePathInDownloads(path string) error {
	downloadDir, err := s.getDownloadDir()
	if err != nil {
		return err
	}

	// Разрешение символических ссылок и относительных путей
	absPath, err := filepath.Abs(path)
	if err != nil {
		return models.NewBadRequestWrapped("неверный путь к файлу", err)
	}
	evalPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// Файл может ещё не существовать (для OpenFolder), пробуем относительный путь
		evalPath = absPath
	}

	absDownloadDir, err := filepath.Abs(downloadDir)
	if err != nil {
		return fmt.Errorf("failed to resolve download directory: %v", err)
	}

	// Убеждаемся, что путь находится внутри папки «Загрузки»
	rel, err := filepath.Rel(absDownloadDir, evalPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return models.NewForbidden("доступ разрешен только к файлам в папке загрузок")
	}

	return nil
}

// OpenFile — открыть файл в приложении по умолчанию
// Разрешено только для файлов в папке «Загрузки» пользователя
func (s *AttachmentService) OpenFile(path string) error {
	if err := s.validatePathInDownloads(path); err != nil {
		return err
	}

	cleanPath := filepath.Clean(path)
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", cleanPath)
	case "darwin":
		cmd = exec.Command("open", cleanPath)
	default:
		cmd = exec.Command("xdg-open", cleanPath)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	return nil
}

// OpenFolder — открыть папку, содержащую файл
// Разрешено только для папок в директории «Загрузки» пользователя
func (s *AttachmentService) OpenFolder(path string) error {
	if err := s.validatePathInDownloads(path); err != nil {
		return err
	}

	dir := filepath.Clean(filepath.Dir(path))
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open folder: %v", err)
	}
	return nil
}

// BulkDeleteOlderThan — массовое удаление файлов, загруженных до указанной даты
func (s *AttachmentService) BulkDeleteOlderThan(dateStr string) (int, error) {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return s.server.BulkDeleteAttachments(ctx, dateStr)
	}
	_, release := serviceOperationContext(s.lifecycle)
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
