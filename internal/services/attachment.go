package services

import (
	"context"
	"docflow/internal/models"
	"docflow/internal/repository"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/uuid"
)

type AttachmentService struct {
	ctx             context.Context
	repo            *repository.AttachmentRepository
	settingsService *SettingsService
	authService     *AuthService
}

func NewAttachmentService(repo *repository.AttachmentRepository, settingsService *SettingsService, authService *AuthService) *AttachmentService {
	return &AttachmentService{
		repo:            repo,
		settingsService: settingsService,
		authService:     authService,
	}
}

func (s *AttachmentService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// Upload — загрузка файла
func (s *AttachmentService) Upload(documentIDStr string, documentType string, filename string, contentBase64 string) (*models.Attachment, error) {
	currentUser, err := s.authService.GetCurrentUser()
	if err != nil {
		return nil, &models.AppError{Code: 401, Message: "Требуется авторизация"}
	}

	if !s.authService.HasRole("clerk") && !s.authService.HasRole("admin") {
		return nil, &models.AppError{Code: 403, Message: "Недостаточно прав: загружать файлы могут только делопроизводители"}
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

	// 4. Сохранение в БД
	attachment := &models.Attachment{
		DocumentID:   documentID,
		DocumentType: documentType,
		Filename:     filename,
		FileSize:     int64(len(data)),
		ContentType:  ext, // упрощённый тип содержимого
		Content:      data,
		UploadedBy:   currentUser.ID,
	}

	if err := s.repo.Create(attachment); err != nil {
		return nil, err
	}
	attachment.FillIDStr()
	attachment.UploadedByName = currentUser.FullName

	return attachment, nil
}

// GetList — получить вложения документа
func (s *AttachmentService) GetList(documentIDStr string) ([]models.Attachment, error) {
	if !s.authService.IsAuthenticated() {
		return nil, &models.AppError{Code: 401, Message: "Требуется авторизация"}
	}

	documentID, err := uuid.Parse(documentIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid document ID")
	}
	return s.repo.GetByDocumentID(documentID)
}

// Download — получить содержимое файла в формате base64
func (s *AttachmentService) Download(idStr string) (*models.DownloadResponse, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid attachment ID")
	}

	// Получение метаданных файла
	attachment, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Получение содержимого из БД
	content, err := s.repo.GetContent(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get file content: %v", err)
	}

	return &models.DownloadResponse{
		Filename: attachment.Filename,
		Content:  base64.StdEncoding.EncodeToString(content),
	}, nil
}

// Delete — удалить вложение
func (s *AttachmentService) Delete(idStr string) error {
	// Проверка прав доступа
	if !s.authService.HasRole("clerk") && !s.authService.HasRole("admin") {
		return &models.AppError{Code: 403, Message: "Недостаточно прав: удалять файлы могут только делопроизводители"}
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid attachment ID")
	}

	// Удаление из БД
	if err := s.repo.Delete(id); err != nil {
		return err
	}

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
	content, err := s.repo.GetContent(id)
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

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
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

	dir := filepath.Dir(path)
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
