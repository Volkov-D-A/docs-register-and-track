package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

// AttachmentService is the desktop HTTP adapter and the attachment Wails API.
type AttachmentService struct {
	server          serverclient.AttachmentClient
	lifecycle       *OperationLifecycle
	metrics         *observability.Registry
	uiMu            sync.RWMutex
	uiContext       context.Context
	openFilesDialog func(context.Context, wailsruntime.OpenDialogOptions) ([]string, error)
	downloadDir     func() (string, error)
	launch          func(string, ...string) error
}

// DesktopAttachmentOptions supplies lifecycle and optional native OS adapters.
// Nil adapters use the native implementation; metrics and lifecycle are optional.
type DesktopAttachmentOptions struct {
	Lifecycle       *OperationLifecycle
	Metrics         *observability.Registry
	OpenFilesDialog func(context.Context, wailsruntime.OpenDialogOptions) ([]string, error)
	DownloadDir     func() (string, error)
	Launch          func(string, ...string) error
}

// NewDesktopAttachmentService returns the service and its private startup callback.
func NewDesktopAttachmentService(client serverclient.AttachmentClient, options DesktopAttachmentOptions) (*AttachmentService, func(context.Context), error) {
	if attachmentDependencyMissing(client) {
		return nil, nil, fmt.Errorf("attachment HTTP client is required")
	}
	s := &AttachmentService{
		server:          client,
		lifecycle:       options.Lifecycle,
		metrics:         options.Metrics,
		openFilesDialog: options.OpenFilesDialog,
		downloadDir:     options.DownloadDir,
		launch:          options.Launch,
	}
	if s.openFilesDialog == nil {
		s.openFilesDialog = wailsruntime.OpenMultipleFilesDialog
	}
	if s.downloadDir == nil {
		s.downloadDir = defaultDownloadDir
	}
	if s.launch == nil {
		s.launch = func(name string, args ...string) error { return exec.Command(name, args...).Start() }
	}
	startup := func(ctx context.Context) {
		s.uiMu.Lock()
		defer s.uiMu.Unlock()
		s.uiContext = ctx
	}
	return s, startup, nil
}

func (s *AttachmentService) pickerContext() context.Context {
	s.uiMu.RLock()
	defer s.uiMu.RUnlock()
	return s.uiContext
}

func (s *AttachmentService) shortOperationContext() (context.Context, func()) {
	parent, release := serviceOperationContext(s.lifecycle)
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	return ctx, func() { cancel(); release() }
}

func (s *AttachmentService) Upload(documentIDStr string) ([]dto.Attachment, error) {
	return measureOperation(s.metrics, "attachments.upload", func() ([]dto.Attachment, error) {
		uiContext := s.pickerContext()
		if uiContext == nil {
			return nil, fmt.Errorf("file picker is not initialized")
		}
		paths, err := s.openFilesDialog(uiContext, wailsruntime.OpenDialogOptions{Title: "Выберите файлы для вложения"})
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

func (s *AttachmentService) UploadForAssignment(assignmentIDStr string) ([]dto.Attachment, error) {
	return measureOperation(s.metrics, "attachments.upload.assignment", func() ([]dto.Attachment, error) {
		uiContext := s.pickerContext()
		if uiContext == nil {
			return nil, fmt.Errorf("file picker is not initialized")
		}
		assignmentID, err := uuid.Parse(assignmentIDStr)
		if err != nil {
			return nil, models.NewBadRequestWrapped("неверный ID поручения", err)
		}
		paths, err := s.openFilesDialog(uiContext, wailsruntime.OpenDialogOptions{Title: "Выберите файлы для отчёта об исполнении"})
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

func (s *AttachmentService) ReconcileStorage() (*models.AttachmentStorageReconciliation, error) {
	ctx, cancel := s.shortOperationContext()
	defer cancel()
	return s.server.ReconcileAttachmentStorage(ctx)
}

func (s *AttachmentService) GetAssignmentFiles(assignmentIDStr string) ([]dto.Attachment, error) {
	ctx, cancel := s.shortOperationContext()
	defer cancel()
	return s.server.ListAssignmentAttachments(ctx, assignmentIDStr)
}

func (s *AttachmentService) GetList(documentIDStr string) ([]dto.Attachment, error) {
	return measureOperation(s.metrics, "attachments.get_list", func() ([]dto.Attachment, error) {
		ctx, cancel := s.shortOperationContext()
		defer cancel()
		return s.server.ListDocumentAttachments(ctx, documentIDStr)
	})
}

func (s *AttachmentService) Delete(idStr string) error {
	ctx, release := serviceOperationContext(s.lifecycle)
	defer release()
	return s.server.DeleteAttachment(ctx, idStr)
}

func (s *AttachmentService) BulkDeleteOlderThan(dateStr string) (int, error) {
	ctx, cancel := s.shortOperationContext()
	defer cancel()
	return s.server.BulkDeleteAttachments(ctx, dateStr)
}

func (s *AttachmentService) DownloadToDisk(idStr string) (string, error) {
	return measureOperation(s.metrics, "attachments.download", func() (string, error) {
		ctx, release := serviceOperationContext(s.lifecycle)
		defer release()
		attachment, content, err := s.server.GetAttachmentContent(ctx, idStr)
		if err != nil {
			return "", err
		}
		defer content.Close()
		downloadDir, err := s.getDownloadDir()
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(downloadDir, 0755); err != nil {
			return "", err
		}
		return writeDownloadFileFromStorage(downloadDir, attachment.Filename, func(file *os.File) error {
			_, copyErr := io.Copy(file, content)
			return copyErr
		})
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

func defaultDownloadDir() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %v", err)
	}
	return filepath.Join(currentUser.HomeDir, "Downloads"), nil
}

func (s *AttachmentService) getDownloadDir() (string, error) { return s.downloadDir() }

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

func (s *AttachmentService) OpenFile(path string) error {
	if err := s.validatePathInDownloads(path); err != nil {
		return err
	}

	cleanPath := filepath.Clean(path)
	var name string
	var args []string

	switch runtime.GOOS {
	case "windows":
		name, args = "rundll32", []string{"url.dll,FileProtocolHandler", cleanPath}
	case "darwin":
		name, args = "open", []string{cleanPath}
	default:
		name, args = "xdg-open", []string{cleanPath}
	}

	if err := s.launch(name, args...); err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	return nil
}

func (s *AttachmentService) OpenFolder(path string) error {
	if err := s.validatePathInDownloads(path); err != nil {
		return err
	}

	dir := filepath.Clean(filepath.Dir(path))
	var name string
	var args []string

	switch runtime.GOOS {
	case "windows":
		name, args = "explorer", []string{dir}
	case "darwin":
		name, args = "open", []string{dir}
	default:
		name, args = "xdg-open", []string{dir}
	}

	if err := s.launch(name, args...); err != nil {
		return fmt.Errorf("failed to open folder: %v", err)
	}
	return nil
}
