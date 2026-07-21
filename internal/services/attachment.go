package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
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
)

// AttachmentService предоставляет бизнес-логику для работы с вложениями (файлами) документов.
type AttachmentService struct {
	repo            AttachmentStore
	settingsService *SettingsService
	authService     *AuthService
	fileStorage     FileStorage
	access          *DocumentAccessService
	lifecycle       *OperationLifecycle
	uiContext       context.Context
	metrics         *observability.Registry
}

type attachmentOutboxStore interface {
	MarkDeletingWithOutbox(attachment models.Attachment) error
	MarkDeletingWithEffects(attachment models.Attachment, effects []models.OutboxEvent) error
	CreateWithOutbox(*models.Attachment, []models.OutboxEvent) error
	MarkDeletingMultipleWithOutbox([]models.Attachment, []models.OutboxEvent) error
	OutboxEnabled() bool
}

type attachmentDeletionScheduler interface {
	EnqueuePendingDeletions() error
	OutboxEnabled() bool
}

type attachmentStoragePathStore interface {
	GetAllStoragePaths() ([]string, error)
}

type objectNameLister interface {
	ListObjectNames(ctx context.Context) ([]string, error)
}

// NewAttachmentService создает новый экземпляр AttachmentService.
func NewAttachmentService(repo AttachmentStore, settingsService *SettingsService, authService *AuthService, fs FileStorage, access *DocumentAccessService) *AttachmentService {
	return &AttachmentService{
		repo:            repo,
		settingsService: settingsService,
		authService:     authService,
		fileStorage:     fs,
		access:          access,
	}
}

func (s *AttachmentService) SetOperationLifecycle(lifecycle *OperationLifecycle) {
	s.lifecycle = lifecycle
}

func (s *AttachmentService) SetOperationMetrics(metrics *observability.Registry) { s.metrics = metrics }

// ReconcileStorage compares database metadata and MinIO without modifying
// either side. It is intentionally available only to administrators.
func (s *AttachmentService) ReconcileStorage() (*models.AttachmentStorageReconciliation, error) {
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
		paths, err := wailsruntime.OpenMultipleFilesDialog(s.uiContext, wailsruntime.OpenDialogOptions{Title: "Выберите файлы для вложения"})
		if err != nil {
			return nil, fmt.Errorf("failed to choose files: %w", err)
		}
		attachments := make([]dto.Attachment, 0, len(paths))
		for _, path := range paths {
			attachment, err := s.uploadPath(documentIDStr, path)
			if err != nil {
				return nil, err
			}
			attachments = append(attachments, *attachment)
		}
		return attachments, nil
	})
}

