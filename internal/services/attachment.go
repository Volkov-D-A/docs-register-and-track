package services

import (
	"context"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AttachmentService предоставляет бизнес-логику для работы с вложениями (файлами) документов.
type AttachmentService struct {
	repo            AttachmentStore
	settingsService *SettingsService
	authService     *AuthService
	journal         *JournalService
	auditService    *AdminAuditLogService
	fileStorage     FileStorage
}

// NewAttachmentService создает новый экземпляр AttachmentService.
func NewAttachmentService(repo AttachmentStore, settingsService *SettingsService, authService *AuthService, journal *JournalService, auditService *AdminAuditLogService, fs FileStorage) *AttachmentService {
	return &AttachmentService{
		repo:            repo,
		settingsService: settingsService,
		authService:     authService,
		journal:         journal,
		auditService:    auditService,
		fileStorage:     fs,
	}
}

// Upload — загрузка файла
func (s *AttachmentService) Upload(documentIDStr string, documentType string, filename string, contentBase64 string) (*dto.Attachment, error) {
	currentUser, err := s.authService.GetCurrentUser()
	if err != nil {
		return nil, models.ErrUnauthorized
	}

	if !s.authService.HasRole("clerk") && !s.authService.HasRole("admin") {
		return nil, models.NewForbidden("Недостаточно прав: загружать файлы могут только делопроизводители")
	}

	documentID, err := uuid.Parse(documentIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid document ID")
	}

	// 1. Декодирование содержимого
	// Удаление префикса data URI, если присутствует (например, "data:application/pdf;base64,")
	if idx := strings.Index(contentBase64, ","); idx != -1 {
		contentBase64 = contentBase64[idx+1:]
	}

	data, err := base64.StdEncoding.DecodeString(contentBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode file content: %v", err)
	}

	// 2. Проверка размера
	maxSize, _ := s.settingsService.GetMaxFileSize() // returns bytes
	if int64(len(data)) > maxSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed size (%d MB)", maxSize/(1024*1024))
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
		return nil, fmt.Errorf("file type '%s' is not allowed", ext)
	}

	objectName := uuid.New().String() + ext
	if err := s.fileStorage.UploadFile(context.Background(), objectName, data, ext); err != nil {
		return nil, fmt.Errorf("failed to upload file to storage: %v", err)
	}

	// 4. Сохранение в БД
	attachment := &models.Attachment{
		DocumentID:   documentID,
		DocumentType: documentType,
		Filename:     filename,
		FileSize:     int64(len(data)),
		ContentType:  ext, // упрощённый тип содержимого
		StoragePath:  objectName,
		UploadedBy:   uuid.MustParse(currentUser.ID),
	}

	if err := s.repo.Create(attachment); err != nil {
		// Попытка откатить (удалить) файл из хранилища, если сохранение в БД не удалось
		_ = s.fileStorage.DeleteFile(context.Background(), objectName)
		return nil, err
	}

	s.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
		DocumentID:   documentID,
		DocumentType: documentType,
		UserID:       uuid.MustParse(currentUser.ID),
		Action:       "FILE_UPLOAD",
		Details:      fmt.Sprintf("Добавлен файл: %s", filename),
	})

	attachment.UploadedByName = currentUser.FullName

	return dto.MapAttachment(attachment), nil
}

// GetList — получить вложения документа
func (s *AttachmentService) GetList(documentIDStr string) ([]dto.Attachment, error) {
	if !s.authService.IsAuthenticated() {
		return nil, models.ErrUnauthorized
	}

	documentID, err := uuid.Parse(documentIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid document ID")
	}
	res, err := s.repo.GetByDocumentID(documentID)
	return dto.MapAttachments(res), err
}

// Download — получить содержимое файла в формате base64
func (s *AttachmentService) Download(idStr string) (*dto.DownloadResponse, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid attachment ID")
	}

	// Получение метаданных файла
	attachment, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Получение содержимого из файлового хранилища
	content, err := s.fileStorage.DownloadFile(context.Background(), attachment.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file content: %v", err)
	}

	return &dto.DownloadResponse{
		Filename: attachment.Filename,
		Content:  base64.StdEncoding.EncodeToString(content),
	}, nil
}

