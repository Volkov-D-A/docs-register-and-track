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
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/Volkov-D-A/docs-register-and-track/internal/attachmentname"
	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/serverclient"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/operations"
)

// AttachmentService is the desktop HTTP adapter and the attachment Wails API.
type AttachmentService struct {
	server          serverclient.AttachmentClient
	lifecycle       *operations.Lifecycle
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
	Lifecycle       *operations.Lifecycle
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
	parent, release := s.lifecycle.OperationContext()
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	return ctx, func() { cancel(); release() }
}

func (s *AttachmentService) Upload(documentIDStr string) (*dto.AttachmentUploadResult, error) {
	return operations.Measure(s.metrics, "attachments.upload", func() (*dto.AttachmentUploadResult, error) {
		return s.uploadSelectedFiles(documentIDStr, "", "Выберите файлы для вложения")
	})
}

func (s *AttachmentService) UploadForAssignment(assignmentIDStr string) (*dto.AttachmentUploadResult, error) {
	return operations.Measure(s.metrics, "attachments.upload.assignment", func() (*dto.AttachmentUploadResult, error) {
		assignmentID, err := uuid.Parse(assignmentIDStr)
		if err != nil {
			return nil, models.NewBadRequestWrapped("неверный ID поручения", err)
		}
		return s.uploadSelectedFiles("", assignmentID.String(), "Выберите файлы для отчёта об исполнении")
	})
}

func (s *AttachmentService) uploadSelectedFiles(documentID, assignmentID, title string) (*dto.AttachmentUploadResult, error) {
	uiContext := s.pickerContext()
	if uiContext == nil {
		return nil, fmt.Errorf("file picker is not initialized")
	}
	paths, err := s.openFilesDialog(uiContext, wailsruntime.OpenDialogOptions{Title: title})
	if err != nil {
		return nil, fmt.Errorf("failed to choose files: %w", err)
	}
	result := &dto.AttachmentUploadResult{Items: make([]dto.AttachmentUploadItem, 0, len(paths))}
	for _, path := range paths {
		attachment, uploadErr := operations.Measure(s.metrics, "attachments.upload.file", func() (*dto.Attachment, error) {
			return s.uploadSelectedPath(documentID, assignmentID, path)
		})
		item := dto.AttachmentUploadItem{Filename: filepath.Base(path), Attachment: attachment}
		if uploadErr != nil {
			item.Attachment = nil
			item.Error = attachmentUploadError(uploadErr)
		}
		result.Items = append(result.Items, item)
	}
	// Returning a Wails error would discard the successful files in this result.
	return result, nil
}

func attachmentUploadError(err error) *dto.AttachmentUploadError {
	result := &dto.AttachmentUploadError{Code: "INTERNAL_ERROR", Message: "Не удалось загрузить файл.", Status: 500, RequestID: models.ErrorRequestID(err)}
	if appErr, ok := models.AsAppError(err); ok {
		result.Code, result.Status = appErr.SafeKind(), appErr.StatusCode()
		if message, public := models.PublicErrorMessage(appErr); public {
			result.Message = message
		}
	}
	return result
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
	ctx, release := s.lifecycle.OperationContext()
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
	return operations.Measure(s.metrics, "attachments.get_list", func() ([]dto.Attachment, error) {
		ctx, cancel := s.shortOperationContext()
		defer cancel()
		return s.server.ListDocumentAttachments(ctx, documentIDStr)
	})
}

func (s *AttachmentService) Delete(idStr string) error {
	ctx, release := s.lifecycle.OperationContext()
	defer release()
	return s.server.DeleteAttachment(ctx, idStr)
}

func (s *AttachmentService) BulkDeleteOlderThan(dateStr string) (int, error) {
	ctx, cancel := s.shortOperationContext()
	defer cancel()
	return s.server.BulkDeleteAttachments(ctx, dateStr)
}

func (s *AttachmentService) DownloadToDisk(idStr string) (string, error) {
	return operations.Measure(s.metrics, "attachments.download", func() (string, error) {
		ctx, release := s.lifecycle.OperationContext()
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
	cleanFilename := attachmentname.Normalize(filename)
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

// validatePathInDownloads returns the canonical path that may be passed to the OS.
func (s *AttachmentService) validatePathInDownloads(path string) (string, error) {
	downloadDir, err := s.getDownloadDir()
	if err != nil {
		return "", err
	}
	absDownloadDir, err := filepath.Abs(downloadDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve download directory: %w", err)
	}
	root, err := filepath.EvalSymlinks(absDownloadDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve download directory: %w", err)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", models.NewBadRequestWrapped("неверный путь к файлу", err)
	}
	resolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// Allow a missing final filename so OpenFolder still works after deletion.
		// Lstat distinguishes it from a dangling symlink. Never fall back to an
		// unchecked path when a parent is missing, inaccessible or a symlink loop.
		if _, statErr := os.Lstat(absPath); !errors.Is(statErr, os.ErrNotExist) {
			return "", models.NewForbidden("доступ разрешен только к файлам в папке загрузок")
		}
		parent, parentErr := filepath.EvalSymlinks(filepath.Dir(absPath))
		if parentErr != nil {
			return "", models.NewForbidden("доступ разрешен только к файлам в папке загрузок")
		}
		resolved = filepath.Join(parent, filepath.Base(absPath))
	}
	rel, err := filepath.Rel(root, resolved)
	if err != nil || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", models.NewForbidden("доступ разрешен только к файлам в папке загрузок")
	}
	return resolved, nil
}

func (s *AttachmentService) OpenFile(path string) error {
	cleanPath, err := s.validatePathInDownloads(path)
	if err != nil {
		return err
	}
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
	resolved, err := s.validatePathInDownloads(path)
	if err != nil {
		return err
	}
	// A path naming the download root must not open its parent outside the root.
	dir, err := s.validatePathInDownloads(filepath.Dir(resolved))
	if err != nil {
		return err
	}
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

// Interface dependencies must also reject a typed nil pointer.
func attachmentDependencyMissing(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
