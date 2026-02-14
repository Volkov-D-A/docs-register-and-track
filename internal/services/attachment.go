package services

import (
	"context"
	"docflow/internal/models"
	"docflow/internal/repository"
	"encoding/base64"
	"fmt"
	"path/filepath"
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

// OpenFile opens the file in default viewer (optional feature)
func (s *AttachmentService) OpenFile(idStr string) error {
	// runtime.BrowserOpenURL could be used if we served files via HTTP
	// For local app, we might just return path to frontend or let frontend handle "download"
	return nil
}