func (s *AttachmentService) uploadPath(documentIDStr, path string) (*dto.Attachment, error) {
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
	if !canUploadDirectly {
		if !s.settingsService.IsAssignmentCompletionAttachmentsEnabled() {
			return nil, models.NewForbidden("загрузка файлов при завершении поручения отключена в настройках")
		}

		hasAssignmentAccess, err := s.access.HasAssignmentAccess(documentID)
		if err != nil {
			return nil, err
		}
		if !hasAssignmentAccess {
			return nil, models.ErrForbidden
		}
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
	filename := filepath.Base(path)

	// Проверка размера до чтения содержимого.
	maxSize, _ := s.settingsService.GetMaxFileSize() // returns bytes
	if info.Size() > maxSize {
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
	if err := s.fileStorage.UploadFile(ctx, objectName, file, info.Size(), contentType); err != nil {
		return nil, fmt.Errorf("failed to upload file to storage: %v", err)
	}

	// 4. Сохранение в БД
	userID, err := uuid.Parse(currentUser.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid current user ID: %w", err)
	}

	attachment := &models.Attachment{
		DocumentID:  documentID,
		Filename:    filename,
		FileSize:    info.Size(),
		ContentType: contentType,
		StoragePath: objectName,
		UploadedBy:  userID,
	}

	outboxRepo, ok := s.repo.(attachmentOutboxStore)
	if !ok || !outboxRepo.OutboxEnabled() {
		_ = s.fileStorage.DeleteFile(ctx, objectName)
		return nil, fmt.Errorf("attachment store must support atomic outbox operations")
	}
	event, buildErr := NewJournalOutboxEvent("attachment:"+objectName+":upload:journal", models.CreateJournalEntryRequest{DocumentID: documentID, UserID: userID, Action: "FILE_UPLOAD", Details: fmt.Sprintf("Добавлен файл: %s", filename)})
	if buildErr != nil {
		return nil, buildErr
	}
	err = outboxRepo.CreateWithOutbox(attachment, []models.OutboxEvent{event})
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

// GetList — получить вложения документа
func (s *AttachmentService) GetList(documentIDStr string) ([]dto.Attachment, error) {
	return measureOperation(s.metrics, "attachments.get_list", func() ([]dto.Attachment, error) {
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
	outboxRepo, ok := s.repo.(attachmentOutboxStore)
	if !ok || !outboxRepo.OutboxEnabled() {
		return fmt.Errorf("attachment store must support atomic outbox operations")
	}
	currentUserID, _ := s.authService.GetCurrentUserUUID()
	event, buildErr := NewJournalOutboxEvent("attachment:"+attachment.ID.String()+":delete:journal", models.CreateJournalEntryRequest{DocumentID: attachment.DocumentID, UserID: currentUserID, Action: "FILE_DELETE", Details: fmt.Sprintf("Удален файл: %s", attachment.Filename)})
	if buildErr != nil {
		return buildErr
	}
	return outboxRepo.MarkDeletingWithEffects(*attachment, []models.OutboxEvent{event})
}

// DownloadToDisk — сохранить файл в папку «Загрузки» пользователя и вернуть полный путь
func (s *AttachmentService) DownloadToDisk(idStr string) (string, error) {
	return measureOperation(s.metrics, "attachments.download", func() (string, error) {
		ctx, release := serviceOperationContext(s.lifecycle)
		defer release()

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

func writeDownloadFileWithoutOverwrite(downloadDir, filename string, content []byte) (string, error) {
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

		if _, err := file.Write(content); err != nil {
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
	cleanFilename := filepath.Base(strings.TrimSpace(filename))
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
	outboxRepo, ok := s.repo.(attachmentOutboxStore)
	if !ok || !outboxRepo.OutboxEnabled() {
		return 0, fmt.Errorf("attachment store must support atomic outbox operations")
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
	if err := outboxRepo.MarkDeletingMultipleWithOutbox(attachments, []models.OutboxEvent{event}); err != nil {
		return 0, fmt.Errorf("failed to queue attachment deletion: %w", err)
	}
	return len(attachments), nil
}

// ProcessPendingDeletions retries deletion intents left by a failed database or
// storage operation. It is safe to invoke repeatedly: MinIO deletion is
// idempotent and a row is only physically removed after it was marked.
func (s *AttachmentService) ProcessPendingDeletions(ctx context.Context) error {
	if scheduler, ok := s.repo.(attachmentDeletionScheduler); ok && scheduler.OutboxEnabled() {
		return scheduler.EnqueuePendingDeletions()
	}
	attachments, err := s.repo.GetPendingDeletion()
	if err != nil {
		return fmt.Errorf("failed to get pending attachment deletions: %w", err)
	}

	var errs []error
	for _, attachment := range attachments {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(errs, err)...)
		}
		if err := s.finalizeDeletion(ctx, attachment); err != nil {
			errs = append(errs, fmt.Errorf("attachment %s: %w", attachment.ID, err))
		}
	}
	return errors.Join(errs...)
}

func (s *AttachmentService) finalizeDeletion(ctx context.Context, attachment models.Attachment) error {
	if err := s.fileStorage.DeleteFile(ctx, attachment.StoragePath); err != nil {
		return fmt.Errorf("failed to delete file from storage: %w", err)
	}
	if err := s.repo.DeleteMarked(attachment.ID); err != nil {
		return fmt.Errorf("failed to delete attachment record from db: %w", err)
	}
	return nil
}
