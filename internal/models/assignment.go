package models

import (
	"time"

	"github.com/google/uuid"
)

// Assignment — поручение
type Assignment struct {
	ID                 uuid.UUID  `json:"id"`
	IncomingDocumentID uuid.UUID  `json:"incomingDocumentId"`
	DocumentNumber     string     `json:"documentNumber,omitempty"`  // join
	DocumentSubject    string     `json:"documentSubject,omitempty"` // join
	Description        string     `json:"description"`
	ExecutorID         uuid.UUID  `json:"executorId"`
	ExecutorName       string     `json:"executorName,omitempty"` // join
	DueDate            *time.Time `json:"dueDate,omitempty"`
	Status             string     `json:"status"` // new, in_progress, completed, overdue
	CompletedAt        *time.Time `json:"completedAt,omitempty"`
	CreatedBy          uuid.UUID  `json:"createdBy"`
	CreatedByName      string     `json:"createdByName,omitempty"` // join
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`

	// Соисполнители
	CoExecutors []CoExecutor `json:"coExecutors,omitempty"`
}

// CoExecutor — соисполнитель
type CoExecutor struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"userId"`
	UserName  string    `json:"userName,omitempty"` // join
	CreatedAt time.Time `json:"createdAt"`
}

// Attachment — вложение
type Attachment struct {
	ID           uuid.UUID `json:"id"`
	DocumentType string    `json:"documentType"` // incoming, outgoing
	DocumentID   uuid.UUID `json:"documentId"`
	Filename     string    `json:"filename"`
	MimeType     string    `json:"mimeType"`
	FileSize     int64     `json:"fileSize"`
	FileData     []byte    `json:"-"` // не отдаём в JSON
	UploadedBy   uuid.UUID `json:"uploadedBy"`
	UploadedAt   time.Time `json:"uploadedAt"`
}

// AssignmentFilter — фильтры для журнала поручений
type AssignmentFilter struct {
	ExecutorID  *uuid.UUID `json:"executorId,omitempty"`
	Status      string     `json:"status,omitempty"`
	DueDateFrom *time.Time `json:"dueDateFrom,omitempty"`
	DueDateTo   *time.Time `json:"dueDateTo,omitempty"`
	Page        int        `json:"page"`
	PageSize    int        `json:"pageSize"`
}

// CreateAssignmentRequest — создание поручения
type CreateAssignmentRequest struct {
	IncomingDocumentID uuid.UUID   `json:"incomingDocumentId"`
	Description        string      `json:"description"`
	ExecutorID         uuid.UUID   `json:"executorId"`
	CoExecutorIDs      []uuid.UUID `json:"coExecutorIds,omitempty"`
	DueDate            *time.Time  `json:"dueDate,omitempty"`
}
