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

// Upload handles file upload
func (s *AttachmentService) Upload(documentIDStr string, documentType string, filename string, contentBase64 string) (*models.Attachment, error) {
	currentUser, err := s.authService.GetCurrentUser()
	if err != nil {
		return nil, &models.AppError{Code: 401, Message: "Unauthorized"}
	}

	if !s.authService.HasRole("clerk") && !s.authService.HasRole("admin") {
		return nil, &models.AppError{Code: 403, Message: "Permission denied: only clerks can upload files"}
	}

	documentID, err := uuid.Parse(documentIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid document ID")
	}

	// 1. Decode content
	// Remove data URI prefix if present (e.g. "data:application/pdf;base64,")
	if idx := strings.Index(contentBase64, ","); idx != -1 {
		contentBase64 = contentBase64[idx+1:]
	}

	data, err := base64.StdEncoding.DecodeString(contentBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode file content: %v", err)
	}

	// 2. Validate Size
	maxSize, _ := s.settingsService.GetMaxFileSize() // returns bytes
	if int64(len(data)) > maxSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed size (%d MB)", maxSize/(1024*1024))
	}

	// 3. Validate Type
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

	// 4. Save to DB
	attachment := &models.Attachment{
		DocumentID:   documentID,
		DocumentType: documentType,
		Filename:     filename,
		FileSize:     int64(len(data)),
		ContentType:  ext, // simplified content type
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

// GetList returns attachments for a document
func (s *AttachmentService) GetList(documentIDStr string) ([]models.Attachment, error) {
	documentID, err := uuid.Parse(documentIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid document ID")
	}
	return s.repo.GetByDocumentID(documentID)
}

// Download returns the file content as base64
func (s *AttachmentService) Download(idStr string) (*models.DownloadResponse, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid attachment ID")
	}

	// Get metadata for filename
	attachment, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Get content from DB
	content, err := s.repo.GetContent(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get file content: %v", err)
	}

	return &models.DownloadResponse{
		Filename: attachment.Filename,
		Content:  base64.StdEncoding.EncodeToString(content),
	}, nil
}

// Delete removes attachment
func (s *AttachmentService) Delete(idStr string) error {
	// Check permissions
	if !s.authService.HasRole("clerk") && !s.authService.HasRole("admin") {
		return &models.AppError{Code: 403, Message: "Permission denied: only clerks can delete files"}
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid attachment ID")
	}

	// Remove from DB
	if err := s.repo.Delete(id); err != nil {
		return err
	}

	return nil
}

// DownloadToDisk saves the file to the user's Downloads directory and returns the full path
func (s *AttachmentService) DownloadToDisk(idStr string) (string, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return "", fmt.Errorf("invalid attachment ID")
	}

	// Get metadata
	attachment, err := s.repo.GetByID(id)
	if err != nil {
		return "", err
	}

	// Get content
	content, err := s.repo.GetContent(id)
	if err != nil {
		return "", fmt.Errorf("failed to get file content: %v", err)
	}

	// Determine download path
	// For Linux, typically ~/Downloads
	// We can try to use os/user to find home directory
	// Note: Wails might have a specific way, but standard Go works too.
	currentUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %v", err)
	}

	// Construct path
	// Assuming standardized "Downloads" folder. 
	// For a more robust solution, xdg-user-dir DOWNLOAD could be used, but ~/Downloads is a safe default for now.
	downloadDir := filepath.Join(currentUser.HomeDir, "Downloads")
	
	// Create directory if not exists (unlikely but safe)
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create download directory: %v", err)
	}

	// Clean filename to prevent path traversal or invalid chars
	cleanFilename := filepath.Base(attachment.Filename)
	fullPath := filepath.Join(downloadDir, cleanFilename)

	// Write file
	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %v", err)
	}

	return fullPath, nil
}

// OpenFile opens the file with the default application
func (s *AttachmentService) OpenFile(path string) error {
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

// OpenFolder opens the folder containing the file
func (s *AttachmentService) OpenFolder(path string) error {
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
