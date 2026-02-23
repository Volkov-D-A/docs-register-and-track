package models

import (
	"time"

	"github.com/google/uuid"
)

type Attachment struct {
	ID             uuid.UUID `json:"-"`
	IDStr          string    `json:"id"`
	DocumentID     uuid.UUID `json:"-"`
	DocumentIDStr  string    `json:"documentId"`
	DocumentType   string    `json:"documentType"` // 'incoming' или 'outgoing'
	Filename       string    `json:"filename"`
	Filepath       string    `json:"filepath"` // внутренний путь
	FileSize       int64     `json:"fileSize"`
	ContentType    string    `json:"contentType"`
	Content        []byte    `json:"-"` // Хранится в БД, не в JSON
	UploadedBy     uuid.UUID `json:"-"`
	UploadedByStr  string    `json:"uploadedBy"`
	UploadedByName string    `json:"uploadedByName,omitempty"` // заполняется при получении
	UploadedAt     time.Time `json:"uploadedAt"`
}

func (a *Attachment) FillIDStr() {
	a.IDStr = a.ID.String()
	a.DocumentIDStr = a.DocumentID.String()
	a.UploadedByStr = a.UploadedBy.String()
}

type DownloadResponse struct {
	Filename string `json:"filename"`
	Content  string `json:"content"` // в формате Base64
}
