package models

import (
	"time"

	"github.com/google/uuid"
)

// Attachment представляет собой прикрепленный к документу файл.
type Attachment struct {
	ID             uuid.UUID `json:"-"`
	DocumentID     uuid.UUID `json:"-"`
	Filename       string    `json:"filename"`
	Filepath       string    `json:"filepath"` // внутренний путь
	FileSize       int64     `json:"fileSize"`
	ContentType    string    `json:"contentType"`
	StoragePath    string    `json:"-"` // Путь к файлу в MinIO
	UploadedBy     uuid.UUID `json:"-"`
	UploadedByName string    `json:"uploadedByName,omitempty"` // заполняется при получении
	UploadedAt     time.Time `json:"uploadedAt"`
}
