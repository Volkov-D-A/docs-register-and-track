package models

import (
	"time"

	"github.com/google/uuid"
)

// Assignment — поручение по документу
type Assignment struct {
	ID           uuid.UUID `json:"-"`
	DocumentID   uuid.UUID `json:"-"`
	DocumentType string    `json:"documentType"` // 'incoming' или 'outgoing'

	ExecutorID   uuid.UUID `json:"-"`
	ExecutorName string    `json:"executorName,omitempty"`

	Content     string     `json:"content"`
	Deadline    *time.Time `json:"deadline,omitempty"`
	Status      string     `json:"status"` // new, in_progress, completed, cancelled, returned, finished
	Report      string     `json:"report,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	DocumentNumber  string `json:"documentNumber,omitempty"`
	DocumentSubject string `json:"documentSubject,omitempty"`

	// Соисполнители
	CoExecutors   []User   `json:"coExecutors,omitempty"`
	CoExecutorIDs []string `json:"coExecutorIds,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type AssignmentFilter struct {
	Search       string `json:"search,omitempty"`
	DocumentID   string `json:"documentId,omitempty"`
	ExecutorID   string `json:"executorId,omitempty"`
	Status       string `json:"status,omitempty"`
	DateFrom     string `json:"dateFrom,omitempty"`
	DateTo       string `json:"dateTo,omitempty"`
	OverdueOnly  bool   `json:"overdueOnly"` // Фильтр просроченных
	ShowFinished bool   `json:"showFinished"`
	Page         int    `json:"page"`
	PageSize     int    `json:"pageSize"`
}