// Delete — удалить вложение
func (s *AttachmentService) Delete(idStr string) error {
	// Проверка прав доступа
	if !s.authService.HasRole("clerk") && !s.authService.HasRole("admin") {
		return models.NewForbidden("Недостаточно прав: удалять файлы могут только делопроизводители")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid attachment ID")
	}

	// Получение вложения для журналирования
	attachment, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	// Удаление из файлового хранилища
	if err := s.fileStorage.DeleteFile(context.Background(), attachment.StoragePath); err != nil {
		return fmt.Errorf("failed to delete file from storage: %v", err)
	}

	// Удаление из БД
	if err := s.repo.Delete(id); err != nil {
		return err
	}

	currentUserID, _ := uuid.Parse(s.authService.GetCurrentUserID())
	s.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
		DocumentID:   attachment.DocumentID,
		DocumentType: attachment.DocumentType,
		UserID:       currentUserID,
		Action:       "FILE_DELETE",
		Details:      fmt.Sprintf("Удален файл: %s", attachment.Filename),
	})

	return nil
}

// DownloadToDisk — сохранить файл в папку «Загрузки» пользователя и вернуть полный путь
func (s *AttachmentService) DownloadToDisk(idStr string) (string, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return "", fmt.Errorf("invalid attachment ID")
	}

	// Получение метаданных
	attachment, err := s.repo.GetByID(id)
	if err != nil {
		return "", err
	}

	// Получение содержимого
	content, err := s.fileStorage.DownloadFile(context.Background(), attachment.StoragePath)
	if err != nil {
		return "", fmt.Errorf("failed to get file content: %v", err)
	}

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

	// Очистка имени файла для предотвращения обхода пути
	cleanFilename := filepath.Base(attachment.Filename)
	fullPath := filepath.Join(downloadDir, cleanFilename)

	// Запись файла
	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %v", err)
	}

	return fullPath, nil
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
		return fmt.Errorf("invalid path: %v", err)
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
		return fmt.Errorf("access denied: path is outside the download directory")
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
	// Проверка прав доступа
	if !s.authService.HasRole("admin") {
		return 0, models.NewForbidden("Недостаточно прав: массовое удаление файлов доступно только администраторам")
	}

	date, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return 0, fmt.Errorf("invalid date format, expected RFC3339: %v", err)
	}

	attachments, err := s.repo.GetOlderThan(date)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch old attachments: %v", err)
	}

	if len(attachments) == 0 {
		return 0, nil
	}

	var successfulIDs []uuid.UUID
	deletedCount := 0

	for _, att := range attachments {
		if err := s.fileStorage.DeleteFile(context.Background(), att.StoragePath); err != nil {
			// Логируем ошибку, но продолжаем удаление других файлов
			fmt.Printf("Failed to delete file %s from MinIO: %v\n", att.StoragePath, err)
			continue
		}
		successfulIDs = append(successfulIDs, att.ID)
		deletedCount++
	}

	if len(successfulIDs) > 0 {
		if err := s.repo.DeleteMultiple(successfulIDs); err != nil {
			return 0, fmt.Errorf("failed to delete records from db: %v", err)
		}
	}

	currentUserID, _ := uuid.Parse(s.authService.GetCurrentUserID())
	var currentUserName string
	if u, err := s.authService.GetCurrentUser(); err == nil {
		currentUserName = u.FullName
	}
	s.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
		DocumentID:   uuid.Nil, // Массовая операция
		DocumentType: "system",
		UserID:       currentUserID,
		Action:       "FILE_BULK_DELETE",
		Details:      fmt.Sprintf("Инструментом массовой очистки удалено %d файлов, загруженных до %s", deletedCount, date.Format("02.01.2006")),
	})

	details := fmt.Sprintf("Массовое удаление файлов: удалено %d, загруженных до %s", deletedCount, date.Format("02.01.2006"))
	s.auditService.LogAction(currentUserID, currentUserName, "FILES_BULK_DELETE", details)

	return deletedCount, nil
}
