package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

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

type AttachmentStoragePathStore interface {
	GetAllStoragePaths() ([]string, error)
}

type ObjectNameLister interface {
	ListObjectNames(ctx context.Context) ([]string, error)
}

type AssignmentAttachmentCreator interface {
	CreateForAssignmentWithOutbox(*models.Attachment, bool, []models.OutboxEvent) error
}

type AssignmentAttachmentStore interface {
	GetByAssignmentID(uuid.UUID) ([]models.Attachment, error)
}
